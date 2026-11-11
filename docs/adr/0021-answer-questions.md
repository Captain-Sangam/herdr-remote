# ADR 0021: The phone answers Claude Code questions

- **Status:** Accepted, 2026-11-12. It replaces [ADR 0011](0011-view-only-prompts.md) for Claude Code questions.
- **Decision:** 21. It is not in the decision table of the technical design.

## Context

- An agent asked a multiple-choice question. The phone said "Answer this on the Mac", and the message box stayed off while the agent waited. The user could do nothing from the phone.
- ADR 0011 made questions view-only, because a digit from a wrong parse can pick the wrong answer.
- A probe on Claude Code 2.1.295 showed how the question dialog works ([interfaces](../interfaces.md#claude-code-questions)):
  - Each option has a number, and its digit acts on it at once.
  - A single choice goes to the next question, or ends the dialog after the last one.
  - A multi-select digit toggles a box. `tab` goes to the next question, and then to a review screen with "1. Submit answers" and "2. Cancel".
  - "Chat about this" declines the question. The agent then stops, so the user can send a message.
  - "Type something" opens a text field. Only the Mac keyboard can fill it.

## Decision

- **Scope:** Claude Code questions are actionable. Codex questions, `auth` prompts and other agents stay view-only.
- **Prompt:** `GET /prompt` reads the screen again for each call. It returns:
  - `block`: the dialog, from the question tabs to the end of the screen, without the rule lines. The prompt hash covers it.
  - `question`: the part of `block` above the options. The phone shows it once, and the options only as keys.
  - `options`: a key for each numbered option on the screen, with the description under it as `detail`. The numbers must count 1, 2, 3 with no gap, or there are no options.
  - `tab` (Next) when the options have check boxes.
  - No key for "Type something".
- **Answer:** `POST /decision {decision: "answer", key}` runs the full approval guard ([ADR 0010](0010-approval-guard.md)). The prompt hash covers the block, so the options that the user saw are the options on the screen. The key must be one of the fresh options.
- **Keys:** the allowed keys are now `1` to `9`, `y`, `esc` and `tab` ([ADR 0003](0003-allowed-hc-use.md)).
- **Result:** a question stays blocked between its steps. The result is `applied` when the turn, the status or the prompt hash changes.
- **Audit:** the decision is `answer:<key>`. The audit never has the option's text.

## Alternatives

| Alternative                       | Why it lost                                                                |
| --------------------------------- | -------------------------------------------------------------------------- |
| Keep questions view-only          | The user cannot unblock the agent from the phone.                          |
| Let the phone type free text      | That is keyboard input into the terminal, which is out of scope.           |
| Parse the options from `msg`      | `msg` names only the first question, and has no option numbers.            |
| Arrow keys and Enter on an option | The cursor position differs. A digit acts on its own option, at any place. |

## Consequences

- A wrong parse cannot press a key that the user did not see. The hash ties the shown options to the screen.
- A free answer goes through "Chat about this", then a message.
- A new Claude Code dialog layout gives no options, so the question becomes view-only again (safe).
