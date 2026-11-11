# ADR 0014: The phone key, the app lock and privacy

- **Status:** Accepted, 2026-11-08. Amended 2026-11-09 after the code reviews. Notifications changed by [ADR 0019](0019-alert-preview-and-actions.md). Copy added by [ADR 0023](0023-copy-transcript-text.md). Screenshots allowed by [ADR 0024](0024-allow-screenshots.md).
- **Decision:** 14 in the [decision table](../history/technical-design.md#31-decision-table).

## Context

- The phone's private key must stay in Android Keystore (SR-4).
- The first plan bound the key to an unlocked device. Then each TLS handshake fails while the phone is locked, which stops background alerts.
- A reviewer found that project data can leak through the recents screenshot, notification text on the lock screen, and logs.
- A code review found that the default notification settings showed the tab name on the lock screen.

## Decision

- **Key:** P-256 in Android Keystore, `PURPOSE_SIGN`, digests `SHA256` and `NONE`, `setUserAuthenticationRequired(false)`. Not bound to an unlocked device.
- **App lock:** after 5 minutes in the background, the app asks for BiometricPrompt (a strong biometric or the device credential) before it shows anything.
- **Screens:** the activity sets `FLAG_SECURE`.
- **Notifications:** private, with no action buttons.
- **Data:** transcripts and terminal bytes stay in memory. No build logs content.
- **Amended 2026-11-09:**
  - Both the private and the public notification text say "An agent needs attention". The tab name appears only in the app.
  - The application registers one observer on the process lifecycle for the app lock. No activity owns it.
- **Changed 2026-11-10 by [ADR 0019](0019-alert-preview-and-actions.md):** an unlocked phone shows the agent name and the prompt, with Approve and Deny. A locked phone shows the agent name and what it needs, with no prompt and no buttons.

## Alternatives

| Alternative                 | Why it lost                        |
| --------------------------- | ---------------------------------- |
| `setUnlockedDeviceRequired` | Stops background alerts.           |
| Full text in notifications  | Shows commands on the lock screen. |

## Consequences

- Alerts work while the phone is locked.
- The lock screen and the recents card show no project data.
- Screenshots of the app are black. Tests use `uiautomator dump`, and the docs use mockups ([design](../design.md)). Do not turn `FLAG_SECURE` off in any build.
- On a phone with no screen lock, the gate passes with a note.

## Sources

- [Technical design, decision 14 and the review amendments](../history/technical-design.md#review-2026-11-09)
- [Debate, planning round 2](../history/debate.md#2-planning-debate-round-2) and [plan review round 1](../history/debate.md#3-plan-review-round-1-blind)
