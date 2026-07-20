package feed

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const maxFeedBytes = 2 << 20

type Item struct {
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
	URL       string    `json:"url"`
	Published time.Time `json:"published"`
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	ID        string     `xml:"id"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Links     []atomLink `xml:"link"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type Client struct {
	HTTPClient *http.Client
}

func NewClient() *Client {
	return &Client{HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Fetch(ctx context.Context, sourceURL string, count, maxTitleRunes int, prefix string) ([]Item, error) {
	data, contentType, err := c.fetchPage(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	if strings.Contains(contentType, "text/html") {
		return c.fetchHTMLPages(ctx, data, sourceURL, count, maxTitleRunes, prefix)
	}
	return parseAtom(data, count, maxTitleRunes, prefix)
}

func (c *Client) fetchPage(ctx context.Context, sourceURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "text/html, application/atom+xml;q=0.9, application/xml;q=0.8")
	req.Header.Set("User-Agent", "claude-geeknews-spinner/1.0 (+https://github.com/saz/claude-geeknews-spinner)")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch GeekNews source: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch GeekNews source: HTTP %s", resp.Status)
	}

	limited := io.LimitReader(resp.Body, maxFeedBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("read GeekNews source: %w", err)
	}
	if len(data) > maxFeedBytes {
		return nil, "", errors.New("GeekNews response exceeds 2 MiB")
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func parseAtom(data []byte, count, maxTitleRunes int, prefix string) ([]Item, error) {
	var parsed atomFeed
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse GeekNews Atom feed: %w", err)
	}
	items := make([]Item, 0, len(parsed.Entries))
	seen := make(map[string]struct{}, len(parsed.Entries))
	for _, entry := range parsed.Entries {
		title := CleanTitle(entry.Title, maxTitleRunes)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		summary := CleanTitle(entry.Summary, maxTitleRunes)
		if summary == "" {
			summary = CleanTitle(entry.Content, maxTitleRunes)
		}
		published := parseTime(entry.Published)
		if published.IsZero() {
			published = parseTime(entry.Updated)
		}
		items = append(items, Item{
			Title:     prefix + title,
			Summary:   summary,
			URL:       alternateURL(entry),
			Published: published,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Published.After(items[j].Published)
	})
	if len(items) > count {
		items = items[:count]
	}
	if len(items) == 0 {
		return nil, errors.New("GeekNews feed contained no usable entries")
	}
	return items, nil
}

func parseHTML(data []byte, sourceURL string, count, maxTitleRunes int, prefix string) ([]Item, error) {
	items, _, err := parseHTMLPage(data, sourceURL, maxTitleRunes, prefix)
	if len(items) > count {
		items = items[:count]
	}
	return items, err
}

func (c *Client) fetchHTMLPages(ctx context.Context, firstPage []byte, sourceURL string, count, maxTitleRunes int, prefix string) ([]Item, error) {
	items := make([]Item, 0, count)
	seen := make(map[string]struct{}, count)
	data := firstPage
	pageURL := sourceURL
	for page := 0; page < 5 && pageURL != "" && len(items) < count; page++ {
		pageItems, nextURL, err := parseHTMLPage(data, pageURL, maxTitleRunes, prefix)
		if err != nil {
			if len(items) > 0 {
				break
			}
			return nil, err
		}
		for _, item := range pageItems {
			if _, exists := seen[item.Title]; exists {
				continue
			}
			seen[item.Title] = struct{}{}
			items = append(items, item)
			if len(items) == count {
				break
			}
		}
		pageURL = nextURL
		if pageURL != "" && len(items) < count {
			var contentType string
			data, contentType, err = c.fetchPage(ctx, pageURL)
			if err != nil || !strings.Contains(contentType, "text/html") {
				break
			}
		}
	}
	if len(items) == 0 {
		return nil, errors.New("GeekNews latest page contained no usable topics")
	}
	return items, nil
}

func parseHTMLPage(data []byte, sourceURL string, maxTitleRunes int, prefix string) ([]Item, string, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("parse GeekNews HTML: %w", err)
	}
	base, err := url.Parse(sourceURL)
	if err != nil {
		return nil, "", err
	}
	items := make([]Item, 0, 20)
	seen := make(map[string]struct{}, 20)
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "topic_row") {
			item := itemFromTopicNode(node, base, maxTitleRunes, prefix)
			if item.Title != "" {
				if _, exists := seen[item.Title]; !exists {
					seen[item.Title] = struct{}{}
					items = append(items, item)
				}
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	if len(items) == 0 {
		return nil, "", errors.New("GeekNews latest page contained no usable topics")
	}
	return items, findNextPage(doc, base), nil
}

func findNextPage(node *html.Node, base *url.URL) string {
	if node.Type == html.ElementNode && node.Data == "div" && hasClass(node, "next") {
		link := findElement(node, "a", "")
		if link != nil {
			href, err := url.Parse(attribute(link, "href"))
			if err == nil && href.String() != "" {
				return base.ResolveReference(href).String()
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findNextPage(child, base); found != "" {
			return found
		}
	}
	return ""
}

func itemFromTopicNode(node *html.Node, base *url.URL, maxTitleRunes int, prefix string) Item {
	id := attribute(node, "data-topic-state-id")
	titleNode := findElement(node, "h2", "topic-title-heading")
	if titleNode == nil {
		return Item{}
	}
	title := CleanTitle(textContent(titleNode), maxTitleRunes)
	if title == "" {
		return Item{}
	}
	timeNode := findElement(node, "time", "")
	published := time.Time{}
	if timeNode != nil {
		published = parseTime(attribute(timeNode, "datetime"))
	}
	topicURL := ""
	if id != "" {
		relative, _ := url.Parse("/topic?id=" + url.QueryEscape(id))
		topicURL = base.ResolveReference(relative).String()
	}
	summary := ""
	if summaryNode := findElement(node, "div", "topicdesc"); summaryNode != nil {
		summary = CleanTitle(textContent(summaryNode), maxTitleRunes)
	}
	return Item{Title: prefix + title, Summary: summary, URL: topicURL, Published: published}
}

func findElement(node *html.Node, tag, class string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tag && (class == "" || hasClass(node, class)) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag, class); found != nil {
			return found
		}
	}
	return nil
}

func hasClass(node *html.Node, expected string) bool {
	for _, class := range strings.Fields(attribute(node, "class")) {
		if class == expected {
			return true
		}
	}
	return false
}

func attribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func textContent(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func CleanTitle(value string, maxRunes int) string {
	value = strings.Map(func(r rune) rune {
		if r == 0x00B7 || r == 0x318D {
			return '-'
		}
		if unicode.IsControl(r) || isBidiControl(r) {
			return -1
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if maxRunes <= 0 || utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maxRunes-1])) + "..."
}

func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069)
}

func alternateURL(entry atomEntry) string {
	for _, link := range entry.Links {
		if link.Rel == "alternate" && link.Href != "" {
			return link.Href
		}
	}
	return entry.ID
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}
