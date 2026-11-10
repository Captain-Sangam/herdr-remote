# Review notes

**User notes.** None were given with the review command.

**What is under review.**
- The whole repo `/Users/zeus/code/herdr-remote`, from the empty tree (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`) to `f858d27`. Everything in it is new.
- The code diff is at `/private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.OdlikHChF5/herdr-remote.diff.patch`. It leaves out `PRD.md`, `DEBATE.md` and `TDD.md` (read those directly), the vendored `xterm.js` and `xterm.css`, the Gradle wrapper jar, and the captured fixture `daemon/testdata/events-probe.jsonl`.

**The plan.**
- `/Users/zeus/code/herdr-remote/TDD.md`. Its section 3 holds the decisions, section 4 the acceptance criteria, section 5 the steps, and section 6 the tests.
- The implementer's dated "As built (2026-11-08)" section at its end records assumptions and deviations. Do not report a deviation listed there as a miss unless it breaks a decision or a criterion.
- `PRD.md` is the product view of the same requirements.

**Settled decisions to respect** (from the TDD decision table). Report a breach. Do not re-argue the choice:
- The trust boundary is the macOS user: same-user processes are trusted (decision 18).
- There is no relay. TLS 1.3 with client certificates, and two listeners (7766 paired, 7767 enrollment only).
- Daemon-mode Codex shows terminal text, not a guessed transcript.
- Approvals cover permission prompts of Claude Code and Codex only. Questions are view-only.
- There is a Mac-in-use guard (`HIDIdleTime`), and the race it narrows is an accepted residual risk.

**As built: deviations and additions that the implementer reported:**
- `pair --json` also returns `text`, the QR content. The app has a "paste the pairing link" field.
- The mod also accepts `herdr-app do herdr-remote.pair`, `.allow`, `.deny` and `.cancel`.
- Commit `0d14b24` (Android step 9) does not build alone. The files were split across commits after the fact.
- The lock-screen text with Android's default "show sensitive content" setting is the tab name and "permission", not "An agent needs attention".
- A refused request (409 or 400) is not remembered. Only a dispatched one is.
- The daemon runs every subprocess with a bare environment.

**Verified by the orchestrator before this review:**
- `go vet ./...` is clean. `go test ./...` passes in about 85 s.
- `./gradlew testDebugUnitTest assembleDebug` passes, with 11 JVM tests.
- `connectedDebugAndroidTest` on the emulator: the WebView test passed. The pairing test was skipped, because it needs a live pairing link.
- The daemon is installed and runs as LaunchAgent `dev.herdr-remote` on `*:7766`. Port 7767 is closed.

**Environment limits.**
- There is no physical phone and no Tailscale. Criteria that need them were not run.
- Reviewers may run one targeted Go test (`cd daemon && go test -run <Name> ./...`). Reviewers must not run the full suites, Gradle builds, the emulator, `install.sh`, or anything that changes the installed daemon, its state directory, the mod, or the user's hc terminals.

**Bot comments.** None.
