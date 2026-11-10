# ADR 0006: QR pairing with a code and Allow on the Mac

- **Status:** Accepted, 2026-11-08.
- **Decision:** 6 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user asked for pairing by a QR code that Herdr shows.
- A QR code can leak through a screen share or a photo.
- A mod's `exec` has a limit of 30 s. A blocking call that waits for the scan breaks the overlay.
- A mod's `image()` and `svg()` load only files. An SVG QR code writes the token to disk.
- Each mod redraw redraws the full Herdr window. A countdown redraws it every second.
- In an early plan, opening the overlay opened a pairing window. So a revoke of a lost phone also opened port 7767.
- The Codex planner proposed a Mac-side confirmation, so a token alone is not enough.

## Decision

- The overlay opens to the device list. Only **Pair a phone** opens a 120 s window and port 7767.
- The overlay draws the QR code with run-length `div()` rows. It writes no file. It shows the expiry as a clock time, with no countdown redraw.
- The token is 32 random bytes in memory. It binds to the first client key that uses it.
- The phone and the Mac show the same 6-digit code, made from both public keys.
- The user clicks Allow on the Mac. Deny, Cancel, a closed overlay or expiry stores nothing.
- The overlay polls `pair status --json` each second. Each call returns at once.

## Alternatives

| Alternative                 | Why it lost                                          |
| --------------------------- | ---------------------------------------------------- |
| A token only                | A QR code seen on a screen share is enough to pair.  |
| A hidden SVG file           | Avoids the mod reload, but writes the token to disk. |
| A blocking `pair wait` call | `exec` stops at 30 s, before a slow scan ends.       |
| A countdown in the overlay  | It redraws the full Herdr window every second.       |

## Consequences

- A leaked QR code alone enrolls nothing. Pairing needs the user at the Mac.
- One QR code pairs at most one phone.
- An agent must not run `pair allow` for the user.
- The Go and the JVM tests share the code vectors in `testdata/pairing-vectors.json`.
- [Architecture: pairing](../architecture.md#31-pairing) gives the full rules.

## Sources

- [Technical design, decision 6 and section 3.3](../history/technical-design.md#33-pairing-flow)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind) and [round 2](../history/debate.md#4-plan-review-round-2)
