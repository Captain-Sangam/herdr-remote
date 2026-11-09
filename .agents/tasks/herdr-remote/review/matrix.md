# Completeness matrix (orchestrator, not shown to reviewers in round 1)

Basis: commits, the As built note, the orchestrator's test runs, and a reading of tls.go and the decision handler in actions.go. The other rows were checked by grepping for each named element (VerifyConnection, SessionTicketsDisabled, TLS13, RequireAnyClientCert, HIDIdleTime, expired_request, live_limit, agent_removed, Sync, parent_thread_id, record too large, herdr-remote-owned, server_time_ms, ": ka", FLAG_SECURE, VISIBILITY_PRIVATE, specialUse, BOOT_COMPLETED, MY_PACKAGE_REPLACED, WebViewAssetLoader, retryOnConnectionFailure, TLS_1_3, BiometricPrompt, setUserAuthenticationRequired, UUIDv7). Not a line-by-line review.

| Row | Status | Note |
|---|---|---|
| S1 | delivered | 0fd87fb; SDK at the cask root, not ~/Library/Android/sdk (routine) |
| S2 | delivered | 67d240b; tests for allow-list; bare env for subprocesses |
| S3 | delivered | e85a376; tls.go mutualTLS: TLS13, RequireAnyClientCert, tickets off, VerifyConnection with per-listener verify func |
| S4 | delivered | 33e1767 |
| S5 | delivered | ab9aeaa; adds 409 live_limit (routine) |
| S6 | delivered | bfaac7b |
| S7 | delivered | 4b261f0; guard order matches 3.5 exactly; refusals not remembered (As built) |
| S8 | drifted (minor) | 38f9802; extra nextCommand handlers .pair/.allow/.deny/.cancel; pair --json returns text |
| S9 | delivered, commit not buildable alone | 0d14b24 |
| S10 | delivered | c57bc5b |
| S11 | delivered | cd696d6, 649c88f |
| S12 | partial | manual checks needing a phone, Tailscale, GUI left to the user |
| C1 | delivered | verified live: *:7766 only, 7767 closed, openssl refusal in As built |
| C2 | partial | Allow with a real phone not done (implementer chose Deny on the real daemon); Go tests cover Allow |
| C3 | delivered (emulator) | |
| C4 | delivered | probes in As built |
| C5 | partial | resize and heavy output not run |
| C6 | delivered | probes through the API and the emulator |
| C7 | delivered | |
| C8 | delivered (emulator) | swipe-away and notifications-denied left to the user |
| C9 | delivered (tests) | |
| C10 | not verified | no Tailscale |
| C11 | delivered (emulator, cold start with PIN) | 5-minute background timeout path: tested by JVM gate test per As built |
| C12 | drifted | lock screen shows tab name + "permission" under Android default; recents not checked (FLAG_SECURE present) |
| C13 | delivered | live audit.log has no secrets |

Changes no step explains: paste-link field in the app; mod nextCommand handlers; manual_test.go harness (test-only, skipped without HR_MANUAL_DIR).
