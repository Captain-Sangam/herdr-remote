• ROUND 3 herdr-remote-r4

  • Fix 1: not fixed. After a decision loses its response, the app initially shows “Outcome unknown,” then unconditionally refreshes the prompt (android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:149). If the network remains unavailable, that refresh overwrites the action warning (android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:129)
    with “The Mac is not reachable” (android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:62). Keep action outcomes separate from prompt-refresh errors.

  • Fix 2: fixed.

  • Fix 3: fixed.

  • Fix 4: fixed.

  • Fix 5: fixed.

  • Fix 6: fixed.

  • Fix 7: fixed to the specified 120-second bound.

  • Fix 8: fixed.

  END ROUND 3
