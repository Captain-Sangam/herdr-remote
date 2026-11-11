# ADR 0003: A fixed list of allowed hc calls

- **Status:** Accepted, 2026-11-08. Amended 2026-11-12: [ADR 0021](0021-answer-questions.md) adds the keys `2` to `9` and `tab`.
- **Decision:** 3 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user scoped control before the debate: watch, approve or deny, and send a message. No raw keyboard into the terminal.
- The phone must not be able to run commands, read files or start programs on the Mac (SR-2).
- Remote hc machines are out of scope.

## Decision

The daemon may use only:

- `hello`, `terminal.list`, `terminal.get`, `terminal.read`, `events.subscribe`
- `terminal.send` with `force: false` and no `caller`
- `terminal.keys` with exactly one key, called only from the approval code: `1` to `9`, `y`, `esc` or `tab` (amended by [ADR 0021](0021-answer-questions.md))
- `hc attach --observe`.

A target must:

- match `^t[0-9a-z]+$`, with no `machine/` prefix
- be `live`
- have `reported: true`.

## Alternatives

| Alternative                      | Why it lost                     |
| -------------------------------- | ------------------------------- |
| `terminal.interrupt`             | Out of scope, by user decision. |
| Keyboard input into the terminal | Out of scope, by user decision. |

## Consequences

- It is the minimum for the requested screens. Each check fails toward a refusal.
- Interrupts, keyboard input, and the start or stop of agents stay on the Mac.
- `allowedMethods` in `daemon/hc.go` holds the list. A new feature that needs another method needs a new ADR.

## Sources

- [Technical design, decision 3](../history/technical-design.md#31-decision-table)
- [Debate, the brief and the user's decisions](../history/debate.md#0-the-brief-and-the-users-decisions)
