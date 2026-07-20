package syncer

import (
	"context"
	"errors"
	"net/url"

	"github.com/saz/claude-geeknews-spinner/internal/claude"
	"github.com/saz/claude-geeknews-spinner/internal/config"
	"github.com/saz/claude-geeknews-spinner/internal/feed"
)

type Fetcher interface {
	Fetch(context.Context, string, int, int, string) ([]feed.Item, error)
}

type Result struct {
	Headlines []string
}

func Run(ctx context.Context, fetcher Fetcher) (Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return Result{}, err
	}
	items, err := fetcher.Fetch(ctx, cfg.SourceURL, cfg.Count, cfg.MaxTitleRunes, cfg.Prefix)
	if err != nil {
		return Result{}, err
	}
	if len(items) > cfg.Count {
		items = items[:cfg.Count]
	}
	titles := make([]string, 0, len(items))
	for _, item := range items {
		if title := formatHeadline(item, cfg.ClickableLinks); title != "" {
			titles = append(titles, title)
		}
	}
	if len(titles) == 0 {
		return Result{}, errors.New("no GeekNews headlines were returned")
	}
	if err := claude.Apply(claude.DisplayOptions{Mode: cfg.DisplayMode, Titles: titles}); err != nil {
		return Result{}, err
	}
	return Result{Headlines: titles}, nil
}

func formatHeadline(item feed.Item, clickable bool) string {
	title := feed.CleanTitle(item.Title, 0)
	summary := feed.CleanTitle(item.Summary, 0)
	headline := title
	if summary != "" {
		headline += " - " + summary
	}
	if headline == "" || !clickable {
		return headline
	}
	link, err := url.Parse(item.URL)
	if err != nil || (link.Scheme != "https" && link.Scheme != "http") || link.Host == "" {
		return headline
	}
	return "\x1b]8;;" + link.String() + "\a" + headline + "\x1b]8;;\a"
}
