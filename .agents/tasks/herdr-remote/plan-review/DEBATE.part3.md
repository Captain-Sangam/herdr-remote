
## 4. Plan review, round 2

Each reviewer got the orchestrator's verdict on every finding from both reviewers, the corrections, and the revised plan. The task was to accept, reject or amend each correction with evidence, and to check the revised plan for anything the corrections broke.

### Fable reviewer

**Accepted** the corrections for TLS resumption, the `exec` limit, half-open sockets, unverified prompt types, the TLS and key settings, logging, the foreground service and the QR drawing. **Withdrew** the message-tagging finding, because the orchestrator's probe refuted it.

**Amended the Codex daemon-mode fix:**
- As written, it still fails on this Mac. t1v (daemon mode) has two candidate rollouts after its start time. One of them belongs to t1x, which runs with `--no-daemon` in the same folder. "Several candidates" sends t1v to terminal text.
- Fix: remove rollouts that the explicit lookups already bind to other terminals.
- Also: the daemon closes idle rollouts, so a scan by modification time is right, and `lsof` on the daemon is not.

**Amended the Mac-in-use guard:**
- `HIDIdleTime` works here, but it sees only local keyboard and mouse input, and perhaps Screen Sharing (not verified). It does not see SSH, or `hc keys` from agents.
- A mouse jiggler or any device with periodic reports would keep the Mac "in use" forever, so the phone could never approve.
- Fix: make the threshold a setting, return the measured idle time in the refusal, and audit refusals.
- A sleeping display is fine, because idle time only grows.

**New findings:**
- The audit line has no prompt hash or decision, so the consumed marker cannot be rebuilt after a restart as the plan claims. Nothing says the `dispatching` line is fsynced before the key, that a failed write refuses the action, or that the reload reads the rotated file.
- `/v1/enroll` long-polls up to 120 s, but OkHttp's default 10 s read timeout would abort it before Allow. The planned check delayed the scan, not the Allow, so it would not catch this.
- Opening the overlay always opened a pairing window, so revoking a lost phone also opened port 7767 for 120 s.
- The testing table asked for a manual check on "a plain shell", which is not an agent under the plan's own rules, so that check could not run.

Its conclusion: nothing else material remained.

### Astra reviewer

**Accepted** most corrections, and **rejected** its own reading of the tagging finding in light of the probe.

**Amended the Codex daemon-mode fix:** a unique candidate is not an identity. Terminal A has had no prompt yet, so it has no rollout. Terminal B starts later in the same folder and makes one. A would show B's transcript. It required an exact binding or the terminal-text fallback.

**Amended the Mac-in-use guard:**
- The check ran before the lock, so a wait for the lock could make it arbitrarily stale.
- Move it to just before the key, fail closed when the idle time cannot be read, and call it a heuristic.
- Criterion 6 promised more than the guard delivers.

**Amended the idempotency fix:**
- Persist the request ID, a digest of the payload and the prompt hash before the action, and fail closed if that write fails.
- Make sure rotation and the 10-minute window cannot forget IDs.
- The message criterion must promise "at most once", not "exactly once".

**Other amendments:**
- Content-free logging must also cover the debug build that the testing section installs.
- The transcript step said "never an error", but the plan also returns 409 for a stale cursor.

**New finding, the most severe of the round:** the TLS decision applied the device-store check to both listeners. Built literally, the enrollment listener would reject every new phone, because a new phone is not in the store yet. First pairing would always fail.

### What the orchestrator verified

- **Codex lock files.** `lsof +D ~/.codex/thread-writer-locks` showed:
  - the daemon-mode app-server holds the lock of t1v's thread (t1v had no prompt yet);
  - each `codex --no-daemon` process holds its own lock;
  - the ChatGPT desktop app's own app-server holds its own.
- **t1x.** It is a `codex --no-daemon` session in the same folder as t1v, started later. This confirmed the Fable reviewer's counter-example.
- **The other items** were checked against the plan text. Each was a real defect in the revision.

### Resolution

- **Codex daemon mode.** The two amendments were combined into one rule: bind by elimination.
  - Candidates: rollouts of live daemon threads (locks held by an app-server) that match the folder, source and start time, minus rollouts that other terminals already own.
  - The binding holds only if there is exactly one candidate and exactly one unbound Codex terminal in that folder.
  - This handles the Fable case (t1x's rollout is excluded), the Astra case (two daemon-mode terminals in one folder fall back to terminal text), and a session that ended (its lock is gone).
- **Approval guard order.** The guard now runs in this order:
  1. request ID;
  2. lock;
  3. state, hash and markers;
  4. consumed marker;
  5. the idle check, last;
  6. a fsynced `dispatching` audit line with the prompt hash and decision;
  7. one key.

  Other rules:
  - A failed audit write gives 503 and no key.
  - The threshold is a setting, and 0 turns the guard off.
  - An `ioreg` failure refuses.
  - The daemon rebuilds IDs and markers from both the current and the rotated audit log.
- **Messages.** A fsynced line with the request ID and a SHA-256 of the text, sent at most once.
- **Pairing and the overlay.**
  - Enrollment gets a 130 s client timeout, and a test with Allow given 60 s late.
  - The overlay opens to the device list, and only **Pair a phone** opens port 7767.
- **Listeners.** The checks are written separately for each listener: the device store on 7766, and only a well-formed certificate plus the token on 7767. A test enrolls a never-seen key through the real 7767 listener.
- **Other fixes:** no content logs in any build; the stale-cursor wording; the testing-table row.
- **The documents.** The orchestrator then wrote the corrected plan as the two final documents, `PRD.md` and `TDD.md`, for the reviewers to check in round 3.
