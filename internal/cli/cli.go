package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saz/claude-geeknews-spinner/internal/claude"
	"github.com/saz/claude-geeknews-spinner/internal/config"
	"github.com/saz/claude-geeknews-spinner/internal/feed"
	"github.com/saz/claude-geeknews-spinner/internal/syncer"
)

var Version = "dev"

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}
	switch args[0] {
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "refresh", "sync":
		return runRefresh(args[1:], stdout)
	case "config":
		return runConfig(args[1:], stdout)
	case "status":
		return runStatus(stdout)
	case "uninstall":
		return runUninstall(args[1:], stdout)
	case "hook":
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		_, _ = syncer.Run(ctx, feed.NewClient())
		return nil
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return nil
	case "help", "--help", "-h":
		printHelp(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q; run with --help", args[0])
	}
}

func runInstall(args []string, stdout, stderr io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet("install", flag.ContinueOnError)
	flags.SetOutput(stderr)
	count := flags.Int("count", cfg.Count, "number of recent headlines")
	display := flags.String("display", cfg.DisplayMode, "display mode: verb, tip, or both")
	clickableLinks := flags.Bool("clickable-links", cfg.ClickableLinks, "wrap headlines in terminal hyperlinks")
	prefix := flags.String("prefix", cfg.Prefix, "headline prefix")
	maxRunes := flags.Int("max-title-runes", cfg.MaxTitleRunes, "maximum title length in Unicode characters")
	if err := flags.Parse(args); err != nil {
		return err
	}
	cfg.Count = *count
	cfg.DisplayMode = strings.ToLower(*display)
	cfg.ClickableLinks = *clickableLinks
	cfg.Prefix = *prefix
	cfg.MaxTitleRunes = *maxRunes
	if err := config.Save(cfg); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	if strings.Contains(executable, string(filepath.Separator)+"go-build") {
		return errors.New("install from a built binary or with go install; go run uses a temporary executable")
	}
	state, err := claude.Install(executable)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Installed Claude Code hooks in %s\n", state.SettingsPath)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, syncErr := syncer.Run(ctx, feed.NewClient())
	if syncErr != nil {
		fmt.Fprintf(stderr, "warning: initial refresh failed: %v\n", syncErr)
		fmt.Fprintln(stdout, "The next Claude Code session will retry automatically.")
		return nil
	}
	fmt.Fprintf(stdout, "Loaded %d current GeekNews headlines.\n", len(result.Headlines))
	return nil
}

func runRefresh(args []string, stdout io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: claude-geeknews-spinner refresh")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	result, err := syncer.Run(ctx, feed.NewClient())
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Fetched and applied %d current GeekNews headlines.\n", len(result.Headlines))
	return nil
}

func runConfig(args []string, stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "show" {
		path, _ := config.Path()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Fprintf(stdout, "%s\n%s\n", path, data)
		return nil
	}
	if args[0] == "path" {
		path, err := config.Path()
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, path)
		return nil
	}
	if args[0] != "set" || len(args) != 3 {
		return errors.New("usage: claude-geeknews-spinner config set <key> <value>")
	}
	if err := config.Set(&cfg, args[1], args[2]); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Set %s to %s.\n", args[1], args[2])
	installed, err := claude.IsInstalled()
	if err != nil {
		return err
	}
	if installed {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		if _, err := syncer.Run(ctx, feed.NewClient()); err != nil {
			return fmt.Errorf("config saved, but refresh failed: %w", err)
		}
		fmt.Fprintln(stdout, "Refreshed the active spinner configuration.")
	}
	return nil
}

func runStatus(stdout io.Writer) error {
	settingsPath, err := claude.SettingsPath()
	if err != nil {
		return err
	}
	configPath, err := config.Path()
	if err != nil {
		return err
	}
	installed, err := claude.IsInstalled()
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Installed: %t\nClaude settings: %s\nConfig: %s\n", installed, settingsPath, configPath)
	fmt.Fprintln(stdout, "Refresh: live on SessionStart and UserPromptSubmit")
	return nil
}

func runUninstall(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	purge := flags.Bool("purge", false, "also delete config data")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := claude.Uninstall()
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Removed hooks from %s and restored the previous spinner settings.\n", result.SettingsPath)
	if result.PreservedUserChanges {
		fmt.Fprintln(stdout, "Preserved spinner values that were changed after installation.")
	}
	if *purge {
		dir, err := config.Dir()
		if err != nil {
			return err
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "Deleted config data.")
	}
	return nil
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `claude-geeknews-spinner shows current GeekNews headlines in Claude Code.

Usage:
  claude-geeknews-spinner install [options]
  claude-geeknews-spinner refresh
  claude-geeknews-spinner config [show|path|set <key> <value>]
  claude-geeknews-spinner status
  claude-geeknews-spinner uninstall [--purge]

Install options:
  --count 10             Number of latest headlines, from 1 to 50
  --display verb         verb, tip, or both
  --clickable-links      Add experimental terminal hyperlinks
  --prefix "[GN] "       Text placed before each title
  --max-title-runes 100  Maximum title length`)
}
