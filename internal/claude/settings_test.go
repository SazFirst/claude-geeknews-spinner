package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestEnvironment(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, ".claude"))
	t.Setenv("CLAUDE_GEEKNEWS_CONFIG_DIR", filepath.Join(root, "tool-config"))
	return root
}

func writeTestSettings(t *testing.T, value any) string {
	t.Helper()
	path, err := SettingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestSettings(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	settings, err := readSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	return settings
}

func TestInstallApplyAndUninstallPreserveExistingSettings(t *testing.T) {
	setupTestEnvironment(t)
	path := writeTestSettings(t, map[string]any{
		"model": "opus",
		"spinnerVerbs": map[string]any{
			"mode":  "append",
			"verbs": []string{"Original"},
		},
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{"hooks": []any{map[string]any{"type": "command", "command": "/keep/me"}}},
			},
		},
	})

	if _, err := Install("/usr/local/bin/claude-geeknews-spinner"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(DisplayOptions{Mode: "verb", Titles: []string{"[GN] Headline"}}); err != nil {
		t.Fatal(err)
	}
	settings := readTestSettings(t, path)
	if string(settings["model"]) != `"opus"` {
		t.Fatalf("model changed: %s", settings["model"])
	}
	if !containsJSON(settings["spinnerVerbs"], "[GN] Headline") {
		t.Fatalf("headline not installed: %s", settings["spinnerVerbs"])
	}
	if !containsJSON(settings["hooks"], "/keep/me") {
		t.Fatalf("existing hook was removed: %s", settings["hooks"])
	}

	result, err := Uninstall()
	if err != nil {
		t.Fatal(err)
	}
	if result.PreservedUserChanges {
		t.Fatal("unexpected user-change warning")
	}
	settings = readTestSettings(t, path)
	if !containsJSON(settings["spinnerVerbs"], "Original") {
		t.Fatalf("original spinner was not restored: %s", settings["spinnerVerbs"])
	}
	if containsJSON(settings["hooks"], "claude-geeknews-spinner") {
		t.Fatalf("tool hook survived uninstall: %s", settings["hooks"])
	}
	if !containsJSON(settings["hooks"], "/keep/me") {
		t.Fatalf("existing hook was removed on uninstall: %s", settings["hooks"])
	}
}

func TestMalformedSettingsAreNeverOverwritten(t *testing.T) {
	setupTestEnvironment(t)
	path, _ := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"model":`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install("/tmp/claude-geeknews-spinner"); err == nil {
		t.Fatal("expected malformed settings to fail")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("malformed settings were changed: %q", after)
	}
}

func TestApplyDetectsUserChanges(t *testing.T) {
	setupTestEnvironment(t)
	path := writeTestSettings(t, map[string]any{"model": "sonnet"})
	if _, err := Install("/tmp/claude-geeknews-spinner"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(DisplayOptions{Mode: "verb", Titles: []string{"First"}}); err != nil {
		t.Fatal(err)
	}
	settings := readTestSettings(t, path)
	settings["spinnerVerbs"] = mustJSON(map[string]any{"mode": "replace", "verbs": []string{"User value"}})
	if err := writeSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	if err := Apply(DisplayOptions{Mode: "verb", Titles: []string{"Second"}}); err == nil {
		t.Fatal("expected user settings drift to be detected")
	}
}

func TestApplySkipsUnchangedSettingsWrite(t *testing.T) {
	setupTestEnvironment(t)
	path := writeTestSettings(t, map[string]any{"model": "sonnet"})
	if _, err := Install("/tmp/claude-geeknews-spinner"); err != nil {
		t.Fatal(err)
	}
	options := DisplayOptions{Mode: "verb", Titles: []string{"Current"}}
	if err := Apply(options); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Unix(1_000_000, 0)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := Apply(options); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("unchanged settings were rewritten at %s", info.ModTime())
	}
}

func containsJSON(value json.RawMessage, needle string) bool {
	return len(value) > 0 && stringContains(string(value), needle)
}

func stringContains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
