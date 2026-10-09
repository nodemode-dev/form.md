import type { Engine, Webhook, WebhookDelivery, Workspace } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import type { Tab } from "./dialog";
import { confirmDialog, simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { actions, busy, button, copyText, field, iconButton, input, section, select, status } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";
import { boundTab, isAdmin } from "./workspace-settings";

function revealSecret(box: HTMLElement, secret: string, note: string): void {
  const copy = iconButton("copy", t("Copy secret"));
  copy.addEventListener("click", async () => toast({ id: "copy", message: (await copyText(secret)) ? t("Secret copied") : t("Could not copy") }));
  box.replaceChildren(h("span", { text: secret }), copy);
  box.hidden = false;
  box.title = note;
}

function hookState(x: Webhook): string {
  if (!x.active) return x.failures > 0 ? t("Paused after repeated failures") : t("Paused");
  if (x.last_status === "failed") return t("Last delivery failed");
  if (x.last_status === "delivered") return t("Delivering");
  return t("Waiting for the first event");
}

function openDeliveries(engine: Engine, x: Webhook): void {
  simpleDialog(
    t("Deliveries"),
    (body) => {
      const st = status();
      const list = h("div", { class: "rows" });
      const empty = h("p", { class: "muted", text: t("Nothing has been sent yet.") });
      empty.hidden = true;
      const row = () => h("div", { class: "row" }, h("span", { class: "row__icon" }), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })), h("div", { class: "row__actions" }));
      const update = (el: HTMLElement, d: WebhookDelivery) => {
        const icon = el.firstElementChild as HTMLElement;
        if (icon.dataset.status !== d.status) {
          icon.dataset.status = d.status;
          icon.replaceChildren(iconNode(d.status === "delivered" ? "success" : d.status === "failed" ? "danger" : "clock"));
        }
        const [title, meta] = el.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, d.event);
        const bits = [formatRelative(d.created_at), plural(d.attempts, "{count} attempt", "{count} attempts")];
        if (d.last_status) bits.push(`HTTP ${d.last_status}`);
        if (d.last_error && d.status !== "delivered") bits.push(d.last_error);
        if (d.status === "pending" && d.attempts > 0) bits.push(t("next try {when}", { when: formatRelative(d.next_at) }));
        setText(meta, bits.join(" · "));
        setAttr(meta, "title", formatDateTime(d.created_at));
        const side = el.querySelector(".row__actions") as HTMLElement;
        if (d.status === "failed" && !side.firstElementChild) {
          const retry = button(t("Retry"), "ghost");
          retry.addEventListener("click", async () => {
            try {
              await busy(retry, () => engine.api.webhooks.retry(d.id));
              retry.remove();
              setTimeout(load, 1500);
            } catch (e) {
              st.error(e);
            }
          });
          side.append(retry);
        }
      };
      const load = () => {
        engine.api.webhooks
          .deliveries(x.id)
          .then((r) => {
            reconcile(list, r.deliveries, (d) => d.id, row, update);
            empty.hidden = r.deliveries.length > 0;
          })
          .catch((e) => st.error(e));
      };
      body.append(h("p", { class: "section__lead", text: x.url }), list, empty, st.el);
      load();
    },
    undefined,
    "md",
  );
}

