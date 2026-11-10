# ADR 0009: Transcripts from exact sources only

- **Status:** Accepted, 2026-11-08. Amended 2026-11-09 after the code reviews.
- **Decision:** 9 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user reads the transcript before an approval. A wrong transcript is worse than terminal text.
- Claude Code maps an agent to its session file through `~/.claude/sessions/<pid>.json`. The planners verified it on a live terminal.
- A Codex with `--no-daemon` keeps its rollout file open, so `lsof` gives an exact mapping.
- A plain `codex` runs its threads in a shared app-server. No process of the terminal holds the rollout. Herdr's own Codex launcher runs plain `codex`.
- Three review rounds tried a folder match for daemon-mode Codex. Each round found a case where it shows another session:
  - a terminal with no prompt yet, beside another Codex in the same folder
  - a Codex outside Herdr in the same folder
  - a session that exited but keeps its lock
  - a resumed thread with an old timestamp.
- A read of the last 2 MiB usually starts inside a JSON record.

## Decision

- Three sources, in order:
  1. the Claude sessions file
  2. the open rollout file of a Codex with `--no-daemon`
  3. terminal text.
- A Codex in daemon mode shows terminal text, with the hint "Start Codex with `--no-daemon` for the full transcript".
- Pages are aligned to records. Cursors are tied to the file and its inode.
- **Amended 2026-11-09:**
  - A read error, or a file with no record that parses, falls back to terminal text.
  - The first request pages back until it has an item.

## Alternatives

| Alternative                           | Why it lost                                                     |
| ------------------------------------- | --------------------------------------------------------------- |
| A folder match for daemon-mode Codex  | Three review rounds found cases where it shows another session. |
| Codex's `state_5.sqlite`              | Undocumented, needs SQLite, and has no terminal binding either. |
| A manual choice on the Mac            | Not needed: `lsof` gives an exact mapping for `--no-daemon`.    |
| The newest rollout by folder and time | A heuristic, not an identity.                                   |

## Consequences

- The user sets Herdr's Codex launcher to `["codex", "--no-daemon"]` once. `install.sh` prints this tip. It does not change Herdr settings.
- Terminal text is only the visible screen while the agent works or is blocked.
- The transcript files are undocumented. If their layout changes, terminal text shows.
- [Architecture: transcripts](../architecture.md#34-transcripts) gives the full rules.

## Sources

- [Technical design, decision 9 and section 3.6](../history/technical-design.md#36-files-and-formats)
- [Debate, final resolution](../history/debate.md#final-resolution-made-after-round-3-at-the-round-cap)
