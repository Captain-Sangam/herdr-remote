# ADR 0024: Screenshots are allowed, and the recents card stays blank

- **Status:** Accepted, 2026-11-12. It amends [ADR 0014](0014-phone-key-and-privacy.md).
- **Decision:** 24. It is not in the decision table of the technical design.

## Context

- `FLAG_SECURE` made every screenshot of the app black. So nobody could check a layout by eye from a screenshot, or share one.
- The user said that screenshots do not have to be blocked.
- `FLAG_SECURE` also blanked the recents card (SR-11). Android 13 and later can blank the recents card alone, with `Activity.setRecentsScreenshotEnabled(false)`.

## Decision

- On Android 13 and later, the activity calls `setRecentsScreenshotEnabled(false)` and does not set `FLAG_SECURE`. Screenshots and screen recordings show the app.
- Before Android 13, the activity keeps `FLAG_SECURE`, because only it blanks the recents card there.
- SR-11 changes: the recents card shows no content. Screenshots can show content.

## Alternatives

| Alternative                     | Why it lost                                                      |
| ------------------------------- | ---------------------------------------------------------------- |
| Keep `FLAG_SECURE`              | The user does not need it, and it blocks checks by eye.          |
| Drop the recents protection too | The recents card shows the screen to anyone who holds the phone. |

## Consequences

- A screenshot, a screen recording or screen sharing can show transcripts and prompts. That is the user's choice.
- `adb exec-out screencap` works for the on-device checks.
