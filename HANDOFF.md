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
  - `checkpoint-pane` (the commit that adds this file)
- **Done since 2026-11-09:**
  - `install.sh` waits for `bootout` to finish before `bootstrap` (`52ee7fe`).
  - Tailscale runs on the Mac (`macbook-pro-5`) and on the phone (`ajays-s26`), with the same account. The daemon was restarted after the install.
  - The app is on the phone (Samsung S26, Android 16, 3-button navigation). It is paired as `S26` (`0b14a90331fe8ee0`). A message from the phone to an agent worked.
  - Two app fixes (`d3cd35f`):
    - "Notify me" crashed: biometric 1.1.0 pulls `fragment` 1.2.5, which rejects the permission request code. `fragment` 1.8.6 is now pinned.
    - Content sat under the 3-button navigation bar. The root now uses `safeDrawingPadding()`, with `adjustResize`.
  - The Pane design system, the app icon and the notification icon (`48d779f`).
  - Every screen uses the Pane components (`f234df4`, `c9cdeab`, `8bc7120`). `DESIGN.md` section 10 lists the known differences from the mockups.
- **Tests pass:**
  - `cd daemon && go test ./...`
  - `cd android && ./gradlew testDebugUnitTest assembleDebug`, with `JAVA_HOME=/opt/homebrew/opt/openjdk@17` and `ANDROID_HOME=/opt/homebrew/share/android-commandlinetools`
  - `scripts/format.sh --check`
- **Not checked yet:**
  - The phone runs an old APK. It does not have the fixes or the Pane screens.
  - Only the pair screen was checked on the emulator. FLAG_SECURE blocks screenshots, so check the other screens by eye on the phone.
- **The emulator** was paired once to reproduce the crash, then revoked on the Mac. Its app still thinks that it is paired.

## What is left, in order

1. **Install the new APK on the phone:** `tailscale file cp ~/code/herdr-remote/android/app/build/outputs/apk/debug/app-debug.apk ajays-s26:`, then open the file on the phone.
2. **Check every screen by eye** against `design/viewer.html`: light and dark mode, font scale 1.3, 3-button navigation. Nothing must sit under the navigation bar or the keyboard.
3. **Codex transcripts:** set Herdr's Codex launcher to `["codex", "--no-daemon"]` (Herdr settings, `new-menu.agents`). Without it, Codex agents show terminal text.
4. **Basic checks:**
   - spaces and agents show, and a status change shows within 2 s;
   - transcripts show for Claude and for `--no-daemon` Codex;
   - the live view is in color, and recovers after a pane resize and after heavy output (`yes | head -c 50000000`).
5. **Approvals and messages:**
   1. Make a test prompt: `hc run --bg --name hr-test --cwd ~/code/herdr-remote -- claude --permission-mode default "Use Bash to run: touch /tmp/hr-ok"`.
   2. Approve one, and deny one.
   3. Tap while you type on the Mac. Expect "Someone is using the Mac".
   4. Send a message to an idle agent.
   5. Then `hc kill hr-test; hc rm hr-test`.
6. **Background alerts and privacy:**
   - Turn "Notify me" on, and allow notifications and the battery exemption.
   - Swipe the app away, then block an agent. Expect an alert within 5 s that shows only "An agent needs attention" on the lock screen.
   - The recents card is blank.
   - After more than 5 minutes away, the app asks for your PIN.
   - After a phone reboot, alerts still come.
7. **Revoke:** click Revoke in the overlay. The phone loses access within 2 s. Then pair again.
8. **Away from home:** on cellular with Tailscale, the app connects.
9. **After a Mac reboot:** `launchctl print gui/$(id -u)/dev.herdr-remote | grep state` shows running. `audit.log` holds lines with no secrets or content.
10. **Optional:**
    - The pairing link lists the Mac's Tailscale name but not its Tailscale address (`100.107.60.83`), although `endpoints()` in `daemon/daemon.go` should add it. Find out why. The name works only while Tailscale DNS is on in the phone app.
    - On the same Wi-Fi without Tailscale, the app uses `192.168.1.3`. Reserve that address for the Mac in the router, or pair again when it changes.
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
