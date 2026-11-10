# ADR 0004: Two TLS listeners with a pinned key

- **Status:** Accepted, 2026-11-08.
- **Decision:** 4 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- Only paired phones may use the API. A new phone must still complete TLS once, to enroll.
- The TLS handshake cannot see the request path.
- Go does not call `VerifyPeerCertificate` on a resumed session. `VerifyConnection` runs on every handshake. Both reviewers found that a revoked phone could resume a session.
- The Mac has different addresses on the LAN and on the tailnet. A certificate name check fails on one of them.
- In plan review round 2, the Astra reviewer found that a device-store check on both listeners rejects every new phone. So the first pairing always failed.

## Decision

- Two TLS listeners share one P-256 self-signed server key. The phone pins its SPKI SHA-256.
  - **Paired listener, port 7766:** always on.
  - **Enrollment listener, port 7767:** exists only while a pairing window is open. It serves only `POST /v1/enroll`.
- Go settings on both: `MinVersion: tls.VersionTLS13`, `ClientAuth: tls.RequireAnyClientCert`, `SessionTicketsDisabled: true`.
- All client checks are in `VerifyConnection`, never in `VerifyPeerCertificate`.
- Checks for each listener:
  - **7766:** the client SPKI hash must be in the device store. Each request looks the device up again.
  - **7767:** the client must present one well-formed P-256 certificate. The token, bound to that SPKI, is the authorization. The device store is not used.

## Alternatives

| Alternative                              | Why it lost                                               |
| ---------------------------------------- | --------------------------------------------------------- |
| One listener with path middleware        | Unknown keys reach the HTTP parser while pairing is open. |
| The device-store check on both listeners | A new phone is not in the store, so it can never enroll.  |
| Certificate names and hostname checks    | The Mac address differs on the LAN and the tailnet.       |
| A relay                                  | The user ruled it out.                                    |

## Consequences

- The paired listener never accepts an unknown key.
- Tickets off, and checks in `VerifyConnection`, stop a revoked phone that tries to resume a session.
- A key pin makes LAN and Tailscale addresses work the same way.
- "Only paired phones complete the handshake" is true for port 7766 only. Port 7767 is the exception: it opens only in a window that the user starts, and the one-time token authorizes it.
- The phone uses its own trust manager and hostname verifier, which check only the pin.

## Sources

- [Technical design, decision 4](../history/technical-design.md#31-decision-table)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind) and [round 2](../history/debate.md#4-plan-review-round-2)
