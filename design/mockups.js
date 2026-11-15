/* herdr-remote mockups: content, design systems and screen renderers. Shared by options.html and viewer.html. */
/* ---------- Content (realistic data for every mockup) ---------- */
const ST = {
  idle: "Idle",
  working: "Working",
  done: "Done",
  blocked: "Blocked",
  error: "Error",
};
const STS = {
  idle: "IDLE",
  working: "WORK",
  done: "DONE",
  blocked: "BLOCKED",
  error: "ERROR",
};
const GLYPH = { blocked: "!", error: "x", done: "+", working: ">", idle: " " };
const A = {
  daemon: {
    tab: "daemon",
    space: "herdr-remote",
    app: "claude",
    state: "blocked",
    kind: "permission",
    msg: "Claude needs your permission to use Bash",
    tool: "Bash",
    age: "1 min",
  },
  android: {
    tab: "android",
    space: "herdr-remote",
    app: "claude",
    state: "working",
    title: "Pairing code in two groups",
    age: "now",
  },
  docs: {
    tab: "docs",
    space: "herdr-remote",
    app: "codex",
    state: "done",
    title: "Update the pairing docs",
    age: "12 min",
  },
  review: {
    tab: "review",
    space: "herdr-remote",
    app: "codex",
    state: "idle",
    title: "Review of PR #14",
    age: "1 h",
  },
  billing: {
    tab: "billing-webhooks",
    space: "shop-api",
    app: "claude",
    state: "blocked",
    kind: "permission",
    msg: "Claude needs your permission to use Edit",
    tool: "Edit",
    age: "4 min",
  },
  migr: {
    tab: "migrations",
    space: "shop-api",
    app: "codex",
    state: "working",
    title: "Migration 0042: refresh tokens",
    age: "now",
  },
  auth: {
    tab: "auth-refresh",
    space: "shop-api",
    app: "claude",
    state: "idle",
    title: "Refresh token flow",
    age: "38 min",
  },
  pricing: {
    tab: "pricing-page",
    space: "website",
    app: "claude",
    state: "done",
    title: "Annual plan toggle",
    age: "6 min",
  },
  og: {
    tab: "og-images",
    space: "website",
    app: "codex",
    state: "idle",
    title: "OG image templates",
    age: "2 h",
  },
  alerts: {
    tab: "grafana-alerts",
    space: "infra",
    app: "codex",
    state: "error",
    title: "p95 latency alert",
    age: "9 min",
  },
};
const SPACES = [
  { name: "herdr-remote", agents: ["daemon", "android", "docs", "review"] },
  { name: "shop-api", agents: ["billing", "migr", "auth"] },
  { name: "website", agents: ["pricing", "og"] },
  { name: "infra", agents: ["alerts"] },
];
const SORTED = [
  "daemon",
  "billing",
  "alerts",
  "pricing",
  "docs",
  "android",
  "migr",
  "auth",
  "review",
  "og",
];
const URG = ["blocked", "error", "done", "working", "idle"];
const DIFF = [
  [41, "", "  const sig = req.headers['stripe-signature'];"],
  [42, "del", "- const event = JSON.parse(req.body);"],
  [42, "add", "+ const event = stripe.webhooks.constructEvent("],
  [43, "add", "+   req.rawBody, sig, env.STRIPE_WEBHOOK_SECRET);"],
  [44, "", "  switch (event.type) {"],
]
  .map(
    ([n, c, t]) =>
      `<span class="dl ${c}"><span class="ln">${n}</span>${t}</span>`,
  )
  .join("");
