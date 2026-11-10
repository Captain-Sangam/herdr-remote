// Herdr Remote: the paired phones with a Revoke for each, and Pair a phone,
// which shows the pairing QR code, then the 6-digit code with Allow and Deny.
// Open or close it with `herdr-app do mod.open --name herdr-remote`. It talks
// to the daemon only through the herdr-remoted CLI (install.sh writes the path
// below); every call returns within a second. It writes no file and never
// redraws on a timer: the status loop notifies only when the state changes.
// Commands for keys.json or scripts: herdr-remote.pair, .allow, .deny, .cancel.

import { View, div } from "gpui-kit";
import { v_flex, h_flex } from "gpui-base";
import { exec, theme, props, closeOverlay, nextCommand } from "herdr";

const DAEMON = "__HERDR_REMOTED__";
const QR_WIDTH = 280; // px for the squares, without the quiet zone

export default class HerdrRemote extends View {
  init(_props, cx) {
    this.devices = [];
    this.error = null;
    this.pair = null; // { matrix, expires_at } while this overlay shows a window
    this.status = { state: "none" };
    this.gen = 0; // bumping it ends the status loop
    this.handle = cx.focus_handle();
    this.handle.focus();
    this.refresh(cx);
    cx.spawn(async (cx) => {
      for (;;) {
        const c = await nextCommand();
        if (c === "pair") this.startPair(cx);
        else if (c === "allow" || c === "deny" || c === "cancel")
          this.decide(cx, c);
        else if (c === "focus") this.handle.focus();
      }
    });
  }

  async call(args) {
    const r = await exec(DAEMON, [...args, "--json"]);
    if (r.code !== 0)
      throw new Error(
        (r.stderr || r.stdout).trim() || `${args.join(" ")} failed`,
      );
    return JSON.parse(r.stdout);
  }

  fail(e) {
    this.error = String(e?.message ?? e);
  }

  refresh(cx) {
    cx.spawn(async (cx) => {
      try {
        this.devices = await this.call(["devices", "ls"]);
        this.error = null;
      } catch (e) {
        this.fail(e);
      }
      cx.notify();
    });
  }

  startPair(cx) {
    cx.spawn(async (cx) => {
      try {
        this.pair = await this.call(["pair"]);
        this.status = { state: "open" };
        this.error = null;
        this.poll(cx);
      } catch (e) {
        this.fail(e);
      }
      cx.notify();
    });
  }

  // Asks the daemon for the pairing state each second; redraws only on a change.
  poll(cx) {
    const gen = ++this.gen;
    cx.spawn(async (cx) => {
      while (gen === this.gen) {
        await cx.sleep(1000);
        if (gen !== this.gen) return;
        let st;
        try {
          st = await this.call(["pair", "status"]);
        } catch (e) {
          this.fail(e);
          st = { state: "none" };
        }
        if (JSON.stringify(st) === JSON.stringify(this.status)) continue;
        this.status = st;
        if (st.state !== "open" && st.state !== "pending") this.endWindow(cx);
        cx.notify();
      }
    });
  }

  endWindow(cx) {
    this.gen++;
    this.pair = null;
    this.refresh(cx);
  }

  decide(cx, what) {
    cx.spawn(async (cx) => {
      try {
        await this.call(["pair", what]);
        this.error = null;
      } catch (e) {
        this.fail(e);
      }
      try {
        this.status = await this.call(["pair", "status"]);
      } catch {
        this.status = { state: "none" };
      }
      this.endWindow(cx);
      cx.notify();
    });
  }

  revoke(cx, id) {
    cx.spawn(async (cx) => {
      try {
        await this.call(["devices", "rm", id]);
        this.error = null;
      } catch (e) {
        this.fail(e);
      }
      this.refresh(cx);
    });
  }

  done(cx) {
    this.status = { state: "none" };
    this.refresh(cx);
  }

  // A closed overlay cancels any window it opened.
  async close(_cx) {
    const open =
      this.pair !== null ||
      this.status.state === "open" ||
      this.status.state === "pending";
    this.gen++;
    if (open) await exec(DAEMON, ["pair", "cancel"]);
  }

  // The QR matrix as run-length rows of plain div()s, with a 4-cell quiet zone.
  qr() {
    const rows = this.pair.matrix;
    const cell = Math.max(2, Math.floor(QR_WIDTH / rows.length));
    return v_flex()
      .p(cell * 4)
      .bg("#ffffff")
      .rounded(6)
      .children(
        rows.map((row) => {
          const runs = [];
          for (let x = 0; x < row.length;) {
            let len = 1;
            while (x + len < row.length && row[x + len] === row[x]) len++;
            runs.push({ dark: row[x] === "1", len });
            x += len;
          }
          return h_flex()
            .h(cell)
            .flex_none()
            .children(
              runs.map((r) =>
                div()
                  .w(r.len * cell)
                  .h(cell)
                  .flex_none()
                  .bg(r.dark ? "#000000" : "#ffffff"),
              ),
            );
        }),
      );
  }

