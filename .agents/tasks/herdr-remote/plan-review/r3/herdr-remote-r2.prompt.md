You are herdr-remote-r2.

This is round three, the last round of the plan review. Same rules as before: read-only evidence commands only, no edits, no hc actions, no herdr-app do, no secrets, no subagents, answer directly in this terminal. Do not read the record or the debate folder.

The orchestrator accepted every round-two item from both reviewers after checking each against the plan and the tools. One extra fact was verified: `lsof +D ~/.codex/thread-writer-locks` shows the daemon-mode app-server (pid 14881) holding the lock of t1v's thread, each `codex --no-daemon` process holding its own lock, and the ChatGPT app's codex app-server holding its own. t1x is `codex --no-daemon` in the same folder as t1v. So r1-1/r1-11 and the Astra amendment were combined into an elimination rule: live daemon threads (locks held by an app-server) whose first-line session_meta matches folder, source and time, minus rollouts bound to other terminals, and exactly one unbound Codex peer terminal in that folder.

Where each round-two item now lives:
- r1-1, r1-11, Astra's r1-1 amendment: TDD 3.6 "Transcript sources" item 3; Step 6 tests.
- r1-12, Astra's r2-2 amendment: TDD 3.5 rules 3, 6, 9, 10; 3.6 audit line, ID lifetime, rotation; decision 12; criterion 7 says at most once.
- r1-13 and Astra's r1-4 amendment: decision 5 (/v1/enroll 130 s); Step 3 test; section 6 row 2.
- r1-14: decision 6, 3.3 rule 1, 3.8, Step 8 check.
- r1-15: section 6 row 4.
- Astra's r2-1 amendment: 3.5 rule 5 (idle check last, under the lock, fail closed, `mac_idle_seconds`, measured idle in the 409, refusal audited, limits stated); criterion 6 and section 7 reworded.
- Astra's r2-4 amendment: decision 14 (no build logs content); SR-9 in the PRD.
- Astra's r2-10 amendment: Step 6 invariant.
- r2-11: decision 4 "Checks per listener"; Step 3 first test.

The plan is now written as the two final documents, as drafts:
- TECHNICAL DESIGN: /Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/plan-review/drafts/TDD.md (the implementer builds from this; it keeps the plan structure: outcome, current behaviour, decisions with architecture, pairing, API, approval guard, files and formats, Android and mod design, scope and criteria, steps, testing, constraints, verification).
- PRODUCT REQUIREMENTS: /Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/plan-review/drafts/PRD.md.
The previous plan revision, for comparison, is /Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/plan-review/plan-v3.md.

Read both drafts in full. Your job:
1. Check that each round-two item above is fixed correctly in TDD.md. Say fixed, or not fixed with evidence.
2. Check the drafts as the documents an implementer and the user will act on, without asking questions: contradictions between PRD and TDD, or inside TDD (decision table vs sections 3.3 to 3.8 vs criteria vs steps vs tests); anything still to guess (a payload, a code formula, a manifest entry, a CLI shape, an error code); a Check that would not catch its failure; a fact that does not match the installed tools; anything a reader without the debate could misread.
3. Give a final verdict line: `VERDICT: READY` if no material defect remains, or `VERDICT: NOT READY` followed by the blocking items.
New findings use new ids continuing your series, in the round-one shape with likelihood and severity. No style nits about wording; the documents follow a house style on purpose.

Budget: 500 words.

Your answer opens with a line containing only `ROUND 3 herdr-remote-r2` and closes with a line containing only `END ROUND 3`. Put nothing after the closing line.
