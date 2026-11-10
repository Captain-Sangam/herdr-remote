# ADR 0012: Messages arrive as plain prompts, at most once

- **Status:** Accepted, 2026-11-08.
- **Decision:** 12 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user asked to send a message to an agent from the phone.
- Text with control characters can break out of a paste.
- A retry after a lost response can send the prompt twice.
- A reviewer asked if hc tags the text as `<hc from=…>`, which agents treat as a report. A probe showed that a send from outside a terminal, with no `caller`, arrives untagged.
- A SHA-256 of a short text, such as "yes", is easy to guess.

## Decision

- `POST /message {request_id, text}` calls `terminal.send` with `force: false` and no `caller`. The text arrives as a plain prompt.
- Targets are live terminals with `reported: true`. A blocked agent refuses the message.
- The daemon removes C0 and C1 control characters, but keeps `\n` and `\t`. The limit is 4000 characters.
- A fsynced `dispatching` audit line `{request_id, terminal, length}` comes first. It has the text length, never the text or a hash of it. If the write fails, the daemon returns 503 and sends nothing.
- Delivery is at most once for each `request_id`.

## Alternatives

| Alternative                 | Why it lost                                             |
| --------------------------- | ------------------------------------------------------- |
| Raw keys                    | Out of scope.                                           |
| A `caller` on the send      | The text arrives tagged as a report from another agent. |
| A hash of the text in audit | A hash of a short text is easy to guess.                |

## Consequences

- The filter stops input that tries to break out of the paste.
- The request ID stops a retry that sends the prompt twice.
- The result can be `outcome_unknown`. Then the phone says so and never sends again by itself.

## Sources

- [Technical design, decision 12](../history/technical-design.md#31-decision-table)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind) and [final resolution](../history/debate.md#final-resolution-made-after-round-3-at-the-round-cap)
