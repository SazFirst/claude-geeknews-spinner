package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/saz/claude-geeknews-spinner/internal/claude"
	"github.com/saz/claude-geeknews-spinner/internal/config"
	"github.com/saz/claude-geeknews-spinner/internal/feed"
	"github.com/saz/claude-geeknews-spinner/internal/store"
)

type Fetcher interface {
	Fetch(context.Context, string, int, int, string) ([]feed.Item, error)
}

type Cache struct {
	FetchedAt  time.Time   `json:"fetchedAt"`
	ConfigHash string      `json:"configHash"`
	Items      []feed.Item `json:"items"`
}

type Result struct {
	Fetched    bool
	UsedCache  bool
	FetchedAt  time.Time
	Headlines  []string
	FetchError error
}

func Run(ctx context.Context, force bool, fetcher Fetcher) (Result, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return Result{}, err
	}
	lock, err := store.AcquireLock(filepath.Join(cacheDir, "sync.lock"), 3*time.Second, 30*time.Second)
	if err != nil {
		return Result{}, err
	}
	defer lock.Release()

	cfg, err := config.Load()
	if err != nil {
		return Result{}, err
	}
	cachePath := filepath.Join(cacheDir, "headlines.json")
	cache := loadCache(cachePath)
	hash := configHash(cfg)
	cacheMatches := cache.ConfigHash == hash && len(cache.Items) > 0
	fresh := cacheMatches && time.Since(cache.FetchedAt) < cfg.Interval()

	result := Result{}
	items := cache.Items
	if force || !fresh {
		fetched, fetchErr := fetcher.Fetch(ctx, cfg.SourceURL, cfg.Count, cfg.MaxTitleRunes, cfg.Prefix)
		if fetchErr == nil {
			items = fetched
			cache = Cache{FetchedAt: time.Now().UTC(), ConfigHash: hash, Items: fetched}
			if err := store.WriteJSON(cachePath, cache, 0o600); err != nil {
				return Result{}, err
			}
			result.Fetched = true
		} else {
			result.FetchError = fetchErr
			if len(items) == 0 {
				return result, fetchErr
			}
			result.UsedCache = true
		}
	} else {
		result.UsedCache = true
	}

	if len(items) > cfg.Count {
		items = items[:cfg.Count]
	}
	titles := make([]string, 0, len(items))
	for _, item := range items {
		if item.Title != "" {
			titles = append(titles, item.Title)
		}
	}
	if len(titles) == 0 {
		return result, errors.New("no cached or fetched GeekNews headlines are available")
	}
	if err := claude.Apply(claude.DisplayOptions{Mode: cfg.DisplayMode, Titles: titles}); err != nil {
		return result, err
	}
	result.FetchedAt = cache.FetchedAt
	result.Headlines = titles
	return result, nil
}

func ReadCache() (Cache, error) {
	dir, err := config.CacheDir()
	if err != nil {
		return Cache{}, err
	}
	path := filepath.Join(dir, "headlines.json")
	var cache Cache
	if err := store.ReadJSON(path, &cache); err != nil {
		return Cache{}, err
	}
	return cache, nil
}

func PurgeCache() error {
	dir, err := config.CacheDir()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove cache directory %s: %w", dir, err)
	}
	return nil
}

func loadCache(path string) Cache {
	var cache Cache
	if err := store.ReadJSON(path, &cache); err != nil {
		return Cache{}
	}
	return cache
}

func configHash(cfg config.Config) string {
	data, _ := json.Marshal(struct {
		Count         int
		SourceURL     string
		Prefix        string
		MaxTitleRunes int
	}{cfg.Count, cfg.SourceURL, cfg.Prefix, cfg.MaxTitleRunes})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}
