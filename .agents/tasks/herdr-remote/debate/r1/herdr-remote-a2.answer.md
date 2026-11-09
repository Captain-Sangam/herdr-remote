• ROUND 1 herdr-remote-a2

  Position:

  Build a Go per-user Mac broker, a small Herdr pairing mod, and a Kotlin/Compose Android app. Use fixed hc connect subprocesses for RPC and fixed herdr-app ls calls for layout (hc connect --help; herdr-app --skill). Pair through a short-lived QR enrollment, local key confirmation, and pinned TLS over LAN or Tailscale. Start with visible-screen snapshots at 4
  Hz (terminal.read). Require explicit transcript binding when identity is uncertain. Support narrowly tested approval profiles; unknown prompts remain Mac-only. Background alerts use a user-started foreground connection service, with missed alerts recovered on reconnect.

  Reasoning:

  1. Security. Expose application operations, never an RPC proxy. Authentication precedes terminal access, transcript reads, or subprocess execution. This boundary matters because the core includes command and file operations (machine.exec, file.write). Pairing establishes a device key; network location grants no authority.

  2. Shipped compatibility. Layout comes from the window; terminal facts come from the core (herdr-app --skill; terminal.get). A block can be permission, question, or authentication, with optional classification (terminal.get, result Status.kind). Approval therefore requires a recognized prompt. Critically, terminal.keys has no conditional status or prompt
     parameter: freshness checks cannot eliminate the race with local interaction.

  3. Smallest useful system. One broker owns authentication, state, transcripts, and decisions. The mod only presents pairing and local administration; mods support execution and image assets (~/.config/herdr-app/herdr.d.ts:13; ~/.config/herdr-app/AGENTS.md:801).

  4. Responsive view. Poll only the selected terminal, one request outstanding, deduplicating unchanged screens. terminal.read(lines:0) supplies the visible screen without history scrolling (hc --skill). Measure latency before adding framed-stream decoding and an Android emulator.

  Outline:

  All paths below are proposed.

  1. protocol/openapi.yaml, broker/auth/, broker/store/. Define exactly state, events, screen, transcript, message, and decision; enrollment is separate. Reject unknown fields, oversized requests, arbitrary paths, RPC names, and remote-machine targets. Store broker secrets with user-only permissions; phone keys use Android Keystore. Test unauthorized access
     against every operation.

  2. broker/pairing/, mods/remote-pair/. QR contains protocol version, broker identity, LAN/Tailscale endpoints, TLS SPKI pin, and a random single-use enrollment secret expiring after two minutes. Phone proves possession of its signing key; Mac confirms its fingerprint before durable enrollment. Subsequent connections authenticate that key through TLS client
     authentication. Check revocation on every request and close revoked streams. Pin identity across both addresses. Expired, replayed, or mismatched enrollment fails toward deny. Test interception, stolen QR replay, address switching, and revocation during an active connection.

  3. broker/herdr/, broker/state/, broker/view/. Internally allow only hello, terminal.list/get/read/send/keys, and events.subscribe, plus fixed herdr-app ls. Restrict targets to validated local terminal identities. Messages use force:false, reject terminal control characters, and never accept client-supplied keys (terminal.send). Handle subscription
     snapshots, sequence numbers, heartbeats, and missed snapshots (events.subscribe). Failed reads produce a timestamped stale view and disable controls. Render terminal output as inert text.

  4. broker/transcript/. Search only the two transcript roots named in .agents/tasks/herdr-remote/brief.md:14. Use explicit session identifiers when available; cwd and modification time merely suggest candidates. Otherwise require local selection. Persist binding against core identity, terminal creation, foreground process, and session identity; invalidate on
     replacement or ambiguity (hello; terminal.get). Tail bounded JSONL pages, tolerate partial records, and reject symlink escapes. Ambiguity fails toward “transcript unavailable”; offer clearly labelled terminal text instead. Test concurrent same-directory sessions, resumes, truncation, and rotation.

  5. broker/approval/, fixtures/prompts/. Show agent, block classification, full fresh prompt, and exact proposed choice. Proposed profiles: Claude permits ["enter"] only when a certified fixture proves single-use Yes is selected; Codex permits ["y"] only when explicitly advertised; denial uses ["esc"] only when advertised. These are certification targets, not
     verified installed bindings. Pi, unknown versions, questions, and authentication fail toward deny. Bind expiring decisions to device, terminal incarnation, turn, and screen hash. Serialize phones and consume durable request IDs before dispatch; crashes return “outcome unknown” without retry, avoiding duplicate actions. Document the remaining read/keys
     race (terminal.keys).

  6. android/, scripts/, tests/. Build the requested screens and connection notification. Deduplicate blocked alerts; reconnect resynchronizes (events.subscribe). Install android-commandlinetools through Homebrew, install pinned SDK/emulator packages, then run Gradle unit tests, assembleDebug, and connectedDebugAndroidTest on a Mac emulator and physical phone.
     Test Doze, force-stop, network changes, stale approvals, and multiple phones.

  Assumptions:

  • Android service policy, dependencies, and prompt profiles require implementation-time verification.
  • Runtime inspection was sandbox-denied (hc ls; herdr-app ls).
  • Fully atomic approval requires an interface beyond shipped terminal.keys.

  Confidence:

  83

  END ROUND 1
