import XCTest

/// Step 5 of the plan: pair by a pasted link in the app, against the manual harness with
/// `HR_MANUAL_AUTOALLOW=1`. `TEST_RUNNER_HR_PAIR_LINK` is a fresh harness link.
final class PairUITests: XCTestCase {
  func testPairByAPastedLink() throws {
    guard let raw = ProcessInfo.processInfo.environment["HR_PAIR_LINK"] else {
      throw XCTSkip("set TEST_RUNNER_HR_PAIR_LINK to a fresh harness link")
    }
    // The harness listens on 127.0.0.1 only.
    let link = raw.replacingOccurrences(
      of: "ep=[^&]*", with: "ep=127.0.0.1", options: .regularExpression)
    let app = XCUIApplication()
    app.launchArguments = ["-reset"]
    app.launch()
    let field =
      app.textViews["pair-paste"].exists
      ? app.textViews["pair-paste"] : app.textFields["pair-paste"]
    XCTAssertTrue(field.waitForExistence(timeout: 10))
    field.tap()
    field.typeText(link)
    app.buttons["Pair with the pasted link"].tap()
    let check = app.buttons["Check the connection"]
    XCTAssertTrue(check.waitForExistence(timeout: 30), "paired")
    check.tap()
    let note = app.staticTexts.matching(
      NSPredicate(format: "label BEGINSWITH 'Connected to herdr-remoted'")
    ).firstMatch
    XCTAssertTrue(note.waitForExistence(timeout: 15))
  }

  /// Without auto-allow: the code shows while the Mac waits. `TEST_RUNNER_HR_UI_OUT` names a
  /// file on the Mac for the code, to compare with `herdr-remoted pair status`. Then
  /// `herdr-remoted pair cancel` closes the window, and the app gives the 410 text.
  func testTheCodeShowsUntilTheWindowCloses() throws {
    guard let raw = ProcessInfo.processInfo.environment["HR_PAIR_LINK_WAIT"] else {
      throw XCTSkip(
        "set TEST_RUNNER_HR_PAIR_LINK_WAIT to a fresh link of a harness without auto-allow")
    }
    let link = raw.replacingOccurrences(
      of: "ep=[^&]*", with: "ep=127.0.0.1", options: .regularExpression)
    let app = XCUIApplication()
    app.launchArguments = ["-reset"]
    app.launch()
    let field =
      app.textViews["pair-paste"].exists
      ? app.textViews["pair-paste"] : app.textFields["pair-paste"]
    XCTAssertTrue(field.waitForExistence(timeout: 10))
    field.tap()
    field.typeText(link)
    app.buttons["Pair with the pasted link"].tap()
    let code = app.descendants(matching: .any).matching(identifier: "pair-code").firstMatch
    XCTAssertTrue(code.waitForExistence(timeout: 15))
    if let out = ProcessInfo.processInfo.environment["HR_UI_OUT"] {
      try? code.label.write(toFile: out, atomically: true, encoding: .utf8)
    }
    XCTAssertTrue(
      app.staticTexts[
        "The pairing window closed before Allow. Click Pair a phone again on the Mac."
      ]
      .waitForExistence(timeout: 150))
  }

  func testAnExpiredLinkSaysSo() {
    let app = XCUIApplication()
    app.launchArguments = ["-reset"]
    app.launch()
    let field =
      app.textViews["pair-paste"].exists
      ? app.textViews["pair-paste"] : app.textFields["pair-paste"]
    XCTAssertTrue(field.waitForExistence(timeout: 10))
    field.tap()
    field.typeText("herdr-remote://pair?v=1&pin=x&ep=127.0.0.1&p=17766&ep_port=17767&t=t&exp=1000")
    app.buttons["Pair with the pasted link"].tap()
    XCTAssertTrue(
      app.staticTexts["This pairing code has expired. Click Pair a phone again on the Mac."]
        .waitForExistence(timeout: 5))
  }
}
