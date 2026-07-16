# Contributing

## Development Setup

1. Install Go 1.21 or later.
2. Fork and clone the repository.
3. Create a focused branch for the change.
4. Run the checks below before opening a pull request.

```bash
gofmt -w cmd internal
go test ./...
go test -race ./...
go vet ./...
```

## Pull Requests

- Keep changes focused and explain user-visible behavior.
- Add or update tests for behavior changes.
- Do not replace structured HTML or XML parsing with regular expressions.
- Preserve existing Claude Code settings and hooks.
- Never write an empty headline list after a network failure.
- Update both README files when commands or configuration change.

## Reporting Bugs

Include the operating system, Claude Code version, tool version, redacted config, and the output of:

```bash
claude-geeknews-spinner status
```

Do not include tokens, credentials, or private Claude settings in an issue.