const P = {
  daemon: {
    state: "ok",
    title: "Claude needs your permission to use Bash",
    kl: "Command · turn 41",
    code: '<span class="cp">$</span> go test ./daemon/... -run TestEnroll -count=1',
    why: "Run the enrollment tests after the pin change",
  },
  billing: {
    state: "refused",
    title: "Claude needs your permission to use Edit",
    kl: "File edit · src/webhooks/stripe.ts",
    code: DIFF,
    note: "Someone is using the Mac right now (idle 4 s). Try again when it is free.",
  },
  auth: {
    state: "viewonly",
    title: "Claude has a question",
    kl: "Question · view only",
    code: "Which token lifetime do you want for the\nrefresh flow?\n\n❯ 1. 15 minutes\n  2. 1 hour\n  3. 24 hours",
  },
};
const TR = {
  daemon: [
    [
      "user",
      "text",
      "Change the enrollment check so it compares the SPKI pin before it reads the token.",
    ],
    [
      "assistant",
      "text",
      "I changed enroll.go so the pin check runs first. Now I will run the enrollment tests.",
    ],
    [
      "assistant",
      "tool_call",
      "Bash  go test ./daemon/... -run TestEnroll -count=1",
    ],
  ],
  billing: [
    [
      "user",
      "text",
      "Stripe webhooks fail the signature check in staging. Find the cause and fix it.",
    ],
    [
      "assistant",
      "text",
      "The handler parses the body before it checks the signature, so the raw bytes are gone. I will pass the raw body to constructEvent.",
    ],
    ["assistant", "tool_call", "Edit  src/webhooks/stripe.ts"],
  ],
  auth: [
    ["user", "text", "Add refresh tokens to the auth service."],
    [
      "assistant",
      "text",
      "The access token lives for 15 minutes now. I need one decision before I write the migration.",
    ],
  ],
  android: [
    [
      "user",
      "text",
      "Show the pairing code as two groups of three digits. Keep the error layout as it is.",
    ],
    [
      "assistant",
      "text",
      "I will change PairScreen.kt so the code shows as two groups, then build the debug APK.",
    ],
    [
      "assistant",
      "tool_call",
      "Edit  android/app/src/main/java/dev/herdrremote/ui/PairScreen.kt",
    ],
    [
      "tool",
      "tool_result",
      "Updated PairScreen.kt with 6 additions and 1 removal",
    ],
    ["assistant", "tool_call", "Bash  ./gradlew :app:assembleDebug"],
    [
      "tool",
      "tool_result",
      "BUILD SUCCESSFUL in 41s\n38 actionable tasks: 6 executed, 32 up-to-date",
    ],
    [
      "assistant",
      "text",
      "The build passes. The code now shows as 482 913. Do you want me to install it on the phone with adb?",
    ],
  ],
  alerts: [
    [
      "terminal",
      "terminal",
      '<span class="tc-a">›</span> Add an alert for p95 latency over 800 ms\n  on api-gateway\n\n• Read dashboards/api-gateway.json\n• Edited alerts/latency.yaml (+18 -2)\n<span class="tc-r">■ stream error: error sending request for url\n  (https://api.openai.com/v1/responses)\n  retrying 5/5 in 3.2s…</span>\n<span class="tc-r">■ stream disconnected before completion</span>',
    ],
  ],
};
const TERM = [
  '<span class="w">⏺</span> I will run the enrollment tests to check the pin change.',
  "",
  '<span class="g">⏺</span> <span class="w">Bash</span>(go test ./daemon/... -run TestEnroll -count=1)',
  "  ⎿  === RUN   TestEnrollPinMismatch",
  '     --- <span class="g">PASS</span>: TestEnrollPinMismatch (0.02s)',
  "     === RUN   TestEnrollWindowClosed",
  '     --- <span class="g">PASS</span>: TestEnrollWindowClosed (0.01s)',
  "     === RUN   TestEnrollOneShotToken",
  '     --- <span class="g">PASS</span>: TestEnrollOneShotToken (0.03s)',
  '     <span class="g">ok</span>   herdr-remote/daemon/enroll   0.412s',
  "",
  '<span class="w">⏺</span> All 3 enrollment tests pass. Next I will update',
  "  docs/api.md with the new pin check.",
  "",
  '<span class="g">⏺</span> <span class="w">Update</span>(docs/api.md)',
  "  ⎿  Updated docs/api.md with 2 additions and 1 removal",
  '     212 <span class="r">-  The daemon reads the token first.</span>',
  '     212 <span class="g">+  The daemon checks the SPKI pin first,</span>',
  '     213 <span class="g">+  then reads the one-time token.</span>',
  "",
  '<span class="o">✻ Thinking…</span> <span class="d">(8s · ↑ 2.1k tokens · esc to interrupt)</span>',
  "",
  '<span class="d">╭' + "─".repeat(50) + "╮</span>",
  '<span class="d">│</span> &gt;' + " ".repeat(48) + '<span class="d">│</span>',
  '<span class="d">╰' + "─".repeat(50) + "╯</span>",
  '  <span class="m">⏵⏵ auto mode on</span> <span class="d">(shift+tab to cycle)</span>',
];

const SCREENS = [
  {
    id: "lock",
    name: "App lock",
    note: "Shows at start and after 5 min in the background. The system prompt asks for a fingerprint or the PIN.",
  },
  {
    id: "notify",
    name: "Notification on the lock screen",
    note: "The lock screen shows only the generic line. It never shows the agent or the command.",
  },
  {
    id: "pair",
    name: "Pair: start",
    note: "First run. Scan the QR code from the Herdr overlay, or paste the link.",
  },
  {
    id: "pairCode",
    name: "Pair: compare the code",
    note: "Both devices show the same 6 digits. The user clicks Allow on the Mac.",
  },
  {
    id: "pairFail",
    name: "Pair: failed",
    note: "The pairing window closed before Allow. The user can try again at once.",
  },
  {
    id: "spacesLoading",
    name: "Spaces: loading",
    note: "First load after unlock. The phone tries Wi-Fi, then Tailscale.",
  },
  {
    id: "spaces",
    name: "Spaces",
    note: "Home. Agents that need the user come first, then the spaces.",
  },
  {
    id: "spacesOffline",
    name: "Spaces: not connected",
    note: "The event stream is down. The list shows the last known data, dimmed.",
  },
  {
    id: "space",
    name: "Space",
    note: "All agents in one space, with live status.",
  },
  {
    id: "all",
    name: "All agents",
    note: "Blocked agents first, then by urgency.",
  },
  {
    id: "allEmpty",
    name: "All agents: empty",
    note: "Herdr runs no agents. Agents start on the Mac only.",
  },
  {
    id: "agentBlocked",
    name: "Agent: approve a command",
    note: "A command prompt. Approve and Deny stay in reach under a long command.",
  },
  {
    id: "agentRefused",
    name: "Agent: file edit, refused",
    note: "A file edit prompt. The daemon refused the approval because someone used the Mac.",
  },
  {
    id: "agentViewOnly",
    name: "Agent: question, view only",
    note: "A question from Claude. The phone shows it but cannot answer it.",
  },
  {
    id: "agentApplied",
    name: "Agent: approved, live view",
    note: "After Approve. The result note shows, and the Live tab shows the terminal.",
  },
  {
    id: "agentTranscript",
    name: "Agent: transcript and message",
    note: "A working agent. The user reads the transcript and types a message.",
  },
  {
    id: "agentError",
    name: "Agent: error, terminal text",
    note: "A Codex agent in daemon mode with an error. Terminal text with a hint, and an unknown send outcome.",
  },
  {
    id: "settings",
    name: "Settings",
    note: "Paired Mac, background watching and Forget this Mac.",
  },
];

