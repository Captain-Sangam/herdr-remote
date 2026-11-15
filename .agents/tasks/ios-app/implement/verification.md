outcome: blocked at step 6 (planned pause for the user's iPad check; steps 1 to 5 built and verified)

# Verification of the implement turn (2026-11-15)

Checked by the orchestrator, not taken from the report.

## Git

- `git status --porcelain`: only `.agents/tasks/ios-app/` files, as `before.md` listed. All step work is committed.
- `git log 866f953..HEAD`: seven commits: steps 1 to 6 (`61522c2`, `558a549`, `9fb42ec`, `553231c`, `4b100f8`, `46bb6dc`) and the As built note (`fbfd324`). Conventional messages, no attribution lines.
- Files outside `ios/`: `.github/workflows/ci.yaml`, `.gitignore`, `scripts/format.sh`, ADR 0025, `docs/README.md`, `docs/architecture.md`, `docs/design.md`, `docs/product.md`, `docs/testing.md`, and `plan.md` in the task folder. All inside the steps' scope. No Android or daemon code changed.

## Commands run by the orchestrator

| Command | Result |
| --- | --- |
| `swift test --package-path ios/Core` | Pass, 28 tests |
| `scripts/format.sh --check` | Pass |
| `cd daemon && go test ./...` | Pass (84.6 s) |
| Android `testDebugUnitTest assembleDebug` | Pass |
| `xcodebuild build` (iPhone 17 simulator, `CODE_SIGNING_ALLOWED=NO`) | Pass |
| `xcodebuild test` (iPad mini simulator), no harness | Pass, harness tests skipped |
| `xcodebuild test-without-building` (iPhone 17) with the scratch harness and one fresh link | 13 unit tests pass (spike, API, pairing). `PairUITests.testPairByAPastedLink` failed. |
| The same UI test alone with its own fresh link | Pass |

## Discrepancies

- **Test isolation:** `PairingTests` and `PairUITests` both read `TEST_RUNNER_HR_PAIR_LINK`. A link pairs one key only, so the second test fails when both run in one pass. `docs/testing.md` ("iOS tests") names one link. This is a test-setup fault, not an app fault. The implementer fixes it in the next turn (separate links or a note).
- **ATS exception:** `ios/App/Info.plist` sets `NSExceptionAllowsInsecureHTTPLoads` for `ts.net`. The plan's step 1 invariant ("no code builds an `http` URL") holds: `ios/Shared/Api.swift:30` builds only `https`, and the redirect tests pass.
- **CI:** the format job now runs on `macos-26` with Xcode 26.6; the Mac has Xcode 27. Not run in CI from here (no push). Risk noted in As built.
- **Step 6:** not run. No iPad was connected and Xcode had no account.

## As built note

Present in `plan.md`, covers steps 1 to 6, the assumptions and the deviations in the report, and the user's step 6 checks.
