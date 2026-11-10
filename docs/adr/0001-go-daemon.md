# ADR 0001: One Go daemon is the only network listener

- **Status:** Accepted, 2026-11-08.
- **Decision:** 1 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- Herdr.app and herdr-core are closed source. herdr-remote can use only their shipped interfaces: `hc`, `herdr-app` and mods.
- The phone needs one endpoint on the Mac that it can reach over the LAN or Tailscale.
- The criteria, in order, were: security, works with Herdr as shipped, the smallest system, a responsive live view.
- Go and Java 17 were installed on the Mac. The Android SDK was not.
- Both planners proposed the same shape in round 1: one Go daemon as the only piece that faces the network.

## Decision

- One Go daemon, `herdr-remoted`, runs as a per-user LaunchAgent (`dev.herdr-remote`).
- It is the only network listener. The phone reaches nothing else on the Mac.
- It uses the Go standard library plus one QR encoder (`github.com/skip2/go-qrcode`).

## Alternatives

| Alternative               | Why it lost                                                         |
| ------------------------- | ------------------------------------------------------------------- |
| A Swift daemon            | The same function, with more macOS-only code.                       |
| A Herdr mod as the server | Mods have no isolation, and Herdr gives a mod a 250 ms quit budget. |
| A relay server            | The user ruled out a relay before the debate.                       |

## Consequences

- One static binary to build and install. `scripts/install.sh` builds it to `~/.local/bin/herdr-remoted`.
- launchd starts it at login and again after a crash (`RunAtLoad`, `KeepAlive`).
- It runs as the user, so processes of the same user are inside the trust boundary ([ADR 0018](0018-trust-boundary.md)).
- All access control lives in one place: the daemon.

## Sources

- [Technical design, section 1 and decision 1](../history/technical-design.md#1-outcome)
- [Debate, planning round 1](../history/debate.md#1-planning-debate-round-1-blind)
