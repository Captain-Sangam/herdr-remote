# Brief: herdr-remote

**Name.** `herdr-remote`. Agents: `herdr-remote-a1` (claude), `herdr-remote-a2` (codex).

**Task.** Plan an Android app, plus whatever must run on the Mac, that lets the user control their Herdr.app sessions from the phone. Pairing: Herdr shows a QR code, the phone scans it and is connected. Screens: a list of spaces; the agents inside a space; an all-agents view; per agent, its chat transcript, approve or deny a pending action, and a live view of its terminal. It must be safe and secure: the phone becomes a remote control for coding agents that run in auto mode on the user's Mac. Done means a plan an implementer can build step by step, for one user, one Mac, one or more phones, sideloaded.

**Repos.**
- `/Users/zeus/code/herdr-remote` (primary, new, branch `main`, no commits). Everything is built here: the Android app and the Mac-side component.
- No source for Herdr. Herdr.app 0.1.13 came from a DMG; herdr-core (`hc`) is 0.1.18. They are fixed dependencies, reached only through their public interfaces:
  - **herdr-core API.** JSON-RPC 2.0, one message per line on a local Unix socket (`$HC_TERMINAL_SOCKET`, here `~/.local/state/herdr-core/default/default.sock`), or over stdio with `hc connect`. Full schema: `hc schema`, copied to `$S/hc-schema.json`. Useful: `terminal.list`, `terminal.get`, `terminal.read` (screen plus history as text), `terminal.attach` (raw PTY stream on a framed connection; `observe: true` = watch only, never types or resizes), `terminal.keys`, `terminal.send`, `terminal.interrupt`, `events.subscribe` (snapshot then status/turn/title/created/finished changes, 15 s heartbeat). Dangerous: `terminal.run`, `machine.exec`, `file.read/write/edit`, `doc.set`, `server.stop`.
  - **Window model.** Spaces and tabs belong to Herdr.app, not the core. `herdr-app ls` returns `spaces[{id,name,folder,machine,tabs[{id,name,terminals[{terminal,pane,status,app}]}]}]`.
  - **Agent status.** Programs report `idle | working | done | blocked | error` over OSC 7501 plus `app` (`claude-code`, `codex`, `pi`). `blocked` means a person is needed (permission or question). `terminal.send` refuses a blocked program unless `force`; `terminal.keys` has no such refusal (schema errors list).
  - **Mods.** Herdr.app runs trusted JavaScript mods from `~/.config/herdr-app/mods/<name>/`. A mod can run programs (`exec`), use the network, and show images and SVGs from its folder (`~/.config/herdr-app/AGENTS.md:801-804, 971, 996`; API in `herdr.d.ts`, e.g. `agents()` at line 181). Shipped mods are Herdr's; copy with `mod.clone` before editing.
  - **Transcripts.** Claude Code writes JSONL under `~/.claude/projects/<cwd-slug>/<session>.jsonl`; Codex under `~/.codex/sessions/YYYY/MM/DD/`. hc terminal facts (`argv`, `cwd`, `pid`, `foreground`) carry no session id.
- Local toolchain: Java 17, Go, Swift. No Android SDK, no `adb`, no Rust.

**Binding rules.** No repo rules yet. The user's global rule: drive Herdr only with `hc` and `herdr-app`, never the legacy `herdr` CLI, never `HERDR_ENV`.

**Criteria**, in order: (1) security: nothing that reaches the Mac's network can run commands, read files or drive agents without a paired device key, and a lost phone can be revoked; (2) works with Herdr as shipped, no patches; (3) smallest system that does the listed screens well; (4) a responsive live terminal view.

**Non-goals.** Changing Herdr.app or herdr-core. iOS. Multi-user or team sharing. Play Store release. Starting new agents, spaces or tabs from the phone. Remote machines added with `hc machine add`.

**Known facts.** All of the above was read from the installed binaries and guides on 2026-11-08. The phone must never get the raw hc API: it includes `machine.exec` and `file.write`.

**User decisions (2026-11-08).**
1. Reach: a direct connection from the phone to the Mac, over the same Wi-Fi or over Tailscale when away. No relay server of ours.
2. Control depth: watch (live view, transcript), approve or deny a blocked action, and send a text message to an agent. No raw keyboard into the terminal in this version.
3. The plan includes installing the Android command-line tools with Homebrew (`android-commandlinetools`) and building with Gradle. No Android Studio.

**Questions for the agents.**
4. Mac-side component: language, how it talks to hc and `herdr-app`, and the exact allow-list it exposes.
5. Pairing and transport: what the QR holds, how the device key is bound, TLS pinning or another channel, revocation, replay. It must work on the LAN address and the Tailscale address.
6. Live view: raw `terminal.attach` stream into an Android terminal emulator, `terminal.read` snapshots, or `herdr-app shot` frames.
7. Transcript: how to map a terminal to its Claude Code or Codex session file, and the fallback.
8. Approvals: what the phone shows for a blocked agent and which keys it sends, per agent kind; and how the phone learns of a blocked agent when the app is in the background, without a relay or push server of ours.