  button(t, id, label, primary, onClick) {
    return h_flex()
      .id(id)
      .h(30)
      .px(12)
      .items_center()
      .rounded(6)
      .border_1()
      .border_color(primary ? t.solid : t.input)
      .bg(primary ? t.solid : t.raised)
      .text_color(primary ? t.solid_foreground : t.foreground)
      .cursor_pointer()
      .hover((s) => s.opacity(0.88))
      .on_click((_e, cx) => onClick(cx))
      .child(label);
  }

  body(t) {
    const st = this.status;
    if (this.pair && st.state === "open") {
      const at = new Date(this.pair.expires_at * 1000).toLocaleTimeString();
      return v_flex()
        .gap(10)
        .items_center()
        .child(
          div()
            .text_color(t.muted)
            .child("Scan this with the Herdr Remote app"),
        )
        .child(this.qr())
        .child(div().text_color(t.dim).text_size(12).child(`Expires at ${at}`))
        .child(
          this.button(t, "cancel", "Cancel", false, (cx) =>
            this.decide(cx, "cancel"),
          ),
        );
    }
    if (st.state === "pending") {
      return v_flex()
        .gap(10)
        .items_center()
        .child(
          div()
            .text_color(t.muted)
            .child(`Phone: ${st.device_name ?? ""}`),
        )
        .child(
          div()
            .text_size(34)
            .font_semibold()
            .child(st.code ?? ""),
        )
        .child(
          div()
            .text_color(t.muted)
            .child("Allow only if the phone shows the same code."),
        )
        .child(
          h_flex()
            .gap(8)
            .child(
              this.button(t, "deny", "Deny", false, (cx) =>
                this.decide(cx, "deny"),
              ),
            )
            .child(
              this.button(t, "allow", "Allow", true, (cx) =>
                this.decide(cx, "allow"),
              ),
            ),
        );
    }
    if (
      st.state === "allowed" ||
      st.state === "denied" ||
      st.state === "expired"
    ) {
      const text =
        st.state === "allowed"
          ? `Paired ${st.device_name ?? "the phone"}.`
          : st.state === "denied"
            ? "Pairing denied."
            : "Pairing expired.";
      return v_flex()
        .gap(10)
        .items_center()
        .child(div().child(text))
        .child(this.button(t, "done", "Done", true, (cx) => this.done(cx)));
    }
    const list = this.devices.length
      ? this.devices.map((d) =>
          h_flex()
            .id(`dev-${d.id}`)
            .items_center()
            .gap(8)
            .py(4)
            .child(div().flex_1().truncate().child(d.name))
            .child(
              div()
                .text_color(t.dim)
                .text_size(12)
                .child((d.added ?? "").slice(0, 10)),
            )
            .child(
              this.button(t, `rm-${d.id}`, "Revoke", false, (cx) =>
                this.revoke(cx, d.id),
              ),
            ),
        )
      : [div().text_color(t.muted).child("No phones are paired.")];
    return v_flex()
      .gap(6)
      .child(div().flex_1().min_h(0).overflow_y_scrollbar().children(list))
      .child(
        h_flex()
          .justify_end()
          .gap(8)
          .pt(8)
          .child(
            this.button(t, "pair", "Pair a phone", true, (cx) =>
              this.startPair(cx),
            ),
          ),
      );
  }

  render(cx) {
    const t = theme();
    return v_flex()
      .id("herdr-remote")
      .track_focus(this.handle)
      .on_key_down((e, cx) => {
        if (e.key !== "escape") return;
        closeOverlay(props().overlay);
        cx.stop_propagation();
      })
      .size_full()
      .min_h(0)
      .p(18)
      .gap(12)
      .bg(t.popover)
      .text_size(13)
      .text_color(t.foreground)
      .child(
        h_flex()
          .items_center()
          .child(
            div().flex_1().text_size(15).font_semibold().child("Herdr Remote"),
          )
          .child(div().text_size(11).text_color(t.dim).child("Esc")),
      )
      .when(Boolean(this.error), (el) =>
        el.child(div().text_color(t.blocked).text_size(12).child(this.error)),
      )
      .child(this.body(t));
  }
}
