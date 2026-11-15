⏺ IMPLEMENT ios-app-impl implement

  I built steps 1 to 5. Step 6 is blocked: its stub is built, but this Mac has no iPad connected, no signing identity and no Xcode account, so nothing could be installed. The As built note in plan.md
  lists the 10 checks the user must do at the iPad.

  Steps
  - Step 1, done (61522c2): The project skeleton, Swift in scripts/format.sh, the CI format job on macos-26, and ADR 0025. The spike passed against the manual harness. Over an IP address and over a
    DNS name, the app presents a keychain identity and pins the Mac's key. A wrong pin, a redirect and TLS 1.2 all fail. Decision 4 holds, so there is no fallback to Network.framework.
    - iOS's built-in HTTPS rules (ATS) do not check IP addresses, but they reject the Mac's self-signed key for any DNS name, even after the app accepts it. One exception for ts.net in Info.plist
      fixes this, and a wrong pin still fails over the name.
    - Tailscale was off, so no real MagicDNS name was tested. The stand-in name localtest.me (it resolves to 127.0.0.1) passed.
  - Step 2, done (558a549): The pure logic in ios/Core. swift test passes 28 tests, with a case for each Android test case. Go parses the certificate the app builds and accepts its signature.
  - Step 3, done (9fb42ec): The device key, the keychain store and the API client. Seven tests pass against a fake TLS server and the harness:
    - A refused first endpoint moves to the next one.
    - A connection that closes after the request gives "Outcome unknown", and the server sees one request.
    - A redirect is refused.
    - An endpoint that never connects is dropped after 10 s.
    - The grep check shows the software key only inside the simulator branch.
  - Step 4, done (553231c): The Pane design system. I checked a preview screen by eye in dark and light mode and fixed two layout faults.
  - Step 5, done (4b100f8): Pairing by QR code or pasted link, against the harness:
    - A wrong pin shows no code and sends no token, and then the same link pairs.
    - The app showed 119 639 and the harness showed 119639.
    - The expired-link, 410 and wrong-pin texts are the Android ones.
  - Step 6, blocked (46bb6dc): The content extension stub builds and is embedded in the app. A signed device build fails with "requires a development team".

  Deviations
  - The plan's test command uses CODE_SIGNING_ALLOWED=NO. With it, the keychain fails (-34018), so the tests use the default ad-hoc simulator signing. The build check keeps the flag.
  - The MagicDNS check moves to the iPad.

  Assumptions
  - Until the app could pair, the tests used the harness's own phone key as the test identity. Nothing printed it.
  - URLSession has no connect timeout. A small per-request check cancels a request that has sent and received nothing after 10 s.
  - Each action uses its own session, so no stale connection can send it twice.
  - The iOS docs are a new section 9 in docs/architecture.md, so no existing links change.
  - The message box is not in step 4. Only the agent screen in step 8 uses it.
  - Until step 7, a paired app shows a stand-in screen with "Check the connection", a test-alert key and "Forget this Mac". The step 6 checks use it.
  - A new build setting, HR_BUNDLE_ID, sets the bundle ID. It is only needed if a free team cannot register dev.herdrremote.
  - To check that the code shows while the Mac waits, I ran the harness without auto-allow and closed the window with pair cancel. No pair allow was run.
  - CI uses Xcode 26.6 and this Mac uses Xcode 27. Their swift-format versions can format Swift differently.

  Left: the step 6 checks by the user, then steps 7 to 14 and 16. Step 15 waits for the membership.

  Files
  - Step 1: ios/project.yml, ios/App/ (start), ios/Shared/Transport.swift, ios/Core/, ios/Tests/, scripts/format.sh, .github/workflows/ci.yaml, .gitignore, ADR 0025, docs/README.md, docs/product.md,
    docs/testing.md.
  - Step 3: ios/Shared/Device.swift, ios/Shared/Api.swift, ios/App/HerdrRemote.entitlements.
  - Step 4: ios/App/Pane*.swift.
  - Step 5: ios/App/PairScreen.swift, ios/App/AppModel.swift, ios/UITests/.
  - Step 6: ios/NotificationContent/.
  - Docs: docs/architecture.md, docs/design.md, docs/testing.md.

  The final commit is fbfd324, the As built note.

  STATUS: blocked at step 6

  END IMPLEMENT implement
