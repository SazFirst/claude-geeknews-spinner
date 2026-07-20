package claude

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/saz/claude-geeknews-spinner/internal/config"
	"github.com/saz/claude-geeknews-spinner/internal/store"
)

const stateVersion = 1

var managedKeys = []string{"spinnerVerbs", "spinnerTipsEnabled", "spinnerTipsOverride"}

type RawValue struct {
	Present bool            `json:"present"`
	Value   json.RawMessage `json:"value,omitempty"`
}

type Snapshot map[string]RawValue

type InstallState struct {
	Version      int      `json:"version"`
	SettingsPath string   `json:"settingsPath"`
	Executables  []string `json:"executables"`
	Original     Snapshot `json:"original"`
	LastApplied  Snapshot `json:"lastApplied,omitempty"`
	InstalledAt  string   `json:"installedAt"`
}

type DisplayOptions struct {
	Mode   string
	Titles []string
}

func SettingsPath() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func StatePath(settingsPath string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(settingsPath))
	return filepath.Join(dir, "installations", hex.EncodeToString(sum[:8])+".json"), nil
}

func LoadState(settingsPath string) (InstallState, error) {
	path, err := StatePath(settingsPath)
	if err != nil {
		return InstallState{}, err
	}
	var state InstallState
	if err := store.ReadJSON(path, &state); err != nil {
		return InstallState{}, err
	}
	if state.Version != stateVersion {
		return InstallState{}, fmt.Errorf("unsupported installation state version %d", state.Version)
	}
	return state, nil
}

func Install(executable string) (InstallState, error) {
	settingsPath, err := SettingsPath()
	if err != nil {
		return InstallState{}, err
	}
	lock, err := acquireSettingsLock(settingsPath)
	if err != nil {
		return InstallState{}, err
	}
	defer lock.Release()
	settings, err := readSettings(settingsPath)
	if err != nil {
		return InstallState{}, err
	}

	state, err := LoadState(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		original := takeSnapshot(settings)
		state = InstallState{
			Version:      stateVersion,
			SettingsPath: settingsPath,
			Original:     original,
			LastApplied:  original,
			InstalledAt:  time.Now().UTC().Format(time.RFC3339),
		}
	} else if err != nil {
		return InstallState{}, err
	}
	state.Executables = appendUnique(state.Executables, executable)

	if err := removeHookHandlers(settings, state.Executables); err != nil {
		return InstallState{}, err
	}
	if err := addHookHandler(settings, "SessionStart", executable); err != nil {
		return InstallState{}, err
	}
	if err := addHookHandler(settings, "UserPromptSubmit", executable); err != nil {
		return InstallState{}, err
	}
	if err := saveState(state); err != nil {
		return InstallState{}, err
	}
	if err := writeSettings(settingsPath, settings); err != nil {
		return InstallState{}, err
	}
	return state, nil
}

func Apply(options DisplayOptions) error {
	if len(options.Titles) == 0 {
		return errors.New("refusing to write an empty headline list")
	}
	settingsPath, err := SettingsPath()
	if err != nil {
		return err
	}
	lock, err := acquireSettingsLock(settingsPath)
	if err != nil {
		return err
	}
	defer lock.Release()
	state, err := LoadState(settingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("not installed; run claude-geeknews-spinner install first")
		}
		return err
	}
	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}
	current := takeSnapshot(settings)
	if state.LastApplied != nil {
		for _, key := range managedKeys {
			if !rawEqual(current[key], state.LastApplied[key]) {
				return fmt.Errorf("Claude setting %s changed after installation; run uninstall before replacing it", key)
			}
		}
	}
	restoreSnapshot(settings, state.Original)

	verbs := map[string]any{"mode": "append", "verbs": options.Titles}
	tips := map[string]any{"excludeDefault": true, "tips": options.Titles}
	switch options.Mode {
	case "verb":
		settings["spinnerVerbs"] = mustJSON(verbs)
	case "tip":
		settings["spinnerTipsEnabled"] = mustJSON(true)
		settings["spinnerTipsOverride"] = mustJSON(tips)
	case "both":
		settings["spinnerVerbs"] = mustJSON(verbs)
		settings["spinnerTipsEnabled"] = mustJSON(true)
		settings["spinnerTipsOverride"] = mustJSON(tips)
	default:
		return fmt.Errorf("unsupported display mode %q", options.Mode)
	}
	desired := takeSnapshot(settings)
	if snapshotsEqual(current, desired) {
		if state.LastApplied == nil {
			state.LastApplied = desired
			return saveState(state)
		}
		return nil
	}
	state.LastApplied = desired
	if err := writeSettings(settingsPath, settings); err != nil {
		return err
	}
	return saveState(state)
}

type UninstallResult struct {
	PreservedUserChanges bool
	SettingsPath         string
}

