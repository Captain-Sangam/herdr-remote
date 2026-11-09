# Handoff: herdr-remote

Read this first in a new session. It gives the state on 2026-11-09 and what is left. Nothing is in flight: no agents run, and the tree is clean.

## What this is

- An Android app that controls Herdr.app sessions on this Mac. It shows spaces, agents, transcripts, a live terminal view, approve and deny, and messages.
- The phone reaches only `herdr-remoted`, a Go daemon (LaunchAgent `dev.herdr-remote`), over mutual TLS:
  - port 7766 for paired phones;
  - port 7767 for enrollment, open only during a pairing window.
- A Herdr mod overlay (`herdr-remote`) shows the pairing QR code, the paired devices and Revoke.
- The design is `TDD.md`. Its "As built" and "Review (2026-11-09)" sections at the end describe the code as it is now. `PRD.md` holds the product requirements. `DEBATE.md` tells why each decision was made.

## Where things are

| What           | Where                                                                                                                                            |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Repo           | `/Users/zeus/code/herdr-remote`, branch `main`, no git remote                                                                                    |
| Daemon code    | `daemon/` (Go)                                                                                                                                   |
| Android app    | `android/` (Kotlin, Compose). The APK is at `android/app/build/outputs/apk/debug/app-debug.apk`                                                  |
| Mod source     | `mod/herdr-remote/`                                                                                                                              |
| Install        | `scripts/install.sh`: builds the daemon to `~/.local/bin/herdr-remoted`, writes the LaunchAgent, copies the mod                                  |
| Formatting     | `scripts/format.sh` (`--check`, `--list`, or paths), `.githooks/pre-commit`, `.github/workflows/ci.yaml`                                         |
| Daemon state   | `~/Library/Application Support/herdr-remote/` (`server.key`, `devices.json`, `audit.log`, `daemon.log`, `admin.sock`). Do not print `server.key` |
| Installed mod  | `~/.config/herdr-app/mods/herdr-remote/`                                                                                                         |
| Council record | `.agents/tasks/herdr-remote/`: `record.md` (full trail), `review.md` and `review-2.md` (code reviews), `plan.md` (superseded by `TDD.md`)        |
| Android SDK    | `/opt/homebrew/share/android-commandlinetools`. `adb` is in `platform-tools/` and is not on the PATH                                             |

## State on 2026-11-09

- **HEAD:** `21e19c8` plus this file.
- **Checkpoint tags:**
  - `checkpoint-pre-review` (`f858d27`)
  - `checkpoint-fixes-applied` (`429c512`)
  - `checkpoint-reviewed` (`b331373`)
  - `checkpoint-formatted` (`21e19c8`)
- **Done:**
  - the plan;
  - the build (12 steps);
  - two code reviews with 12 fixes;
  - the formatting tooling, the pre-commit hook (`core.hooksPath=.githooks` is set in this clone) and the CI check;
  - the reinstall of the fixed daemon.
- **Tests pass:**
  - `cd daemon && go test ./...`
  - `cd android && ./gradlew testDebugUnitTest assembleDebug` (19 JVM tests), with `JAVA_HOME=/opt/homebrew/opt/openjdk@17` and `ANDROID_HOME=/opt/homebrew/share/android-commandlinetools`
  - `scripts/format.sh --check`
- **The installed daemon** runs the latest build (reinstalled at 14:57 on 2026-11-09). It listens on 7766, and 7767 is closed.
- **No phone is paired yet.**

## What is left, in order

1. **Optional: fix the `install.sh` restart race.**
   - Problem: `scripts/install.sh:42-43` runs `launchctl bootout` and then `launchctl bootstrap` at once. While the service is loaded, `bootstrap` fails with "5: Input/output error", the script stops, and the daemon stays down until a second run.
   - Fix: after `bootout`, wait (up to about 5 s) until `launchctl print gui/$(id -u)/dev.herdr-remote` fails, then `bootstrap`.
2. **Tailscale, for access away from home.**
   - Install it on the Mac and on the phone, with the same account.
   - Then run `launchctl kickstart -k gui/$(id -u)/dev.herdr-remote`. The daemon looks for the `tailscale` binary only when it starts.
   - Do this before you pair, so the QR code includes the Tailscale address. (The app also refreshes the address list from `/v1/hello` later.)
3. **Install the app on the phone.**
   - Turn on Developer options and USB debugging, then connect the phone by USB.
   - Run `export PATH="/opt/homebrew/share/android-commandlinetools/platform-tools:$PATH"`, then `adb devices`.
   - Then `adb install -r ~/code/herdr-remote/android/app/build/outputs/apk/debug/app-debug.apk`.
4. **Pair.**
   1. Put the phone on the same Wi-Fi as the Mac.
   2. Run `herdr-app do mod.open --name herdr-remote`, then click **Pair a phone**.
   3. Scan the QR code with the app, and check that the 6-digit code matches.
   4. Click **Allow**. Allow `herdr-remoted` if the macOS firewall asks.
5. **Codex transcripts:** set Herdr's Codex launcher to `["codex", "--no-daemon"]` (Herdr settings, `new-menu.agents`). Without it, Codex agents show terminal text.
6. **Basic checks:**
   - spaces and agents show, and a status change shows within 2 s;
   - transcripts show for Claude and for `--no-daemon` Codex;
   - the live view is in color, and recovers after a pane resize and after heavy output (`yes | head -c 50000000`).
7. **Approvals and messages:**
   1. Make a test prompt: `hc run --bg --name hr-test --cwd ~/code/herdr-remote -- claude --permission-mode default "Use Bash to run: touch /tmp/hr-ok"`.
   2. Approve one, and deny one.
   3. Tap while you type on the Mac. Expect "Someone is using the Mac".
   4. Send a message to an idle agent.
   5. Then `hc kill hr-test; hc rm hr-test`.
8. **Background alerts and privacy:**
   - Turn watching on, and allow notifications and the battery exemption.
   - Swipe the app away, then block an agent. Expect an alert within 5 s that shows only "An agent needs attention" on the lock screen.
   - The recents card is blank.
   - After more than 5 minutes away, the app asks for your PIN.
   - After a phone reboot, alerts still come.
9. **Revoke:** click Revoke in the overlay. The phone loses access within 2 s. Then pair again.
10. **Away from home:** on cellular with Tailscale, the app connects.
11. **After a Mac reboot:** `launchctl print gui/$(id -u)/dev.herdr-remote | grep state` shows running. `audit.log` holds lines with no secrets or content.
12. **Optional:**
    - Push to a GitHub repo so that `ci.yaml` runs.
    - Fix any known issue in the `TDD.md` review note, for example `maxLines = 3` on the prompt title line (`android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt`).
    - Add the shfmt checksums for Intel Macs and arm64 Linux to `scripts/format.sh`.

## Conventions for the next session

- **Herdr:** drive it only with `hc` and `herdr-app` (see the `herdr` skill). Never use the legacy `herdr` CLI.
- **Council:** larger work goes through the council skill (`/council plan | implement | review`), with the task folder `.agents/tasks/herdr-remote/`.
  - `.agents/` is tracked in git, so commit changes there before a council review. Its pre-flight check expects a clean tree.
  - Use new reviewer and implementer names that do not clash with live terminals.
- **Commits:** commit on `main` and never push without asking. The pre-commit hook formats staged files and refuses a file that also has unstaged changes.
- **The user's preferences:**
  - Save and checkpoint (commit plus an annotated `checkpoint-*` tag) when asked.
  - Reviews must not nitpick. Surface only what matters.
  - Plan and design docs follow the house style (the `language` skill).
