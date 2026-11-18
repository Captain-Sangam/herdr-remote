# ADR 0020: The transcript folds the steps and follows the end

- **Status:** Accepted, 2026-11-10. [ADR 0026](0026-show-agent-text.md) amends the folds: all text shows, and only tool steps fold.
- **Decision:** 20. It is not in the decision table of the technical design.

## Context

- The transcript showed every tool call and every tool result. A turn of 20 steps filled many screens, so the answer was hard to find.
- The view followed new items only while the end showed. After a send, the open keyboard pushed the end out of view, so the user had to tap "Bottom ↓".
- A free scroll stopped in the middle of messages, and nothing showed where the view was in the transcript.

## Decision

- **Folds:** the prompts and each turn's answer show. The steps between them go into one row for each turn: "▸ 5 steps · Bash git status". A tap opens or closes the row.
  - The answer is the text after the last tool step. A turn that runs has no answer yet, so all of it folds, and the row shows the last call.
  - Thinking blocks never reach the phone ([API](../api.md)).
- **Follow:** while it is on, the view stays at the end when items come, the keyboard opens or the last fold grows.
  - A scroll up, "Load earlier" or opening a fold turns it off.
  - A send, "Bottom ↓" or a scroll to the end turns it on.
- **Steps:** a fling comes to rest on the nearest message start, with a haptic tick for each message that passes. A message taller than the screen scrolls freely.
- **Scroll bar:** a thin bar on the right shows the place in the loaded items while the user scrolls.

## Alternatives

| Alternative                                        | Why it lost                                                       |
| -------------------------------------------------- | ----------------------------------------------------------------- |
| Compose's snap fling (`rememberSnapFlingBehavior`) | It snaps inside a tall message too, so its middle cannot be read. |
| Show the text between tool calls                   | It is mostly "Let me check…". The fold keeps it one tap away.     |

## Consequences

- The scroll bar measures items, not pixels. A long answer moves the bar less than its height.
- The open folds reset when the user leaves the transcript tab.
