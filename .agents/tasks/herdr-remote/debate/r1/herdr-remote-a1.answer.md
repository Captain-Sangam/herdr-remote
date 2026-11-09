⏺ ROUND 1 herdr-remote-a1

  Position
  One Go daemon, herdr-remoted, started by a LaunchAgent, is the only thing the phone can reach. It talks to hc through hc connect subprocesses (JSON-RPC on stdio) and to the window through herdr-app ls. It serves a fixed HTTPS and WebSocket API on :7766, behind mutual TLS.

  - Pairing: a small Herdr mod, opened with mod.open, shows a QR code (SVG). The QR holds the daemon's SPKI SHA-256, its LAN and Tailscale addresses, the port, and a single-use token that expires after 120 s.
  - Device key: the phone makes a P-256 key in Android Keystore. It connects with the daemon's hash pinned, presents a self-signed client cert, and redeems the token. The daemon stores the client's SPKI hash. Pinning by key hash, not by hostname, means any address works.
  - Revocation: herdr-remoted devices rm (or a button in the mod) deletes the device and closes its open connections.
  - Live view: the daemon runs hc attach --observe <t> inside a PTY sized to the terminal. It streams the bytes over a WebSocket to xterm.js in a WebView and throws away anything the phone sends on that socket. Fallback: terminal.read.
  - Transcript, Claude: Terminal.pid → ~/.claude/sessions/<pid>.json (sessionId, cwd) → ~/.claude/projects/<slug>/<sessionId>.jsonl. I checked this chain on t20.
  - Transcript, Codex: the newest rollout whose session_meta.cwd matches and whose timestamp is at or after created_ms. If the match is ambiguous or missing, fall back to terminal.read with history.
  - Approvals: the phone shows status.kind, status.msg and the last lines of the screen. Approve or Deny sends {turn, msg_hash, req_id}. The daemon checks again that the agent is still blocked with the same turn and msg and that the expected option text is on screen. Then it sends one fixed key:
    - Claude: 1 to approve, esc to deny.
    - Codex: y to approve, esc to deny.
    - Questions: a digit from 1 to 9.
    - auth and unknown apps: view only.
  - Background alerts: an opt-in Android foreground service keeps the /v1/events WebSocket open and posts a local notification when an agent turns blocked.

  Reasoning
  1. Security.
     - Raw hc exposes machine.exec, file.write and terminal.run (schema methods). The daemon holds a fixed list of methods: hello, terminal.list, terminal.get, terminal.read, events.subscribe, terminal.send (never with force), and terminal.keys (only keys from the fixed map).
     - terminal.keys has no blocked refusal, but terminal.send does (schema errors), so the daemon must check the turn itself.
     - turn "goes up each time the status leaves working" (schema Terminal.turn), so turn plus msg identifies one dialog.
     - mTLS rejects unknown keys during the handshake, before any HTTP is read.
     - Message text has C0 and C1 control characters stripped, except \n and \t, so ESC[201~ cannot escape the bracketed paste.
  2. Works with Herdr as shipped.
     - The plan uses only public CLIs and a user mod: mod.open (herdr-app commands), and exec and image() (AGENTS.md:801-804, 971).
     - The framed wire layout is not in the schema; only frame names appear (AttachResult.channel). So the daemon does not speak it. hc attach --observe gives the same stream (hc attach --help).
  3. Smallest system.
     - One binary, one mod, one app.
     - herdr-app shot captures the whole window (herdr-app --skill, around lines 127-131), not one agent.
     - Terminals already carry their space id in the label herdr.space (hc get t20).
  4. Responsive live view. Attach sends only changes, in color. Polling terminal.read gives plain text only.

  Outline
  - daemon/ (Go):
    - main.go: the serve, pair and devices commands.
    - hc.go: RPC calls, limited to the allowed methods.
    - tls.go, pair.go, api.go.
    - live.go: the PTY, using creack/pty.
    - transcript.go.
    - Dependencies: creack/pty, coder/websocket and a QR library.
  - mod/herdr-remote/: main.js and mod.json. It runs herdr-remoted pair --svg and shows the QR with image().
  - install.sh: installs the mod and the LaunchAgent plist.
  - android/ (Kotlin, Compose):
    - Screens: Pair (GMS code scanner), Spaces, Space agents, All agents, and Agent (Transcript, Live, Approve/Deny, Message).
    - WatchService for background alerts.
    - OkHttp with a Keystore KeyManager and a pinned TrustManager; xterm.js bundled in assets; the Keystore key set to setUnlockedDeviceRequired.
  - Build order: hc.go with tests → TLS and pairing → API → mod → Android pairing and lists → agent screens → live view → background service.
  - Security invariants and failure direction:
    - Unknown or revoked key: the handshake fails (toward deny).
    - Pairing token used or expired: refused (toward deny).
    - Turn, msg or screen does not match: HTTP 409 and no key is pressed (toward deny).
    - hc is down: HTTP 503, and the phone shows its last data with its age (toward a stale view).
    - hc reports missed events: the daemon takes a new snapshot (stale view, then fresh).
    - A POST is retried with the same req_id: the daemon returns the cached result for 10 minutes (no duplicate action).
    - Transcript paths are built only on the daemon, and their real path must be under ~/.claude/projects or ~/.codex/sessions (toward deny).
    - Terminal addresses must match ^t[0-9a-z]+$ and be a local agent. machine/… is rejected (toward deny).
  - Verification, daemon:
    - go test ./... with a fake hc on stdio. The tests assert that no method outside the list is ever sent and that a stale turn presses no key.
    - Integration tests run against a separate core (hc connect --session herdr-remote-test), never the user's own.
    - openssl s_client without a client cert must fail.
  - Verification, Android on this Mac:
    - brew install --cask android-commandlinetools
    - sdkmanager "platform-tools" "platforms;android-35" "build-tools;35.0.0"
    - ./gradlew testDebugUnitTest assembleDebug: JVM tests for the QR parser, pinning and transcript parsing.
    - Instrumented tests: sdkmanager "emulator" "system-images;android-35;google_apis;arm64-v8a", avdmanager create avd, emulator -no-window, then ./gradlew connectedDebugAndroidTest.
    - Install on the phone with adb install over USB or wireless debugging.

  Assumptions
  - Claude Code accepts 1 in its permission menu, and Codex accepts y. Check this against real screens before shipping.
  - hc attach --observe runs correctly when its terminal is a PTY whose size matches the agent's terminal.
  - Codex has no pid-to-session file. I wanted to check lsof for its open rollout file, but that is outside the allowed commands.
  - ~/.claude/sessions/<pid>.json is an undocumented Claude Code file. The terminal.read fallback covers it if it changes.
  - herdr-app ls and the hc socket can be reached from a LaunchAgent environment.
  - An Android 14+ foreground service of type specialUse is fine for a sideloaded app.
  - The format of OSC 7501 test reports for the isolated test core still needs to be read from its docs.

  Confidence
  72

  END ROUND 1
