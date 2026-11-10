# ADR 0002: Three hc connections and one subprocess helper

- **Status:** Accepted, 2026-11-08.
- **Decision:** 2 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The raw hc API includes dangerous methods, such as `machine.exec` and `file.write`.
- `terminal.send` and `terminal.keys` wait until nobody types at the Mac for about 3 s. They must not block reads.
- After `events.subscribe`, a connection carries only notifications. It cannot serve other calls.
- The daemon also needs other programs: `herdr-app`, `lsof`, `ps`, `ioreg` and `tailscale`.
- In round 2, the Claude planner asked for a separate connection for actions, and for one helper with fixed arguments, with tests.

## Decision

- The daemon keeps three long-lived `hc connect` subprocesses: reads, events and actions.
- It runs every other program through one helper (`daemon/run.go`):
  - It accepts only fixed program paths, found at start: `hc`, `herdr-app`, `lsof`, `ps`, `ioreg`, `tailscale`.
  - It accepts only fixed argument shapes. It uses no shell, and a timeout.
  - It gives each program a bare environment: `HOME`, `PATH=/usr/bin:/bin`, `LANG`.
- There is no generic pass-through to hc. Typed functions in `daemon/hc.go` make each call.

## Alternatives

| Alternative                           | Why it lost                                                                                             |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| A JSON-RPC proxy with a method filter | One filter bug exposes everything.                                                                      |
| One shared `hc connect` connection    | An action that waits for the keyboard blocks reads. A subscribed connection carries only notifications. |
| Direct access to the hc socket        | Not a shipped interface.                                                                                |

## Consequences

- The Go tests use a fake `hc` binary that records each request. They fail on a method outside the list, `force: true`, a `caller`, another key, a `machine/` target, and a program outside the helper list.
- The bare environment means that the daemon never inherits `HC_TERMINAL_ID`. So its messages stay untagged ([ADR 0012](0012-messages.md)).
- The hc client reconnects with backoff. While hc is lost, the API returns 503.

## Sources

- [Technical design, decision 2 and step 2](../history/technical-design.md#31-decision-table)
- [Debate, planning round 2](../history/debate.md#2-planning-debate-round-2)
