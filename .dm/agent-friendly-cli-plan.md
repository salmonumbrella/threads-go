# Agent-Friendly CLI Refactor Plan (threads-cli)

Date started: 2026-02-06

Goal: make `threads` maximally usable by an automated agent. That means fewer round-trips, fewer prompts, stable machine-readable output, and “desire paths” that accept what agents naturally paste (IDs, `#IDs`, URLs).

## Principles

- Stdout is for data. Stderr is for status/progress/errors.
- JSON mode is non-interactive by default (no prompts). `--yes` (or an alias) is required for destructive actions.
- Errors in JSON mode are structured and stable (agents branch on `kind` and `code`, not strings).
- Aliases optimize for common verbs (`ls`, `rm`, `show`, etc.) and reduce token tax.

## Phase 1: Output Contract (Completed)

- Add `--json` shortcut for `--output json`.
- Add `--no-color` shortcut for `--color never`.
- Add `--no-prompt` alias for `--yes`.
- Ensure `threads version -o json` emits JSON to stdout.

## Phase 2: JSON Errors + Stderr Hygiene (Completed)

- Route UI/status output to stderr when in JSON mode (so stdout remains machine-readable).
- Emit a structured JSON error envelope in JSON mode:
  - `error.kind` = `auth|rate_limit|validation|network|api|unknown`
  - includes API fields when available (`code`, `type`, `field`, `request_id`, `retry_after_seconds`)

## Phase 3: Confirmation Rules (Completed)

- In JSON output mode:
  - refuse to prompt for confirmation
  - require `--yes` / `--no-prompt` for destructive actions (delete/remove/unrepost/etc.)

## Phase 4: Desire Paths + Aliases (Completed)

- Add command aliases to reduce navigation and improve discoverability:
  - `auth a`, `auth login signin`, `auth token set-token`, `auth refresh renew`, `auth status st`, `auth list ls`, `auth remove rm|del`
  - `posts p`, `posts create new|add`, `posts get show`, `posts list ls`, `posts delete del|rm`, `posts carousel car`, `posts quote qt`, `posts repost boost`, `posts unrepost undo-repost`
  - `replies r`, `replies list ls`, `replies create new|add`, `replies hide rm|del`, `replies unhide restore`, `replies conversation thread`
  - `webhooks wh`, `webhooks subscribe sub|add`, `webhooks list ls`, `webhooks delete del|rm`
  - `users u`, `users get show`, `users lookup find|search`

## Next (Planned)

- Accept pasted Threads permalinks/URLs wherever an ID is accepted (posts/replies/users where feasible).
- Add `help-json` (machine-readable command discovery). (Completed)
- Consider `--emit id|url|json` for “find/search” style commands to make chaining trivial without `jq`.

## Phase 5: Help + ID Shorthands (Completed)

- Add `threads help-json [command...]` for machine-readable command discovery.
- Normalize common ID shorthands for agent inputs:
  - `#123`
  - `post:123` / `reply:123` / `user:123` (validated by context)

## Phase 6: URL + Best-Search Desire Paths (Completed)

- Accept pasted permalinks where post IDs are expected (e.g. `https://www.threads.net/t/<id>`).
- Add non-interactive search selection:
  - `threads search <query> --best --emit id|json|url`
- Let `threads users get @username` (and profile URLs) delegate to lookup.