/* ---------- Icons ---------- */
const ICONS = {
  back: '<path d="M19 12H5M11 6l-6 6 6 6"/>',
  chev: '<path d="M9 6l6 6-6 6"/>',
  gear: '<path d="M4 7h10M18 7h2M4 17h4M12 17h8"/><circle cx="16" cy="7" r="2"/><circle cx="10" cy="17" r="2"/>',
  send: '<path d="M4 12l16-8-6 16-3-7z"/>',
  arrow: '<path d="M5 12h14M13 6l6 6-6 6"/>',
  refresh: '<path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"/>',
  fp: '<path d="M6.5 10.5a5.5 5.5 0 0 1 11 0v1.5M9 20c1-2 1.5-4.5 1.5-8a1.5 1.5 0 0 1 3 0c0 3-.6 6-2 9M15.6 20c.6-1.8.9-3.8.9-6M7 14c0 2.2-.4 3.8-1 5"/>',
  check: '<path d="M5 12.5l4.5 4.5L19 7.5"/>',
  x: '<path d="M6 6l12 12M18 6 6 18"/>',
  spaces:
    '<rect x="4" y="4" width="7" height="7" rx="1.5"/><rect x="13" y="4" width="7" height="7" rx="1.5"/><rect x="4" y="13" width="7" height="7" rx="1.5"/><rect x="13" y="13" width="7" height="7" rx="1.5"/>',
  agents:
    '<circle cx="9" cy="8" r="3"/><path d="M3 19c0-3 2.7-5 6-5s6 2 6 5"/><path d="M16 5a3 3 0 0 1 0 6M18 14c2 .6 3 2.4 3 5"/>',
  zoom: '<path d="M4 9V4h5M20 15v5h-5M4 4l6 6M20 20l-6-6"/>',
  qr: '<rect x="4" y="4" width="6" height="6"/><rect x="14" y="4" width="6" height="6"/><rect x="4" y="14" width="6" height="6"/><path d="M14 14h2v2h-2zM18 18h2v2h-2zM14 18h2M18 14h2"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>',
  lock: '<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>',
  bell: '<path d="M6 16v-5a6 6 0 0 1 12 0v5l2 2H4zM10 20a2 2 0 0 0 4 0"/>',
  offline:
    '<path d="M2 8.5a15 15 0 0 1 20 0M5 12a10 10 0 0 1 14 0M8.5 15.5a5 5 0 0 1 7 0"/><circle cx="12" cy="19" r=".8"/><path d="M3 3l18 18"/>',
  mac: '<rect x="4" y="5" width="16" height="11" rx="1.5"/><path d="M2 19h20"/>',
  key: '<circle cx="8" cy="15" r="4"/><path d="M11 12l9-9M17 6l3 3M15 8l2 2"/>',
  tail: '<circle cx="6" cy="6" r="1.6"/><circle cx="12" cy="6" r="1.6"/><circle cx="18" cy="6" r="1.6"/><circle cx="6" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="18" cy="12" r="1.6"/><circle cx="12" cy="18" r="1.6"/>',
  alert: '<path d="M12 3.5l9.5 16.5h-19z"/><path d="M12 10v4.5M12 17.5v.3"/>',
  ask: '<circle cx="12" cy="12" r="9"/><path d="M9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.7.3-1 .9-1 1.7M12 17v.3"/>',
  phone:
    '<path d="M5 4h4l2 5-2.5 1.5a11 11 0 0 0 5 5L15 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/>',
  camera:
    '<path d="M4 8h3l2-3h6l2 3h3v11H4z"/><circle cx="12" cy="13" r="3.5"/>',
};
const ic = (n, c = "") =>
  `<svg class="ic ${c}" viewBox="0 0 24 24" aria-hidden="true">${ICONS[n]}</svg>`;
const SB_R =
  '<svg viewBox="0 0 18 14"><path d="M9 13.5 0 4.2a13 13 0 0 1 18 0z"/></svg><svg viewBox="0 0 18 14"><rect x="0" y="10" width="3" height="4" rx=".6"/><rect x="5" y="7" width="3" height="7" rx=".6"/><rect x="10" y="4" width="3" height="10" rx=".6"/><rect x="15" y="0" width="3" height="14" rx=".6"/></svg><span>82%</span><svg viewBox="0 0 26 13"><rect x=".75" y=".75" width="21.5" height="11.5" rx="3" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="3" y="3" width="14" height="7" rx="1.4"/><rect x="23.5" y="4" width="2" height="5" rx="1"/></svg>';
const NAV =
  '<div class="nb"><svg viewBox="0 0 24 24"><path d="M8 6.5v11M12 6.5v11M16 6.5v11"/></svg><svg viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="4"/></svg><svg viewBox="0 0 24 24"><path d="M14.5 5.5 8 12l6.5 6.5"/></svg></div>';
const PAIR_ART =
  '<svg class="art" viewBox="0 0 220 96" aria-hidden="true"><rect x="20" y="12" width="96" height="62" rx="6" class="a1"/><path d="M8 82h120" class="a1"/><g class="a2"><rect x="50" y="25" width="10" height="10"/><rect x="76" y="25" width="10" height="10"/><rect x="50" y="51" width="10" height="10"/><rect x="65" y="40" width="6" height="6"/><rect x="76" y="51" width="10" height="10"/></g><path d="M128 44h32" class="a3"/><rect x="166" y="8" width="44" height="80" rx="9" class="a1"/><path d="M181 16h14" class="a1"/><rect x="176" y="36" width="24" height="24" rx="3" class="a4"/></svg>';

