# ADR 0023: Transcript text can be copied

- **Status:** Accepted, 2026-11-12. It amends [ADR 0014](0014-phone-key-and-privacy.md).
- **Decision:** 23. It is not in the decision table of the technical design.

## Context

- The user asked to copy text from the app.
- A copy puts text on the system clipboard, where other apps can read it.
- Android 13 and later shows a preview of each copy, unless the copy has `ClipDescription.EXTRA_IS_SENSITIVE`.

## Decision

- A long press on a transcript message selects its text. The user drags the handles, then taps Copy.
- Each copy from the transcript has `EXTRA_IS_SENSITIVE`, so Android does not show its text in the preview.

## Alternatives

| Alternative                          | Why it lost                                 |
| ------------------------------------ | ------------------------------------------- |
| A long press copies the full message | The user chose a selection, to copy a part. |
| No copy                              | The user asked for it.                      |

## Consequences

- The text that the user copies leaves the app. That is the user's choice for each copy.
- A selection covers one message. To copy more, copy each message.
