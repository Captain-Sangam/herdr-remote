• ROUND 2 app-features-a1

  Position

  Accept Reply first provisionally; swap Deny (#2) with the decision queue (#5). Keep catch-up sixth and helpers eleventh. Defer receipts, but correct the claim that transcripts make them redundant.

  Reasoning

  Deny mainly adds focus to a composer already enabled when unblocked (android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:199). The queue reduces navigation across several decisions. That offers broader value without more authority.

  Receipts have a distinct case: an approval response is lost, then the terminal exits. Its recorded result survives (daemon/actions.go:578, daemon/actions.go:20), while transcript access requires a live agent (daemon/transcript.go:635, daemon/api.go:68, daemon/state.go:75). A receipt can recover that result; it cannot resolve an unrecorded outcome.
  Deferral is reasonable.

  Helpers’ value is inferred, not measured. Eleventh is appropriate. Their verified relationship source is Herdr’s layout (daemon/state.go:301), not an established hc parent field; feature 11 should leave the security-change grouping.

  Outline

  • Catch-up: timestamps cannot identify the last seen item precisely: multiple blocks share one timestamp (daemon/transcript.go:383). Add stable session/item identity, bounded backfill and genuinely visible-item tracking. The proposed changes cannot deliver the current “Done when.”
  • Typed answers: one key plus one message contradicts the draft’s retained SR-6 constraint (docs/product.md:164). Prefer separately confirmed steps. Reading the field before sending reduces risk but does not eliminate a later screen change.
  • Interrupt: help promises a stop key and an outcome report, not guaranteed process preservation. Make dispatch semantics a prerequisite; turn checks alone cannot establish atomicity. Existing keys lack conditional dispatch (docs/interfaces.md:36). Adjust “Done when” accordingly.
  • Images: first-byte checks do not establish a valid image. Require decoding, dimension limits, metadata policy, bounded staging and cleanup. Upload-plus-message needs crash handling and explicit SR-6 treatment; “arrives once” is currently unsupported.
  • Launch/end: request IDs and confirmation alone are insufficient. Specify durable dispatch records, unknown outcomes, target identity and state rechecks. Launchers must fix executable, arguments and directory; validate task text as literal data.
  • Reply parity: “Depends on nothing” understates the work. Specify missing-transcript fallback, stale-agent handling and iOS presentation under its existing unlock policy (docs/adr/0025-ios-app.md:49). Android-only sizing cannot establish parity cost.

  Assumptions

  • Reply frequency remains unmeasured; “most frequent trip” should be labelled an assumption.
  • No implementation or behavioural probes ran.

  Confidence

  89

  END ROUND 2
