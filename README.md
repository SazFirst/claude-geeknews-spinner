# Claude GeekNews Spinner

Show the latest [GeekNews](https://news.hada.io/new) headlines while Claude Code is working.

```text
[GN] A new open source database engine
[GN] How a team reduced build time by 80 percent
[GN] A practical guide to coding agents
```

The default pool contains the 10 newest posts from the GeekNews latest page. The count, refresh interval, prefix, title length, source URL, and display location are configurable.

[Korean documentation](README.ko.md)

## How It Stays Current

Claude Code cannot read spinner values directly from a URL. This tool installs two lightweight asynchronous hooks:

- `SessionStart` checks for fresh headlines when a session starts or resumes.
- `UserPromptSubmit` checks again when you submit a prompt.

The default cache lifetime is 15 seconds, matching the public cache policy of the GeekNews latest page. A fresh cache exits without a network request. A stale cache is refreshed in the background, so Claude Code does not wait for the request. Running sessions pick up the updated settings automatically.

There is no daemon and no process stays resident between hook events.

## Install

### With Go

Go 1.21 or later is required.

```bash
go install github.com/saz/claude-geeknews-spinner/cmd/claude-geeknews-spinner@latest
claude-geeknews-spinner install
```

Pushing a version tag publishes prebuilt archives for Linux, macOS, and Windows through GitHub Releases.

### Customize During Installation

```bash
claude-geeknews-spinner install \
  --count 20 \
  --interval 30s \
  --display verb \
  --prefix "[GeekNews] " \
  --max-title-runes 120
```

## Configuration

Show the active configuration and its path:

```bash
claude-geeknews-spinner config
claude-geeknews-spinner config path
```

Change one value and refresh the active spinner immediately:

```bash
claude-geeknews-spinner config set count 20
claude-geeknews-spinner config set interval 1m
claude-geeknews-spinner config set display tip
```

Default configuration:

```json
{
  "count": 10,
  "refreshInterval": "15s",
  "sourceUrl": "https://news.hada.io/new",
  "prefix": "[GN] ",
  "maxTitleRunes": 100,
  "displayMode": "verb"
}
```

Supported values:

| Setting | Values | Description |
| --- | --- | --- |
| `count` | 1 to 50 | Number of newest headlines. Pagination is followed when needed. |
| `refreshInterval` | 15 seconds to 24 hours | Minimum age before another request is made. |
| `sourceUrl` | Absolute HTTP or HTTPS URL | Supports the GeekNews HTML layout and Atom feeds. |
| `prefix` | Any string | Text shown before each title. |
| `maxTitleRunes` | 20 to 500 | Maximum title length before truncation. |
| `displayMode` | `verb`, `tip`, `both` | Where Claude Code displays the headlines. |

`verb` puts headlines in the main spinner phrase. Claude Code may also use that phrase in its turn completion text. `tip` uses the secondary tips area. `both` writes to both locations.

## Commands

```text
claude-geeknews-spinner install [options]
claude-geeknews-spinner refresh
claude-geeknews-spinner config [show|path|set <key> <value>]
claude-geeknews-spinner status
claude-geeknews-spinner uninstall [--purge]
```

`refresh` forces an immediate network request. `status` reports the installation, config, cache size, and last successful fetch. `uninstall` removes only this tool's hooks and restores the spinner values that existed before installation. Add `--purge` to remove its config and cache too.

## Safety

- Existing Claude Code settings and unrelated hooks are preserved.
- Settings updates use a lock and an atomic file replacement.
- Invalid Claude settings are never replaced with an empty object.
- A failed or empty network response leaves the last successful cache active.
- Existing spinner values are saved at installation and restored at uninstall.
- User changes made directly to managed spinner keys are detected instead of overwritten.
- Control characters and bidirectional text controls are removed from remote titles.
- `CLAUDE_CONFIG_DIR` is honored for alternate Claude Code profiles.

The tool requests only the configured source URL. It does not collect telemetry.

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/claude-geeknews-spinner
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution details.
The [design notes](docs/design.md) compare the refresh model with related spinner projects.

## License

MIT
