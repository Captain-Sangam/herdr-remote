import AVFoundation
import Core
import SwiftUI

/// Pairing (docs/architecture.md section 3.1): scan or paste the code, show the 6 digits, wait
/// for Allow on the Mac. The words are those of `ui/PairScreen.kt`.
struct PairScreen: View {
  let model: AppModel

  enum PairState: Equatable {
    case idle, connecting
    case code(String)
    case failed(String)
  }

  @State private var state = PairState.idle
  @State private var pasted = ""
  @State private var name = Pairing.defaultName(model: UIDevice.current.model, machine: machine)
  @State private var scanning = false
  @Environment(\.pane) private var c

  /// The hardware identifier, "iPad13,18". The simulator reports the device it simulates.
  static var machine: String {
    if let sim = ProcessInfo.processInfo.environment["SIMULATOR_MODEL_IDENTIFIER"] { return sim }
    var u = utsname()
    uname(&u)
    return withUnsafeBytes(of: &u.machine) {
      String(decoding: $0.prefix { $0 != 0 }, as: UTF8.self)
    }
  }

  func start(_ text: String) {
    let link: PairingLink
    do {
      link = try Pairing.parse(text)
    } catch {
      state = .failed(Pairing.failure(error))
      return
    }
    let name = name.trimmingCharacters(in: .whitespaces)
    guard Pairing.validName(name) else {
      state = .failed("Give this phone a name of 1 to 64 characters")
      return
    }
    state = .connecting
    Task {
      do {
        let identity = try DeviceKey.ensure()
        let deviceId = try await Enroll.run(link: link, name: name, identity: identity) { code in
          Task { @MainActor in state = .code(code) }
        }
        model.store.update {
          $0.pin = link.pin
          $0.endpoints = link.endpoints
          $0.lastGood = nil
          $0.port = link.port
          $0.deviceId = deviceId
          $0.macName = link.endpoints.first ?? ""
        }
        model.didPair()
      } catch {
        state = .failed(Pairing.failure(error))
      }
    }
  }

  func scan() {
    switch AVCaptureDevice.authorizationStatus(for: .video) {
    case .authorized: scanning = true
    case .notDetermined:
      Task {
        if await AVCaptureDevice.requestAccess(for: .video) {
          scanning = true
        } else {
          state = .failed(Self.cameraOff)
        }
      }
    default: state = .failed(Self.cameraOff)
    }
  }

  static let cameraOff =
    "Scanner unavailable: camera access is off. Allow it in Settings, or paste the link."

  var body: some View {
    ScrollView {
      VStack(alignment: .leading, spacing: 14) {
        PairArt()
        switch state {
        case .code(let code): codeView(code)
        case .connecting:
          Title(text: "Pair with your Mac")
          Waiting(text: "Connecting to the Mac…")
        case .idle, .failed: startView
        }
      }
      .padding(.horizontal, PaneSpace.xl)
      .padding(.vertical, 18)
      .frame(maxWidth: 560, alignment: .leading)
      .frame(maxWidth: .infinity)
    }
    .fullScreenCover(isPresented: $scanning) {
      ScannerSheet(
        onCode: {
          scanning = false
          start($0)
        },
        onFail: {
          scanning = false
          state = .failed("Scanner unavailable: \($0)")
        },
        onCancel: { scanning = false }
      )
      .modifier(PaneTheme())
    }
  }

  @ViewBuilder private func codeView(_ code: String) -> some View {
    Title(text: "Check the code")
    Text("The Mac shows a 6-digit code too.").pane(PaneType.body).foregroundStyle(c.muted)
    HStack(spacing: 14) {
      ForEach([String(code.prefix(3)), String(code.suffix(3))], id: \.self) { group in
        Text(group).pane(PaneType.code).foregroundStyle(c.done)
          .padding(.horizontal, 10).padding(.vertical, 4)
          .overlay(Rectangle().stroke(c.line, lineWidth: PaneSpace.line))
      }
    }
    .accessibilityElement(children: .combine)
    .accessibilityIdentifier("pair-code")
    Marked(text: "If the Mac shows the same code, click **Allow** there. Otherwise click **Deny**.")
    Waiting(text: "Waiting for Allow on the Mac")
    Text("This phone: \(name.trimmingCharacters(in: .whitespaces))").pane(PaneType.caption)
      .foregroundStyle(c.muted)
  }

