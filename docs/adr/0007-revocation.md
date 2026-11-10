# ADR 0007: Revocation ends access at once

- **Status:** Accepted, 2026-11-08. Amended 2026-11-09 after the code reviews.
- **Decision:** 7 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- A lost phone must lose access at once, with its open streams (FR-4: within 2 s).
- A revoked phone must not connect again, also with a resumed TLS session (SR-5).
- A code review found that a failed write of `devices.json` could leave a revoked device in memory.

## Decision

- `herdr-remoted devices rm <id>`, or Revoke in the overlay, deletes the device.
- The daemon closes the device's connections and ends its live-view subprocesses.
- **Amended 2026-11-09:** a revoke marks the device as revoked in memory, closes its connections and live views, and then writes `devices.json`. A failed write keeps the mark and returns the error, so a retry writes again.

## Alternatives

| Alternative             | Why it lost |
| ----------------------- | ----------- |
| Certificate expiry only | Too slow.   |

## Consequences

- Each request looks the device up again ([ADR 0004](0004-two-tls-listeners.md)). So a revoke also stops a request on an open connection.
- No session tickets and checks in `VerifyConnection` stop a resumed session.
- A revoke does not open the enrollment port.
- On the phone tested, the Mac dropped the phone in 0.21 s.
- A revoked phone shows "Not connected" and retries. It does not say that it was removed. This is an open issue in [operations](../operations.md#open).

## Sources

- [Technical design, decision 7 and the review amendments](../history/technical-design.md#review-2026-11-09)
