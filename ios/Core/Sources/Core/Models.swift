import Foundation

// The API shapes (docs/api.md). A missing field takes its default, as in the Android app.

extension KeyedDecodingContainer {
  fileprivate func get<T: Decodable>(_ key: Key, _ fallback: T) throws -> T {
    try decodeIfPresent(T.self, forKey: key) ?? fallback
  }
}

public struct Agent: Codable, Equatable, Sendable {
  public var address: String
  public var space = ""
  public var tab = ""
  public var `where` = ""
  public var app = ""
  public var state = ""
  public var kind: String?
  public var msg: String?
  public var turn: Int64 = 0
  public var title = ""
  public var cols = 0
  public var rows = 0

  public init(
    address: String, space: String = "", tab: String = "", where: String = "", app: String = "",
    state: String = "", kind: String? = nil, msg: String? = nil, turn: Int64 = 0,
    title: String = ""
  ) {
    self.address = address
    self.space = space
    self.tab = tab
    self.where = `where`
    self.app = app
    self.state = state
    self.kind = kind
    self.msg = msg
    self.turn = turn
    self.title = title
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    address = try c.decode(String.self, forKey: .address)
    space = try c.get(.space, "")
    tab = try c.get(.tab, "")
    `where` = try c.get(.where, "")
    app = try c.get(.app, "")
    state = try c.get(.state, "")
    kind = try c.decodeIfPresent(String.self, forKey: .kind)
    msg = try c.decodeIfPresent(String.self, forKey: .msg)
    turn = try c.get(.turn, 0)
    title = try c.get(.title, "")
    cols = try c.get(.cols, 0)
    rows = try c.get(.rows, 0)
  }

  public var blocked: Bool { state == "blocked" }

  public var label: String { !tab.isEmpty ? tab : !`where`.isEmpty ? `where` : address }

  /// The name an alert shows: the space and tab, or the label.
  public var alertName: String { `where`.isEmpty ? label : `where` }
}

public struct Space: Codable, Equatable, Sendable {
  public var id: String
  public var name: String
  public var agents: [Agent]

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    id = try c.decode(String.self, forKey: .id)
    name = try c.decode(String.self, forKey: .name)
    agents = try c.get(.agents, [])
  }
}

public struct TranscriptItem: Codable, Equatable, Hashable, Sendable {
  public var role: String
  public var kind: String
  public var text: String
  public var ts: Int64 = 0

  public init(role: String, kind: String, text: String, ts: Int64 = 0) {
    self.role = role
    self.kind = kind
    self.text = text
    self.ts = ts
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    role = try c.decode(String.self, forKey: .role)
    kind = try c.decode(String.self, forKey: .kind)
    text = try c.decode(String.self, forKey: .text)
    ts = try c.get(.ts, 0)
  }
}

public struct TranscriptPage: Decodable, Equatable, Sendable {
  public var source: String
  public var hint: String?
  public var model = ""
  public var items: [TranscriptItem] = []
  public var nextCursor = ""

  enum CodingKeys: String, CodingKey {
    case source, hint, model, items
    case nextCursor = "next_cursor"
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    source = try c.decode(String.self, forKey: .source)
    hint = try c.decodeIfPresent(String.self, forKey: .hint)
    model = try c.get(.model, "")
    items = try c.get(.items, [])
    nextCursor = try c.get(.nextCursor, "")
  }
}

/// One key of a question: a digit for an option, or "tab" for Next after a multi-select.
/// `detail` is the description under the option.
public struct PromptOption: Codable, Equatable, Sendable {
  public var key: String
  public var label: String
  public var detail = ""

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    key = try c.decode(String.self, forKey: .key)
    label = try c.decode(String.self, forKey: .label)
    detail = try c.get(.detail, "")
  }
}

public struct PromptView: Decodable, Equatable, Sendable {
  public var app = ""
  public var kind: String?
  public var msg = ""
  public var block = ""
  /// A question's block above its options: the options show as keys, not twice.
  public var question = ""
  public var options: [PromptOption] = []
  public var turn: Int64 = 0
  public var promptHash = ""
  public var actionable = false
  public var reason: String?

