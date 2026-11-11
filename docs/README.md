# herdr-remote docs

This folder holds the documentation of herdr-remote. Start with the product and the architecture. Use the other pages when you need them.

## Start here

| Doc                             | Read it to                                                                |
| ------------------------------- | ------------------------------------------------------------------------- |
| [Product](product.md)           | Learn what the app does, for whom, and the requirements (FR, SR, NFR).    |
| [Architecture](architecture.md) | Learn the components, the data flows, and the security and privacy model. |
| [Deployment](deployment.md)     | Install the daemon and the mod, build the app, pair a phone, and update.  |

## Reference

| Doc                                         | Read it to                                                                      |
| ------------------------------------------- | ------------------------------------------------------------------------------- |
| [API](api.md)                               | Find an endpoint, a shape, an error code, a timeout or the pairing link format. |
| [Herdr and agent interfaces](interfaces.md) | Check the hc, Herdr, Claude Code and Codex facts that the code depends on.      |
| [Operations](operations.md)                 | Use the admin CLI, read the audit log, revoke a phone, fix a problem.           |
| [Testing](testing.md)                       | Run the tests, use the emulator and the probe agents, check a real phone.       |
| [Design](design.md)                         | Use the Pane design system: tokens, type, components and screen rules.          |

## Decisions (ADR)

Each ADR keeps the number of its decision in the [decision table](history/technical-design.md#31-decision-table). ADR 0013 is decision 13.

| ADR                                             | Decision                                        | Status              |
| ----------------------------------------------- | ----------------------------------------------- | ------------------- |
| [0001](adr/0001-go-daemon.md)                   | One Go daemon is the only network listener.     | Accepted            |
| [0002](adr/0002-hc-subprocesses-and-helper.md)  | Three hc connections and one subprocess helper. | Accepted            |
| [0003](adr/0003-allowed-hc-use.md)              | A fixed list of allowed hc calls.               | Accepted, amended   |
| [0004](adr/0004-two-tls-listeners.md)           | Two TLS listeners with a pinned key.            | Accepted            |
| [0005](adr/0005-http-sse-and-streams.md)        | HTTP with SSE and chunked streams.              | Accepted, amended   |
| [0006](adr/0006-pairing.md)                     | QR pairing with a code and Allow on the Mac.    | Accepted            |
| [0007](adr/0007-revocation.md)                  | Revocation ends access at once.                 | Accepted, amended   |
| [0008](adr/0008-live-view.md)                   | Live view from hc attach in xterm.js.           | Accepted            |
| [0009](adr/0009-transcripts.md)                 | Transcripts from exact sources only.            | Accepted, amended   |
| [0010](adr/0010-approval-guard.md)              | The approval guard.                             | Accepted, amended   |
| [0011](adr/0011-view-only-prompts.md)           | Questions and unknown prompts are view-only.    | Replaced in part    |
| [0012](adr/0012-messages.md)                    | Messages arrive as plain prompts, at most once. | Accepted            |
| [0013](adr/0013-background-alerts.md)           | Background alerts from a foreground service.    | Accepted, corrected |
| [0014](adr/0014-phone-key-and-privacy.md)       | The phone key, the app lock and privacy.        | Accepted, amended   |
| [0015](adr/0015-android-stack.md)               | The Android stack.                              | Accepted            |
| [0016](adr/0016-spaces-and-agents.md)           | Spaces and agents from `herdr-app ls`.          | Accepted            |
| [0017](adr/0017-logging-and-audit.md)           | Content-free logs and an audit log.             | Accepted            |
| [0018](adr/0018-trust-boundary.md)              | Processes of the same user are trusted.         | Accepted            |
| [0019](adr/0019-alert-preview-and-actions.md)   | Alert previews, Approve, finish alerts.         | Accepted            |
| [0020](adr/0020-transcript-folds-and-scroll.md) | Transcript folds, follow, message steps.        | Accepted            |
| [0021](adr/0021-answer-questions.md)            | The phone answers Claude Code questions.        | Accepted            |
| [0022](adr/0022-skill-picker.md)                | A skill picker from the transcript's list.      | Accepted            |
| [0023](adr/0023-copy-transcript-text.md)        | Transcript text can be copied.                  | Accepted            |
| [0024](adr/0024-allow-screenshots.md)           | Screenshots allowed, recents card blank.        | Accepted            |

To add a decision:

1. Copy an ADR and give it the next number.
2. Fill in the status, context, decision, alternatives and consequences.
3. Add a row to the table above.
4. Change the docs that describe the code, for example the architecture.

## History

These files are the record of how the design was made. They describe the plan at that time. The pages above describe the code now.

| File                                            | Content                                                                                                                                      |
| ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| [Technical design](history/technical-design.md) | The approved design of 2026-11-08 (was `TDD.md`): the decision table, the 12 build steps, the "As built" notes and the review of 2026-11-09. |
| [Design debate](history/debate.md)              | The planning debate and the three plan reviews (was `DEBATE.md`): who argued what, and what was verified.                                    |

The council record of the planning, the build and the reviews is in `.agents/tasks/herdr-remote/` at the repo root. The council skill expects it there.

## Other places

- [`design/`](../design/viewer.html): the Pane mockups. `viewer.html` shows one screen at a time. `options.html` shows all screens, the tokens and the components.
- [`assets/`](assets/): the logo and the screens in the root README.
- [`../HANDOFF.md`](../HANDOFF.md): the state of the work for the next session.
- [`../AGENTS.md`](../AGENTS.md): the short guide for coding agents.