/* ---------- Design systems ---------- */
const T = (k, n, w, s, l, f, smp, x = "") => ({ k, n, w, s, l, f, smp, x });
const SYS = [
  {
    id: "pane",
    letter: "A",
    name: "Pane",
    def: "dark",
    nav: "bottom",
    lead: "glyph",
    idea: "The app looks like the terminal it controls. Monospace text, box-drawn panes, ANSI colours and key hints. Dense, so more agents fit on one screen.",
    pros: [
      "Matches Herdr and the terminal that the agents run in.",
      "Dense. About 30% more rows fit on one screen.",
    ],
    cons: ["Small text. Less friendly at a glance."],
    sig: "Panes with their title in the border, and a tmux-style status line as the navigation.",
    fontNote:
      "Bundle JetBrains Mono for all text. The mockup uses SF Mono or Menlo.",
    fonts: {
      d: "'JetBrains Mono',ui-monospace,'SF Mono',Menlo,monospace",
      b: "'JetBrains Mono',ui-monospace,'SF Mono',Menlo,monospace",
      m: "'JetBrains Mono',ui-monospace,'SF Mono',Menlo,monospace",
    },
    type: [
      T("num", "Pairing code", 700, 44, 52, "d", "482 913"),
      T("display", "Display", 700, 18, 26, "d", "Pair with your Mac"),
      T("head", "Headline", 700, 16, 24, "d", "No agents"),
      T("title", "Title", 700, 14, 20, "d", "~/spaces/herdr-remote"),
      T(
        "body",
        "Body",
        400,
        13,
        20,
        "b",
        "Claude needs your permission to use Bash",
      ),
      T("label", "Label", 400, 12, 18, "b", "[y] approve"),
      T("cap", "Caption", 400, 11, 16, "b", "claude · 1 min"),
      T(
        "mono",
        "Mono",
        400,
        12,
        18,
        "m",
        "go test ./daemon/... -run TestEnroll",
      ),
    ],
    space: [4, 8, 10, 12, 16],
    radius: [["All", 0]],
    light: {
      bg: "#EEF1F6",
      sf: "#FFFFFF",
      sf2: "#E2E7EF",
      ln: "#B7C0D1",
      tx: "#222836",
      mu: "#5B667D",
      ac: "#2E7D8C",
      onac: "#FFFFFF",
      idle: "#7A8499",
      work: "#3A6EA5",
      done: "#4F7F2E",
      block: "#B23A48",
      err: "#B4561E",
    },
    dark: {
      bg: "#191C26",
      sf: "#1F2330",
      sf2: "#272C3B",
      ln: "#3B4258",
      tx: "#D8DEE9",
      mu: "#7D879E",
      ac: "#88C0D0",
      onac: "#191C26",
      idle: "#6C7891",
      work: "#81A1C1",
      done: "#A3BE8C",
      block: "#BF616A",
      err: "#D08770",
    },
    term: {
      bg: "#14161E",
      fg: "#D8DEE9",
      g: "#A3BE8C",
      r: "#BF616A",
      y: "#EBCB8B",
      b: "#81A1C1",
      m: "#B48EAD",
      o: "#D08770",
    },
    wall: "linear-gradient(175deg,#14161E 0%,#2B3142 70%,#3B4252 100%)",
    icon: {
      bg: '<rect width="108" height="108" fill="#191C26"/>',
      fg: '<rect x="31" y="33" width="46" height="42" rx="4" fill="#222736" stroke="#D8DEE9" stroke-width="3.5"/><path d="M56 33v42M56 54h21" stroke="#D8DEE9" stroke-width="3.5"/><rect x="59.5" y="36.5" width="14" height="14" fill="#BF616A"/><path d="M37 47l6 5-6 5" fill="none" stroke="#88C0D0" stroke-width="3.5" stroke-linecap="square"/><rect x="45" y="58" width="6" height="3.5" fill="#A3BE8C"/>',
      mono: '<rect x="31" y="33" width="46" height="42" rx="4" fill="none" stroke="var(--mf)" stroke-width="3.5"/><path d="M56 33v42M56 54h21" stroke="var(--mf)" stroke-width="3.5"/><rect x="59.5" y="36.5" width="14" height="14" fill="var(--mf)"/><path d="M37 47l6 5-6 5" fill="none" stroke="var(--mf)" stroke-width="3.5" stroke-linecap="square"/><rect x="45" y="58" width="6" height="3.5" fill="var(--mf)"/>',
      iconNote: "A split terminal pane. One pane is red: an agent that waits.",
    },
  },
  {
    id: "dispatch",
    letter: "B",
    name: "Dispatch",
    def: "light",
    nav: "top3",
    lead: "shape",
    idea: "Big type, a strict grid and numbers first. You see how many agents need you from arm's length. Black, white and cobalt. Status uses shape and colour.",
    pros: [
      "Easy to read at a glance and in sunlight.",
      "Status has a shape (square, triangle, ring), so it works for colour-blind users.",
    ],
    cons: ["Big type leaves less room for transcripts."],
    sig: "A giant numeral for blocked agents, and status shapes in every row.",
    fontNote:
      "Bundle Inter Tight (Regular to ExtraBold) and JetBrains Mono. The mockup uses Helvetica Neue.",
    fonts: {
      d: "'Inter Tight','Helvetica Neue',Helvetica,Arial,sans-serif",
      b: "'Inter Tight','Helvetica Neue',Helvetica,Arial,sans-serif",
      m: "'JetBrains Mono',ui-monospace,Menlo,monospace",
    },
    type: [
      T("num", "Numeral", 800, 112, 96, "d", "2", "letter-spacing:-.06em"),
      T(
        "display",
        "Display",
        800,
        40,
        42,
        "d",
        "herdr-remote",
        "letter-spacing:-.035em",
      ),
      T(
        "head",
        "Headline",
        700,
        26,
        28,
        "d",
        "agents need you",
        "letter-spacing:-.02em",
      ),
      T(
        "title",
        "Title",
        700,
        20,
        24,
        "d",
        "billing-webhooks",
        "letter-spacing:-.02em",
      ),
      T(
        "body",
        "Body",
        400,
        15,
        22,
        "b",
        "Claude needs your permission to use Bash",
      ),
      T(
        "label",
        "Label",
        700,
        11,
        14,
        "b",
        "APPROVE · BLOCKED",
        "letter-spacing:.08em",
      ),
      T("cap", "Caption", 500, 12, 16, "b", "4 min ago"),
      T(
        "mono",
        "Mono",
        500,
        13,
        19,
        "m",
        "go test ./daemon/... -run TestEnroll",
      ),
    ],
    space: [4, 8, 12, 20, 24, 40],
    radius: [
      ["All", 0],
      ["Rule", "2 px"],
    ],
    light: {
      bg: "#FFFFFF",
      sf: "#FFFFFF",
      sf2: "#F1F1F1",
      ln: "#0A0A0A",
      tx: "#0A0A0A",
      mu: "#6B6B6B",
      ac: "#1D3BFF",
      onac: "#FFFFFF",
      idle: "#9A9A9A",
      work: "#1D3BFF",
      done: "#00A05A",
      block: "#F0301A",
      err: "#FF9500",
    },
    dark: {
      bg: "#0A0A0A",
      sf: "#0A0A0A",
      sf2: "#1A1A1A",
      ln: "#FAFAFA",
      tx: "#FAFAFA",
      mu: "#8E8E8E",
      ac: "#5B73FF",
      onac: "#FFFFFF",
      idle: "#6E6E6E",
      work: "#5B73FF",
      done: "#2FD07F",
      block: "#FF5440",
      err: "#FFB020",
    },
    term: {
      bg: "#000000",
      fg: "#F2F2F2",
      g: "#2FD07F",
      r: "#FF5440",
      y: "#FFB020",
      b: "#7A8CFF",
      m: "#E07AFF",
      o: "#FF8A3D",
    },
    wall: "linear-gradient(180deg,#1D3BFF 0%,#1D3BFF 62%,#0A0A0A 62%)",
    icon: {
      bg: '<rect width="108" height="108" fill="#FFFFFF"/>',
      fg: '<path d="M41 30v48M41 60c0-9 5-13 12-13s12 4 12 13v18" fill="none" stroke="#0A0A0A" stroke-width="11"/><rect x="62" y="27" width="12" height="12" fill="#1D3BFF"/>',
      mono: '<path d="M41 30v48M41 60c0-9 5-13 12-13s12 4 12 13v18" fill="none" stroke="var(--mf)" stroke-width="11"/><rect x="62" y="27" width="12" height="12" fill="var(--mf)"/>',
      iconNote: "A heavy lowercase h with one cobalt square.",
    },
  },
];
const TOK = [
  ["bg", "Background"],
  ["sf", "Surface"],
  ["sf2", "Surface 2"],
  ["ln", "Line"],
  ["tx", "Text"],
  ["mu", "Muted text"],
  ["ac", "Accent"],
  ["onac", "On accent"],
  ["idle", "Idle"],
  ["work", "Working"],
  ["done", "Done"],
  ["block", "Blocked"],
  ["err", "Error"],
];
const VMAP = {
  idle: "s-idle",
  work: "s-work",
  done: "s-done",
  block: "s-block",
  err: "s-err",
};

