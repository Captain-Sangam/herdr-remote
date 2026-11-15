⏺ ROUND 2 ios-app-a2

  Position.
  CONSENSUS: I agree with the draft's decisions, with the fixes below. All iOS claims here are from memory.

  Reasoning.
  1. Store: conceded. One keychain item in the shared access group (decision 6) replaces my UserDefaults.
  2. QR scanner: conceded. DataScannerViewController.isSupported needs an A12 or later.
  3. Notification text: confirmed. msg stays out of the title and the body.
  4. Dynamic actions:
     - extensionContext.notificationActions can be set at run time (iOS 12 or later). I am not certain that it works when the category starts with none.
     - The extension gets the tap in didReceive(_:completionHandler:), and .doNotDismiss keeps the alert open for the result.
     - simctl push with mutable-content runs a service extension in recent Xcode versions. I am not certain of this.

  Outline. Plan check:
  - Step 1, ordering: the spike uses a software key. The largest risks are a Secure Enclave key forming an identity that URLSession presents, and a Personal Team signing the shared access group for an
    extension. Both are first tested in step 11, after ten steps are built. Add a device spike after step 3.
  - Step 5, contradicts decision 4: Android handshakes on a raw socket (net/Api.kt:261-285), and URLSession cannot handshake without a request. Take the server key from the trust challenge of the
    enroll request, or name one NWConnection.
  - Decision 6, omission: the store item needs AfterFirstUnlockThisDeviceOnly, or the service extension cannot read the pin on a locked device.
  - Decision 12, omission and contradiction:
    - The payload needs the category, or the content extension does not attach when the service extension fails.
    - It also needs the generic title, which "only the address and the kind" excludes.
  - Decision 13, omission: if the Mac has no APNs key, PUT /v1/push must fail, so that the app keeps its local alerts. Otherwise there are no alerts at all.
  - Steps 10 and 13, omission: no step adds the willPresent handler, so an alert while the app is open does not show. The step 10 check expects one.
  - Decision 10:
    - Add a fallback: buttons in the extension's own view (UNNotificationExtensionUserInteractionEnabled), if dynamic actions fail without static ones.
    - The probe can read as unlocked for about 10 s after a lock.
  - Step 7, weak check: it tests only actions. Criterion 5 (the folds, older pages, a cursor 409, the stop at a message start) has no check. Step 8 does not check resize or zoom (criterion 6).
  - Step 11: check the content extension over a LAN endpoint, for the local-network permission.
  - References: the ones I checked match: net/DeviceKey.kt:29, watch/WatchService.kt:214, :288, :305 and :339, and hr.written in assets/xterm/live.html:71.

  Assumptions.
  - An extension inherits the app's local-network permission.

  Confidence. 78

  END ROUND 2
