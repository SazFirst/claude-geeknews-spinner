# Design Notes

This project updates Claude Code's spinner from a remote news page. That is a
different problem from installing a fixed list of themed verbs, so the refresh
trigger and settings update strategy matter as much as feed parsing.

## Related Implementations

### claudenews

[tolibear/claudenews](https://github.com/tolibear/claudenews) is the closest
reference. It is a Node.js CLI with a source picker for Hacker News, Reddit,
Lobsters, dev.to, and GitHub Trending. A `SessionStart` hook fetches headlines,
and a `PreToolUse` hook refreshes a cache after 30 minutes. It then writes a
combined list to `spinnerVerbs`.

This project does not copy that architecture. A tool-use hook can run many
times during one turn and is unrelated to when the user starts waiting. Here,
`SessionStart` primes the data and `UserPromptSubmit` starts a non-blocking
live request at the point a new spinner is about to be useful. Unlike
`claudenews`, this project does not persist a headline cache.

### One-shot remote updater

[spinnerverbs.py](https://gist.github.com/rashikichi/c695a3ccbbc943f84d31e2e3789d5158)
fetches GitHub release notes, extracts bullet points, and atomically replaces
`spinnerVerbs`. It has response-size limits and careful TLS defaults, but it is
run manually and has no lifecycle hook. It also falls back to a new settings
object after malformed JSON, with a backup.

This project keeps the useful bounded-response and atomic-write properties. It
instead refuses to modify malformed Claude settings because silently replacing
unrecognized configuration is too risky for an automatic hook.

### Settings managers and static packs

[claude-code-tool-manager](https://github.com/tylergraydev/claude-code-tool-manager)
provides a desktop CRUD interface backed by a database and syncs selected verbs
to Claude settings. Static packs and generator prompts, such as
[create-spinnerverbs](https://gist.github.com/reggiechan74/12e89296f8574b338eac1576fea26a49),
write a fixed list once. These approaches are appropriate for user-authored
phrases, but neither supplies a lightweight remote refresh lifecycle.

[bhpark1013/claudenews](https://github.com/bhpark1013/claudenews) takes another
route: a Claude plugin shows several news sources in a dynamic status line and
can translate and summarize them. That is richer than a spinner, but it changes
the display surface and adds plugin-specific behavior that this focused CLI
does not need.

## Chosen Architecture

- Fetch the exact GeekNews latest page at `https://news.hada.io/new` rather than
  mixing its ordering with another feed.
- Parse HTML structurally and follow pagination for a configurable 1 to 50
  titles. The default is 10.
- Install async `SessionStart` and `UserPromptSubmit` exec-form hooks. Claude is
  never blocked by the network refresh.
- Fetch live on every hook event. A failed request leaves the currently applied
  settings untouched, and no daemon or persistent headline cache is used.
- Merge unrelated Claude settings and hooks, and use atomic replacement,
  and snapshot managed spinner values for uninstall.
- Optionally wrap sanitized titles and validated HTTP or HTTPS URLs in OSC 8
  hyperlinks while retaining a plain-text fallback.
- Ship one dependency-light Go binary with no runtime package manager.

Claude Code watches settings files for changes, so a successful background
refresh becomes available without restarting the running session. A verb
already selected for the current turn can remain in component state; the new
pool is reliable for subsequent selections.
