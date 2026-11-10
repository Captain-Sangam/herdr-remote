# ADR 0013: Background alerts from a foreground service

- **Status:** Accepted, 2026-11-08. Episode rule corrected 2026-11-10 (`6c8e729`).
- **Decision:** 13 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The user ruled out a relay server and a push service.
- Android 15 limits `dataSync` foreground services to 6 hours a day.
- Android 15 stops `BOOT_COMPLETED` from starting some foreground service types. `specialUse` is not on that list.
- A reviewer found that the plan had no permissions, no service type property and no restart after a reboot or an update.
- hc sends the status and the turn as two events. The turn goes up after the blocked status arrives. Before `6c8e729`, the turn was part of the episode key, so each block gave two alerts.

## Decision

- An opt-in foreground service of type `specialUse` holds `/v1/events`.
- It shows one notification for each blocked episode. An episode starts when an agent becomes blocked, or shows another `msg` while blocked. The turn is not part of it.
- After a reconnect, the snapshot shows the agents that are still blocked.
- `BOOT_COMPLETED` and `MY_PACKAGE_REPLACED` receivers start the service again if the toggle is on.
- Alerts come only while the service runs and is connected.

## Alternatives

| Alternative              | Why it lost                             |
| ------------------------ | --------------------------------------- |
| Firebase Cloud Messaging | Needs a server and Google credentials.  |
| A `dataSync` service     | Limited to 6 hours a day on Android 15. |

## Consequences

- No server outside the Mac.
- The service shows an ongoing notification, "Watching N agents".
- The app asks for an exemption from battery optimization. If notifications are denied, the toggle stays off and explains why.
- If the OS stops the service, alerts stop. The criterion says "while the service runs and is connected".
- On the phone tested, one alert came 3 to 4 s after a block, also after a reboot.

## Sources

- [Technical design, decision 13 and section 3.7](../history/technical-design.md#37-android-app)
- [Debate, plan review round 1](../history/debate.md#3-plan-review-round-1-blind)
