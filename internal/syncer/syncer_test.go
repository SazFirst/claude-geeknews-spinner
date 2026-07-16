package syncer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/saz/claude-geeknews-spinner/internal/claude"
	"github.com/saz/claude-geeknews-spinner/internal/config"
	"github.com/saz/claude-geeknews-spinner/internal/feed"
)

type fakeFetcher struct {
	items []feed.Item
	err   error
	calls int
}

func (f *fakeFetcher) Fetch(context.Context, string, int, int, string) ([]feed.Item, error) {
	f.calls++
	return f.items, f.err
}

func setupSyncTest(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, ".claude"))
	t.Setenv("CLAUDE_GEEKNEWS_CONFIG_DIR", filepath.Join(root, "tool-config"))
	t.Setenv("CLAUDE_GEEKNEWS_CACHE_DIR", filepath.Join(root, "tool-cache"))
	settingsPath, _ := claude.SettingsPath()
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	if _, err := claude.Install("/tmp/claude-geeknews-spinner"); err != nil {
		t.Fatal(err)
	}
}

func TestRunFetchesThenFallsBackToCache(t *testing.T) {
	setupSyncTest(t)
	now := time.Now()
	live := &fakeFetcher{items: []feed.Item{
		{Title: "One", Published: now},
		{Title: "Two", Published: now.Add(-time.Minute)},
	}}
	result, err := Run(context.Background(), true, live)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Fetched || len(result.Headlines) != 2 {
		t.Fatalf("unexpected live result: %+v", result)
	}

	offline := &fakeFetcher{err: errors.New("offline")}
	result, err = Run(context.Background(), true, offline)
	if err != nil {
		t.Fatalf("cached fallback failed: %v", err)
	}
	if !result.UsedCache || result.FetchError == nil || len(result.Headlines) != 2 {
		t.Fatalf("unexpected fallback result: %+v", result)
	}
}

func TestRunUsesFreshCacheWithoutNetwork(t *testing.T) {
	setupSyncTest(t)
	fetcher := &fakeFetcher{items: []feed.Item{{Title: "Cached", Published: time.Now()}}}
	if _, err := Run(context.Background(), true, fetcher); err != nil {
		t.Fatal(err)
	}
	fetcher.err = errors.New("should not be called")
	if _, err := Run(context.Background(), false, fetcher); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", fetcher.calls)
	}
}

func TestFormatHeadlineCreatesOnlySafeHTTPLinks(t *testing.T) {
	item := feed.Item{Title: "\x1bUnsafe title", URL: "https://news.hada.io/topic?id=123"}
	want := "\x1b]8;;https://news.hada.io/topic?id=123\aUnsafe title\x1b]8;;\a"
	if got := formatHeadline(item, true); got != want {
		t.Fatalf("linked headline = %q, want %q", got, want)
	}
	if got := formatHeadline(item, false); got != "Unsafe title" {
		t.Fatalf("plain headline = %q", got)
	}
	item.URL = "javascript:alert(1)"
	if got := formatHeadline(item, true); got != "Unsafe title" {
		t.Fatalf("unsafe URL should fall back to plain text: %q", got)
	}
}

func TestRunAppliesClickableHeadlines(t *testing.T) {
	setupSyncTest(t)
	cfg := config.Default()
	cfg.ClickableLinks = true
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	fetcher := &fakeFetcher{items: []feed.Item{{
		Title: "Linked",
		URL:   "https://news.hada.io/topic?id=456",
	}}}
	result, err := Run(context.Background(), true, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b]8;;https://news.hada.io/topic?id=456\aLinked\x1b]8;;\a"
	if len(result.Headlines) != 1 || result.Headlines[0] != want {
		t.Fatalf("headlines = %q, want %q", result.Headlines, want)
	}
}
