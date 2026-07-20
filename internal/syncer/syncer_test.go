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

func TestRunFetchesEveryTime(t *testing.T) {
	setupSyncTest(t)
	now := time.Now()
	live := &fakeFetcher{items: []feed.Item{
		{Title: "One", Published: now},
		{Title: "Two", Published: now.Add(-time.Minute)},
	}}
	result, err := Run(context.Background(), live)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Headlines) != 2 {
		t.Fatalf("unexpected live result: %+v", result)
	}
	if _, err := Run(context.Background(), live); err != nil {
		t.Fatal(err)
	}
	if live.calls != 2 {
		t.Fatalf("fetch calls = %d, want 2", live.calls)
	}
}

func TestRunFailureLeavesSpinnerUnchanged(t *testing.T) {
	setupSyncTest(t)
	fetcher := &fakeFetcher{items: []feed.Item{{Title: "Current", Published: time.Now()}}}
	if _, err := Run(context.Background(), fetcher); err != nil {
		t.Fatal(err)
	}
	settingsPath, err := claude.SettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	fetcher.err = errors.New("offline")
	if _, err := Run(context.Background(), fetcher); err == nil {
		t.Fatal("expected live fetch failure")
	}
	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("settings changed after a failed live fetch")
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
	result, err := Run(context.Background(), fetcher)
	if err != nil {
		t.Fatal(err)
	}
	want := "\x1b]8;;https://news.hada.io/topic?id=456\aLinked\x1b]8;;\a"
	if len(result.Headlines) != 1 || result.Headlines[0] != want {
		t.Fatalf("headlines = %q, want %q", result.Headlines, want)
	}
}
