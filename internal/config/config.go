package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultSourceURL = "https://news.hada.io/new"
	appDirName       = "claude-geeknews-spinner"
)

type Config struct {
	Count           int    `json:"count"`
	RefreshInterval string `json:"refreshInterval"`
	SourceURL       string `json:"sourceUrl"`
	Prefix          string `json:"prefix"`
	MaxTitleRunes   int    `json:"maxTitleRunes"`
	DisplayMode     string `json:"displayMode"`
	ClickableLinks  bool   `json:"clickableLinks"`
}

func Default() Config {
	return Config{
		Count:           10,
		RefreshInterval: "15s",
		SourceURL:       DefaultSourceURL,
		Prefix:          "[GN] ",
		MaxTitleRunes:   100,
		DisplayMode:     "verb",
		ClickableLinks:  false,
	}
}

func (c Config) Validate() error {
	if c.Count < 1 || c.Count > 50 {
		return fmt.Errorf("count must be between 1 and 50, got %d", c.Count)
	}
	interval, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return fmt.Errorf("invalid refreshInterval %q: %w", c.RefreshInterval, err)
	}
	if interval < 15*time.Second || interval > 24*time.Hour {
		return errors.New("refreshInterval must be between 15s and 24h")
	}
	if c.SourceURL == "" {
		return errors.New("sourceUrl cannot be empty")
	}
	parsedURL, err := url.Parse(c.SourceURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		return fmt.Errorf("sourceUrl must be an absolute HTTP or HTTPS URL, got %q", c.SourceURL)
	}
	if c.MaxTitleRunes < 20 || c.MaxTitleRunes > 500 {
		return fmt.Errorf("maxTitleRunes must be between 20 and 500, got %d", c.MaxTitleRunes)
	}
	switch c.DisplayMode {
	case "verb", "tip", "both":
	default:
		return fmt.Errorf("displayMode must be verb, tip, or both, got %q", c.DisplayMode)
	}
	return nil
}

func (c Config) Interval() time.Duration {
	d, _ := time.ParseDuration(c.RefreshInterval)
	return d
}

func Dir() (string, error) {
	if dir := os.Getenv("CLAUDE_GEEKNEWS_CONFIG_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func CacheDir() (string, error) {
	if dir := os.Getenv("CLAUDE_GEEKNEWS_CACHE_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return LoadFile(path)
}

func LoadFile(path string) (Config, error) {
	c := Default()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("open config %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(io.LimitReader(f, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("parse config %s: unexpected trailing JSON", path)
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config %s: %w", path, err)
	}
	return c, nil
}

func Save(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'), 0o600)
}

func Set(c *Config, key, value string) error {
	next := *c
	switch strings.ToLower(key) {
	case "count":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("count must be an integer: %w", err)
		}
		next.Count = n
	case "interval", "refreshinterval":
		next.RefreshInterval = value
	case "prefix":
		next.Prefix = value
	case "maxtitlerunes", "max-title-runes":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("maxTitleRunes must be an integer: %w", err)
		}
		next.MaxTitleRunes = n
	case "display", "displaymode":
		next.DisplayMode = strings.ToLower(value)
	case "clickablelinks", "clickable-links":
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("clickableLinks must be true or false: %w", err)
		}
		next.ClickableLinks = enabled
	case "sourceurl", "source-url":
		next.SourceURL = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

func atomicWrite(path string, data []byte, defaultMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	mode := defaultMode
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
