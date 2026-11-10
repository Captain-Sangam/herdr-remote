
## 3. Plan review, round 1 (blind)

Two fresh agents, which had not seen the planning debate, reviewed the plan against the brief. Each was asked for at most ten findings, each with evidence, likelihood (1 to 10), severity (1 to 10) and a proposed correction. They could not see each other's findings.

### Fable reviewer: 10 findings

1. **Codex in daemon mode breaks the transcript lookup** (likelihood 8, severity 6).
   - A Codex started as plain `codex`, which is how Herdr's agent menu and the user's own terminals launch it, runs its threads in a shared `codex app-server` daemon. No process of the terminal holds the rollout file.
   - The fallback is also thinner than the plan said: `terminal.read` returns only the visible screen while an agent is working or blocked.
2. **A revoked phone could resume a TLS session** (6, 7). Go skips `VerifyPeerCertificate` on resumed connections, and both Go and Android cache sessions by default. A revoked phone could resume past the key check, and the criterion "later handshakes fail" would be false.
3. **The overlay cannot wait 120 s** (9, 5). A mod's `exec` has a 30-second limit (`herdr.d.ts:10`), so a blocking `pair wait` breaks the overlay on any slow scan. `exec` also resolves the binary through `PATH`.
4. **Dead connections go unnoticed** (7, 5). With the read timeout off, a socket left half-open by a network change or Mac sleep is never detected, so background alerts stop silently.
5. **The guard was only tested on default-mode command prompts** (5, 6). Auto-mode prompts and Codex file-edit approvals might look different, and the Check would not notice.
6. **TLS and key settings are unstated** (7, 5). Each one, if guessed wrong, means every handshake fails: `RequireAnyClientCert`, TLS 1.3, the Keystore key's purpose and digests, an OkHttp hostname verifier that accepts IP addresses and MagicDNS names.
7. **No logging policy** (9, 4). Nothing says what is logged, or that tokens, codes and transcript text must never be. The device name sent during enrollment is attacker-controlled and unbounded.
8. **Foreground-service details are missing** (8, 4): the permissions, the service type property, and a restart after reboot or app update.
9. **`hc` might tag the phone's message** (5, 5). `hc` may wrap sent text as `<hc from=…>`, which the `hc` guide tells agents to treat as a report rather than a user instruction.
10. **The QR drawing is over-built** (5, 2). Drawing the QR with thousands of `div()` cells and a one-second countdown redraws the whole Herdr window every second.

Its completeness verdict: transcript, approve or deny, and security were partial; everything else was delivered.

### Astra reviewer: 10 findings

1. **A held key can approve an unseen action** (6, 9).
   - Someone typing at the Mac makes `hc` hold the phone's key. By the time it lands, the prompt may have changed, and the key approves an action nobody saw.
   - The per-terminal lock cannot exclude local input. It asked for this to be treated as unresolved.
2. **Restarts and retries can repeat actions** (6, 8). A daemon restart loses the in-memory marker, so a replay could press again. Messages have no idempotency at all. It wanted durable action IDs.
3. **Same-user processes are not separated** (5, 9). Any process running as the same user can read the server key or run `pair allow`, and file permissions cannot stop it. Either declare such processes trusted, or redesign the boundary.
4. **Project data leaks through phone surfaces** (7, 7): the recents screenshot after the biometric timeout, notification text on the lock screen, and logs.
5. **TLS resumption** (5, 5): the same issue as the Fable reviewer's second finding.
6. **Hostname verification** (8, 7). A certificate with the right key but without the current address still fails OkHttp's hostname check.
7. **The 30-second `exec` limit**: the same as the Fable reviewer's third finding.
8. **Foreground-service permissions**: the same as the Fable reviewer's eighth.
9. **xterm.js can overflow** (7, 6). Unbounded base64 delivery can outpace xterm.js, which drops data past its write buffer.
10. **Transcript pages can start mid-record** (9, 6). Reading the last 2 MB usually starts inside a JSON record, which would turn a valid transcript into terminal text. Pagination needs record alignment and cursors.

