• ROUND 1 app-features-a1

  Position.

  Build features that shorten the cycle of checking, understanding and unblocking an agent. Prioritise guided decisions and catching up, then add narrowly guarded controls for stopping work, answering
  more questions and starting prepared tasks.

  Reasoning.

  The brief prioritises limited time away from the desk. Blocked-agent lists already exist (ListScreens.kt:167 (android/app/src/main/java/dev/herdrremote/ui/ListScreens.kt:167)); the next gain is
  reviewing decisions with less navigation. Existing transcript, prompt and action endpoints make this relatively small (daemon/api.go:21 (daemon/api.go:21)).

  Broader controls deserve lower confidence: the daemon restricts hc methods (daemon/hc.go:44 (daemon/hc.go:44)), and key dispatch retains a race (ADR 0010:53 (docs/adr/0010-approval-guard.md:53)).
  Every proposal below can use the shared API from native Android and iOS screens, consistent with ADR 0025:9 (docs/adr/0025-ios-app.md:9).

  Outline.

  All items require phone UI. Herdr.app and shipped mods stay unchanged; hc calls stay unchanged unless specified. Sizes include eventual iOS parity.

  1. Guided decision queue. Review each waiting prompt beside its recent transcript. Move to the next unresolved agent after each result, reducing repeated navigation.
     Changes: phone workflow using existing prompt, transcript and decision endpoints (daemon/api.go:21 (daemon/api.go:21)). Security: one explicit decision per tap, fresh prompt checks, no bulk
     approval. S; depends on the existing approval guard.

  2. Catch up since last read. Open an agent at the first unread message and jump to its latest answer. This makes returning after an interruption faster.
     Changes: phone read markers; add stable session/item identifiers to /transcript, whose current items contain role, kind, text and timestamp (docs/api.md:44 (docs/api.md:44)). Security: persist
     identifiers only; content stays in memory. M; depends on exact session identity and resetting markers when it changes.

  3. Stop the current turn. Request that a working agent stop, then inspect the reported outcome. This helps redirect mistaken work before it grows.
     Changes: dedicated interrupt endpoint and hc interrupt, advertised by its help. Needs a security change: interrupts are excluded (docs/product.md:69 (docs/product.md:69)); SR-2 restricts
     operations (docs/product.md:160 (docs/product.md:160)). Risk: interrupting the wrong process or turn. Safeguards: confirmed target, fresh state, Mac-idle guard, audited request ID and no
     automatic retry. M; depends on proving single-key behaviour and a new ADR.

  4. Expanded tool details. Open a tool step to inspect its longer command, edit payload or failure output. This helps assess the agent’s work without returning to the Mac.
     Changes: bounded transcript-item detail endpoint and phone viewer; tool text currently truncates at 2,000 characters (daemon/transcript.go:21 (daemon/transcript.go:21), daemon/transcript.go:390
     (daemon/transcript.go:390)). Security: only the identified transcript record, memory-only display, strict size limits; no arbitrary file access. M; depends on stable item handles and explicit
     truncation indicators.

  5. Action receipts. Reopen a recent send or decision to check the daemon’s recorded result after a connection failure. An unresolved outcome remains visibly unknown, discouraging accidental repeats.
     Changes: phone receipt list and authenticated /requests/{id} lookup, building on retained results (daemon/actions.go:62 (daemon/actions.go:62)). Security: return only that device’s action
     metadata, never content or raw logs; lookup never dispatches. M; depends on retaining device ownership and defining receipt expiry.

  6. Transcript search. Find a phrase, command or filename within the current agent’s conversation. Load earlier pages explicitly to extend the search, saving long scrolling sessions.
     Changes: phone search over existing paginated reads (AgentViewModel.kt:185 (android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:185)); no new endpoint. Security: memory-only index and
     queries, bounded loading, clear searched-range label. S; depends on pagination and distinguishing unavailable or truncated text from “no match.”

  7. Agent family view. See which helpers belong to a parent agent and which child needs attention. Open the relevant helper directly when a larger task stalls.
     Changes: expose parent relationships in agent responses and render a tree; current layout handling assigns children their parent’s placement (daemon/state.go:317 (daemon/state.go:317)). Security:
     read-only, local agents only. M; depends on fresh herdr-app ls relationships and fixing open-space membership refresh where it blocks this view (docs/operations.md:127 (docs/operations.md:127)).

  8. Codex choice questions. Answer recognised Codex multiple-choice questions from the phone. This removes another reason to return to the desk.
     Changes: prompt parser, decision validation and phone option rendering. Needs a security change: Codex questions are excluded (docs/product.md:68 (docs/product.md:68)); FR-12 currently permits
     Claude questions only (docs/product.md:150 (docs/product.md:150)). Risk: a misparsed answer reaches another prompt. Safeguards: verified layouts, fresh hashes, one key per action, unknown layouts
     view-only. L; depends on proven selection semantics, fixtures and a new ADR.

  9. Launch approved presets. Choose a task preset prepared on the Mac and supply its task text. Start useful work while away, even when no suitable agent is running.
     Changes: preset-list and launch endpoints; narrowly permitted hc run, currently outside the method allowlist (daemon/hc.go:44 (daemon/hc.go:44)). Needs a security change: SR-2 and the agent-start
     exclusion (docs/product.md:70 (docs/product.md:70)). Risks: arbitrary execution and duplicate launches. Safeguards: Mac-owned fixed executable/folder, literal task argument, confirmation, quotas
     and durable deduplication. L; depends on preset provisioning, reliable Herdr placement and a new ADR.

  Assumptions.

  • Ranking reflects inferred frequency and value; no usage telemetry was supplied.
  • Help establishes interrupt and template availability, not their complete safety semantics.
  • Search and catch-up initially cover live agents with exact transcript mappings.
  • Sizes are relative estimates, not delivery commitments.
  • I did not investigate external products, credentials, ignored files or the fixed iOS plan.
  • No files changed; no tests, builds or agent-control commands ran.

  Confidence.

  85

  END ROUND 1
