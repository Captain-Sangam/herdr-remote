• ROUND 1 herdr-remote-r6

  1. herdr-remote-r6-1 — Activity destruction can bypass the authentication gate.
     Path: android/app/src/main/java/dev/herdrremote/MainActivity.kt:86; android/app/src/main/java/dev/herdrremote/Gate.kt:25.
     Evidence: With watching running and Android’s “Don’t keep activities” enabled, unlock and press Home. Activity destruction removes the observer before the delayed process ON_STOP. The only background timestamp setter is in MainActivity.kt:29, so it never runs. Reopening after five minutes leaves Gate.locked false and displays content without
     authentication.
     Likelihood: 3/10 — requires activity destruction before the process callback while the service keeps the process alive.
     Severity: 8/10 — exposes transcripts and remote controls without the required authentication.
     Label: implementation defect. Disposition: new.
     Fix: Register the process lifecycle observer once in Application, independently of activity lifetime.

  2. herdr-remote-r6-2 — Forgetting the Mac leaves its event stream and watcher running.
     Path: android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:43; android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:28.
     Evidence: While connected with watching enabled, select “Forget this Mac”. stop() only cancels the coroutine; the synchronous read at AgentsRepository.kt:82 neither checks cancellation nor closes the connection. Keepalives prevent its read timeout (daemon/api.go:118). Subsequent events repopulate cleared state at AgentsRepository.kt:87, and the unstopped
     service continues notifying at android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:56. Pairing another Mac can consequently mix the previous Mac’s terminal records into the new connection’s state.
     Likelihood: 8/10 — occurs when forgetting a connected Mac.
     Severity: 6/10 — retains unwanted access and can display agents from the wrong Mac.
     Label: implementation defect. Disposition: new.
     Fix: Cancel and close the active SSE call, await reader termination, and stop the watch service before clearing pairing state.

  3. herdr-remote-r6-3 — Long permission dialogs make approval controls inaccessible.
     Path: android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:64.
     Evidence: Supply an actionable Codex dialog containing a 45-line command on a normal portrait phone. The prompt Text has horizontal scrolling only and consumes the Card’s available height; the Approve/Deny row follows it at AgentScreen.kt:67. Neither surrounding Column scrolls (AgentScreen.kt:48, AgentScreen.kt:60), leaving the buttons clipped or with no
     remaining height. The user cannot answer this valid permission prompt.
     Likelihood: 6/10 — multiline commands, shorter screens, or larger accessibility fonts trigger it.
     Severity: 6/10 — blocks the app’s primary remote approval workflow.
     Label: implementation defect. Disposition: new.
     Fix: Bound the prompt’s height and make its contents vertically scrollable while keeping decision buttons visible.

  Checklist rows not fully delivered in code: S9/C11 (finding 1), S10/C3 (finding 2), S11/C6 (finding 3).
  Bot comments: none
  END ROUND 1