func Uninstall() (UninstallResult, error) {
	settingsPath, err := SettingsPath()
	if err != nil {
		return UninstallResult{}, err
	}
	state, err := LoadState(settingsPath)
	if err != nil {
		return UninstallResult{}, err
	}
	lock, err := acquireSettingsLock(settingsPath)
	if err != nil {
		return UninstallResult{}, err
	}
	defer lock.Release()
	settings, err := readSettings(settingsPath)
	if err != nil {
		return UninstallResult{}, err
	}
	if err := removeHookHandlers(settings, state.Executables); err != nil {
		return UninstallResult{}, err
	}

	preserved := false
	current := takeSnapshot(settings)
	for _, key := range managedKeys {
		if state.LastApplied != nil && rawEqual(current[key], state.LastApplied[key]) {
			restoreValue(settings, key, state.Original[key])
		} else if state.LastApplied != nil && !rawEqual(current[key], state.LastApplied[key]) {
			preserved = true
		}
	}
	if err := writeSettings(settingsPath, settings); err != nil {
		return UninstallResult{}, err
	}
	statePath, err := StatePath(settingsPath)
	if err != nil {
		return UninstallResult{}, err
	}
	if err := os.Remove(statePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return UninstallResult{}, err
	}
	return UninstallResult{PreservedUserChanges: preserved, SettingsPath: settingsPath}, nil
}

func IsInstalled() (bool, error) {
	settingsPath, err := SettingsPath()
	if err != nil {
		return false, err
	}
	_, err = LoadState(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func readSettings(path string) (map[string]json.RawMessage, error) {
	settings := make(map[string]json.RawMessage)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return settings, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Claude settings %s: %w", path, err)
	}
	if len(data) > 5<<20 {
		return nil, fmt.Errorf("Claude settings exceed 5 MiB: %s", path)
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse Claude settings %s without modifying it: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]json.RawMessage) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := store.AtomicWrite(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write Claude settings %s: %w", path, err)
	}
	return nil
}

func takeSnapshot(settings map[string]json.RawMessage) Snapshot {
	snapshot := make(Snapshot, len(managedKeys))
	for _, key := range managedKeys {
		value, ok := settings[key]
		snapshot[key] = RawValue{Present: ok, Value: cloneRaw(value)}
	}
	return snapshot
}

func restoreSnapshot(settings map[string]json.RawMessage, snapshot Snapshot) {
	for _, key := range managedKeys {
		restoreValue(settings, key, snapshot[key])
	}
}

func restoreValue(settings map[string]json.RawMessage, key string, value RawValue) {
	if !value.Present {
		delete(settings, key)
		return
	}
	settings[key] = cloneRaw(value.Value)
}

func saveState(state InstallState) error {
	path, err := StatePath(state.SettingsPath)
	if err != nil {
		return err
	}
	return store.WriteJSON(path, state, 0o600)
}

func addHookHandler(settings map[string]json.RawMessage, event, executable string) error {
	hooks, err := decodeHooks(settings["hooks"])
	if err != nil {
		return err
	}
	handler := map[string]any{
		"type":    "command",
		"command": executable,
		"args":    []string{"hook"},
		"async":   true,
		"timeout": 15,
	}
	hooks[event] = append(hooks[event], map[string]any{"hooks": []any{handler}})
	settings["hooks"] = mustJSON(hooks)
	return nil
}

func removeHookHandlers(settings map[string]json.RawMessage, executables []string) error {
	hooksRaw, ok := settings["hooks"]
	if !ok {
		return nil
	}
	hooks, err := decodeHooks(hooksRaw)
	if err != nil {
		return err
	}
	for event, groups := range hooks {
		filteredGroups := make([]map[string]any, 0, len(groups))
		for _, group := range groups {
			handlers, _ := group["hooks"].([]any)
			filteredHandlers := make([]any, 0, len(handlers))
			for _, rawHandler := range handlers {
				handler, _ := rawHandler.(map[string]any)
				if isOurHandler(handler, executables) {
					continue
				}
				filteredHandlers = append(filteredHandlers, rawHandler)
			}
			if len(filteredHandlers) > 0 {
				group["hooks"] = filteredHandlers
				filteredGroups = append(filteredGroups, group)
			}
		}
		if len(filteredGroups) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = filteredGroups
		}
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = mustJSON(hooks)
	}
	return nil
}

func decodeHooks(raw json.RawMessage) (map[string][]map[string]any, error) {
	hooks := make(map[string][]map[string]any)
	if len(raw) == 0 {
		return hooks, nil
	}
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return nil, fmt.Errorf("parse existing Claude hooks without modifying them: %w", err)
	}
	return hooks, nil
}

func acquireSettingsLock(settingsPath string) (*store.Lock, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(settingsPath))
	path := filepath.Join(dir, "locks", "settings-"+hex.EncodeToString(sum[:8])+".lock")
	return store.AcquireLock(path, 3*time.Second, 30*time.Second)
}

func isOurHandler(handler map[string]any, executables []string) bool {
	command, _ := handler["command"].(string)
	args, _ := handler["args"].([]any)
	if len(args) != 1 || args[0] != "hook" {
		return false
	}
	for _, executable := range executables {
		if command == executable {
			return true
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func rawEqual(a, b RawValue) bool {
	if a.Present != b.Present {
		return false
	}
	if !a.Present {
		return true
	}
	var av, bv any
	if json.Unmarshal(a.Value, &av) != nil || json.Unmarshal(b.Value, &bv) != nil {
		return bytes.Equal(a.Value, b.Value)
	}
	return reflect.DeepEqual(av, bv)
}

func snapshotsEqual(a, b Snapshot) bool {
	for _, key := range managedKeys {
		if !rawEqual(a[key], b[key]) {
			return false
		}
	}
	return true
}

func mustJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}
