⏺ ROUND 2 herdr-remote-a1

  Position
  1. Concede. A plain pipe works (fact 1). A PTY only adds queries that we would have to discard.
  2. Concede lsof, with two refinements:
     - npm Codex shows node as the foreground process, so walk its descendants for codex.
     - If more than one rollout file is open, use the one whose session_meta.parent_thread_id is null.
  3. Accept decision 14.
  4. Accept, with keepalives (Outline d).
  5. Accept both.
  6. Object. The failing sequence:
     - Two phones tap Approve, or one phone retries.
     - Request 1 passes the guard and presses 1.
     - Request 2 gets the lock immediately. Claude has not reported working yet, so state, turn and hash all still match.
     - A second 1 is pressed. It goes into the input box, or into the next dialog if one opens during the ~3 s quiet wait.

     Fix, with no durable table: keep an in-memory pressed(turn, hash) marker per terminal, and make the guard refuse a match. After the key is pressed, compare KeysResult.turn and status with the expected values and report a mismatch.

  Reasoning
  - hc get t21 gives foreground.name: node. The Codex session_meta keys include parent_thread_id (round-1 jq).
  - terminal.keys follows the "Same quiet rule as send" and has no blocked refusal. KeysResult returns status and turn (schema).
  - Status is the program's own OSC 7501 report (hc --skill:199-200), so it comes after the key.

  Outline
  - a. Step 3's Check revokes a live stream, but no stream exists until Steps 4 and 5. Move that check to Step 5.
  - b. Decision 6 keeps the token only in memory. But image() accepts only a file path (gpui-kit.d.ts:20290), so Step 8 would write the token to disk. Draw the QR with div() (gpui-kit.d.ts:20271) from pair --json instead.
  - c. Decision 4: the TLS handshake cannot see the request path.
    - Accept unknown keys only while a pairing window is open, and use middleware to limit them to /v1/enroll.
    - Enroll long-polls until Allow.
    - Bind the token to the first SPKI that uses it, so that the same phone can retry after a dropped connection.
  - d. Decisions 5 and 13: hc heartbeats every 15 s, and an idle terminal sends no bytes. OkHttp's default read timeout is 10 s. Send SSE comments, and turn off the read timeout on both streams.
  - e. Decision 2: send and keys wait for quiet. If one hc connect handles requests one at a time, reads stall behind them. Give actions their own connection.
  - f. Decision 8: when resize_notifications is false, the observer must track size itself (AttachResult). A repaint after a resize is not verified, so reopen the stream on each size event.
  - g. Step 2's invariant covers RPC only. Run every subprocess (hc attach, herdr-app ls, lsof, pgrep) through one helper with fixed arguments, and test it.
  - h. Decision 13: deduplicate by turn plus prompt hash.
  - i. Criterion 3: renaming a space sends no hc event. Run herdr-app ls again on each /v1/spaces request, and drop spaces that have machine set.
  - j. Step 10: add the POST_NOTIFICATIONS permission and the specialUse foreground-service type.

  Assumptions
  - A single hc connect may handle requests one at a time.
  - OkHttp's default read timeout is 10 s.
  - Claude's working report reaches hc after the lock passes to request 2.

  Confidence
  78

  END ROUND 2
