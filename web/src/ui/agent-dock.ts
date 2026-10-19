import "@xterm/xterm/css/xterm.css";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import type { Engine } from "../engine";
import { ApiError, request, stream } from "../engine/api";
import { formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { reconcile, render } from "../extensions/blocks/ai";
import { markdownReady } from "../extensions/blocks/markdown";
import { h, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { type MenuEntry, openMenu } from "./menu";
import { toast } from "./toast";

interface TermInfo { id: string; title: string; kind: "shell" | "claude"; alive: boolean }
interface Part { kind: "text" | "tool"; text?: string; id?: string; name?: string; input?: string; status?: string }
interface Msg { role: "user" | "assistant"; text?: string; note?: string; parts?: Part[]; error?: string; meta?: { seconds?: number; turns?: number; cost_usd?: number; input_tokens?: number; output_tokens?: number } }
interface ChatState { busy: boolean; session: string; messages: Msg[]; usage: { input: number; output: number; cost_usd: number; context: number; window: number; model: string } }
interface Harness { enabled: boolean; windows: boolean; folder: string; claude?: { installed: boolean; path?: string; version?: string } }

export interface AgentDock {
  toggle(): void;
  show(): Promise<void>;
  newTerminal(): Promise<void>;
  claude(): Promise<void>;
  chat(): Promise<void>;
  isOpen(): boolean;
}

const HEIGHT_KEY = "formmd.dock.h";
const HEIGHT_DEFAULT = 280;
const MAC = /Mac|iPhone|iPad/i.test(navigator.platform || navigator.userAgent);

function isToggle(e: KeyboardEvent): boolean {
  return (MAC ? e.metaKey : e.ctrlKey) && !e.altKey && !e.shiftKey && e.code === "KeyJ";
}

function tokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(n >= 100_000 ? 0 : 1)}k`;
  return String(n);
}

let viewSeq = 0;

class TermView {
  readonly pane: HTMLElement;
  private readonly scope = `.dock__term[data-view="${++viewSeq}"]`;
  readonly term: Terminal;
  private readonly fit = new FitAddon();
  private ws: WebSocket | null = null;
  private readonly queued: string[] = [];
  private opened = false;
  private readonly sheet = new CSSStyleSheet();
  private readonly styles: HTMLStyleElement[] = [];
  private readonly restyler = new MutationObserver(() => this.syncSheet());
  private readonly adopter = new MutationObserver((ms) => {
    for (const m of ms) for (const n of m.addedNodes) if (n instanceof HTMLStyleElement) this.adoptStyle(n);
  });
  constructor(readonly info: TermInfo, private readonly onExit: (v: TermView) => void) {
    this.pane = h("div", { class: "dock__term", attrs: { role: "tabpanel", "data-view": String(viewSeq) } });
    this.term = new Terminal({
      cursorBlink: true,
      fontFamily: monoFamily(),
      fontSize: 12,
      lineHeight: 1.2,
      scrollback: 5000,
      allowProposedApi: false,
      theme: themeOf(),
    });
    this.term.attachCustomKeyEventHandler((e) => !isToggle(e));
    this.term.loadAddon(this.fit);
    this.term.onData((data) => this.send({ type: "input", data }));
    this.term.onResize(({ cols, rows }) => this.send({ type: "resize", cols, rows }));
  }
  mount(): void {
    if (this.opened) return;
    this.opened = true;
    document.adoptedStyleSheets = [...document.adoptedStyleSheets, this.sheet];
    this.adopter.observe(this.pane, { childList: true, subtree: true });
    this.term.open(this.pane);
    for (const s of this.pane.querySelectorAll("style")) this.adoptStyle(s);
    this.resize();
    this.connect();
  }
  private adoptStyle(s: HTMLStyleElement): void {
    if (this.styles.includes(s)) return;
    this.styles.push(s);
    s.remove();
    this.restyler.observe(s, { childList: true, characterData: true, subtree: true });
    this.syncSheet();
  }
  private syncSheet(): void {
    const raw = new CSSStyleSheet();
    try {
      raw.replaceSync(this.styles.map((s) => s.textContent ?? "").join("\n"));
    } catch {
      return;
    }
    this.sheet.replaceSync([...raw.cssRules].map((r) => (r instanceof CSSStyleRule ? `${this.scope} { ${r.cssText} }` : r.cssText)).join("\n"));
  }
  connect(): void {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${proto}//${location.host}/api/terminal/${encodeURIComponent(this.info.id)}/attach`);
    ws.binaryType = "arraybuffer";
    ws.onopen = () => {
      this.send({ type: "resize", cols: this.term.cols, rows: this.term.rows });
      for (const raw of this.queued.splice(0)) ws.send(raw);
    };
    ws.onmessage = (m: MessageEvent<ArrayBuffer | string>) => {
      if (typeof m.data === "string") {
        try {
          if ((JSON.parse(m.data) as { type?: string }).type === "exit") {
            this.info.alive = false;
            this.term.write(`\r\n\x1b[2m${t("[process ended: restart or close this tab]")}\x1b[0m\r\n`);
            this.onExit(this);
          }
        } catch {
          return;
        }
        return;
      }
      this.term.write(new Uint8Array(m.data));
    };
    this.ws = ws;
  }
  send(msg: { type: string; data?: string; cols?: number; rows?: number }): void {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(msg));
    else if (msg.type === "input" && this.ws?.readyState === WebSocket.CONNECTING) this.queued.push(JSON.stringify(msg));
  }
  reattach(): void {
    if (!this.opened) {
      this.info.alive = true;
      this.mount();
      return;
    }
    const old = this.ws;
    if (old) {
      old.onmessage = null;
      old.close();
    }
    this.queued.length = 0;
    this.info.alive = true;
    this.term.reset();
    this.connect();
  }
  resize(): void {
    if (!this.opened || !this.pane.isConnected || this.pane.hidden) return;
    try {
      this.fit.fit();
    } catch {
      return;
    }
  }
  restyle(): void {
    this.term.options.theme = themeOf();
  }
  dispose(): void {
    this.ws?.close();
    this.adopter.disconnect();
    this.restyler.disconnect();
    this.term.dispose();
    this.pane.remove();
    document.adoptedStyleSheets = document.adoptedStyleSheets.filter((x) => x !== this.sheet);
  }
}

