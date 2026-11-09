• ROUND 3 herdr-remote-r6

  1. Fixed. The process observer is registered in android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:21, survives activity destruction, and the activity reads the resulting gate state in android/app/src/main/java/dev/herdrremote/MainActivity.kt:38.

  2. Not fixed completely. android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:64 still renders a.msg without a height or line limit, before the bounded block and decision buttons at :72. A command message that wraps beyond the available card height can consume that height and leave the buttons inaccessible. The surrounding Column at :51 cannot scroll.
     Messages pass through without truncation at daemon/state.go:343. The emulator result establishes that its particular message fits; it does not establish a bound. Bound or scroll the title too, while reserving space for the buttons.

  3. Fixed. The post-decision refresh uses manual = false at android/app/src/main/java/dev/herdrremote/ui/AgentViewModel.kt:152; refresh failures preserve the existing action note through :70 and :132.

  4. Fixed. android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:52 cancels the SSE Call, and :81 prevents endpoint fallback after cancellation. android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:32 stops watching before clearing pairing. android/app/src/main/java/dev/herdrremote/watch/WatchService.kt:73 waits for collectors before
     removing the ongoing notification.

  END ROUND 3
