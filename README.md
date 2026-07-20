# Claude GeekNews Spinner

Show the latest [GeekNews](https://news.hada.io/new) headlines while Claude Code is working.

```text
[GN] A new open source database engine
[GN] How a team reduced build time by 80 percent
[GN] A practical guide to coding agents
```

The default pool contains the 10 newest posts from the GeekNews latest page. The count, prefix, title length, source URL, display location, and experimental terminal hyperlinks are configurable.

[Korean documentation](README.ko.md)

## How It Stays Current

Claude Code cannot read spinner values directly from a URL. This tool installs two lightweight asynchronous hooks:

- `SessionStart` fetches headlines when a session starts or resumes.
- `UserPromptSubmit` fetches them again when you submit a prompt.

Each hook performs a live request. The asynchronous hook does not make Claude Code wait for the network, and a successful result updates the pool used by subsequent spinner selections. There is no persistent headline cache.

There is no daemon and no process stays resident between hook events.

`spinnerVerbs` is a selection pool, not a timed playlist. Claude Code normally
chooses one entry for a turn and keeps it until the turn resets. Built-in
thinking progress, tool activity, or task status can temporarily replace the
visible label, but Claude Code does not cycle through custom verbs on a timer.
Because `UserPromptSubmit` runs asynchronously, its newly fetched pool is
reliably available for subsequent selections rather than guaranteed for the
turn that triggered the hook.

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
claude-geeknews-spinner config set display tip
claude-geeknews-spinner config set clickable-links true
```

Default configuration:

```json
{
  "count": 10,
  "sourceUrl": "https://news.hada.io/new",
  "prefix": "[GN] ",
  "maxTitleRunes": 100,
  "displayMode": "verb",
  "clickableLinks": false
}
```

Supported values:

| Setting | Values | Description |
| --- | --- | --- |
| `count` | 1 to 50 | Number of newest headlines. Pagination is followed when needed. |
| `sourceUrl` | Absolute HTTP or HTTPS URL | Supports the GeekNews HTML layout and Atom feeds. |
| `prefix` | Any string | Text shown before each title. |
| `maxTitleRunes` | 20 to 500 | Maximum title length before truncation. |
| `displayMode` | `verb`, `tip`, `both` | Where Claude Code displays the headlines. |
| `clickableLinks` | `true`, `false` | Wrap titles in experimental OSC 8 terminal links. |

`verb` appends headlines to Claude Code's built-in spinner verbs, so the default verbs remain available. Claude Code may also use that phrase in its turn completion text. Each entry combines the title with the first summary line when GeekNews provides one. `tip` uses the secondary tips area. `both` writes to both locations.

When `clickableLinks` is enabled, supported terminals can open a headline with Cmd+click on macOS or Ctrl+click on Linux and Windows. Claude Code does not document OSC 8 support for `spinnerVerbs`, so unsupported renderers may fall back to plain text or strip the link.

## Commands

```text
claude-geeknews-spinner install [options]
claude-geeknews-spinner refresh
claude-geeknews-spinner config [show|path|set <key> <value>]
claude-geeknews-spinner status
claude-geeknews-spinner uninstall [--purge]
```

`refresh` performs an immediate network request. `status` reports the installation and config paths. `uninstall` removes only this tool's hooks and restores the spinner values that existed before installation. Add `--purge` to remove its config too.

## Safety

- Existing Claude Code settings and unrelated hooks are preserved.
- Settings updates use an atomic file replacement.
- Invalid Claude settings are never replaced with an empty object.
- A failed or empty network response leaves the currently installed spinner settings unchanged.
- Existing spinner values are saved at installation and restored at uninstall.
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
