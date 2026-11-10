# ADR 0005: HTTP with SSE and chunked streams

- **Status:** Accepted, 2026-11-08. Amended 2026-11-09 after the code reviews.
- **Decision:** 5 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The live view is watch-only. The phone never sends input to the terminal.
- An idle terminal sends no bytes. hc sends a heartbeat only every 15 s. OkHttp's default read timeout is 10 s.
- After a network change or a Mac sleep, a socket can stay half-open. With no read timeout, the phone never finds out, and background alerts stop with no sign.
- `/v1/enroll` long-polls for up to 120 s, so the default timeout aborts it before Allow.
- An action that the phone retries by itself can press a key twice.

## Decision

- HTTP on the TLS listeners:
  - JSON for reads and actions
  - Server-Sent Events (SSE) for `/v1/events`, with a comment line every 10 s
  - a chunked `application/octet-stream` body for the live view.
- Phone read timeouts:
  - SSE: 30 s, then reconnect
  - live view: none. The phone reopens each live view when SSE reconnects.
  - `/v1/enroll`: 130 s
  - actions: 150 s (amended)
  - other calls: 15 s.
- Action calls set `retryOnConnectionFailure(false)`.
- The listeners use TCP keepalive (the Go default, 15 s).
- **Amended 2026-11-09:**
  - An action changes endpoint only if the connection failed before the request was sent. After that, the app never retries.
  - Any failure that is not an explicit daemon reply shows "Outcome unknown. Check the transcript before you send again".
  - The app calls `/v1/hello` on each SSE connect, and after `expired_request`, without a re-post.

## Alternatives

| Alternative         | Why it lost                                              |
| ------------------- | -------------------------------------------------------- |
| WebSocket           | Needs a Go library and adds nothing for one-way streams. |
| No SSE read timeout | A dead socket after a network change is never found.     |

## Consequences

- The daemon needs only the Go standard library for HTTP.
- The SSE keepalive gives a liveness signal for the whole connection. The phone reconnects after a network change.
- The phone parses SSE by hand over OkHttp.
- The phone never sends an action twice by itself ([ADR 0010](0010-approval-guard.md), [ADR 0012](0012-messages.md)).

## Sources

- [Technical design, decision 5 and the review amendments](../history/technical-design.md#review-2026-11-09)
- [Debate, planning round 2](../history/debate.md#2-planning-debate-round-2) and [plan review round 1](../history/debate.md#3-plan-review-round-1-blind)
