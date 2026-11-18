# ADR 0026: The transcript shows all the agent's text

- **Status:** Accepted, 2026-11-18. It amends [ADR 0020](0020-transcript-folds-and-scroll.md).
- **Decision:** 26. It is not in the decision table of the technical design.

## Context

- ADR 0020 folded all of a turn up to its last tool step. The text between tool calls went into the fold too.
- The user said that the fold hid text that must show. Agents now write progress notes between tool calls, for example "Round one is in: both debaters answered". A turn that runs showed no text at all.
- The answers showed their markdown marks as written: `**bold**`, `` `code` `` and `# heading`. In the monospace font, the marks looked like code and made the text hard to read.

## Decision

- **Folds:** every text shows. Each run of tool calls and results between two texts goes into one fold: "▸ 3 steps · Bash git status".
- **Markdown:** the messages get light markdown. `# heading` and `**bold**` show bold, and `` `code` `` shows in the accent colour. Their marks go.
  - Lists, tables and fenced code stay as written.
  - Tool calls, tool results and terminal text stay as written, because their marks are part of the output.

## Alternatives

| Alternative                            | Why it lost                                                        |
| -------------------------------------- | ------------------------------------------------------------------ |
| Keep one fold for each turn (ADR 0020) | The user said that it hid text that must show.                     |
| A markdown library                     | Three marks cover most of the problem. The app adds no dependency. |

## Consequences

- A turn with many notes has more rows. Each note is short, so the answer is still near the end.
- A copy from a message has no marks.
