# Fix turn 1 verification (2026-11-09)

- Implementer herdr-remote-impl2 (Claude Fable 5.1), turn 19:47:05Z to 20:16:53Z (about 30 min). Report: review/fix1/report.md, all eight fixes done.
- Commits since the reviewed HEAD f858d27, one per fix:
  - 05423de fix 7 (key wait 120 s)
  - 62562af fix 6 (revoke mark, close, then write)
  - 05f8ce8 fix 5 (transcript fallback and paging)
  - b360936 fix 1 (no action retry after send; outcome unknown; 150 s)
  - c8682c0 fix 3 (hello refresh)
  - cb5e759 fix 2 (generic notification)
  - 0e367cb fix 8 (app lock)
  - 429c512 fix 4 (cancel the live Call)
- Scope: each commit touches only files of its fix (daemon/actions*, devices.go, daemon.go, api_test.go, transcript*; Android Api.kt, AgentViewModel.kt, AgentsRepository.kt, WatchService.kt, Gate.kt, MainActivity.kt, LiveView.kt, plus new tests). Nothing under .agents/, no attribution lines, no push.
- Orchestrator runs:
  - `go vet ./...` clean
  - `go test ./...` ok (84 s)
  - `./gradlew testDebugUnitTest assembleDebug` ok, 15 JVM tests in 5 suites, 0 failures, APK rebuilt.
- Code read: revoke() now marks, closes connections, ends live views, then drops and writes (daemon.go). keyWait is 120 s (actions.go).
- Leftovers: no hr-probe terminals, no emulator, no scratch daemon.
- The installed daemon (~/.local/bin/herdr-remoted) still runs the pre-fix build. The user reinstalls with scripts/install.sh.
