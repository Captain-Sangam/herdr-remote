import CryptoKit
import Foundation
import LocalAuthentication
import Testing

@testable import Core

// GateTest.kt, PinTest.kt (the gate part), SentMarkTest.kt (as a rule), and the certificate and
// the push payload.

@Test func lockedAtStartAndAfterFiveMinutesInTheBackground() {
  var g = Gate()
  #expect(g.locked)
  g.unlock()
  #expect(!g.locked)
  g.onBackground(nowMs: 1_000)
  g.onForeground(nowMs: 1_000 + Gate.lockAfterMs - 1)
  #expect(!g.locked, "under the limit")
  g.onBackground(nowMs: 5_000)
  g.onForeground(nowMs: 5_000 + Gate.lockAfterMs)
  #expect(g.locked, "at the limit")
  #expect(Gate.shouldLock(locked: true, backgroundedAt: nil, nowMs: 0))
}

@Test func aShortBackgroundKeepsTheGateOpen() {
  var g = Gate()
  g.unlock()
  g.onBackground(nowMs: 1_000)
  g.onForeground(nowMs: 2_000)
  #expect(!g.locked)
  // A foreground with no background before it (a second scene) keeps it open too.
  g.onForeground(nowMs: 10 * Gate.lockAfterMs)
  #expect(!g.locked)
}

@Test func onlyADeviceWithNoPasscodeOpensWithoutAPrompt() {
  #expect(Gate.check(nil) == .prompt)
  #expect(Gate.check(.passcodeNotSet) == .open)
  for code: LAError.Code in [
    .biometryNotAvailable, .biometryLockout, .biometryNotEnrolled, .notInteractive,
    .systemCancel, .authenticationFailed,
  ] {
    #expect(Gate.check(code) == .error, "\(code)")
  }
}

@Test func anActionTriesTheNextEndpointOnlyBeforeTheRequestStarted() {
  // A refused connection, or no route: never sent.
  for code: URLError.Code in [
    .cannotConnectToHost, .cannotFindHost, .timedOut, .notConnectedToInternet, .cancelled,
  ] {
    #expect(ActionRule.mayTryNext(code, requestStarted: false), "\(code)")
    #expect(!ActionRule.mayTryNext(code, requestStarted: true), "\(code)")
  }
  // The request may have arrived: no retry of any kind.
  #expect(!ActionRule.mayTryNext(.networkConnectionLost, requestStarted: false))
  #expect(!ActionRule.mayTryNext(.badServerResponse, requestStarted: false))
  #expect(!ActionRule.mayTryNext(nil, requestStarted: false))
}

@Test func theCertificateCarriesTheKeyAndAValidSignature() throws {
  let key = P256.Signing.PrivateKey()
  var tbs = Data()
  var sig = Data()
  let der = try Certificate.selfSigned(
    point: key.publicKey.x963Representation, serial: Data([0x80, 1, 2, 3]),
    notBefore: Date(timeIntervalSince1970: 1_700_000_000),
    notAfter: Date(timeIntervalSince1970: 2_600_000_000)
  ) {
    tbs = $0
    sig = try key.signature(for: $0).derRepresentation
    return sig
  }
  // The DER parses, and its key is ours.
  let cert = try #require(SecCertificateCreateWithData(nil, der as CFData))
  let pub = try #require(SecCertificateCopyKey(cert))
  #expect(SecKeyCopyExternalRepresentation(pub, nil) as Data? == key.publicKey.x963Representation)
  #expect(SecCertificateCopySubjectSummary(cert) as String? == "herdr-remote")
  // The certificate holds the to-be-signed part and ends with its signature, which checks.
  #expect(der.range(of: tbs) != nil)
  #expect(der.suffix(sig.count) == sig)
  #expect(
    SecKeyVerifySignature(pub, .ecdsaSignatureMessageX962SHA256, tbs as CFData, sig as CFData, nil))
  // 2052 needs GeneralizedTime.
  #expect(Certificate.time(Date(timeIntervalSince1970: 2_600_000_000)).first == 0x18)
  #expect(Certificate.time(Date(timeIntervalSince1970: 1_700_000_000)).first == 0x17)
  #expect(Certificate.int(Data([0x80])) == Data([0x02, 0x02, 0x00, 0x80]))
  #expect(Certificate.int(Data([0, 0, 5])) == Data([0x02, 0x01, 0x05]))
  #expect(Certificate.tlv(0x04, Data(count: 200)).prefix(3) == Data([0x04, 0x81, 200]))
}

@Test func aPushNamesTheAgentOnlyAfterTheFetch() throws {
  let p = try #require(
    PushPayload([
      "aps": ["alert": ["title": AlertText.genericBlocked], "mutable-content": 1],
      "address": "t1", "kind": "blocked",
    ]))
  #expect(p == PushPayload(["address": "t1", "kind": "blocked"]))
  var a = Agent(
    address: "t1", where: "Niti / main", state: "blocked", kind: "permission", msg: "approve x")
  #expect(p.title(a) == "Niti / main needs approval")
  a.kind = "question"
  #expect(p.title(a) == "Niti / main needs attention")
  a.state = "done"
  #expect(
    try #require(PushPayload(["address": "t1", "kind": "done"])).title(a) == "Niti / main finished")
  a.state = "error"
  #expect(
    try #require(PushPayload(["address": "t1", "kind": "done"])).title(a)
      == "Niti / main stopped with an error")
  #expect(
    try #require(PushPayload(["address": "t1", "kind": "clear"])).title(a)
      == "Niti / main no longer waits")
  #expect(PushPayload(["address": "t1", "kind": "other"]) == nil)
  #expect(PushPayload(["kind": "blocked"]) == nil)
  #expect(PushPayload(["address": "", "kind": "blocked"]) == nil)
}
