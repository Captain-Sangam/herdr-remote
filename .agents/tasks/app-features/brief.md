# Brief: app-features

## Name

`app-features`. The agents are `app-features-a1` (Codex, `gpt-6-astra`) and `app-features-a2` (Claude, `fable`).

## Task

Find the features that the mobile app (Android now, iOS at parity later) should get next, and describe each one. The user is one developer who controls the agents in Herdr.app on one Mac from the phone, away from the desk, with little time and a small screen. Done is a list of features. Each feature has a name, a description of what the user can do, why it helps, what it changes (phone, daemon, Herdr), its security effect, and a size (S, M or L). The list is ranked. This stage writes no code.

## Repos

- **herdr-remote** (primary): `/Users/zeus/code/herdr-remote`, branch `main`, HEAD `247206a`.
  - **What the app does today:** `docs/product.md` section 7 (FR-1 to FR-15) and section 5 (scope and out of scope). Pairing by QR code, spaces and agents, transcript, live read-only terminal, approve or deny, answers to Claude Code questions, messages up to 4000 characters, a skill picker, copy, background alerts.
  - **Daemon API:** `daemon/api.go:15-25` (11 routes), described in `docs/api.md`. The daemon is the only part the phone reaches.
  - **Allowed hc calls:** `docs/adr/0003-allowed-hc-use.md`. Only `hello`, `terminal.list/get/read`, `events.subscribe`, `terminal.send` (no force), `terminal.keys` with one key from the approval code, and `hc attach --observe`.
  - **Transcripts:** `daemon/transcript.go`, from Claude Code and Codex session files only.
  - **Phone app:** `android/app/src/main/java/dev/herdrremote/`: `ui/` (`ListScreens.kt`, `AgentScreen.kt`, `LiveView.kt`, `SettingsScreen.kt`, `PairScreen.kt`), `net/Api.kt`, `watch/` (background alerts).
  - **hc and Herdr facts:** `docs/interfaces.md`.
  - **Open issues:** `docs/operations.md` "Known issues".
  - **iOS app:** in progress, paused at plan step 6 (`.agents/tasks/ios-app/plan.md`). Its plan is fixed and is not part of this task.

## Binding rules

- **Security** (`docs/product.md` section 8): the phone can use only the operations of FR-5 to FR-13 and cannot run commands, read or write files, or start programs (SR-2). One user action is at most one key press or one message (SR-6). No log holds transcript text, terminal bytes or message text (SR-9). The phone keeps transcripts in memory only (SR-12). The live view cannot send input (SR-13).
- **Herdr:** the daemon drives Herdr only with `hc` and `herdr-app`, never the legacy `herdr` CLI (`AGENTS.md`). Changes to Herdr.app or its shipped mods are out of scope (`docs/product.md` section 5).
- **Decisions:** a feature that changes a requirement or an ADR needs a new ADR (`AGENTS.md`).

## Criteria

The user gave none. The council optimises for, in this order:

1. Value to the user away from the desk: fewer trips to the Mac, faster unblocking, better awareness of what the agents do.
2. Fit with the security model. A feature that crosses a rule in section 8 or an item in "Out of scope" is allowed in the list, but it must name the rule, the risk and the safeguard.
3. Size, from the existing code and the allowed hc calls.
4. Parity: the feature must also work on iOS.

## Non-goals

- No code, branches or commits.
- Not the iOS parity plan, the Play Store or the App Store.
- Not the open bugs in `docs/operations.md`, unless a bug blocks a feature.

## Known facts

- The transcript fold and the light markdown were fixed on 2026-11-18 (ADR 0026, PR #8). They are done.
- One phone action may press only one key, and only from the approval code (ADR 0003, ADR 0021).

## Questions

- **For the agents:** which hc or Herdr capabilities (from `hc --help`, `hc <command> --help` and `docs/interfaces.md`) the app does not use yet, and what they make possible.
- **For the user:** may the list include features that the product now puts out of scope (keyboard input into a terminal, interrupts, start or stop of agents, a push relay)? The default is yes, with each marked "needs a security change".