  enum CodingKeys: String, CodingKey {
    case app, kind, msg, block, question, options, turn, actionable, reason
    case promptHash = "prompt_hash"
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    app = try c.get(.app, "")
    kind = try c.decodeIfPresent(String.self, forKey: .kind)
    msg = try c.get(.msg, "")
    block = try c.get(.block, "")
    question = try c.get(.question, "")
    options = try c.get(.options, [])
    turn = try c.get(.turn, 0)
    promptHash = try c.get(.promptHash, "")
    actionable = try c.get(.actionable, false)
    reason = try c.decodeIfPresent(String.self, forKey: .reason)
  }

  /// A permission prompt that Approve and Deny answer. A question has `options` instead.
  public var approvable: Bool { actionable && options.isEmpty }
}

public struct Skills: Decodable, Sendable {
  public var skills: [String]

  enum CodingKeys: String, CodingKey { case skills }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    skills = try c.get(.skills, [])
  }
}

public struct Hello: Decodable, Sendable {
  public var version = ""
  public var serverTimeMs: Int64 = 0
  public var endpoints: [String] = []

  enum CodingKeys: String, CodingKey {
    case version = "daemon_version"
    case serverTimeMs = "server_time_ms"
    case endpoints
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    version = try c.get(.version, "")
    serverTimeMs = try c.get(.serverTimeMs, 0)
    endpoints = try c.get(.endpoints, [])
  }
}

public struct ResultBody: Decodable, Sendable {
  public var result: String
}

public struct ErrorBody: Decodable, Sendable {
  public var error = ""
  public var reason: String?
  public var idleMs: Int64?

  enum CodingKeys: String, CodingKey {
    case error, reason
    case idleMs = "idle_ms"
  }

  public init(from decoder: Decoder) throws {
    let c = try decoder.container(keyedBy: CodingKeys.self)
    error = try c.get(.error, "")
    reason = try c.decodeIfPresent(String.self, forKey: .reason)
    idleMs = try c.decodeIfPresent(Int64.self, forKey: .idleMs)
  }
}

/// `decision` is approve, deny or answer. An answer names the option's `key`.
public struct DecisionRequest: Encodable, Sendable {
  public var requestId: String
  public var decision: String
  public var turn: Int64
  public var promptHash: String
  public var key: String?

  public init(requestId: String, decision: String, turn: Int64, promptHash: String, key: String?) {
    self.requestId = requestId
    self.decision = decision
    self.turn = turn
    self.promptHash = promptHash
    self.key = key
  }

  enum CodingKeys: String, CodingKey {
    case decision, turn, key
    case requestId = "request_id"
    case promptHash = "prompt_hash"
  }
}

public struct MessageRequest: Encodable, Sendable {
  public var requestId: String
  public var text: String

  public init(requestId: String, text: String) {
    self.requestId = requestId
    self.text = text
  }

  enum CodingKeys: String, CodingKey {
    case text
    case requestId = "request_id"
  }
}

public struct EnrollRequest: Encodable, Sendable {
  public var token: String
  public var name: String

  public init(token: String, name: String) {
    self.token = token
    self.name = name
  }
}

public struct EnrollResponse: Decodable, Sendable {
  public var deviceId: String

  enum CodingKeys: String, CodingKey { case deviceId = "device_id" }
}

/// A reply that is not 2xx, with the daemon's error body.
public struct ApiError: Error, Equatable {
  public let status: Int
  public let error: String
  public let reason: String?
  public let idleMs: Int64?

  public init(status: Int, error: String, reason: String? = nil, idleMs: Int64? = nil) {
    self.status = status
    self.error = error
    self.reason = reason
    self.idleMs = idleMs
  }

  /// From a reply body: the JSON error, or the first 200 characters of the text.
  public init(status: Int, body: Data) {
    if let e = try? JSONDecoder().decode(ErrorBody.self, from: body) {
      self.init(
        status: status, error: e.error.isEmpty ? "HTTP \(status)" : e.error, reason: e.reason,
        idleMs: e.idleMs)
    } else {
      let text = String(decoding: body.prefix(200), as: UTF8.self)
      self.init(status: status, error: text.isEmpty ? "HTTP \(status)" : text)
    }
  }
}

/// No endpoint answered. The view keeps its last data and shows its age.
public struct UnreachableError: Error, Equatable {
  public let message: String

  public init(_ message: String) { self.message = message }
}
