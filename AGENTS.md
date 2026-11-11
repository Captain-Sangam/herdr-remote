# AGENTS.md

herdr-remote lets an Android phone control the agents in Herdr.app on one Mac. It has three parts:

- `daemon/`: `herdr-remoted`, a Go daemon (LaunchAgent `dev.herdr-remote`). It is the only part that the phone can reach, over mutual TLS on port 7766 (and port 7767 during pairing).
- `mod/herdr-remote/`: a Herdr mod overlay for pairing and revoke.
- `android/`: the Kotlin and Compose app (`dev.herdrremote`).

Read [`HANDOFF.md`](HANDOFF.md) first for the state of the work. Everything else is in [`docs/`](docs/README.md).

## Docs

| Doc                                            | Content                                                             |
| ---------------------------------------------- | ------------------------------------------------------------------- |
| [`docs/README.md`](docs/README.md)             | The index, and the list of ADRs.                                    |
| [`docs/product.md`](docs/product.md)           | The product requirements (FR, SR, NFR).                             |
| [`docs/architecture.md`](docs/architecture.md) | Components, data flows, the security and privacy model.             |
| [`docs/adr/`](docs/README.md#decisions-adr)    | One ADR per design decision. ADR 0013 is decision 13.               |
| [`docs/api.md`](docs/api.md)                   | Endpoints, shapes, errors, timeouts, the pairing link.              |
| [`docs/interfaces.md`](docs/interfaces.md)     | The hc, Herdr, Claude Code and Codex facts that the code uses.      |
| [`docs/deployment.md`](docs/deployment.md)     | Install, LaunchAgent, state files, Tailscale, pairing, updates.     |
| [`docs/operations.md`](docs/operations.md)     | Admin CLI, audit log, revoke, troubleshooting, known issues.        |
| [`docs/testing.md`](docs/testing.md)           | Tests, the emulator, probe agents, the on-device checks over `adb`. |
| [`docs/design.md`](docs/design.md)             | The Pane design system. Mockups are in `design/`.                   |
| [`docs/history/`](docs/README.md#history)      | The original technical design and the design debate.                |

## Commands

| Task                       | Command                                                                                                                                                    |
| -------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Test the daemon            | `cd daemon && go test ./...` (about 80 s)                                                                                                                  |
| Test and build the app     | `cd android && JAVA_HOME=/opt/homebrew/opt/openjdk@17 ANDROID_HOME=/opt/homebrew/share/android-commandlinetools ./gradlew testDebugUnitTest assembleDebug` |
| Check the format           | `scripts/format.sh --check`                                                                                                                                |
| Format files               | `scripts/format.sh [PATH...]`                                                                                                                              |
| Install the daemon and mod | `scripts/install.sh`                                                                                                                                       |
| Install the app            | `adb install -r android/app/build/outputs/apk/debug/app-debug.apk` (`adb` is in `/opt/homebrew/share/android-commandlinetools/platform-tools/`)            |

- Run the three checks (daemon tests, app tests and build, format) before each commit.
- Turn on the pre-commit hook once per clone: `git config core.hooksPath .githooks`.

## Conventions

- **Herdr:** drive it only with `hc` and `herdr-app` (see the `herdr` skill). Never use the legacy `herdr` CLI.
- **Helper agents:** start them with `hc run --bg --label herdr.space=<space id>` so that they show as a tab, and name the tab.
- **Pairing:** under auto mode, a helper may not pair a device. `herdr-remoted pair allow` needs the user's approval.
- **Council:** larger work goes through the council skill (`/council plan | implement | review`), with the task folder `.agents/tasks/herdr-remote/`.
  - `.agents/` is tracked in git, so commit changes there before a council review. Its pre-flight check expects a clean tree.
  - Use new reviewer and implementer names that do not clash with live terminals.
- **Commits:** commit on `main` and never push without asking. The pre-commit hook formats staged files and refuses a file that also has unstaged changes.
- **Checkpoints:** when the user asks, save and checkpoint: a commit plus an annotated `checkpoint-*` tag.
- **Reviews:** do not nitpick. Surface only what matters.
- **Docs:** plan and design docs follow the house style (the `language` skill).
- **Decisions:** a new design decision gets a new ADR in `docs/adr/`. Update the docs that describe the code in the same commit.
- **Secrets:** do not print `server.key`. The recents card stays blank in every build. Screenshots are allowed ([ADR 0024](docs/adr/0024-allow-screenshots.md)).
