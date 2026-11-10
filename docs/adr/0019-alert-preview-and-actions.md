# ADR 0019: Alert previews, Approve from the notification, and finish alerts

- **Status:** Accepted, 2026-11-10. Changes [ADR 0013](0013-background-alerts.md) and [ADR 0014](0014-phone-key-and-privacy.md).
- **Decision:** 19. It comes after the decision table in the technical design.

## Context

- The user asked for three changes to the alerts:
  - an alert when an agent finishes, not only when it is blocked
  - a preview of the prompt in the expanded notification
  - Approve from the notification, with no need to open the app.
- ADR 0014 kept the text generic and the notifications without buttons. A code review found that the default Android lock screen setting shows the private text.
- Herdr reports `done` only when nobody looked at the agent. Otherwise it reports `idle`.
- The ongoing notification changed its text for each new agent count and each reconnect. Each update brought back a notification that the user dismissed. The user saw this as random alerts.

## Decision

- **Finish alerts:** an agent that leaves `working` for `done`, `idle` or `error` alerts once, on the channel "Agent finished". So does an agent that goes from `blocked` to `done`. The first snapshot after a start gives no finish alert.
- **Prompt preview:** the watch service gets `/v1/terminals/{t}/prompt` for each blocked episode. The expanded notification shows `msg` and the prompt block.
- **Approve and Deny:**
  - The notification has Approve and Deny when the prompt is actionable.
  - Each button sends the `turn` and the `prompt_hash` of the prompt that the notification shows. The daemon checks them as for a tap in the app ([ADR 0010](0010-approval-guard.md)).
  - Each button has `setAuthenticationRequired(true)`. The phone must be unlocked to use it.
  - The service never retries. The notification shows the result in the words of the app.
  - After a refusal (for example `mac_in_use`), if the agent is still blocked, the alert comes back with its buttons and the reason on top.
- **Lock screen and watch:** a locked phone shows only the agent name and what it needs. There is no prompt and there are no buttons.
  - Examples: "Herdr Remote / watch-probe needs approval", "Niti / 2 finished".
  - A Galaxy Watch gets the same alert, because it mirrors the phone. The phone is usually locked when the user looks at the watch.
  - The service shows the redacted alerts when the screen goes off.
  - When the screen comes on, the service checks the keyguard each 500 ms. It shows the full alerts again when the phone is unlocked. `ACTION_USER_PRESENT` did not reach the app on Android 17 (the S26).
  - So SR-11 holds with any lock screen setting.
- **One notification for each agent:** a new alert replaces the agent's last one. An agent that leaves `blocked` some other way loses its alert, so a stale Approve does not stay.
- **Ongoing notification:** the service posts "Herdr Remote is watching" once and never updates it.
- **Start of watching:** the service runs the agents that the app knows already through the alert rules when it subscribes. The app can open the event stream before the service, and then the service misses the first snapshot.

## Alternatives

| Alternative                                   | Why it lost                                                                           |
| --------------------------------------------- | ------------------------------------------------------------------------------------- |
| Only `VISIBILITY_PRIVATE` for the preview     | The default lock screen setting shows the private text.                               |
| A `BroadcastReceiver` for the buttons         | An action can wait 150 s. A receiver has about 10 s.                                  |
| Buttons on the redacted alert                 | The user could approve a prompt that they did not see.                                |
| Generic text on the lock screen and the watch | The watch said only "An agent needs attention". The user chose to see the agent name. |
| Approve only in the app                       | The user asked for Approve in the notification.                                       |

## Consequences

- The lock screen and the watch show agent names (the space and the tab). SR-11 changed to allow this on 2026-11-10, at the user's request. Before, they showed only "An agent needs attention" or "An agent finished".

- An unlocked phone can approve without the app lock. The phone unlock is the only check. This is weaker than the app lock ([ADR 0014](0014-phone-key-and-privacy.md)), which asks again after 5 minutes.
- The notification shade of an unlocked phone shows agent names and commands.
- The user can turn off the "Background watching" channel in the Android settings. Watching and alerts continue.

## Sources

- The user's request on 2026-11-10.
- [ADR 0013](0013-background-alerts.md) and [ADR 0014](0014-phone-key-and-privacy.md).
