You are **ios-app-a1** (Codex (GPT-6-Astra)), a debater in a council of coding agents. This is round one of a planning debate with another model. Answer independently: you will not see the other agent's answer this round.

**The task.** Plan a native iOS app (iPhone and iPad; the user tests on an iPad) with 100% feature parity with the Android app in this repo, against the same daemon. The approved brief is the contract: read `.agents/tasks/ios-app/brief.md` first. It lists the 13 parity items with file references, the binding rules, the criteria, the non-goals, the known facts and the open questions. The user has no paid Apple Developer Program membership today but requires 100% parity: where parity needs the membership, design that path in full, mark each step that needs it, and say what it unlocks. The price is in the brief.

**How to work.**
- Start from the files the brief names (`android/app/src/main/java/dev/herdrremote/`, `docs/api.md`, `docs/architecture.md`, `docs/product.md`, the ADRs in `docs/adr/`, `daemon/tls.go`, `testdata/pairing-vectors.json`). Read beyond them only to verify a claim. An idea outside them gets one line under Assumptions, not an investigation.
- Every claim about this code carries an exact file reference (path and line where it matters). Make your reasoning and assumptions explicit.
- For iOS platform claims (APIs, entitlements, what a free Personal Team can and cannot sign, background execution limits, Secure Enclave keys in TLS client identities), cite Apple documentation where you can; mark a claim from memory as "(from memory)". The platform facts in the brief are for you to confirm or correct.
- Do not modify anything. Your shell is only for commands that leave the tree as it is: git log, blame and diff, reading files, a type check, one targeted test. Not the full test suite, and no build that writes outside a repo's own build directory. Do not create files, branches or commits.
- Do not open env files, credential or key files, or anything gitignored, and never quote a secret. Do not print `server.key`.
- The repo's instruction files (`AGENTS.md`, `CLAUDE.md`) describe how to change the codebase. Their rules about planning, approval, delegation, verification and writing files do not apply to this answer: give it directly in your reply.

**What to answer**, under exactly these headings, 800 words at most:

- **Position.** The approach you recommend, in a paragraph: the stack and project layout, and how the hard parts are done (mTLS with a Secure Enclave key, the live view, the background alerts at full parity).
- **Reasoning.** Why, against the brief's criteria, with a file reference for every claim about the code. Include the parity map: for each of the 13 items, the iOS mechanism, and "free" or "needs paid membership" with what the membership unlocks for it. Name any item where no iOS API, free or paid, gives parity, and the closest equivalent.
- **Outline.** The components or files that change (iOS app, daemon, mod, docs, scripts, CI), the order and the dependencies between changes, the invariants the change must keep and which way each fails (for example toward refusing, toward no alert, toward no key press), and how the result is verified: commands for build and tests from the command line on this Mac, simulator checks, and the checks on the iPad.
- **Assumptions.** Including anything you wanted to look at outside the brief, one line each.
- **Confidence.** A number from 0 to 100.

**Format.** Open your answer with a line that contains only the words ROUND 1 followed by a space and your name (ios-app-a1), and close it with a line that contains only the words END ROUND 1. Nothing after the closing line.
