package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Count != 10 {
		t.Fatalf("default count = %d, want 10", cfg.Count)
	}
	if cfg.RefreshInterval != "15s" {
		t.Fatalf("default interval = %q, want 15s", cfg.RefreshInterval)
	}
	if cfg.SourceURL != "https://news.hada.io/new" {
		t.Fatalf("default source URL = %q", cfg.SourceURL)
	}
	if cfg.ClickableLinks {
		t.Fatal("clickable links should be disabled by default")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}
}

func TestLoadMergesDefaultsAndRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"count": 20}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Count != 20 || cfg.RefreshInterval != "15s" {
		t.Fatalf("unexpected merged config: %+v", cfg)
	}

	if err := os.WriteFile(path, []byte(`{"count": 10, "typo": true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected unknown field to fail")
	}

	if err := os.WriteFile(path, []byte(`{"count": 10} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected trailing JSON to fail")
	}
}

func TestSetValidatesValues(t *testing.T) {
	cfg := Default()
	if err := Set(&cfg, "count", "25"); err != nil {
		t.Fatal(err)
	}
	if cfg.Count != 25 {
		t.Fatalf("count = %d", cfg.Count)
	}
	if err := Set(&cfg, "interval", "1s"); err == nil {
		t.Fatal("expected too-short interval to fail")
	}
	if cfg.RefreshInterval != "15s" {
		t.Fatalf("invalid value mutated config: %q", cfg.RefreshInterval)
	}
	if err := Set(&cfg, "clickable-links", "true"); err != nil {
		t.Fatal(err)
	}
	if !cfg.ClickableLinks {
		t.Fatal("clickable links were not enabled")
	}
	if err := Set(&cfg, "clickable-links", "sometimes"); err == nil {
		t.Fatal("expected invalid boolean to fail")
	}
}
