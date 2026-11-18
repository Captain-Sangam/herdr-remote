# Build: usage limits on the spaces screen

The user locked the design: the "Side meter" variation in `design/usage.html`, plus a manual refresh key. Build it on Android, with the daemon endpoint and the docs.

## Rules

- Read `AGENTS.md` first. Commit on `main`, one commit for each step below. Never push.
- Stage only your own files. Do not commit `.agents/`.
- Before each commit, run the three checks in `AGENTS.md`: the daemon tests, the app tests and build, and `scripts/format.sh --check`.
- Do not print or open `server.key`, `~/.codex/auth.json` or any credential file.
- Out of scope: iOS (paused), the agent screen, alerts and notifications.

## Step 1: lock the mockup

- In `design/usage.html`, keep only the Side meter variation. Remove the other three.
- Add a refresh key to the panel, in the Pane style (a Quiet key). Show its "Loading…" state.
- Keep the state switcher (normal, near limit, limit reached, stale, reset time passed, no data), dark and light.
- Commit: `design: lock the usage side meter with a refresh key`.

## Step 2: the daemon

### Data sources (read-only)

- **Claude Code:** a JSON file that the user's Claude Code status line writes (`~/.claude/statusline.sh`, line 70).
  - Path: a new `config.json` setting `claude_usage_file` (`daemon/daemon.go:20`). Default: `~/.cache/herdr-slack/claude-usage.json`.
  - Shape: `{"at": <unix s>, "rate_limits": {"five_hour": {"used_percentage": 72, "resets_at": <unix s>}, "seven_day": {...}}}`.
  - Read at most 64 KiB. A missing, bad or too large file gives no Claude data.
- **Codex:** the newest `rollout-*.jsonl` (by modification time) under `~/.codex/sessions/` (folders `YYYY/MM/DD`).
  - Read only the tail (at most 1 MiB). Find the last line with `"type":"event_msg"`, `payload.type == "token_count"` and a `payload.rate_limits` that is not null.
  - Use `rate_limits.primary` and `rate_limits.secondary` (each `{used_percent, window_minutes, resets_at}` in unix s, either can be null), `rate_limits.plan_type`, and the line's `timestamp` as the read time.
  - The real path must be under `~/.codex/sessions` (reuse `underRoot` in `daemon/transcript.go`). Read nothing else in `~/.codex`.

### API

- `GET /v1/usage`, on the API port with the same mutual TLS as the other GETs. No audit line.
- Response: `{"claude": <source or null>, "codex": <source or null>}`.
  - A source: `{"as_of": <unix ms>, "plan": "<string, optional>", "windows": [{"name": "5h", "used_percent": 72, "resets_at": <unix ms>}]}`.
  - Window names: `five_hour` is `5h`, `seven_day` is `7d`. For Codex, name the window from `window_minutes`: 300 is `5h`, 10080 is `7d`, else hours or days.
- Logs stay free of content (SR-9).

### Tests

- Go tests with fixtures in a temp folder, like `daemon/transcript_test.go`:
  - the Claude file: good, missing, bad JSON, too large
  - the Codex tail: the last `token_count` wins, a null `rate_limits` is skipped, no rollout at all, a path outside the root is refused
  - the window names.

### Docs, in the same commit

- A new ADR, `docs/adr/0027-usage-limits.md`: the two sources, read-only, no credentials and no network calls, the `claude_usage_file` setting, and the freshness limit (the Claude numbers update only while a Claude Code session draws its status line). Add its row to `docs/README.md`.
- `docs/api.md`: the endpoint and its shape.
- `docs/interfaces.md`: the two facts (the status line gets `rate_limits`; Codex writes `token_count` events with `rate_limits`).
- `docs/deployment.md`: the `claude_usage_file` setting, and that the status line must write the file.
- Commit: `feat(daemon): GET /v1/usage from the Claude status line file and Codex rollouts`.

## Step 3: the Android app

- **Icon:** a gauge icon in the `~/spaces` header (`PaneHeader` at `android/app/src/main/java/dev/herdrremote/ui/ListScreens.kt:149`), beside "● online". It follows the state as in the mockup: a warning mark at 90% or more, the error colour at 100%.
- **Panel:** a tap opens the narrow panel from the right over the spaces screen, as in the mockup. A tap outside, Back, or `[x]` closes it.
- **Panel content**, as in the mockup:
  - "Mac read HH:MM · N min ago"
  - one group for each source, with a bar, the used percentage and "Resets …" for each window
  - the Codex plan
  - the note "Claude Code updates only during a session."
  - the refresh key, which gets `/v1/usage` again and shows "Loading…" while it waits.
- **States:**
  - near the limit (90% or more): the warning colour
  - limit reached (100%): the error colour and the reset time
  - stale (`as_of` older than 30 min): dim, with the age
  - reset time passed: the window says "reset" and shows no old percentage
  - no data: "No data" for that source.
- **When the phone gets the data:** when the spaces screen opens, when the panel opens and on refresh. No polling.
- The data stays in memory.
- **Tests:** a JVM test for the state logic as a pure function (near, reached, stale, reset passed, no data).
- **Docs, in the same commit:** `docs/product.md` (one new FR line), `docs/design.md` (the icon and the right panel).
- Commit: `feat(android): usage limits icon and side panel on the spaces screen`.

## When you are done

End your reply with: the commits, the check results, and anything that you did not build or could not check.