function vars(S, mode) {
  const t = S[mode];
  let v = Object.entries(t).map(([k, x]) => `--${VMAP[k] || k}:${x}`);
  Object.entries(S.term).forEach(([k, x]) =>
    v.push(k === "bg" || k === "fg" ? `--t-${k}:${x}` : `--a-${k}:${x}`),
  );
  v.push(
    `--fd:${S.fonts.d}`,
    `--fb:${S.fonts.b}`,
    `--fm:${S.fonts.m}`,
    `--wall:${S.wall}`,
  );
  S.type.forEach((r) =>
    v.push(`--T-${r.k}:${r.w} ${r.s}px/${r.l}px var(--f${r.f})`),
  );
  return v.join(";");
}

/* ---------- Components ---------- */
const cap = (s) => s[0].toUpperCase() + s.slice(1);
const urgent = (ags) =>
  URG.find((u) => ags.some((a) => a.state === u)) || "idle";
const badge = (st) =>
  `<span class="bdg st-${st}" data-s="${STS[st]}"><i></i><b>${ST[st]}</b></span>`;
const sec = (l) => `<div class="sec"><span>${l}</span></div>`;
const btn = (k, l, i, key) =>
  `<span class="btn ${k}"${key ? ` data-k="${key}"` : ""}>${i ? ic(i) : ""}${l}</span>`;
const fld = (l, v, ph) =>
  `<div class="fld"><span class="fld-l">${l}</span><span class="fld-v ${v ? "" : "ph"}">${v || ph}</span></div>`;
const note = (k, t) =>
  `<div class="note ${k}">${ic(k === "ok" ? "check" : k === "err" ? "x" : "alert")}<span>${t}</span></div>`;
