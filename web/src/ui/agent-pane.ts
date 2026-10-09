import type { AgentChat, AgentMessage, AgentSegment, AgentState, Engine } from "../engine";
import { ApiError } from "../engine/api";
import { formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { reconcile, render } from "../extensions/blocks/ai";
import { markdownReady } from "../extensions/blocks/markdown";
import { byId, h, setAttr, setText } from "./dom";
import { type IconName, iconNode } from "./icons";
import { hasLayers } from "./layers";
import { type MenuEntry, openMenu } from "./menu";
import { toast } from "./toast";

interface Bubble {
  el: HTMLElement;
  body: HTMLElement;
  foot: HTMLElement;
  text: HTMLElement | null;
  raw: string;
  steps: HTMLElement[];
}

export interface AgentPane {
  toggle(): void;
  open(): void;
}

const SUGGESTIONS: [string, IconName][] = [
  ["Summarize this page", "text"],
  ["Make a task database with status, owner and due date", "database"],
  ["Find pages I edited this week", "search"],
];

export function mountAgentPane(engine: Engine, btn: HTMLButtonElement): AgentPane {
  const { state } = engine;
  const title = h("h2", { class: "apane__title", text: t("Agent") });
  const historyBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Chats"), "data-tip": t("Chats"), "aria-haspopup": "menu" } }, iconNode("clock"));
  const newBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("New chat"), "data-tip": t("New chat") } }, iconNode("plus"));
  const setBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Agent settings"), "data-tip": t("Agent settings"), "aria-expanded": "false" } }, iconNode("sliders"));
  const closeBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Close agent"), "data-tip": t("Close agent") } }, iconNode("close"));
  const settings = h("div", { class: "apane__settings" });
  settings.hidden = true;
  const list = h("div", { class: "apane__list", attrs: { role: "log", "aria-live": "polite", "aria-label": t("Conversation") } });
  const empty = h("div", { class: "apane__empty" }, h("span", { class: "apane__empty-icon" }, iconNode("bot")), h("p", { class: "apane__lead", text: t("Ask the agent to find, write or organize anything in this workspace. It works through the same tools as connected apps, and every change can be undone.") }));
  const chips = h("div", { class: "apane__chips" });
  for (const [label, icon] of SUGGESTIONS) {
    const c = h("button", { class: "chip chip--btn", attrs: { type: "button" } }, iconNode(icon), h("span", { text: t(label) }));
    c.addEventListener("click", () => {
      input.value = t(label);
      void send();
    });
    chips.append(c);
  }
  empty.append(chips);
  const notice = h("p", { class: "apane__notice", attrs: { role: "status" } });
  notice.hidden = true;
  const ctxChip = h("button", { class: "chip chip--btn apane__ctx", attrs: { type: "button", "aria-pressed": "true" } }, iconNode("page"), h("span"));
  const input = h("textarea", { class: "apane__input", attrs: { rows: "1", "aria-label": t("Message the agent"), placeholder: t("Ask or tell the agent anything") } });
  const sendBtn = h("button", { class: "icon-btn icon-btn--primary apane__send", attrs: { type: "submit", "aria-label": t("Send"), "data-tip": t("Send") } }, iconNode("arrow-up"));
  const stopBtn = h("button", { class: "icon-btn apane__send", attrs: { type: "button", "aria-label": t("Stop"), "data-tip": t("Stop") } }, iconNode("close"));
  stopBtn.hidden = true;
  const form = h("form", { class: "apane__composer" }, h("div", { class: "apane__tools" }, ctxChip), h("div", { class: "apane__row" }, input, sendBtn, stopBtn));
  const pane = h(
    "aside",
    { class: "apane", attrs: { "aria-label": t("Agent"), tabindex: "-1" } },
    h("header", { class: "apane__head" }, title, h("span", { class: "apane__actions" }, historyBtn, newBtn, setBtn, closeBtn)),
    settings,
    h("div", { class: "apane__scroll" }, empty, list),
    notice,
    form,
  );
  pane.hidden = true;
  byId("overlay-root").append(pane);
  const scroller = pane.querySelector(".apane__scroll") as HTMLElement;

  let open = false;
  let ws = "";
  let chat: AgentChat | null = null;
  let data: AgentState | null = null;
  let busy: AbortController | null = null;
  let useCtx = true;
  let ready = false;
  void markdownReady().then((ok) => {
    ready = ok;
  });

  const pageId = () => {
    const r = state.get("route");
    return r.name === "page" ? r.id : "";
  };
  const workspace = () => {
    const id = pageId();
    return (id && engine.pages.get(id)?.workspace_id) || state.get("workspaceId") || "";
  };

  const paintCtx = () => {
    const id = pageId();
    const p = id ? engine.pages.get(id) : undefined;
    ctxChip.hidden = !id;
    setText(ctxChip.querySelector("span") as HTMLElement, p?.title || t("This page"));
    setAttr(ctxChip, "aria-pressed", useCtx ? "true" : "false");
    setAttr(ctxChip, "data-tip", useCtx ? t("The agent sees this page. Click to leave it out.") : t("Click to let the agent see this page."));
  };

  const nearBottom = () => scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 80;
  const stick = (force = false) => {
    if (force || nearBottom()) scroller.scrollTop = scroller.scrollHeight;
  };

  const stepEl = (seg: AgentSegment, running = false): HTMLElement => {
    const icon = h("span", { class: "astep__icon" });
    const el = h("div", { class: "astep" }, icon, h("span", { class: "astep__label", text: seg.label ?? seg.tool ?? "" }), h("span", { class: "astep__summary", text: seg.summary ?? "" }));
    paintStep(el, seg, running);
    return el;
  };
  const paintStep = (el: HTMLElement, seg: AgentSegment, running: boolean) => {
    const icon = el.querySelector(".astep__icon") as HTMLElement;
    const want = running ? "running" : seg.ok ? "ok" : "error";
    if (el.dataset.state !== want) {
      el.dataset.state = want;
      icon.replaceChildren(running ? h("span", { class: "astep__dot" }) : iconNode(seg.ok ? "check" : "alert"));
    }
    setText(el.querySelector(".astep__summary") as HTMLElement, running ? "" : (seg.summary ?? ""));
  };

  const footFor = (b: Bubble, m: Pick<AgentMessage, "run_id" | "run_status" | "run_ops">) => {
    b.foot.replaceChildren();
    if (!m.run_id || !m.run_ops) return;
    const undone = m.run_status === "undone";
    const n = m.run_ops;
    const label = h("span", { class: "abubble__ops", text: n === 1 ? t("1 change") : t("{n} changes", { n }) });
    const undo = h("button", { class: "btn btn--ghost btn--sm", attrs: { type: "button" }, text: undone ? t("Undone") : t("Undo") });
    undo.disabled = undone;
    undo.addEventListener("click", async () => {
      undo.disabled = true;
      try {
        await engine.api.me.undoRun(m.run_id as string);
        setText(undo, t("Undone"));
        toast({ message: t("The agent's changes were undone."), tone: "success" });
      } catch (e) {
        undo.disabled = false;
        engine.reportError(e);
      }
    });
    b.foot.append(label, undo);
  };

  const bubble = (role: "user" | "assistant"): Bubble => {
    const body = h("div", { class: "abubble__body" });
    const foot = h("div", { class: "abubble__foot" });
    const el = h("div", { class: `abubble abubble--${role === "user" ? "user" : "agent"}` }, body, foot);
    list.append(el);
    return { el, body, foot, text: null, raw: "", steps: [] };
  };

  const addText = (b: Bubble, text: string) => {
    if (!b.text) {
      b.text = h("div", { class: "abubble__text doc-body" });
      b.raw = "";
      b.body.append(b.text);
    }
    b.raw += text;
    reconcile(b.text, render(b.raw, ready));
  };

  const showMessage = (m: AgentMessage) => {
    const b = bubble(m.role);
    if (m.role === "user") {
      b.body.append(h("div", { class: "abubble__plain", text: m.text }));
      return;
    }
    for (const seg of m.parts) {
      if (seg.type === "text" && seg.text) addText(b, seg.text);
      else if (seg.type === "tool") {
        b.text = null;
        b.body.append(stepEl(seg));
      }
    }
    footFor(b, m);
  };

  const paintEmpty = () => {
    empty.hidden = list.childElementCount > 0;
  };

  const loadChat = async (id: string | null) => {
    list.replaceChildren();
    chat = null;
    if (id) {
      const r = await engine.api.agent.chat(id);
      chat = r.chat;
      for (const m of r.messages) showMessage(m);
    }
    setText(title, chat?.title || t("Agent"));
    paintEmpty();
    stick(true);
  };

  const checkAI = async () => {
    try {
      const c = await engine.api.ai.models(ws);
      const on = c.enabled && c.models.some((m) => !m.audio);
      notice.hidden = on;
      setText(notice, on ? "" : c.models.length ? t("AI is turned off here. An admin can turn it on.") : t("No AI model is set up yet. An admin can add one under Admin settings, AI."));
      input.disabled = !on;
      sendBtn.disabled = !on;
    } catch {
      notice.hidden = true;
    }
  };

  const loadState = async () => {
    ws = workspace();
    if (!ws) return;
    data = await engine.api.agent.state(ws);
    void checkAI();
  };

  const send = async () => {
    const text = input.value.trim();
    if (!text || busy || !ws) return;
    input.value = "";
    fit();
    const abort = new AbortController();
    busy = abort;
    sendBtn.hidden = true;
    stopBtn.hidden = false;
    bubble("user").body.append(h("div", { class: "abubble__plain", text }));
    let agent: Bubble | null = bubble("assistant");
    agent.el.classList.add("is-thinking");
    paintEmpty();
    stick(true);
    let running: { el: HTMLElement; seg: AgentSegment } | null = null;
    const body: { text: string; page_id?: string } = { text };
    if (useCtx && pageId()) body.page_id = pageId();
    try {
      if (!chat) {
        chat = (await engine.api.agent.newChat(ws)).chat;
        data?.chats.unshift(chat);
      }
      await engine.api.agent.send(
        chat.id,
        body,
        (event, raw) => {
          const d = raw as Record<string, unknown>;
          if (event === "delta" && agent) {
            agent.el.classList.remove("is-thinking");
            addText(agent, String(d.t ?? ""));
            stick();
          } else if (event === "tool" && agent) {
            agent.el.classList.remove("is-thinking");
            agent.text = null;
            const seg: AgentSegment = { type: "tool", tool: String(d.tool ?? ""), label: String(d.label ?? "") };
            const el = stepEl(seg, true);
            agent.body.append(el);
            running = { el, seg };
            stick();
          } else if (event === "tool_done" && running) {
            running.seg.ok = d.ok === true;
            running.seg.summary = String(d.summary ?? "");
            paintStep(running.el, running.seg, false);
            running = null;
          } else if (event === "done" && agent) {
            const m = raw as AgentMessage;
            agent.el.classList.remove("is-thinking");
            if (!agent.body.childElementCount && m.text) addText(agent, m.text);
            footFor(agent, m);
            stick();
          }
        },
        abort.signal,
      );
    } catch (e) {
      if (!(e instanceof ApiError && e.code === "cancelled")) {
        const b = agent ?? bubble("assistant");
        b.el.classList.remove("is-thinking");
        b.body.append(h("p", { class: "abubble__error", text: e instanceof Error ? e.message : t("The agent stopped.") }));
      }
    } finally {
      if (busy === abort) busy = null;
      sendBtn.hidden = false;
      stopBtn.hidden = true;
      if (chat && !chat.title) {
        chat.title = text.slice(0, 80);
        setText(title, chat.title);
      }
      input.focus();
    }
  };

  const fit = () => {
    input.style.height = "auto";
    input.style.height = `${Math.min(input.scrollHeight, 160)}px`;
  };

  const paintSettings = () => {
    settings.replaceChildren();
    if (!data) return;
    const sel = h("select", { class: "select", attrs: { "aria-label": t("Instructions page") } });
    sel.append(h("option", { attrs: { value: "" }, text: t("None") }));
    const all = engine.pages.isLoaded(ws) ? engine.pages.all(ws).filter((p) => p.kind === "doc") : [];
    for (const p of all) sel.append(h("option", { attrs: { value: p.id }, text: p.title || t("Untitled") }));
    sel.value = data.prefs.instructions_page;
    const mem = h("input", { attrs: { type: "checkbox", role: "switch" } });
    mem.checked = data.prefs.memory;
    const save = async () => {
      if (!data) return;
      const next = { instructions_page: sel.value, memory: mem.checked };
      try {
        await engine.api.agent.setPrefs(ws, next);
        data.prefs = next;
      } catch (e) {
        engine.reportError(e);
        sel.value = data.prefs.instructions_page;
        mem.checked = data.prefs.memory;
      }
    };
    sel.addEventListener("change", () => void save());
    mem.addEventListener("change", () => void save());
    const memList = h("ul", { class: "apane__memory" });
    for (const m of data.memory) {
      const del = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Forget") } }, iconNode("close"));
      const li = h("li", {}, h("span", { text: m.text }), del);
      del.addEventListener("click", async () => {
        try {
          await engine.api.agent.forget(m.id);
          if (data) data.memory = data.memory.filter((x) => x.id !== m.id);
          li.remove();
        } catch (e) {
          engine.reportError(e);
        }
      });
      memList.append(li);
    }
    settings.append(
      h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Instructions page") }), sel, h("span", { class: "field__hint", text: t("The agent reads this page before every answer.") })),
      h("label", { class: "dpick__toggle" }, h("span", { text: t("Remember things I tell it") }), mem),
      memList,
    );
  };

  const setOpen = (next: boolean) => {
    if (open === next) return;
    open = next;
    pane.hidden = !open;
    setAttr(btn, "aria-pressed", open ? "true" : "false");
    if (open) {
      if (document.documentElement.dataset.comments === "open") engine.commands.run("page.comments");
      paintCtx();
      requestAnimationFrame(() => pane.classList.add("is-visible"));
      const w = workspace();
      if (w !== ws || !data) {
        void loadState()
          .then(() => loadChat(null))
          .catch(engine.reportError);
      }
      input.focus({ preventScroll: true });
    } else pane.classList.remove("is-visible");
  };

  closeBtn.addEventListener("click", () => setOpen(false));
  newBtn.addEventListener("click", () => {
    busy?.abort();
    void loadChat(null);
    input.focus();
  });
  historyBtn.addEventListener("click", () => {
    const entries: MenuEntry[] = [{ kind: "heading", label: t("Recent chats") }];
    for (const c of data?.chats ?? []) entries.push({ label: c.title || t("Untitled chat"), hint: formatRelative(c.updated_at), checked: c.id === chat?.id, run: () => void loadChat(c.id).catch(engine.reportError) });
    if (entries.length === 1) entries.push({ label: t("No chats yet"), run: () => undefined });
    openMenu(historyBtn, entries, { side: "bottom", align: "end", label: t("Chats") });
  });
  setBtn.addEventListener("click", () => {
    settings.hidden = !settings.hidden;
    setAttr(setBtn, "aria-expanded", settings.hidden ? "false" : "true");
    if (!settings.hidden) paintSettings();
  });
  ctxChip.addEventListener("click", () => {
    useCtx = !useCtx;
    paintCtx();
  });
  stopBtn.addEventListener("click", () => busy?.abort());
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    void send().catch(engine.reportError);
  });
  input.addEventListener("input", fit);
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
      e.preventDefault();
      void send().catch(engine.reportError);
    }
  });
  pane.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !hasLayers() && !e.defaultPrevented) {
      setOpen(false);
      btn.focus();
    }
  });
  state.watch("route", () => {
    if (!open) return;
    paintCtx();
    if (workspace() !== ws) void loadState().then(() => loadChat(null), engine.reportError);
  });

  return { toggle: () => setOpen(!open), open: () => setOpen(true) };
}
