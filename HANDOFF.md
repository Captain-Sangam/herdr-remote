# Handoff: herdr-remote

Read this first in a new session. It gives the state on 2026-11-12 and what is left. Nothing is in flight: no agents run, and the tree is clean.

## What this is

- An Android app that controls Herdr.app sessions on this Mac. It shows spaces, agents, transcripts, a live terminal view, approve and deny, and messages.
- The phone reaches only `herdr-remoted`, a Go daemon (LaunchAgent `dev.herdr-remote`), over mutual TLS:
  - port 7766 for paired phones
  - port 7767 for enrollment, open only during a pairing window.
- A Herdr mod overlay (`herdr-remote`) shows the pairing QR code, the paired devices and Revoke.
- The docs are in [`docs/`](docs/README.md). [`AGENTS.md`](AGENTS.md) has the commands and the conventions. [`README.md`](README.md) has the local setup.
  - The current design is [`docs/architecture.md`](docs/architecture.md), with one ADR for each decision in [`docs/adr/`](docs/README.md#decisions-adr). [`docs/product.md`](docs/product.md) holds the product requirements.
  - The original technical design (was `TDD.md`) and the design debate (was `DEBATE.md`) are in [`docs/history/`](docs/README.md#history). They are the record, not the current state.
- The visual design is [`docs/design.md`](docs/design.md) (was `DESIGN.md`, the Pane design system). [`design/viewer.html`](design/viewer.html) shows every screen in Pane, beside the runner-up, Dispatch.

## Where things are

| What           | Where                                                                                                                                                               |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Repo           | `/Users/zeus/code/herdr-remote`, branch `main`. Remote: `Captain-Sangam/herdr-remote` on GitHub. Work lands through pull requests                                   |
| Docs           | `docs/` (index `docs/README.md`), `AGENTS.md`, `README.md`. Images in `docs/assets/`                                                                                |
| Daemon code    | `daemon/` (Go)                                                                                                                                                      |
| Android app    | `android/` (Kotlin, Compose). The APK is at `android/app/build/outputs/apk/debug/app-debug.apk`                                                                     |
| Pane theme     | `android/app/src/main/java/dev/herdrremote/ui/theme/` (`PaneTheme.kt`, `PaneComponents.kt`)                                                                         |
| Mod source     | `mod/herdr-remote/`                                                                                                                                                 |
| Install        | `scripts/install.sh`: builds the daemon to `~/.local/bin/herdr-remoted`, writes the LaunchAgent, copies the mod                                                     |
| Formatting     | `scripts/format.sh` (`--check`, `--list`, or paths), `.githooks/pre-commit`, `.github/workflows/ci.yaml`                                                            |
| Daemon state   | `~/Library/Application Support/herdr-remote/` (`server.key`, `devices.json`, `audit.log`, `daemon.log`, `admin.sock`). Do not print `server.key`                    |
| Daemon admin   | `herdr-remoted pair [status\|allow\|deny\|cancel]`, `herdr-remoted devices ls\|rm <id>`                                                                             |
| Installed mod  | `~/.config/herdr-app/mods/herdr-remote/`                                                                                                                            |
| Council record | `.agents/tasks/herdr-remote/`: `record.md` (full trail), `review.md` and `review-2.md` (code reviews), `plan.md` (superseded by `docs/history/technical-design.md`) |
| Android SDK    | `/opt/homebrew/share/android-commandlinetools`. `adb` is in `platform-tools/` and is not on the PATH                                                                |
| Emulator       | AVD `herdr-remote` (API 35, 3-button navigation on). Start it with `emulator -avd herdr-remote -no-window` from `emulator/`                                         |

## State on 2026-11-12

- **Checkpoint tags:**
  - `checkpoint-pre-review` (`f858d27`)
  - `checkpoint-fixes-applied` (`429c512`)
  - `checkpoint-reviewed` (`b331373`)
  - `checkpoint-formatted` (`21e19c8`)
  - `checkpoint-pane` (`764495b`)
  - `checkpoint-device-tested` (the commit that adds this line)
- **Done on 2026-11-12 (ADRs [0021](docs/adr/0021-answer-questions.md) to [0024](docs/adr/0024-allow-screenshots.md)):**
  - The phone answers Claude Code questions: a key for each option, Next after a multi-select, the full approval guard. "Type something" stays on the Mac. "Chat about this" then a message gives a free answer.
  - A skill picker: a draft that starts with "/" lists the agent's skills, from its transcript and its skill folders (so `council` shows).
  - A long press selects transcript text to copy. Each copy is marked sensitive.
  - Screenshots are allowed. The recents card stays blank.
  - Working agents blink on every screen. The transcript ends with a spinner and the seconds since the last item, and reloads each 1 s.
  - Checked on the phone with screenshots: questions (single, two questions, multi-select, review), the picker with `/coun` and `/deb`, the working row, a long press and Copy ("Copied."). Not checked: that the clipboard preview hides the text.
- **Done on 2026-11-10, last ([ADR 0020](docs/adr/0020-transcript-folds-and-scroll.md)):**
  - The transcript folds each turn's tool steps into one row. Prompts and answers show.
  - Follow stays on after a send and with the keyboard open. A scroll up turns it off.
  - A fling stops at a message start, with a haptic tick. A scroll bar shows while scrolling.
  - Checked on the phone over `adb`: folds, follow, "Bottom ↓", the keyboard, a fold that opens. Not checked: the tick, the bar, a send.
- **Done on 2026-11-10, later (PRs #2 and #3):**
  - The transcript updates live while the agent works, follows new items and has "Bottom ↓". The header shows the model.
  - Alerts when an agent finishes. The blocked alert shows the prompt, with Approve and Deny ([ADR 0019](docs/adr/0019-alert-preview-and-actions.md)).
  - A locked phone and the Galaxy Watch show only the agent name and what it needs. SR-11 changed to allow the agent name.
  - The "Watching N agents" notification no longer comes back after a dismissal.
  - CI runs the format check and the short daemon tests (`go test -short`), each capped at 30 s. The full daemon suite and the app tests and build run locally.
- **Done since 2026-11-09:**
  - `install.sh` waits for `bootout` to finish before `bootstrap` (`52ee7fe`).
  - Tailscale runs on the Mac and on the phone, with the same account.
  - Two app fixes (`d3cd35f`):
    - "Notify me" crashed: biometric 1.1.0 pulls `fragment` 1.2.5, which rejects the permission request code. `fragment` 1.8.6 is now pinned.
    - Content sat under the 3-button navigation bar. The root now uses `safeDrawingPadding()`, with `adjustResize`.
  - The Pane design system, the app icon and the notification icon (`48d779f`). Every screen uses the Pane components (`f234df4`, `c9cdeab`, `8bc7120`). `docs/design.md` section 10 lists the known differences from the mockups.
  - Every block gave two alerts: hc sends the status and the turn as two events, and the turn went into the episode key. Fixed (`6c8e729`).
  - The tab bar has no window numbers: "0:spaces" read as "0 spaces".
  - The docs moved under `docs/`: focused docs, 18 ADRs, `README.md`, `AGENTS.md` and `CLAUDE.md`. Code comments cite the docs and the ADRs (`51bc97a` to the commit that adds this line).
- **The phone:** Samsung S26 (SM-S942B), Android 17, 3-button navigation, dark mode, font scale 0.9. USB debugging is on. It runs the latest APK and is paired as `SM-S942B` (`af453d9d2350e234`). "Notify me" is on.
- **Checked on the phone over adb on 2026-11-10 (all pass):**
  - Layout at font scale 0.9 and 1.3, with the keyboard open: nothing under the navigation bar, no tap target under 48 dp.
  - A status change shows in about 1.2 s.
  - Transcripts for Claude and for `--no-daemon` Codex.
  - The live view: in color, follows a pane resize, recovers after 50 MB of output.
  - Approve, deny and a message: applied on the Mac and in `audit.log`. A tap while the Mac is in use gives "Someone is using the Mac" (the limit is 10 s of idle).
  - "Notify me": the two prompts, no crash. An alert 3 to 4 s after a block, once, with only "An agent needs attention". The recents card is blank.
  - The app asks for the fingerprint on a cold start and after 5 minutes away.
  - After a phone reboot, the watcher starts by itself and alerts come.
  - Revoke: the Mac drops the phone in 0.21 s. Pairing again works.
  - Wi-Fi off: the app connects over Tailscale.
- **Tests pass:**
  - `cd daemon && go test ./...`
  - `cd android && ./gradlew testDebugUnitTest assembleDebug`, with `JAVA_HOME=/opt/homebrew/opt/openjdk@17` and `ANDROID_HOME=/opt/homebrew/share/android-commandlinetools`
  - `scripts/format.sh --check`
- **How the phone was tested:** `adb` and `uiautomator dump` for the screens, and the WebView debugging port for the live view (a debug build only). Screenshots of the app work since ADR 0024 (`adb exec-out screencap -p`). A plain terminal shows on the phone as an agent if it prints `ESC ] 7501 ; state=idle BEL`.
- **The emulator** was paired once to reproduce the crash, then revoked on the Mac. Its app still thinks that it is paired.

## What is left, in order

0. **Look at the transcript scroll by eye** ([ADR 0019](docs/adr/0019-alert-preview-and-actions.md)): it follows new items while the end shows, and "Bottom ↓" shows after a scroll up. Only the unit tests cover this.
   - Checked over `adb` on 2026-11-10: the live transcript, the model in the header, the blocked alert with the prompt, Approve from the shade (`applied`), the refusal with the buttons back, finish alerts, and the redaction on the lock screen and its end after an unlock.
   - Do not lock, turn off or reboot the phone in a test without asking the user first.
   - The Galaxy Watch 8 gets the alerts: Herdr Remote is on in Galaxy Wearable. The bridge holds an alert while the phone is in use (`pending:USING_PHONE`). Check by eye that the watch shows the agent name after a phone lock. The logs did not confirm it.
   - Approve on the watch needs a new decision: a watch button skips the phone unlock.
1. **After a Mac reboot:** `launchctl print gui/$(id -u)/dev.herdr-remote | grep state` shows running. `audit.log` holds lines with no secrets or content.
2. **Look at the screens by eye** in light mode. Screenshots work now (ADR 0024).
3. **Optional:**
   - A revoked phone keeps its last data on the screen with "Not connected" and retries for good. It does not say that it was removed or offer to pair again.
   - The phone learns new Mac addresses only while it is connected. Without Tailscale, a new Wi-Fi address on the Mac needs a new pairing.
   - Add the shfmt checksums for Intel Macs and arm64 Linux to `scripts/format.sh`.

## Conventions for the next session

The conventions are in [`AGENTS.md`](AGENTS.md#conventions). Claude Code loads them through `CLAUDE.md`.