function lead(S, a, sz = "") {
  if (S.lead === "glyph") return `<span class="ind">[${GLYPH[a.state]}]</span>`;
  return '<span class="ind"></span>';
}
function spaceLead(S, sp, i, u, ags) {
  switch (S.id) {
    case "pane":
      return `<span class="ind">[${GLYPH[u]}]</span>`;
    case "dispatch":
      return `<span class="num">${ags.length}</span>`;
    default:
      return '<span class="ind"></span>';
  }
}
function agentRow(S, a, withSpace) {
  const meta = [withSpace && a.space, a.app, a.age].filter(Boolean).join(" · ");
  return `<div class="row ag st-${a.state}">${lead(S, a)}<div class="mid"><div class="r1">${a.tab}</div><div class="r2">${meta}</div><div class="r3">${a.state === "blocked" ? a.msg : a.title}</div></div><div class="trail">${badge(a.state)}</div></div>`;
}
function spaceRow(S, sp, i) {
  const ags = sp.agents.map((k) => A[k]),
    u = urgent(ags),
    nb = ags.filter((a) => a.state === "blocked").length,
    ne = ags.filter((a) => a.state === "error").length;
  const cnt = `${ags.length} agent${ags.length > 1 ? "s" : ""}${nb ? ` · ${nb} blocked` : ""}${ne ? ` · ${ne} error` : ""}`;
  return `<div class="row sp st-${u}">${spaceLead(S, sp, i, u, ags)}<div class="mid"><div class="r1">${sp.name}</div><div class="r2">${cnt}</div></div><div class="trail"><span class="pips">${ags.map((a) => `<i class="st-${a.state}"></i>`).join("")}</span>${ic("chev")}</div></div>`;
}
function hero(S) {
  return `<div class="hero"><div class="hero-k">Needs you</div><div class="hero-n"><b>2</b> <span>agents need you</span></div><div class="hero-l">${[A.daemon, A.billing].map((a) => `<div class="hero-i st-blocked">${lead(S, a, "sm")}<div class="mid"><div class="r1">${a.tab}</div><div class="r2">${a.tool} · ${a.space}</div></div>${ic("chev")}</div>`).join("")}</div></div>`;
}
function appBar(S, o) {
  let title = o.title;
  if (S.id === "pane" && o.path) {
    const i = o.path.lastIndexOf("/");
    title = `<span class="pth">${o.path.slice(0, i + 1)}</span>${o.path.slice(i + 1)}`;
  }
  const back = o.back
    ? `<span class="ibtn back">${ic("back")}<span class="bl">${o.back}</span></span>`
    : "";
  return `<div class="ab ${o.big ? "big" : ""}">${back}<div class="t">${title}</div><span class="act"><span class="link ${o.off ? "off" : "on"}">Link</span>${o.action || ""}</span></div>${o.sub ? `<div class="sub">${o.sub}</div>` : ""}`;
}
function nav(S, active, count = 2) {
  const em = (k) => (k === "agents" && count ? `<em>${count}</em>` : "");
  if (S.nav === "top3") {
    const it = [
      ["spaces", "Spaces"],
      ["agents", "Agents"],
      ["settings", "Settings"],
    ];
    return {
      top: `<div class="tn">${it.map(([k, l]) => `<span class="tn-i ${k === active ? "on" : ""}">${l}${em(k)}</span>`).join("")}</div>`,
      bottom: "",
    };
  }
  const it = [
    ["spaces", "Spaces", "spaces"],
    ["agents", "Agents", "agents"],
    ["settings", "Settings", "gear"],
  ];
  return {
    top: "",
    bottom: `<div class="bn">${it.map(([k, l, i], n) => `<span class="bn-i ${k === active ? "on" : ""}" data-n="${n}"><span class="pill">${ic(i)}${em(k)}</span><b>${l}</b></span>`).join("")}</div>`,
  };
}
function approval(S, p) {
  const act =
    p.state === "viewonly"
      ? `<div class="appr-mac">${ic("mac")}<span>Answer this on the Mac.</span></div>`
      : `<div class="appr-act">${btn("pri", "Approve", "check", "y")}${btn("alt", "Deny", "x", "n")}${btn("txt", "Reload", "refresh", "r")}</div>`;
  return `<div class="appr ${p.state}" data-title="${p.state === "viewonly" ? "question" : "permission"}"><div class="appr-h"><span class="appr-ic">${ic(p.state === "viewonly" ? "ask" : "alert")}</span><div><div class="appr-t">${p.title}</div><div class="appr-k">${p.kl}</div></div></div><div class="appr-code">${p.code}</div>${p.why ? `<div class="appr-why">${p.why}</div>` : ""}${p.note ? `<div class="appr-note">${ic("x")}<span>${p.note}</span></div>` : ""}${act}</div>`;
}
function composer(o) {
  const inner = o.value
    ? `<span class="v">${o.value}</span><span class="caret"></span><span class="cnt">${o.cnt || ""}</span>`
    : `<span class="ph">${o.ph}</span>`;
  return `<div class="cmp ${o.off ? "off" : ""}"><div class="cmp-in">${inner}</div><span class="cmp-send">${ic("send", "i-send")}${ic("arrow", "i-arrow")}<b>Send</b></span></div>`;
}
const tabs = (t) =>
  `<div class="tabs"><span class="tab ${t === "transcript" ? "on" : ""}">Transcript</span><span class="tab ${t === "live" ? "on" : ""}">Live</span></div>`;
function transcript(a, items, pre = "") {
  const L = (it) =>
    ({
      user: ["You", "you ›"],
      tool_call: ["Tool call", "tool $"],
      tool_result: ["Result", "  ⎿"],
      terminal: ["Terminal", ""],
    })[it[1] === "text" ? it[0] : it[1]] || [cap(a.app), a.app + " ›"];
  return `<div class="tr">${pre}${items
    .map((it) => {
      const [l, p] = L(it);
      return `<div class="ti ${it[0]} ${it[1]}"><div class="ti-l" data-p="${p}">${l}</div><div class="ti-x">${it[2]}</div></div>`;
    })
    .join("")}</div>`;
}
const live = () =>
  `<div class="live"><div class="term">${TERM.join("\n")}</div><div class="live-cap">${ic("zoom")}<span>Live · 96 × 34 · pinch to zoom</span></div></div>`;
