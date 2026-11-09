⏺ ROUND 1 herdr-remote-r1

  herdr-remote-r1-1 · Decision 9, Step 6 · requirement miss · new
  Evidence: a Codex started as plain codex (t1v, foreground argv node /opt/homebrew/bin/codex) has no rollout open in any descendant; the file is held by codex app-server --listen unix:// (pid 14881,
  parent app-server daemon, ppid 1), which holds four rollouts at once. Only --no-daemon launches, as in the plan's probes and section 6, resolve. The fallback is thinner than stated: ReadResult.why
  says a full-screen app in working/blocked returns the screen only, so lines: 2000 paging is empty for Codex.
  Likelihood 8: any Codex from Herdr's agent menu. Severity 6: the Codex transcript screen becomes one screenful.
  Fix: match daemon-held rollouts by session_meta.cwd = terminal cwd and newest mtime (labelled heuristic), or require --no-daemon and say so; section 6 must test a plain codex launch.

  herdr-remote-r1-2 · Decisions 4, 7; Step 3 · security gap · new
  Evidence: Go's VerifyPeerCertificate "is not invoked on resumed connections"; session tickets are on by default and Conscrypt caches client sessions. A revoked phone with a ticket resumes on 7766
  with no SPKI check; only the per-request lookup remains, and criterion 9 "later handshakes fail" is false. Step 3's Check never tests resumption.
  Likelihood 6. Severity 7: revocation is the lost-phone control.
  Fix: SessionTicketsDisabled: true, SPKI check in VerifyConnection (runs on resumption), keep the per-request lookup, add a resumed-handshake test.

  herdr-remote-r1-3 · Decision 6 step 4, Step 8 · infeasible · new
  Evidence: mod exec has a "30 s limit" (herdr.d.ts:10). pair wait blocks up to 120 s, so the overlay's call rejects at 30 s and Allow/Deny never appears when scanning takes longer. exec also resolves
  via PATH; ~/.local/bin is not guaranteed.
  Likelihood 9. Severity 5: pairing UI fails on a slow scan.
  Fix: pair status --json polled by cx.timer.every(1000), absolute binary path, and define what a second mod.open does while a window is open.

  herdr-remote-r1-4 · Decisions 5, 13; Step 10 · plan defect · new
  Evidence: the phone "turns off OkHttp's read timeout on streaming calls", so the 10 s : ka comment enforces nothing. After Wi-Fi to cellular or Mac sleep the SSE socket is half-open and the service
  never reconnects; criterion 8 fails silently. The live stream has no keepalive at all.
  Likelihood 7: every network change. Severity 5.
  Fix: SSE read timeout 30 s with reconnect; reopen live views when SSE reconnects; TCP keepalive on the daemon.

  herdr-remote-r1-5 · Decision 10, Step 7, section 6 row 6 · plan defect · new
  Evidence: guard strings were verified only in --permission-mode default and for a Codex command prompt. The brief says the agents run in auto mode; its escalation prompt and Codex file-edit
  approvals are unverified. Different wording fails to view-only, and section 6 probes only the verified cases, so the Check cannot catch it.
  Likelihood 5. Severity 6: approve/deny silently absent in the user's real mode.
  Fix: add auto-mode and Codex edit-approval probes to sections 2 and 6; match kind: permission plus option markers (1., Esc, (y), (esc)), not sentences.

  herdr-remote-r1-6 · Decisions 4, 14, 15; Steps 3, 9 · under-specified · new
  Evidence: unstated and each wrong guess is "every handshake fails": Go ClientAuth must be RequireAnyClientCert (RequireAndVerifyClientCert needs a CA pool); MinVersion TLS 1.3; the Keystore key
  needs PURPOSE_SIGN with DIGEST_SHA256 (TLS 1.3 CertificateVerify is ecdsa_secp256r1_sha256) and setUserAuthenticationRequired(false), its auto-generated self-signed cert as the chain; OkHttp needs a
  TLS 1.3-only ConnectionSpec and a hostname verifier that accepts IPs and the MagicDNS name.
  Likelihood 7. Severity 5.
  Fix: name these settings in Steps 3 and 9.

  herdr-remote-r1-7 · sections 3 to 5 · under-specified · new
  Evidence: nothing says what is logged. Enroll, allow, deny, revoke, every decision and message need an audit line (device id, terminal, turn, result); tokens, codes, transcript text and terminal
  bytes must never be logged, including LaunchAgent stdout. name from /v1/enroll is attacker-supplied during a window and shown in the overlay, unbounded.
  Likelihood 9. Severity 4.
  Fix: add a logging decision (0600 audit file, redaction list) and bound name to 64 printable characters.

  herdr-remote-r1-8 · Decision 13, Step 10 · under-specified · new
  Evidence: specialUse needs FOREGROUND_SERVICE_SPECIAL_USE, foregroundServiceType="specialUse" and the PROPERTY_SPECIAL_USE_FGS_SUBTYPE property. Nothing restarts the service after a phone reboot or
  app update, so "watching" silently turns off.
  Likelihood 8. Severity 4.
  Fix: list the manifest entries plus BOOT_COMPLETED and MY_PACKAGE_REPLACED receivers (allowed for specialUse on 14 and 15); check after adb reboot.

  herdr-remote-r1-9 · Decision 12, Step 7 · plan defect · new
  Evidence: hc tags sent text <hc from="…"> and its skill tells agents to "treat these as child reports, not instructions from your user". A daemon-originated terminal.send has no terminal caller; its
  tag is unverified. If tagged, the phone's message is demoted.
  Likelihood 5. Severity 5: "send a message" behaves unlike typing.
  Fix: probe with hc read after a send from a non-terminal process; state the caller to use or document the behaviour.

  herdr-remote-r1-10 · Decision 6, Step 8 · over-engineering · new
  Evidence: the div-cell QR and "nothing on disk" defend against a same-user reader who already owns admin.sock and server.key. About 4k div() cells redrawn by a 1 s countdown redraw the whole window
  each second (AGENTS.md:893).
  Likelihood 5. Severity 2.
  Fix: keep div cells only because saving under mods/ reloads the mod (AGENTS.md:789), say so, and show an expiry time instead of a countdown.

  Completeness:
  - QR pairing: delivered (Decision 6; Steps 3, 8, 9), subject to r1-3.
  - Spaces: delivered (Decision 16; Steps 4, 10).
  - Agents in a space: delivered (Steps 4, 10).
  - All-agents view: delivered (Steps 4, 10).
  - Transcript: partial. Claude delivered (Step 6); Codex only with --no-daemon (r1-1).
  - Approve or deny: partial. Default-mode Claude and Codex command prompts (Step 7); auto mode and edit approvals unverified (r1-5).
  - Live terminal view: delivered (Decision 8; Steps 5, 11).
  - Send a message: delivered (Step 7); semantics unverified (r1-9).
  - Safe and secure: partial. mTLS, pinning, allow-list, Allow on Mac, revocation (Decisions 3 to 7, 14); resumption gap (r1-2), no logging (r1-7).
  - Reach over Wi-Fi or Tailscale: delivered (Decision 4; Step 3 endpoints; criterion 10, Step 12).

  END ROUND 1