function monoFamily(): string {
  return getComputedStyle(document.documentElement).getPropertyValue("--font-mono").trim() || "JetBrains Mono, monospace";
}

function themeOf(): Record<string, string> {
  const cs = getComputedStyle(document.documentElement);
  const v = (name: string, fallback: string) => cs.getPropertyValue(name).trim() || fallback;
  return { background: v("--sheet", "#101012"), foreground: v("--text", "#e8e8ea"), cursor: v("--accent", "#f97316"), selectionBackground: v("--selection", "#f9731655") };
}

export function mountAgentDock(engine: Engine): AgentDock {
  const fetchFn: typeof fetch = (input, init) => fetch(input, init);
  const tabs = h("div", { class: "dock__tabs", attrs: { role: "tablist", "aria-label": t("Terminals") } });
  const btn = (icon: Parameters<typeof iconNode>[0], label: string, tip = label) =>
    h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": label, "data-tip": tip } }, iconNode(icon));
  const newBtn = btn("plus", t("New terminal"));
  const chatBtn = btn("agent-chat", t("Chat with Claude Code"));
  const claudeBtn = btn("agent", t("Claude Code in a terminal"));
  const restartBtn = btn("refresh", t("Restart this tab"));
  restartBtn.disabled = true;
  const closeBtn = btn("close", t("Hide the panel"), t("Hide the panel ({keys}); programs keep running", { keys: engine.keymap.chordFor("dock.toggle")?.join("+") ?? "" }));
  const grip = h("div", { class: "dock__grip", attrs: { role: "separator", "aria-orientation": "horizontal", "aria-label": t("Resize the panel") } });
  const head = h("div", { class: "dock__head" }, tabs, h("span", { class: "dock__spacer" }), newBtn, chatBtn, claudeBtn, restartBtn, closeBtn);
  const idle = h("p", { class: "dock__empty", text: t("No terminal is open. Start a shell with the plus button, Claude Code with the agent button, or Chat with the speech bubble.") });
  idle.hidden = true;
  const body = h("div", { class: "dock__body", attrs: { "data-own-keys": "dock.toggle" } }, idle);
  const askInput = h("input", { class: "input dock__ask-input", attrs: { type: "text", spellcheck: "false", autocomplete: "off", "aria-label": t("Ask Claude Code"), placeholder: t("Ask Claude Code to work on this page") } });
  const askSend = h("button", { class: "icon-btn", attrs: { type: "submit", "aria-label": t("Send to Claude Code"), "data-tip": t("Send to Claude Code") } }, iconNode("send"));
  const ask = h("form", { class: "dock__ask", attrs: { autocomplete: "off" } }, h("span", { class: "dock__ask-mark" }, iconNode("agent")), askInput, askSend);
  const root = h("section", { class: "dock", attrs: { "aria-label": t("Terminal and Claude Code") } }, grip, head, body, ask);
  root.hidden = true;
  document.body.appendChild(root);
  body.addEventListener("keydown", (e) => {
    if (!isToggle(e)) e.stopPropagation();
  });

  let stored = HEIGHT_DEFAULT;
  try {
    stored = Number(localStorage.getItem(HEIGHT_KEY)) || HEIGHT_DEFAULT;
  } catch {
    stored = HEIGHT_DEFAULT;
  }
  root.style.blockSize = `${Math.max(160, Math.min(stored, Math.round(innerHeight * 0.8)))}px`;

  const views = new Map<string, TermView>();
  const tabEls = new Map<string, HTMLButtonElement>();
  let active = "";
  let harness: Harness | null = null;

  const chat = buildChat();
  body.appendChild(chat.pane);
  chat.pane.hidden = true;
  const chatX = h("span", { class: "dock__tab-x", attrs: { role: "button", "aria-label": t("Close the Chat tab"), tabindex: "-1" } }, iconNode("close"));
  const chatTab = h("button", { class: "dock__tab is-chat", attrs: { type: "button", role: "tab", "aria-selected": "false" } }, iconNode("agent-chat"), h("span", { text: t("Chat") }), chatX);
  chatTab.hidden = true;
  chatTab.addEventListener("click", (e) => {
    if ((e.target as HTMLElement).closest(".dock__tab-x")) closeChat();
    else select("chat");
  });
  tabs.prepend(chatTab);

  const ro = new ResizeObserver(() => views.get(active)?.resize());
  ro.observe(body);
  new MutationObserver(() => {
    for (const v of views.values()) v.restyle();
  }).observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });

  function select(id: string): void {
    active = id;
    for (const [k, v] of views) {
      const on = k === id;
      if (v.pane.hidden === on) v.pane.hidden = !on;
      const tab = tabEls.get(k);
      if (tab) setAttr(tab, "aria-selected", on ? "true" : "false");
    }
    const chatOn = id === "chat";
    if (chat.pane.hidden === chatOn) chat.pane.hidden = !chatOn;
    setAttr(chatTab, "aria-selected", chatOn ? "true" : "false");
    const v = views.get(id);
    if (v) {
      v.mount();
      requestAnimationFrame(() => {
        v.resize();
        v.term.focus();
      });
    }
    if (chatOn) askInput.focus();
    syncChrome();
  }

  function syncChrome(): void {
    const none = views.size === 0 && chatTab.hidden;
    if (idle.hidden === none) idle.hidden = !none;
    const noRestart = !views.has(active);
    if (restartBtn.disabled !== noRestart) restartBtn.disabled = noRestart;
    const hint = active === "chat" ? t("Ask Claude about this page (Enter to send)") : t("Ask Claude Code to work on this page");
    if (askInput.placeholder !== hint) askInput.placeholder = hint;
  }

  function closeChat(): void {
    chatTab.hidden = true;
    chat.pane.hidden = true;
    if (active === "chat") {
      active = "";
      const next = [...views.keys()].at(-1);
      if (next) select(next);
    }
    syncChrome();
  }

  function addView(info: TermInfo): TermView {
    const v = new TermView(info, (x) => tabEls.get(x.info.id)?.classList.add("is-ended"));
    views.set(info.id, v);
    v.pane.hidden = true;
    body.appendChild(v.pane);
    const label = h("span", { text: info.kind === "claude" ? t("Claude Code") : info.title });
    const x = h("span", { class: "dock__tab-x", attrs: { role: "button", "aria-label": t("Close terminal"), tabindex: "-1" } }, iconNode("close"));
    const tab = h("button", { class: `dock__tab${info.kind === "claude" ? " is-claude" : ""}`, attrs: { type: "button", role: "tab", "aria-selected": "false" } }, iconNode(info.kind === "claude" ? "agent" : "terminal"), label, x);
    if (!info.alive) tab.classList.add("is-ended");
    tab.addEventListener("click", (e) => {
      if ((e.target as HTMLElement).closest(".dock__tab-x")) void closeTerm(info.id);
      else select(info.id);
    });
    tab.addEventListener("dblclick", () => void restart(info.id));
    tabs.appendChild(tab);
    tabEls.set(info.id, tab);
    return v;
  }

  async function closeTerm(id: string): Promise<void> {
    await request(fetchFn, "DELETE", `/api/terminal/${encodeURIComponent(id)}`).catch(fail);
    views.get(id)?.dispose();
    views.delete(id);
    tabEls.get(id)?.remove();
    tabEls.delete(id);
    if (active === id) {
      active = "";
      const next = [...views.keys()].at(-1);
      if (next) select(next);
      else if (!chatTab.hidden) select("chat");
    }
    syncChrome();
  }

  async function restart(id: string): Promise<void> {
    try {
      await request(fetchFn, "POST", `/api/terminal/${encodeURIComponent(id)}/restart`);
      const v = views.get(id);
      if (!v) return;
      v.reattach();
      tabEls.get(id)?.classList.remove("is-ended");
      if (active === id) v.term.focus();
    } catch (e) {
      fail(e);
    }
  }

  function fail(e: unknown): void {
    const message = e instanceof ApiError ? e.message : e instanceof Error ? e.message : String(e);
    toast({ id: "dock", tone: "danger", message });
  }

  async function ready(): Promise<Harness> {
    if (harness) return harness;
    const [h] = await Promise.all([request<Harness>(fetchFn, "GET", "/api/claude"), document.fonts.load(`13px ${monoFamily()}`).catch(() => [])]);
    harness = h;
    if (!harness.enabled) throw new Error(t("The terminal and Claude Code run only on a form.md on this computer, for the instance owner. Set FORMMD_TERMINAL=on to allow them on a server."));
    const list = await request<{ terminals: TermInfo[] }>(fetchFn, "GET", "/api/terminal");
    for (const info of list.terminals) if (!views.has(info.id)) addView(info);
    return harness;
  }

  function setOpen(on: boolean): void {
    if (root.hidden === !on) return;
    root.hidden = !on;
    document.documentElement.toggleAttribute("data-dock", on);
    if (on) requestAnimationFrame(() => views.get(active)?.resize());
  }

  function context(): string {
    const r = engine.state.get("route");
    if (r.name !== "page") return "form.md window: no page open";
    const p = engine.pages.get(r.id);
    return `form.md window: page id ${r.id}${p ? `, titled "${p.title || t("Untitled")}"` : ""}`;
  }

  function note(): string {
    const r = engine.state.get("route");
    if (r.name !== "page") return "";
    const p = engine.pages.get(r.id);
    return p ? t('with the page "{title}"', { title: p.title || t("Untitled") }) : "";
  }

  async function open(kind: "shell" | "claude", resume = "", prompt = ""): Promise<void> {
    await ready();
    if (kind === "claude" && !harness?.claude?.installed) {
      harness = null;
      const again = await ready();
      if (!again.claude?.installed) throw new Error(t("Claude Code is not installed: install it from claude.com/claude-code, run claude once in a terminal to sign in, then try again."));
    }
    const size = views.get(active)?.term;
    const info = await request<TermInfo>(fetchFn, "POST", "/api/terminal", { kind, cols: size?.cols ?? 120, rows: size?.rows ?? 30, resume, context: prompt });
    setOpen(true);
    addView(info);
    select(info.id);
  }

  async function askClaude(text: string): Promise<void> {
    const where = context();
    const line = where ? `${text} (${where})` : text;
    const claudes = [...views.values()].filter((v) => v.info.kind === "claude" && v.info.alive);
    const live = claudes.find((v) => v.info.id === active) ?? claudes.at(-1);
    if (!live) {
      await open("claude", "", line);
      return;
    }
    setOpen(true);
    select(live.info.id);
    live.send({ type: "input", data: line });
    setTimeout(() => live.send({ type: "input", data: "\r" }), 120);
  }

  async function sendAsk(): Promise<void> {
    const text = askInput.value.trim();
    if (!text) return;
    if (active === "chat" && chat.isBusy()) return;
    askInput.value = "";
    try {
      if (active === "chat") await chat.send(text);
      else await askClaude(text);
    } catch (e) {
      if (!askInput.value) askInput.value = text;
      fail(e);
    }
  }

  async function show(): Promise<void> {
    await ready();
    const first = active && (views.has(active) || active === "chat") ? active : [...views.keys()][0];
    if (!first) return open("shell");
    setOpen(true);
    select(first);
  }

  function buildChat() {
    const log = h("div", { class: "dchat__log", attrs: { role: "log", "aria-live": "polite", "aria-label": t("Conversation") } });
    const empty = h("p", { class: "dchat__empty", text: t("Ask Claude Code about this workspace: write a page, build a database, tidy a section. It works through form.md's tools, every change shows here and in the window, and agent changes can be undone.") });
    log.appendChild(empty);
    const status = h("span", { class: "dchat__status" });
    const meter = h("span", { class: "dchat__usage", attrs: { "data-tip": t("Memory: how much of Claude's working memory this conversation uses (it reads all of it again with each message). Reply: tokens written so far. Chat: everything sent and written in this conversation, and what it has cost.") } });
    const stop = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Stop") });
    stop.hidden = true;
    const adopt = h("button", { class: "btn btn--sm", attrs: { type: "button", "data-tip": t("Bring a conversation from a Claude Code tab into this chat") }, text: t("From Claude Code") });
    const handoff = h("button", { class: "btn btn--sm", attrs: { type: "button", "data-tip": t("Continue this conversation in a Claude Code terminal tab") }, text: t("Open in Claude Code") });
    const fresh = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("New chat") });
    const bar = h("div", { class: "dchat__bar" }, status, meter, h("span", { class: "dock__spacer" }), stop, adopt, handoff, fresh);
    const pane = h("div", { class: "dchat", attrs: { role: "tabpanel" } }, bar, log);
    let busy = false;
    let loaded = false;
    let mdReady = false;
    void markdownReady().then((ok) => {
      mdReady = ok;
    });
    let reply: HTMLElement | null = null;
    let textEl: HTMLElement | null = null;
    let raw = "";
    const chips = new Map<string, HTMLElement>();

    const scroll = () => {
      if (log.scrollHeight - log.scrollTop - log.clientHeight < 80) log.scrollTop = log.scrollHeight;
    };
    const setBusy = (on: boolean) => {
      busy = on;
      if (stop.hidden === on) stop.hidden = !on;
      setText(status, on ? t("Claude is working...") : "");
    };
    const user = (text: string, note = "") => {
      if (!empty.hidden) empty.hidden = true;
      const n = h("div", { class: "dchat__msg is-user" }, h("span", { text }));
      if (note) n.appendChild(h("span", { class: "dchat__note", text: note }));
      log.appendChild(n);
      scroll();
    };
    const begin = () => {
      if (!empty.hidden) empty.hidden = true;
      reply = h("div", { class: "dchat__msg is-claude" });
      textEl = null;
      raw = "";
      log.appendChild(reply);
      scroll();
    };
    const delta = (text: string) => {
      if (!reply) begin();
      if (!textEl) {
        textEl = h("div", { class: "dchat__text md" });
        raw = "";
        reply!.appendChild(textEl);
      }
      raw += text;
      reconcile(textEl, render(raw, mdReady));
      scroll();
    };
    const tool = (p: Part) => {
      if (!reply) begin();
      textEl = null;
      const chip = h("div", { class: `dchat__tool is-${p.status ?? "running"}`, attrs: { title: p.input ?? "" } }, h("span", { class: "dchat__tool-mark" }, iconNode("terminal")), h("span", { class: "dchat__tool-name", text: p.name ?? "" }), h("span", { class: "dchat__tool-input", text: p.input ?? "" }));
      if (p.id) chips.set(p.id, chip);
      reply!.appendChild(chip);
      scroll();
    };
    const setChip = (id: string, s: string) => {
      const c = chips.get(id);
      if (!c) return;
      c.classList.remove("is-running", "is-done", "is-error");
      c.classList.add(`is-${s}`);
    };
    const close = (error: string, meta: Msg["meta"]) => {
      if (!reply) begin();
      for (const c of reply!.querySelectorAll(".dchat__tool.is-running")) {
        c.classList.remove("is-running");
        c.classList.add("is-done");
      }
      if (error) reply!.appendChild(h("div", { class: "dchat__error", text: error }));
      if (meta && meta.seconds !== undefined) {
        const used = (meta.input_tokens ?? 0) + (meta.output_tokens ?? 0);
        reply!.appendChild(h("div", { class: "dchat__meta", text: `${meta.seconds} s${meta.turns ? ` · ${meta.turns} ${t("steps")}` : ""}${used ? ` · ${tokens(used)} ${t("tokens")}` : ""}${meta.cost_usd ? ` · $${meta.cost_usd.toFixed(3)}` : ""}` }));
      }
      reply = null;
      textEl = null;
      setBusy(false);
      scroll();
    };
    let totals: ChatState["usage"] = { input: 0, output: 0, cost_usd: 0, context: 0, window: 0, model: "" };
    const usage = (u: Partial<ChatState["usage"]>, replyTokens = -1) => {
      totals = { ...totals, ...u };
      const parts: string[] = [];
      if (totals.context) parts.push(`${t("Memory")} ${tokens(totals.context)}${totals.window ? ` / ${tokens(totals.window)}` : ""}`);
      if (replyTokens >= 0) parts.push(`${t("Reply")} ${tokens(replyTokens)} ${t("tokens")}`);
      if (totals.input || totals.output) parts.push(`${t("Chat")} ${tokens(totals.input + totals.output)} ${t("tokens")}${totals.cost_usd ? ` · $${totals.cost_usd.toFixed(totals.cost_usd < 1 ? 3 : 2)}` : ""}`);
      setText(meter, parts.join(" · "));
      meter.classList.toggle("is-high", totals.window > 0 && totals.context / totals.window > 0.8);
    };
    const clear = () => {
      if (log.firstElementChild !== empty || log.childElementCount > 1) log.replaceChildren(empty);
      if (empty.hidden) empty.hidden = false;
      chips.clear();
      reply = null;
      textEl = null;
      totals = { input: 0, output: 0, cost_usd: 0, context: 0, window: 0, model: "" };
      usage({});
    };
    const show = (s: ChatState) => {
      clear();
      for (const m of s.messages) {
        if (m.role === "user") {
          user(m.text ?? "", m.note ?? "");
          continue;
        }
        begin();
        for (const p of m.parts ?? []) {
          if (p.kind === "text") delta(p.text ?? "");
          else tool(p);
        }
        close(m.error ?? "", m.meta);
      }
      usage(s.usage);
      setBusy(s.busy);
    };
    const load = async () => {
      if (loaded) return;
      loaded = true;
      try {
        show(await request<ChatState>(fetchFn, "GET", "/api/claude/chat"));
      } catch (e) {
        loaded = false;
        fail(e);
      }
    };
    const send = async (text: string) => {
      if (busy) return;
      setBusy(true);
      try {
        await stream(fetchFn, "/api/claude/chat", { text, context: context(), note: note() }, ({ event, data }) => {
          const d = data as Record<string, unknown>;
          if (event === "start") {
            user(String(d.text ?? ""), String(d.note ?? ""));
            begin();
          } else if (event === "delta") delta(String(d.text ?? ""));
          else if (event === "tool") tool({ kind: "tool", id: String(d.id ?? ""), name: String(d.name ?? ""), input: String(d.input ?? ""), status: "running" });
          else if (event === "toolDone") setChip(String(d.id ?? ""), String(d.status ?? "done"));
          else if (event === "usage") usage({ context: Number(d.context ?? 0), window: Number(d.window ?? 0) || totals.window }, Number(d.output ?? 0));
          else if (event === "done") {
            close(String(d.error ?? ""), d.meta as Msg["meta"]);
            if (d.usage) usage(d.usage as ChatState["usage"]);
          }
        });
      } catch (e) {
        if (busy) close(e instanceof Error ? e.message : String(e), undefined);
      }
    };
    stop.addEventListener("click", () => void request(fetchFn, "POST", "/api/claude/chat/cancel").catch(fail));
    fresh.addEventListener("click", () => void request(fetchFn, "POST", "/api/claude/chat/reset").then(() => (clear(), setBusy(false)), fail));
    handoff.addEventListener("click", () => {
      void (async () => {
        const s = await request<ChatState>(fetchFn, "GET", "/api/claude/chat");
        if (!s.session) throw new Error(t("Start a conversation here first; then it can continue in Claude Code"));
        if (s.busy) throw new Error(t("Wait for Claude to finish, or press Stop, before moving the conversation"));
        await open("claude", s.session);
      })().catch(fail);
    });
    adopt.addEventListener("click", () => {
      void (async () => {
        const r = await request<{ sessions: { id: string; title: string; modified: number; current: boolean }[] }>(fetchFn, "GET", "/api/claude/sessions");
        if (!r.sessions.length) throw new Error(t("No Claude Code conversation in form.md yet: open one with the Claude Code button first"));
        const entries: MenuEntry[] = [{ kind: "heading", label: t("Continue from Claude Code") }];
        for (const s of r.sessions) {
          entries.push({
            label: s.title || t("Untitled chat"),
            hint: formatRelative(s.modified * 1000),
            checked: s.current,
            run: () => void request<ChatState>(fetchFn, "POST", "/api/claude/adopt", { session: s.id }).then(show, fail),
          });
        }
        openMenu(adopt, entries, { side: "top", align: "end", label: t("Continue from Claude Code") });
      })().catch(fail);
    });
    return { pane, load, send, isBusy: () => busy };
  }

  const startDrag = (e: PointerEvent) => {
    e.preventDefault();
    grip.classList.add("is-dragging");
    const start = e.clientY;
    const base = root.getBoundingClientRect().height;
    const move = (m: PointerEvent) => {
      const next = Math.max(160, Math.min(Math.round(innerHeight * 0.85), base + (start - m.clientY)));
      root.style.blockSize = `${next}px`;
    };
    const up = () => {
      grip.classList.remove("is-dragging");
      removeEventListener("pointermove", move);
      removeEventListener("pointerup", up);
      try {
        localStorage.setItem(HEIGHT_KEY, String(Math.round(root.getBoundingClientRect().height)));
      } catch {
        return;
      }
    };
    addEventListener("pointermove", move);
    addEventListener("pointerup", up);
  };
  grip.addEventListener("pointerdown", startDrag);

  const api: AgentDock = {
    isOpen: () => !root.hidden,
    toggle: () => {
      if (root.hidden) void show().catch(fail);
      else setOpen(false);
    },
    show,
    newTerminal: async () => {
      await open("shell");
    },
    claude: async () => {
      await open("claude");
    },
    chat: async () => {
      await ready();
      await chat.load();
      setOpen(true);
      if (chatTab.hidden) {
        tabs.appendChild(chatTab);
        chatTab.hidden = false;
      }
      select("chat");
    },
  };
  newBtn.addEventListener("click", () => void api.newTerminal().catch(fail));
  claudeBtn.addEventListener("click", () => void api.claude().catch(fail));
  chatBtn.addEventListener("click", () => void api.chat().catch(fail));
  restartBtn.addEventListener("click", () => void restart(active));
  closeBtn.addEventListener("click", () => setOpen(false));
  ask.addEventListener("submit", (e) => {
    e.preventDefault();
    void sendAsk();
  });
  return api;
}
