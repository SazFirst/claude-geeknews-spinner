# Contributing

## Development Setup

1. Install Node.js 18 or later.
2. Fork and clone the repository.
3. Create a focused branch for the change.
4. Run the checks below before opening a pull request.

```bash
node --test plugins/claude-geeknews-spinner/scripts/*.test.mjs
node --check plugins/claude-geeknews-spinner/scripts/refresh.mjs
claude plugin validate .
```

## Pull Requests

- Keep changes focused and explain user-visible behavior.
- Add or update tests for behavior changes.
- Keep the plugin free of runtime package dependencies.
- Preserve unrelated Claude Code settings.
- Never write an empty headline list after a network failure.
- Update both README files when plugin behavior changes.

## Reporting Bugs

Include the operating system, Claude Code version, plugin version, and a redacted copy of the relevant Claude settings.

Do not include tokens, credentials, or private Claude settings in an issue.
