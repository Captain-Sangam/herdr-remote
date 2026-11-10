# ADR 0010: The approval guard

- **Status:** Accepted, 2026-11-08. Amended 2026-11-09 after the code reviews.
- **Decision:** 10 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- Probes verified the keys on command and file-edit prompts:
  - Claude: `1` approves at any cursor position. `esc` denies.
  - Codex: `y` approves. `esc` denies.
  - "No" was option 4 in one Claude prompt and option 3 in another. A digit is not a safe deny.
- `terminal.keys` has no status condition. A check followed by a key press can never be fully atomic.
- Both planners showed the same failure: two phones, or one retry, pass the state check before the agent redraws. Then a second key lands.
- hc holds a key while someone types at the Mac. When it lands, the prompt can be different.
- A daemon restart loses state in memory.
- `ioreg` reports `HIDIdleTime`, the time since the last keyboard or mouse input. It can serve as a "Mac in use" guard.
- Request IDs with no time cannot tell a replay from a new request after the daemon forgets them.

## Decision

The guard runs in this order:

1. Only `blocked` with `permission`, for `claude-code` or `codex`.
2. A known request ID returns its recorded result.
3. A per-terminal lock.
4. The same turn, prompt hash and option markers.
5. A consumed marker.
6. The Mac-in-use guard (`mac_idle_seconds`, default 10, 0 turns it off). An `ioreg` failure refuses.
7. A fsynced `dispatching` audit line.
8. Exactly one key.

More rules:

- Request IDs are UUIDv7. The daemon refuses an ID more than 5 minutes from its clock, and remembers each ID for 10 minutes.
- A rate limit of 2 actions a second for each device keeps the audit log small, so a rotation cannot drop a valid ID.
- The result is `applied`, `sent_unconfirmed`, `uncertain` (Mac input during the call) or `outcome_unknown`.
- **Amended 2026-11-09:** the daemon waits up to 120 s for `terminal.keys`.

## Alternatives

| Alternative                     | Why it lost                                                       |
| ------------------------------- | ----------------------------------------------------------------- |
| A separate durable store        | The audit log already is one.                                     |
| `enter` on a selected Yes       | Depends on the cursor.                                            |
| A digit to deny                 | The option number of "No" differs between prompts.                |
| No guard                        | A key could approve a prompt that nobody saw.                     |
| A 10-minute cache of random IDs | A replay after the cache forgets the ID looks like a new request. |
| Leave the race open             | The idle guard closes the main path. The rest is written down.    |

## Consequences

- The marker and request IDs stop retries and second phones. The audit log keeps them through a restart.
- WARNING: A residual race remains. The guard does not see SSH input or input from other agents. [Architecture: residual risk](../architecture.md#64-residual-risk) gives the details.
- A device that keeps the Mac "in use", such as a mouse jiggler, needs `mac_idle_seconds` set to 0.
- [Architecture: approvals](../architecture.md#35-approvals) gives the full rules. The comments in `daemon/actions.go` use its rule numbers.

## Sources

- [Technical design, decision 10 and section 3.5](../history/technical-design.md#35-approval-guard)
- [Debate, how each dispute was resolved](../history/debate.md#how-each-dispute-was-resolved), [round 2 resolution](../history/debate.md#resolution) and [final resolution](../history/debate.md#final-resolution-made-after-round-3-at-the-round-cap)
