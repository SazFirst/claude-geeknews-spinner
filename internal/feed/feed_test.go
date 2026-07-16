package feed

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestParseHTMLReturnsLatestRequestedTopics(t *testing.T) {
	data, err := os.ReadFile("testdata/latest.html")
	if err != nil {
		t.Fatal(err)
	}
	items, err := parseHTML(data, "https://news.hada.io/new", 2, 100, "[GN] ")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Title != "[GN] 첫 번째 새 소식" {
		t.Fatalf("first title = %q", items[0].Title)
	}
	if items[1].Title != "[GN] 두 번째 & 중요한 소식" {
		t.Fatalf("second title = %q", items[1].Title)
	}
	if items[0].URL != "https://news.hada.io/topic?id=101" {
		t.Fatalf("first URL = %q", items[0].URL)
	}
	_, nextURL, err := parseHTMLPage(data, "https://news.hada.io/new", 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if nextURL != "https://news.hada.io/new?page=2" {
		t.Fatalf("next URL = %q", nextURL)
	}
}

func TestClientFollowsLatestPagePagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `<div class="topic_row" data-topic-state-id="2"><h2 class="topic-title-heading">Second</h2></div>`)
			return
		}
		fmt.Fprintf(w, `<div class="topic_row" data-topic-state-id="1"><h2 class="topic-title-heading">First</h2></div><div class="next"><a href="%s?page=2">Next</a></div>`, r.URL.Path)
	}))
	defer server.Close()

	client := NewClient()
	items, err := client.Fetch(context.Background(), server.URL+"/new", 2, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Title != "First" || items[1].Title != "Second" {
		t.Fatalf("unexpected paginated items: %+v", items)
	}
}

func TestCleanTitleRemovesControlsAndTruncatesRunes(t *testing.T) {
	got := CleanTitle("  제목\x1b  사이\u202e  공백  ", 6)
	if strings.ContainsRune(got, '\x1b') || strings.ContainsRune(got, '\u202e') {
		t.Fatalf("control character survived: %q", got)
	}
	if got != "제목 사이..." {
		t.Fatalf("clean title = %q", got)
	}
}

func TestCleanTitleReplacesDisallowedPunctuation(t *testing.T) {
	got := CleanTitle("앞\u00b7뒤", 100)
	if got != "앞-뒤" {
		t.Fatalf("clean title = %q", got)
	}
}

func TestParseAtomSortsAndLimitsEntries(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry><title>Older</title><id>old</id><published>2026-01-01T00:00:00Z</published></entry>
  <entry><title>Newer</title><id>new</id><published>2026-01-02T00:00:00Z</published></entry>
</feed>`)
	items, err := parseAtom(data, 1, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Newer" {
		t.Fatalf("unexpected items: %+v", items)
	}
}
