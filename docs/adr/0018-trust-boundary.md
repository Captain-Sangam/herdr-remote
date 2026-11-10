# ADR 0018: Processes of the same user are trusted

- **Status:** Accepted, 2026-11-08.
- **Decision:** 18 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The Astra reviewer found that any process of the same macOS user can read `server.key`, read `devices.json` or run `herdr-remoted pair allow`.
- File permissions cannot separate processes of one user.
- Such processes, agents included, can already drive hc directly.

## Decision

- Processes that run as the same macOS user are trusted.
- The design protects against:
  - the network (the LAN and the tailnet)
  - leaked QR codes
  - lost phones
  - second phones and replays
  - hostile bytes in terminal output or transcripts.

## Alternatives

| Alternative                                       | Why it lost                                                     |
| ------------------------------------------------- | --------------------------------------------------------------- |
| A separate macOS user or a Keychain-protected key | Real isolation, but much more setup. Agents can still drive hc. |

## Consequences

- A clear statement prevents false assurance. File modes on the state files are hygiene, not a boundary.
- An agent in auto mode could run `pair allow`. The convention is that a helper agent never pairs a device without the user.
- [Architecture: security and privacy model](../architecture.md#6-security-and-privacy-model) builds on this boundary.

## Sources

- [Technical design, decision 18](../history/technical-design.md#31-decision-table)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind)