  @ViewBuilder private var startView: some View {
    Title(text: "Pair with your Mac")
    if case .failed(let message) = state {
      PaneNote(text: message, kind: .error).accessibilityIdentifier("pair-error")
    } else {
      VStack(alignment: .leading, spacing: 10) {
        ForEach(
          Array(
            [
              "On the Mac, open Herdr Remote and click **Pair a phone**.",
              "Scan the QR code that the Mac shows.",
              "If both codes match, click **Allow** on the Mac.",
            ].enumerated()), id: \.offset
        ) { i, step in
          HStack(alignment: .firstTextBaseline, spacing: 12) {
            Text("\(i + 1).").pane(PaneType.body).foregroundStyle(c.accent)
            Marked(text: step, color: c.text)
          }
        }
      }
    }
    PaneField(label: "This phone's name", text: $name)
    KeyButton(
      state == .idle ? "Scan the QR code" : "Scan the QR code again", kind: .go, action: scan)
    HStack(spacing: 10) {
      Rule(color: c.line, dashed: true)
      Text("or").pane(PaneType.caption).foregroundStyle(c.muted)
      Rule(color: c.line, dashed: true)
    }
    PaneField(
      label: "Or paste the pairing link", text: $pasted, placeholder: "herdr-remote://pair?t=…",
      singleLine: false
    )
    .accessibilityIdentifier("pair-paste")
    KeyButton(
      "Pair with the pasted link",
      enabled: !pasted.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    ) { start(pasted) }
  }
}

/// "# Pair with your Mac": a display title with an accent "# ".
private struct Title: View {
  let text: String
  @Environment(\.pane) private var c

  var body: some View {
    (Text("# ").foregroundColor(c.accent) + Text(text).foregroundColor(c.text))
      .pane(PaneType.display)
      .accessibilityAddTraits(.isHeader)
  }
}

/// "**Allow**" in bold text colour, the rest as it is.
private struct Marked: View {
  let text: String
  var color: Color?
  @Environment(\.pane) private var c

  var body: some View {
    text.components(separatedBy: "**").enumerated().reduce(Text("")) { out, part in
      out
        + (part.offset % 2 == 0
          ? Text(part.element).foregroundColor(color ?? c.muted)
          : Text(part.element).font(PaneType.body.with(bold: true).font).foregroundColor(c.text))
    }
    .pane(PaneType.body)
  }
}

/// A braille spinner and what the app waits for.
private struct Waiting: View {
  let text: String
  @Environment(\.pane) private var c

  var body: some View {
    HStack(spacing: 10) {
      Spinner()
      Text(text).pane(PaneType.label).foregroundStyle(c.muted)
    }
  }
}

/// The Mac shows a QR code and the phone scans it (PAIR_ART in design/mockups.js).
private struct PairArt: View {
  @Environment(\.pane) private var c

  var body: some View {
    Canvas { ctx, _ in
      let thin = StrokeStyle(lineWidth: 1.6)
      let bold = StrokeStyle(lineWidth: 2.5)
      func line(_ a: CGPoint, _ b: CGPoint) -> Path {
        var p = Path()
        p.move(to: a)
        p.addLine(to: b)
        return p
      }
      // The Mac and its QR code.
      ctx.stroke(
        Path(CGRect(x: 20, y: 12, width: 96, height: 62)), with: .color(c.text), style: thin)
      ctx.stroke(
        line(CGPoint(x: 8, y: 82), CGPoint(x: 128, y: 82)), with: .color(c.text), style: thin)
      for (x, y) in [(50, 25), (76, 25), (50, 51), (76, 51)] {
        ctx.fill(Path(CGRect(x: x, y: y, width: 10, height: 10)), with: .color(c.text))
      }
      ctx.fill(Path(CGRect(x: 65, y: 40, width: 6, height: 6)), with: .color(c.text))
      // The scan, and the phone with its viewfinder.
      ctx.stroke(
        line(CGPoint(x: 128, y: 44), CGPoint(x: 160, y: 44)), with: .color(c.accent),
        style: StrokeStyle(lineWidth: 2.5, dash: [4, 5]))
      ctx.stroke(
        Path(CGRect(x: 166, y: 8, width: 44, height: 80)), with: .color(c.text), style: thin)
      ctx.stroke(
        line(CGPoint(x: 181, y: 16), CGPoint(x: 195, y: 16)), with: .color(c.text), style: thin)
      ctx.stroke(
        Path(CGRect(x: 176, y: 36, width: 24, height: 24)), with: .color(c.accent), style: bold)
    }
    .frame(width: 220, height: 96)
    .accessibilityHidden(true)
  }
}

