# ADR 0011: Questions and unknown prompts are view-only

- **Status:** Accepted, 2026-11-08.
- **Decision:** 11 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user scoped control to approve, deny and message.
- Question menus differ from prompt to prompt. A digit can pick the wrong answer.
- The Claude planner proposed digits 1 to 9 for questions. The Codex planner proposed Mac only.

## Decision

- Questions (`kind: question`), `auth` prompts, Pi and unknown apps are view-only.
- The phone shows the prompt and says "Answer this on the Mac".

## Alternatives

| Alternative   | Why it lost                           |
| ------------- | ------------------------------------- |
| Digits 1 to 9 | One wrong parse answers for the user. |

## Consequences

- An unknown prompt, or a known agent with changed prompt text, fails safe: it is view-only.
- The user answers questions and logins on the Mac (FR-12).
- The Pane design shows a view-only prompt in a blue "question" pane with no keys.

## Sources

- [Technical design, decision 11](../history/technical-design.md#31-decision-table)
- [Debate, summary of positions](../history/debate.md#6-summary-of-positions)
