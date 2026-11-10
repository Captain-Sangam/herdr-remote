# Handoff: herdr-remote

Read this first in a new session. It gives the state on 2026-11-10 and what is left. Nothing is in flight: no agents run, and the tree is clean.

## What this is

- An Android app that controls Herdr.app sessions on this Mac. It shows spaces, agents, transcripts, a live terminal view, approve and deny, and messages.
- The phone reaches only `herdr-remoted`, a Go daemon (LaunchAgent `dev.herdr-remote`), over mutual TLS:
  - port 7766 for paired phones;
  - port 7767 for enrollment, open only during a pairing window.
- A Herdr mod overlay (`herdr-remote`) shows the pairing QR code, the paired devices and Revoke.
- The design is `TDD.md`. Its "As built" and "Review (2026-11-09)" sections at the end describe the code. `PRD.md` holds the product requirements. `DEBATE.md` tells why each decision was made.
- The visual design is `DESIGN.md` (the Pane design system). `design/viewer.html` shows every screen in Pane, beside the runner-up, Dispatch.

## Where things are

| What           | Where                                                                                                                                            |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Repo           | `/Users/zeus/code/herdr-remote`, branch `main`, no git remote                                                                                    |
| Daemon code    | `daemon/` (Go)                                                                                                                                   |
| Android app    | `android/` (Kotlin, Compose). The APK is at `android/app/build/outputs/apk/debug/app-debug.apk`                                                  |
| Pane theme     | `android/app/src/main/java/dev/herdrremote/ui/theme/` (`PaneTheme.kt`, `PaneComponents.kt`)                                                      |
| Mod source     | `mod/herdr-remote/`                                                                                                                              |
| Install        | `scripts/install.sh`: builds the daemon to `~/.local/bin/herdr-remoted`, writes the LaunchAgent, copies the mod                                  |
| Formatting     | `scripts/format.sh` (`--check`, `--list`, or paths), `.githooks/pre-commit`, `.github/workflows/ci.yaml`                                         |
| Daemon state   | `~/Library/Application Support/herdr-remote/` (`server.key`, `devices.json`, `audit.log`, `daemon.log`, `admin.sock`). Do not print `server.key` |
| Daemon admin   | `herdr-remoted pair [status\|allow\|deny\|cancel]`, `herdr-remoted devices ls\|rm <id>`                                                          |
| Installed mod  | `~/.config/herdr-app/mods/herdr-remote/`                                                                                                         |
| Council record | `.agents/tasks/herdr-remote/`: `record.md` (full trail), `review.md` and `review-2.md` (code reviews), `plan.md` (superseded by `TDD.md`)        |
| Android SDK    | `/opt/homebrew/share/android-commandlinetools`. `adb` is in `platform-tools/` and is not on the PATH                                             |
| Emulator       | AVD `herdr-remote` (API 35, 3-button navigation on). Start it with `emulator -avd herdr-remote -no-window` from `emulator/`                      |

## State on 2026-11-10

- **Checkpoint tags:**
  - `checkpoint-pre-review` (`f858d27`)
  - `checkpoint-fixes-applied` (`429c512`)
  - `checkpoint-reviewed` (`b331373`)
  - `checkpoint-formatted` (`21e19c8`)
  - `checkpoint-pane` (`764495b`)
- **Done since 2026-11-09:**
  - `install.sh` waits for `bootout` to finish before `bootstrap` (`52ee7fe`).
  - Tailscale runs on the Mac (`macbook-pro-5`, `100.107.60.83`) and on the phone (`ajays-s26`, `100.111.88.15`), with the same account.
  - Two app fixes (`d3cd35f`):
    - "Notify me" crashed: biometric 1.1.0 pulls `fragment` 1.2.5, which rejects the permission request code. `fragment` 1.8.6 is now pinned.
    - Content sat under the 3-button navigation bar. The root now uses `safeDrawingPadding()`, with `adjustResize`.
  - The Pane design system, the app icon and the notification icon (`48d779f`). Every screen uses the Pane components (`f234df4`, `c9cdeab`, `8bc7120`). `DESIGN.md` section 10 lists the known differences from the mockups.
  - Every block gave two alerts: hc sends the status and the turn as two events, and the turn went into the episode key. Fixed (`6c8e729`).
  - The tab bar has no window numbers: "0:spaces" read as "0 spaces".
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
- **How the phone was tested:** `adb` and `uiautomator dump` for the screens, and the WebView debugging port for the live view (a debug build only). FLAG_SECURE blocks screenshots of the app. A plain terminal shows on the phone as an agent if it prints `ESC ] 7501 ; state=idle BEL`.
- **The emulator** was paired once to reproduce the crash, then revoked on the Mac. Its app still thinks that it is paired.

## What is left, in order

1. **After a Mac reboot:** `launchctl print gui/$(id -u)/dev.herdr-remote | grep state` shows running. `audit.log` holds lines with no secrets or content.
2. **Look at the screens by eye** in light mode. FLAG_SECURE blocked a check of the colours.
3. **Optional:**
   - A revoked phone keeps its last data on the screen with "Not connected" and retries for good. It does not say that it was removed or offer to pair again.
   - The phone learns new Mac addresses only while it is connected. Without Tailscale, a new Wi-Fi address on the Mac needs a new pairing.
   - Push to a GitHub repo so that `ci.yaml` runs.
   - Add the shfmt checksums for Intel Macs and arm64 Linux to `scripts/format.sh`.

## Conventions for the next session

- **Herdr:** drive it only with `hc` and `herdr-app` (see the `herdr` skill). Never use the legacy `herdr` CLI.
- **Helper agents:** start them with `hc run --bg --label herdr.space=<space id>` so that they show as a tab, and name the tab. Under auto mode, a helper may not pair a device: `herdr-remoted pair allow` needs the user's approval.
- **Council:** larger work goes through the council skill (`/council plan | implement | review`), with the task folder `.agents/tasks/herdr-remote/`.
  - `.agents/` is tracked in git, so commit changes there before a council review. Its pre-flight check expects a clean tree.
  - Use new reviewer and implementer names that do not clash with live terminals.
- **Commits:** commit on `main` and never push without asking. The pre-commit hook formats staged files and refuses a file that also has unstaged changes.
- **The user's preferences:**
  - Save and checkpoint (commit plus an annotated `checkpoint-*` tag) when asked.
  - Reviews must not nitpick. Surface only what matters.
  - Plan and design docs follow the house style (the `language` skill).