/// The camera and a Cancel key. It reads one QR code.
private struct ScannerSheet: View {
  let onCode: (String) -> Void
  let onFail: (String) -> Void
  let onCancel: () -> Void

  var body: some View {
    VStack(spacing: 0) {
      PaneHeader("~/pair/scan", onBack: onCancel)
      QRScanner(onCode: onCode, onFail: onFail)
      KeyButton("Cancel", kind: .quiet, action: onCancel).padding(PaneSpace.s)
    }
  }
}

/// AVCaptureMetadataOutput with QR codes only (decision 8): it works on every iPad with a
/// camera.
private struct QRScanner: UIViewControllerRepresentable {
  let onCode: (String) -> Void
  let onFail: (String) -> Void

  func makeUIViewController(context: Context) -> ScannerController {
    ScannerController(onCode: onCode, onFail: onFail)
  }

  func updateUIViewController(_ vc: ScannerController, context: Context) {}
}

final class ScannerController: UIViewController, AVCaptureMetadataOutputObjectsDelegate {
  private let session = AVCaptureSession()
  private let onCode: (String) -> Void
  private let onFail: (String) -> Void
  private var preview: AVCaptureVideoPreviewLayer?
  private var done = false

  init(onCode: @escaping (String) -> Void, onFail: @escaping (String) -> Void) {
    self.onCode = onCode
    self.onFail = onFail
    super.init(nibName: nil, bundle: nil)
  }

  required init?(coder: NSCoder) { nil }

  override func viewDidLoad() {
    super.viewDidLoad()
    view.backgroundColor = .black
    let output = AVCaptureMetadataOutput()
    guard let device = AVCaptureDevice.default(for: .video),
      let input = try? AVCaptureDeviceInput(device: device), session.canAddInput(input),
      session.canAddOutput(output)
    else {
      DispatchQueue.main.async { self.onFail("no camera") }
      return
    }
    session.addInput(input)
    session.addOutput(output)
    output.setMetadataObjectsDelegate(self, queue: .main)
    output.metadataObjectTypes = [.qr]
    let layer = AVCaptureVideoPreviewLayer(session: session)
    layer.videoGravity = .resizeAspectFill
    view.layer.addSublayer(layer)
    preview = layer
    nonisolated(unsafe) let session = session
    DispatchQueue.global(qos: .userInitiated).async { session.startRunning() }
  }

  override func viewDidLayoutSubviews() {
    super.viewDidLayoutSubviews()
    preview?.frame = view.bounds
  }

  override func viewWillDisappear(_ animated: Bool) {
    super.viewWillDisappear(animated)
    nonisolated(unsafe) let session = session
    DispatchQueue.global(qos: .userInitiated).async { session.stopRunning() }
  }

  nonisolated func metadataOutput(
    _ output: AVCaptureMetadataOutput, didOutput objects: [AVMetadataObject],
    from connection: AVCaptureConnection
  ) {
    let value = objects.compactMap { ($0 as? AVMetadataMachineReadableCodeObject)?.stringValue }
      .first
    MainActor.assumeIsolated {
      guard let value, !done else { return }
      done = true
      onCode(value)
    }
  }
}
