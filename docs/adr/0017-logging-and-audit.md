# ADR 0017: Content-free logs and an audit log

- **Status:** Accepted, 2026-11-08.
- **Decision:** 17 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user needs a record of what a phone did, for example a lost phone.
- The first plan said nothing about logs. A reviewer asked that tokens, codes and transcript text never appear in them. The device name from the phone is attacker-controlled.
- A reviewer asked that the debug build also logs no content.
- The approval guard must rebuild its request IDs and markers after a restart ([ADR 0010](0010-approval-guard.md)). So the audit line needs the prompt hash and the decision, and must be durable before the key.

## Decision

- `audit.log` records every pairing event, revocation, decision, refusal and message.
- `dispatching` lines are fsynced before any key or send. If the write fails, the daemon refuses.
- On start, the daemon rebuilds request IDs and markers from `audit.log` and `audit.log.1`.
- Logs never contain tokens, codes, keys, transcript text, terminal bytes or message text. A message line has only the length.
- The device name is limited to 1 to 64 printable characters.

## Alternatives

| Alternative                              | Why it lost                          |
| ---------------------------------------- | ------------------------------------ |
| No audit                                 | No way to see what a lost phone did. |
| A separate durable store for request IDs | The audit log already is one.        |

## Consequences

- The audit log is the durable store of the approval guard.
- A refusal line carries the reason in `result`.
- The daemon sets `daemon.log` to mode 0600 at start, because launchd makes it 0644.
- [Operations: audit log](../operations.md#audit-log) gives the line format.

## Sources

- [Technical design, decision 17 and section 3.6](../history/technical-design.md#36-files-and-formats)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind) and [round 2](../history/debate.md#4-plan-review-round-2)