export function hookList(engine: Engine, w: Workspace, kind: Webhook["kind"]): { el: HTMLElement; load(): void } {
  const st = status();
  const list = h("div", { class: "rows" });
  const reveal = h("div", { class: "secret-box" });
  reveal.hidden = true;
  const row = (x: Webhook) => {
    const r = h("div", { class: "row" }, h("span", { class: "row__icon" }, iconNode(kind === "siem" ? "shield" : "link")), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })), h("div", { class: "row__actions" }));
    const side = r.lastElementChild as HTMLElement;
    const log = button(t("Deliveries"), "ghost");
    log.addEventListener("click", () => openDeliveries(engine, x));
    const test = iconButton("play", t("Send a test event"));
    test.addEventListener("click", async () => {
      try {
        await engine.api.webhooks.test(x.id);
        toast({ message: t("Test event queued.") });
        setTimeout(load, 2500);
      } catch (e) {
        st.error(e);
      }
    });
    const pause = iconButton("eye-off", t("Pause"));
    pause.addEventListener("click", async () => {
      try {
        await engine.api.webhooks.update(x.id, { active: !(r.dataset.active === "true") });
        load();
      } catch (e) {
        st.error(e);
      }
    });
    const rotate = iconButton("key", t("New signing secret"));
    rotate.addEventListener("click", async () => {
      const ok = await confirmDialog({ title: t("Replace the signing secret?"), body: t("The old secret stops working right away. Update the receiver with the new one."), confirm: t("Replace") });
      if (!ok) return;
      try {
        const res = await engine.api.webhooks.rotate(x.id);
        revealSecret(reveal, res.secret, t("Shown only once."));
        st.say(t("New secret shown below. It is shown only once."), "success");
      } catch (e) {
        st.error(e);
      }
    });
    const remove = iconButton("trash", t("Delete"));
    remove.addEventListener("click", async () => {
      const ok = await confirmDialog({ title: t("Delete this destination?"), body: t("Nothing more is sent to {url}.", { url: x.url }), confirm: t("Delete"), variant: "danger" });
      if (!ok) return;
      try {
        await engine.api.webhooks.remove(x.id);
        load();
      } catch (e) {
        st.error(e);
      }
    });
    side.append(log, test, pause, kind === "webhook" ? rotate : h("span"), remove);
    return r;
  };
  const update = (r: HTMLElement, x: Webhook) => {
    r.dataset.active = String(x.active);
    const [title, meta] = r.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
    setText(title, x.description || x.url);
    const bits = [x.description ? x.url : "", kind === "siem" ? (x.format === "splunk" ? t("Splunk HEC") : t("JSON")) : x.events.join(", "), hookState(x), x.last_at ? t("last {when}", { when: formatRelative(x.last_at) }) : ""];
    setText(meta, bits.filter(Boolean).join(" · "));
    const pause = r.querySelectorAll(".row__actions .icon-btn")[1] as HTMLButtonElement;
    const label = x.active ? t("Pause") : t("Resume");
    if (pause.getAttribute("aria-label") !== label) {
      setAttr(pause, "aria-label", label);
      setAttr(pause, "data-tip", label);
      pause.replaceChildren(iconNode(x.active ? "eye-off" : "eye"));
    }
  };
  const load = () => {
    engine.api.webhooks
      .list(w.id)
      .then((r) => reconcile(list, r.webhooks.filter((x) => x.kind === kind), (x) => x.id, row, update))
      .catch((e) => st.error(e));
  };

  const url = input({ type: "url", placeholder: kind === "siem" ? "https://splunk.example.com:8088/services/collector" : "https://example.com/hooks/form-md" });
  url.setAttribute("aria-label", t("Address"));
  const events = input({ placeholder: t("* or page.*, row.create") });
  const desc = input({ maxlength: 200, placeholder: t("What receives it (optional)") });
  const format = select<"json" | "splunk">([
    ["json", t("JSON, one event per request")],
    ["splunk", t("Splunk HTTP Event Collector")],
  ], "json");
  const authHeader = input({ type: "password", autocomplete: "off", placeholder: "Splunk 0000-0000" });
  const add = button(kind === "siem" ? t("Start streaming") : t("Add webhook"), "primary", "plus", "submit");
  const fst = status();
  const fields = kind === "siem"
    ? [field(t("Collector address"), url), field(t("Format"), format), field(t("Authorization header"), authHeader, t("Sent as is with every request, for example a Splunk token."))]
    : [field(t("Address"), url), field(t("Events"), events, t("Comma separated. page.* matches every page event; * matches everything.")), field(t("Description"), desc)];
  const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, ...fields, actions(add), fst.el);
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    fst.clear();
    try {
      const res = await busy(add, () =>
        engine.api.webhooks.create(w.id, kind === "siem"
          ? { kind, url: url.value.trim(), format: format.value as "json" | "splunk", auth_header: authHeader.value }
          : { kind, url: url.value.trim(), events: events.value.split(",").map((x) => x.trim()).filter(Boolean), description: desc.value }),
      );
      url.value = events.value = desc.value = authHeader.value = "";
      if (kind === "webhook") {
        revealSecret(reveal, res.secret, t("Shown only once."));
        fst.say(t("Added. Copy the signing secret now; it is shown only once."), "success");
      } else {
        fst.say(t("Streaming. Every audit event from now on is sent to the collector."), "success");
      }
      load();
    } catch (err) {
      fst.error(err);
    }
  });
  const el = h("div", { class: "form-grid" }, list, reveal, st.el, form);
  return { el, load };
}

export function webhooksTab(engine: Engine): Tab {
  return boundTab(
    engine,
    "webhooks",
    t("Webhooks"),
    "link",
    (panel, w) => {
      const sec = section(t("Webhooks"), t("form.md posts a signed JSON event to each address when pages, rows, comments, sharing or members change. Check the X-FormMD-Signature header against the signing secret. Failed deliveries are retried for about a day."));
      const hooks = hookList(engine, w, "webhook");
      sec.append(hooks.el);
      panel.append(sec);
      hooks.load();
      return hooks.load;
    },
    () => {
      const w = engine.workspace();
      return !!w && isAdmin(w);
    },
  );
}
