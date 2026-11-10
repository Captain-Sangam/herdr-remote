• ROUND 2 herdr-remote-r6

  • G1: Amend the trigger wording; accept L6/S7 and the fix.
    android/app/src/main/java/dev/herdrremote/MainActivity.kt:86 removes the observer responsible for recording background time at MainActivity.kt:29. The swipe trigger holds when activity destruction precedes the delayed process ON_STOP and watching keeps the process alive. It does not hold for every swipe: if ON_STOP already ran while recents was open, the
    timestamp exists. This ordinary interaction raises my likelihood from L3 to L6 without requiring a developer setting. Also, “until the process dies” is too strong: a subsequent normal background cycle can record the missing timestamp (android/app/src/main/java/dev/herdrremote/Gate.kt:18). Registering the observer once in HerdrRemoteApp.onCreate addresses
    the ownership problem.

  • G2: Accept the defect, proposed fix, and lower L4/S5.
    android/app/src/main/java/dev/herdrremote/watch/AgentsRepository.kt:44 cancels the job without cancelling the blocking read at AgentsRepository.kt:82; android/app/src/main/java/dev/herdrremote/HerdrRemoteApp.kt:28 does not stop WatchService. I accept L4 because the trigger requires forgetting an actively connected Mac; contamination of another pairing
    additionally requires re-pairing before that connection ends. My original cost assessment overstated the action risk: fresh prompt validation remains enforced at daemon/actions.go:379. Keep this as a nonblocking follow-up outside the severity-6 findings.

  • G3: Accept L5/S6 and the fix.
    android/app/src/main/java/dev/herdrremote/ui/AgentScreen.kt:64 permits horizontal scrolling only, with decision buttons following at AgentScreen.kt:67. A tall command block can exhaust the available height and make those controls inaccessible. Bounding the block and adding vertical scrolling preserves access to the primary approval workflow.

  END ROUND 2
