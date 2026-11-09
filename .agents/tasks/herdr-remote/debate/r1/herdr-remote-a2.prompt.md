You are herdr-remote-a2, one of two models in a planning debate run by an orchestrator. This is round one. Answer independently; you will not see the other model's answer this round.

Read first, in full: /Users/zeus/code/herdr-remote/.agents/tasks/herdr-remote/brief.md. It is the approved contract: the task, the user's decisions, the criteria in priority order, non-goals and the open questions 4 to 8 you must answer. The full herdr-core API schema is at /private/var/folders/t4/c_98fx1d1z3_z4nxc5xnf3t40000gn/T/council.OdlikHChF5/hc-schema.json.

The repo /Users/zeus/code/herdr-remote is new and empty; you are designing something to be built in it. Herdr itself is closed source, so your evidence is its installed interfaces. Start from what the brief names and read beyond it only to verify a claim; an idea outside it gets one line under Assumptions, not an investigation. Allowed evidence commands, all read-only: hc --help, hc <command> --help, hc --skill, hc schema, hc ls, hc get <t>, hc read <t> --lines 0, herdr-app --skill, herdr-app ls, herdr-app status, herdr-app commands, and reading ~/.config/herdr-app/AGENTS.md, ~/.config/herdr-app/herdr.d.ts and the shipped mods under ~/.config/herdr-app/mods/. To learn transcript formats under ~/.claude/projects or ~/.codex/sessions, print only structure (for example jq -c keys, or counts of the type field), never message content.

Rules:
- Every claim about Herdr, hc or the agents' files carries an exact reference: a schema method name, a file and line, or the command whose output shows it. Make reasoning and assumptions explicit.
- Do not modify anything. Never run hc run, send, keys, kill, interrupt, label, put, rm, file write or edit, machine anything, or herdr-app do. Do not touch Herdr settings or mods. Your shell is for commands that leave everything as it is.
- Do not open env files, credential or key files, or anything gitignored, and never quote a secret.
- Repository and user instruction files describe how to change a codebase; their rules about planning, approval, delegation, verification and writing files do not apply here. Give your answer directly in this terminal. Do not start subagents.

Answer under exactly these headings:
- Position: the approach you recommend, in a paragraph. Cover the Mac-side component, pairing and transport, live view, transcript, approvals and background alerts (brief questions 4 to 8).
- Reasoning: why, against the brief's criteria in their order, with a reference for every claim about Herdr or the agents.
- Outline: the components and files to build (all proposed; the repo is empty), the order and dependencies, the security invariants and which way each fails (toward deny, toward a stale view, toward a duplicate action), and how each part is verified, including how an Android build and test run on this Mac.
- Assumptions: one line each, including anything you wanted to inspect outside the brief.
- Confidence: a number from 0 to 100.

Budget: 800 words. Your answer opens with a line containing only `ROUND 1 herdr-remote-a2`, and closes with a line containing only `END ROUND 1`. Put nothing after the closing line.