Its completeness verdict: almost every capability was partial.

### What the orchestrator verified

- **Codex daemon mode: confirmed.**
  - The user's terminal t1v runs `node /opt/homebrew/bin/codex`. Its processes hold only Unix sockets. The shared `codex app-server` holds a lock file per thread under `~/.codex/thread-writer-locks/`.
  - A Codex that has not been prompted has no rollout file yet.
  - Every rollout's first line records `cwd`, `source` and `parent_thread_id`, which is enough for a strict folder match.
  - A `threads` table in `~/.codex/state_5.sqlite` was also found. It was not used, because it is undocumented and would need SQLite.
- **TLS resumption: confirmed** from Go's documentation. `VerifyPeerCertificate` is not called on resumption, while `VerifyConnection` runs on every connection.
- **The 30-second `exec` limit: confirmed** (`herdr.d.ts:10`). Two more facts came out of the guide:
  - saving any non-hidden file in a mod folder reloads the mod (`AGENTS.md:789`);
  - every mod redraw redraws the whole window (`AGENTS.md:893`).
- **Other prompt kinds: verified with new probes.**
  - Claude Code's Write-tool prompt ("Do you want to create …?", "1. Yes", "Esc to cancel") and Codex's apply_patch prompt ("Would you like to make the following edits?", "1. Yes, proceed (y)", "3. No … (esc)") use the same markers as the command prompts.
  - A Claude auto-mode prompt cannot be produced on demand, so it remains an assumption that fails toward view-only.
- **Message tagging: refuted for this design.** A message sent with `hc send` from a process outside any terminal arrived as a plain prompt with no `<hc` tag, and the probe agent answered it ("PONG").
- **The held-key race: confirmed as residual, and a mitigation was found.** `ioreg -c IOHIDSystem` reports `HIDIdleTime`, the time since the last keyboard or mouse input on the Mac. It can serve as a "Mac in use" guard before and after the key press.
- **Everything else** was accepted on the reasoning given, with no contrary evidence.

### Corrections made for round 2

- **TLS:** TLS 1.3 only, `RequireAnyClientCert`, session tickets off, and the key check in `VerifyConnection`. Plus a test that a revoked phone cannot resume.
- **Pairing:** the overlay polls a `pair status` call that returns at once. It calls the daemon by an absolute path. The device name is limited to 64 printable characters. The QR is drawn as run-length rows with an expiry time instead of a countdown.
- **Connections:** the SSE read timeout is 30 s with automatic reconnect. Live views reopen after a reconnect, and the phone never retries an action by itself.
- **Live view flow control:** chunks of at most 64 KiB, each acknowledged by xterm.js. Above 1 MiB pending, the stream is reopened for a fresh paint.
- **Transcripts:** a third source for daemon-mode Codex, which matches by folder only when exactly one session fits. Pages are aligned to records, and a cursor is tied to the file's inode.
- **Approvals and messages:**
  - request IDs on both;
  - the audit log writes a `dispatching` line before the key and a result line after, and is reloaded on start, so a restart returns `outcome_unknown` and never presses twice;
  - the Mac-in-use guard (`mac_in_use` if the Mac had input in the last 10 s, `uncertain` if input arrives during the press);
  - messages never carry a `caller`.
- **Android:** the full list of foreground-service permissions, with restart on boot and on app update. The Keystore key parameters are spelled out. `FLAG_SECURE` is set, notifications are private, and transcript and terminal data stay in memory only.
- **Two new decisions:**
  - an audit log that never contains tokens, codes, transcript text, terminal bytes or message text;
  - a stated threat-model boundary: processes running as the same macOS user are trusted, because they can already drive `hc` directly.
- **Rejected or narrowed:**
  - a separate durable action store, because the audit log already is one;
  - leaving the race unresolved, because it is now mitigated and the remaining part is documented;
  - dropping the `div()` QR, because a file would put the token on disk;
  - reading Codex's SQLite index.
