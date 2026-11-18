⏺ ROUND 2 app-features-a2

  Position. I accept the list and the three added features, with two rank changes and four corrections. Helpers under their parent (11) moves to 10, because it crosses no rule. Saved replies (7) moves above Catch up (6), because it is S with no open dependency. Reply from the alert stays first at S. Send an image stays 13th.

  Reasoning.
  - Challenge 1. I accept all three.
    - Next blocked agent is phone-only, and 5th is right.
    - Catch up is M and waits on a session marker, so it goes below Saved replies.
    - Full tool steps at 9th is right.
  - Challenge 2, parity. Parity is the fourth criterion and the sizes are for Android, so neither changes. On iOS the feature follows alert delivery.
  - Challenge 2, SR-12. There is no new kind of conflict. ADR 0019 already puts the prompt block in the system notification. But agent text in the notification shade is outside app memory, and Android notification history can keep it. The ADR 0019 amendment must say so and cap the preview length.
  - Challenge 3. I keep it, marked. The brief allows a feature that crosses a rule when it names the rule, the risk and the safeguard. Its value is real but narrow, so 13th is fair.

  Outline. Plan check:
  1. Feature 1, stale alert. A finished alert stays after the agent works again: Reducer.alerts returns null for done to working (android/.../watch/Reducer.kt:82-96). A Reply then lands in the middle of other work. Add: hide the alert when the agent leaves its finished state, and add that to "Done when".
  2. Feature 1, request ID. Make the ID at send time, not when the alert posts. The daemon refuses an ID older than 5 minutes (daemon/actions.go:19).
  3. Feature 2. The idle state after esc is verified for Claude only. Codex raises its turn after esc (docs/interfaces.md:154), and its state then is not recorded. Add a probe.
  4. Contradiction. Section 1 says features 10 to 14 need a security change. Feature 11 does not.
  5. Contradiction. Decision 5 says four features start, end or write. Only 12, 13 and 14 do.
  6. Contradiction. Section 7 says each feature keeps SR-6. Feature 10 changes it.
  7. Feature 11. The daemon already reads sub_agents from herdr-app ls (daemon/state.go:301-317), so the dependency is met. The size is S to M.
  8. Feature 13 safeguard. A check of the first bytes does not stop a file that is an image and something else. What limits the risk is that hc picks the path and nothing runs the file. Add a count cap and a clean-up, against a full disk.
  9. Feature 12. State the working folder: the space folder from herdr-app ls (docs/interfaces.md:88), never from the phone.
  10. Reference. The Status struct is at daemon/hc.go:77-82, not 76-81.

  Assumptions.
  - Android notification history is off by default. I did not check it.
  - I did not check ListScreens.kt:48 or state.go:315-318 line by line.

  Confidence. 80

  END ROUND 2
