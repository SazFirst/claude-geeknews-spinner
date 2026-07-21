# Changelog

All notable changes to this project will be documented in this file.

The format follows Keep a Changelog, and releases use semantic versioning.

## Unreleased

### Changed

- Replaced the Go CLI with a Claude Code plugin.
- The plugin refreshes the latest 10 GeekNews headlines through `SessionStart` and `UserPromptSubmit` hooks.
- The plugin has no runtime package dependencies.
