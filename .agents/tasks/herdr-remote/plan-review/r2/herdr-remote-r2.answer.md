• ROUND 2 herdr-remote-r2

  • herdr-remote-r1-1: amend. Decision 9’s unique candidate is not an identity. Terminal A has no rollout; B starts later in the same folder and produces one. A selects B’s transcript (§2 confirms
    unprompted terminals have none). Require an exact binding or terminal-text fallback.

  • herdr-remote-r1-2 / herdr-remote-r2-5: accept. Decision 4 disables tickets and uses VerifyConnection; Step 3 tests revocation.

  • herdr-remote-r1-3 / herdr-remote-r2-7: accept. Immediate status polling replaces the overlong exec; Step 8 tests delayed scanning.

  • herdr-remote-r1-4: amend. Decision 5 covers SSE/live recovery, but enrollment’s 120-second long-poll still lacks an explicit client timeout.

  • herdr-remote-r1-5: accept. §2 supplies command/edit evidence; §7 retains view-only fallback for unmatched prompts.

  • herdr-remote-r1-6 / herdr-remote-r2-6: accept. Decisions 4/14 and Step 9 specify Keystore signing, both pin checks and endpoint-name tests.

  • herdr-remote-r1-7: accept. Decisions 6/17 bound names and exclude sensitive audit content.

  • herdr-remote-r1-8 / herdr-remote-r2-8: accept. Decision 13 specifies permissions, service type, receivers and foreground startup.

  • herdr-remote-r1-9: reject original finding. The verified outside-terminal probe supports untagged delivery; Step 2 prohibits caller.

  • herdr-remote-r1-10: accept. Run-length rows and change-only redraws address the identified concern.

  • herdr-remote-r2-1: amend. Decision 10 checks idle before acquiring the lock, so waiting can make it arbitrarily stale, not merely milliseconds old. SSH/hc input does not require HID activity;
    remote-desktop coverage remains unverified. Move the check immediately before dispatch, fail closed on unreadable idle state, and test remote input plus sleep/wake. Describe this as a heuristic,
    including unrelated-mouse false refusals; criterion 6 still overpromises prevention.

  • herdr-remote-r2-2: amend. Decision 17 omits prompt_hash, so consumed markers cannot be reconstructed. No fsync/failure ordering is specified; rotation or the ten-minute cutoff can forget IDs and
    permit replay. Persist identity, payload digest and prompt hash before dispatch, fail closed on persistence failure, and preserve records through a defined replay lifetime. Test crashes and
    rotation. Criterion 7 must promise at-most-once dispatch, not exactly-once delivery.

  • herdr-remote-r2-3: accept. Decision 18 explicitly establishes the same-UID trust boundary.

  • herdr-remote-r2-4: amend. Privacy controls address exposure, but content-free logging must cover the debug APK installed by §6, not only release builds.

  • herdr-remote-r2-9: accept. Decision 8 bounds delivery; Step 11 exercises heavy output.

  • herdr-remote-r2-10: amend. Alignment/cursor tests address paging, but Step 6’s “never … an error” contradicts Decision 9’s stale-cursor 409.

  herdr-remote-r2-11 — Decision 4, Step 3; plan defect; disposition new.
  Evidence: Decision 4 applies the client-SPKI check and device lookup to both listeners. A fresh phone is absent from the store, so literal implementation rejects enrollment before Allow.
  Likelihood 8: shared TLS/middleware implementation.
  Severity 8: first pairing fails.
  Correction: Explicitly restrict device-store authorization to 7766; 7767 requires certificate possession and token validation. Test fresh enrollment through the actual listener.

  END ROUND 2
