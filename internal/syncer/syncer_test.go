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
