# ADR 0016: Spaces and agents from herdr-app ls

- **Status:** Accepted, 2026-11-08.
- **Decision:** 16 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- Spaces and tabs belong to the Herdr window, not to herdr-core.
- A space rename causes no hc event.
- Agents on remote machines are out of scope.
- Herdr.app can be closed while its terminals run.

## Decision

- `/v1/spaces` runs `herdr-app ls` on each request, and also on `created`, `removed` and `labels` events.
- It drops spaces whose `machine` is set.
- An agent is a live local terminal with `reported: true`. Its space comes from `herdr-app ls`, or else from its `herdr.space` label.
- If Herdr.app is not open, the API groups agents by their `where` label under "Herdr window closed".

As built:

- Agents in no space go under "Not in a space".
- Hidden children, from `sub_agents`, go in the space and tab of their parent.

## Alternatives

| Alternative  | Why it lost                                              |
| ------------ | -------------------------------------------------------- |
| A poll timer | A refresh on each request is simpler and always current. |

## Consequences

- A renamed space shows on the next `/v1/spaces` call.
- An open space screen does not add new agents. This is an accepted issue in [operations](../operations.md#known-issues).

## Sources

- [Technical design, decision 16 and the As built notes](../history/technical-design.md#as-built-2026-11-08)
- [Debate, planning round 2](../history/debate.md#2-planning-debate-round-2)