const scr = (o) =>
  `<div class="scr ${o.bottom && o.bottom.includes('class="bn"') ? "hasbn" : ""}">${o.top || ""}<div class="body ${o.col ? "col" : ""}">${o.body || ""}</div>${o.bottom || ""}</div>`;
const aicon = (S, mask, size = 72) =>
  `<div class="aicon m-${mask}" style="width:${size}px;height:${size}px"><svg viewBox="18 18 72 72" aria-hidden="true">${S.icon.bg}${S.icon.fg}</svg></div>`;
const setRow = (i, r1, r2, trail = "") =>
  `<div class="set"><span class="set-ic">${ic(i)}</span><div class="mid"><div class="r1">${r1}</div>${r2 ? `<div class="r2">${r2}</div>` : ""}</div>${trail}</div>`;

/* ---------- Screens ---------- */
function spacesScreen(S, m) {
  const n = nav(S, "spaces");
  const sub =
    m === "loading" ? "Connecting to the Mac…" : "10 agents · 2 blocked";
  let top =
    appBar(S, {
      title: "Spaces",
      path: "~/spaces",
      big: true,
      sub,
      off: m === "offline",
    }) + n.top;
  if (m === "offline")
    top += `<div class="ban">${ic("offline")}<span>Not connected to the Mac: last update 48 s ago</span></div>`;
  let body;
  if (m === "loading") {
    body =
      S.id === "pane"
        ? `<div class="tlog"><div><b>$ herdr-remote connect</b></div><div>› trying 192.168.1.24 … <span class="e">timeout</span></div><div>› trying zeus-mbp.tail3c2e.ts.net …</div><div><span class="caret" style="display:inline-block;width:8px;height:15px;vertical-align:middle;background:var(--tx)"></span></div></div>`
        : `<div class="skel"><div class="sk sk-hero"></div>${sec("Spaces")}<div class="list">${[1, 2, 3, 4].map(() => `<div class="row"><i class="sk sk-a"></i><div class="mid"><i class="sk sk-l"></i><i class="sk sk-l s"></i></div></div>`).join("")}</div></div>`;
  } else
    body = `<div class="${m === "offline" ? "stale" : ""}">${hero(S)}${sec("4 spaces")}<div class="list">${SPACES.map((sp, i) => spaceRow(S, sp, i)).join("")}</div></div>`;
  return scr({ top, body, bottom: n.bottom });
}
function spaceScreen(S) {
  const sp = SPACES[0];
  const top = appBar(S, {
    title: sp.name,
    back: "Spaces",
    path: "~/spaces/herdr-remote",
    sub: "4 agents · 1 blocked",
  });
  return scr({
    top,
    body: `${sec("Agents")}<div class="list">${sp.agents.map((k) => agentRow(S, A[k], false)).join("")}</div>`,
  });
}
function allScreen(S, empty) {
  const n = nav(S, "agents", empty ? 0 : 2);
  const top =
    appBar(S, {
      title: "All agents",
      path: "~/agents",
      big: true,
      sub: empty ? "Nothing runs on the Mac" : "10 agents · blocked first",
    }) + n.top;
  let body;
  if (empty) {
    const art = {
      pane: '<div class="ascii">\u250C\u2500 hc agents \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2510\n\u2502                 \u2502\n\u2502   (0 agents)    \u2502\n\u2502                 \u2502\n\u2514\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2518</div>',
      dispatch: '<div class="zero">0</div>',
    }[S.id];
    body = `<div class="empty">${art}<div class="empty-t">No agents</div><div class="empty-b">Start an agent in Herdr on the Mac. It shows here within 2 s.</div></div>`;
  } else {
    const bl = SORTED.filter((k) => A[k].state === "blocked"),
      rest = SORTED.filter((k) => A[k].state !== "blocked");
    body = `${sec("Needs you · 2")}<div class="list">${bl.map((k) => agentRow(S, A[k], true)).join("")}</div>${sec("Others · 8")}<div class="list">${rest.map((k) => agentRow(S, A[k], true)).join("")}</div>`;
  }
  return scr({ top, body, bottom: n.bottom });
}
function agentScreen(S, o) {
  const a = o.a;
  const top = appBar(S, {
    title: a.tab,
    back: a.space,
    path: `~/${a.space}/${a.tab}`,
    sub: `${badge(a.state)}<span>${a.app} · ${a.space}${a.kind ? ` · ${a.kind}` : ""}</span>`,
  });
  let b = "";
  if (o.prompt) b += approval(S, o.prompt);
  if (o.note) b += note(...o.note);
  b += tabs(o.tab);
  b += o.tab === "live" ? live() : transcript(a, o.items, o.pre || "");
  return scr({ top, body: b, col: true, bottom: composer(o.cmp) });
}
function pairScreen(S, st) {
  const art = `<div>${PAIR_ART}</div>`;
  let b;
  if (st === "code")
    b = `${art}<div class="pair-t">Check the code</div><p class="pair-b">The Mac shows a 6-digit code too.</p><div class="code6"><span>482</span><span>913</span></div><p class="pair-b">If the Mac shows the same code, click <b>Allow</b> there. Otherwise click <b>Deny</b>.</p><div class="wait"><span class="spin"></span>Waiting for Allow on the Mac</div><p class="hint">This phone: Galaxy S26</p>`;
  else
    b = `${art}<div class="pair-t">Pair with your Mac</div>${st === "fail" ? note("err", "The pairing window closed before Allow. Click Pair a phone again on the Mac.") : `<ol class="steps"><li><span>On the Mac, open Herdr Remote and click <b>Pair a phone</b>.</span></li><li><span>Scan the QR code that the Mac shows.</span></li><li><span>If both codes match, click <b>Allow</b> on the Mac.</span></li></ol>`}${fld("This phone's name", "Galaxy S26")}${btn("pri blk", st === "fail" ? "Scan the QR code again" : "Scan the QR code", "qr")}<div class="or">or</div>${fld("Or paste the pairing link", "", "herdr-remote://pair?t=…")}${btn("alt blk off", "Pair with the pasted link", "link")}`;
  return scr({ body: `<div class="pair">${b}</div>` });
}
function lockScreen(S, mode) {
  return scr({
    body: `<div class="lock">${aicon(S, "circle", 84)}<div class="lock-t">Herdr Remote</div><div class="lock-s">${ic("lock")}Locked</div>${btn("pri", "Unlock")}</div><div class="bio ${mode === "dark" ? "dk" : ""}"><div class="bio-t">Herdr Remote</div><div class="bio-s">Unlock to see your agents</div><div class="bio-fp">${ic("fp")}</div><div class="bio-c">Touch the fingerprint sensor</div><div class="bio-pin">Use PIN</div></div>`,
  });
}
function notifyScreen(S) {
  const i = `<span class="nt-ic"><svg viewBox="18 18 72 72" aria-hidden="true">${S.icon.mono}</svg></span>`;
  return `<div class="scr"><div class="ls"><div class="ls-lock">${ic("lock")}</div><div class="ls-clock">10:24</div><div class="ls-date">Mon 9 November</div><div class="ls-n"><div class="nt">${i}<div class="mid"><div class="nt-a">Herdr Remote · now</div><div class="nt-t">An agent needs attention</div></div></div><div class="nt">${i}<div class="mid"><div class="nt-a">Herdr Remote</div><div class="nt-t">Herdr Remote is watching</div></div></div></div><div class="ls-ft"><span>${ic("phone")}</span><span>${ic("camera")}</span></div></div></div>`;
}
function settingsScreen(S) {
  const n = nav(S, "settings");
  const top =
    appBar(S, { title: "Settings", path: "~/settings", big: true }) + n.top;
  const body = `<div class="setw">${sec("Paired Mac")}<div class="grp">${setRow("mac", "192.168.1.24", "Wi-Fi")}${setRow("tail", "zeus-mbp.tail3c2e.ts.net", "Tailscale")}${setRow("key", "Device id", "d_01JC8Y4K2M")}</div>${sec("Background watching")}<div class="grp">${setRow("bell", "Notify me when an agent is blocked", "", '<span class="sw on"><i></i></span>')}</div><p class="hint">Alerts come only while this phone can reach the Mac. The notification never shows the command.</p>${btn("dang blk", "Forget this Mac")}<p class="hint">The Mac keeps this phone listed until you revoke it there.</p></div>`;
  return scr({ top, body, bottom: n.bottom });
}
const RENDER = {
  lock: (S, m) => ({ html: lockScreen(S, m) }),
  notify: (S) => ({ html: notifyScreen(S), wall: true, sbn: true }),
  pair: (S) => ({ html: pairScreen(S, "idle") }),
  pairCode: (S) => ({ html: pairScreen(S, "code") }),
  pairFail: (S) => ({ html: pairScreen(S, "fail") }),
  spacesLoading: (S) => ({ html: spacesScreen(S, "loading"), sbn: true }),
  spaces: (S) => ({ html: spacesScreen(S, "ok"), sbn: true }),
  spacesOffline: (S) => ({ html: spacesScreen(S, "offline"), sbn: true }),
  space: (S) => ({ html: spaceScreen(S), sbn: true }),
  all: (S) => ({ html: allScreen(S, false), sbn: true }),
  allEmpty: (S) => ({ html: allScreen(S, true), sbn: true }),
  agentBlocked: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: A.daemon,
      prompt: P.daemon,
      tab: "transcript",
      items: TR.daemon,
      cmp: { ph: "Answer the prompt first", off: true },
    }),
  }),
  agentRefused: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: A.billing,
      prompt: P.billing,
      tab: "transcript",
      items: TR.billing,
      cmp: { ph: "Answer the prompt first", off: true },
    }),
  }),
  agentViewOnly: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: { ...A.auth, state: "blocked", kind: "question" },
      prompt: P.auth,
      tab: "transcript",
      items: TR.auth,
      cmp: { ph: "Answer the prompt first", off: true },
    }),
  }),
  agentApplied: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: { ...A.daemon, state: "working", kind: null },
      note: ["ok", "Done. The agent continues."],
      tab: "live",
      cmp: { ph: "Message the agent" },
    }),
  }),
  agentTranscript: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: A.android,
      tab: "transcript",
      items: TR.android,
      pre: '<div class="more">Load earlier</div>',
      cmp: {
        value: "Yes, install it and open the pairing screen",
        cnt: "44/4000",
      },
    }),
  }),
  agentError: (S) => ({
    sbn: true,
    html: agentScreen(S, {
      a: A.alerts,
      note: [
        "warn",
        "Outcome unknown. Check the transcript before you send again.",
      ],
      tab: "transcript",
      items: TR.alerts,
      pre: '<div class="trsrc">Terminal text · Start Codex with --no-daemon to see the transcript.</div>',
      cmp: { ph: "Message the agent" },
    }),
  }),
  settings: (S) => ({ html: settingsScreen(S), sbn: true }),
};
function phone(S, k, mode) {
  const r = RENDER[k](S, mode);
  const sbn = r.sbn
    ? `<span class="sb-n"><svg viewBox="18 18 72 72" aria-hidden="true">${S.icon.mono}</svg></span>`
    : "";
  return `<div class="ui phone s-${S.id} md-${mode} ${r.wall ? "wallp" : ""}" style="${vars(S, mode)}"><div class="sb"><span>10:24</span>${sbn}<span class="cam"></span><span class="sb-r">${SB_R}</span></div>${r.html}${NAV}</div>`;
}
