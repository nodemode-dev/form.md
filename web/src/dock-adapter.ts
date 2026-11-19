import "./dock/dock.css";
import type { DockAdapter, DockAgentStatus, DockChannel, DockChatEngine, DockChatEvent, DockChatHandlers, DockErrorKind, DockKind, DockLegacy, DockProgramStatus, DockTerminal } from "./dock/adapter";
import { Dock } from "./dock/dock";
import type { Engine } from "./engine";
import { ApiError, request, stream } from "./engine/api";
import { formatRelative } from "./engine/format";
import { t } from "./engine/i18n";
import { reconcile, render } from "./extensions/blocks/ai";
import { markdownReady } from "./extensions/blocks/markdown";
import { type MenuEntry, openMenu } from "./ui/menu";

export const DOCK_KEY = "formmd.dock";
const OLD_HEIGHT_KEY = "formmd.dock.h";
const KINDS: DockKind[] = ["shell", "claude", "codex", "cursor", "chat"];
const OFF = "The terminal and Claude Code run only on a form.md on this computer, for the instance owner. Set FORMMD_TERMINAL=on to allow them on a server.";

interface TermInfo { id: string; title: string; kind: string; alive: boolean; cwd?: string }
interface Part { kind: "text" | "tool"; text?: string; id?: string; name?: string; input?: string; status?: string }
interface Meta { seconds?: number; turns?: number; cost_usd?: number; input_tokens?: number; output_tokens?: number }
interface Msg { role: "user" | "assistant"; text?: string; note?: string; parts?: Part[]; error?: string; meta?: Meta }
interface Usage { input: number; output: number; cost_usd: number; context: number; window: number; model: string }
interface ChatState { busy: boolean; session: string; messages: Msg[]; live?: Msg; usage: Usage }
interface Availability { kinds: string[]; code: string; reason: string }
interface Pane { sheet: CSSStyleSheet; styles: HTMLStyleElement[]; scope: string; watch: MutationObserver }

export interface AgentDock {
  toggle(): void;
  newTerminal(): Promise<void>;
  claude(): Promise<void>;
  chat(): Promise<void>;
  adopt(): Promise<void>;
  isOpen(): boolean;
}

const NO_USAGE: Usage = { input: 0, output: 0, cost_usd: 0, context: 0, window: 0, model: "" };

function tokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(n >= 100_000 ? 0 : 1)}k`;
  return String(n);
}

function monoFamily(): string {
  return getComputedStyle(document.documentElement).getPropertyValue("--font-mono").trim() || "JetBrains Mono, monospace";
}

function record(info: TermInfo): DockTerminal {
  const kind = info.kind === "claude" || info.kind === "codex" || info.kind === "cursor" ? info.kind : "shell";
  return { id: info.id, title: kind === "claude" ? t("Claude Code") : info.title, alive: info.alive, kind, ...(info.cwd ? { tip: info.cwd } : {}) };
}

function errorKind(err: unknown): DockErrorKind | null {
  if (!(err instanceof ApiError)) return null;
  if (err.status === 0) return "unreachable";
  if (err.code === "claude_missing" || err.code === "agent_missing") return "notInstalled";
  if (err.code === "busy") return "busy";
  if (err.code === "terminal_off" || err.code === "owner_only" || err.code === "session_only" || err.status === 401 || err.status === 403) return "refused";
  return null;
}

function channel(id: string): DockChannel {
  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const ws = new WebSocket(`${proto}//${location.host}/api/terminal/${encodeURIComponent(id)}/attach`);
  ws.binaryType = "arraybuffer";
  const queued: string[] = [];
  let data: (d: string | Uint8Array) => void = () => undefined;
  let exit: () => void = () => undefined;
  let closed: () => void = () => undefined;
  const push = (raw: string): void => {
    if (ws.readyState === WebSocket.OPEN) ws.send(raw);
    else if (ws.readyState === WebSocket.CONNECTING) queued.push(raw);
  };
  ws.onopen = () => {
    data(new Uint8Array(0));
    for (const raw of queued.splice(0)) ws.send(raw);
  };
  ws.onmessage = (m: MessageEvent<ArrayBuffer | string>) => {
    if (typeof m.data !== "string") {
      data(new Uint8Array(m.data));
      return;
    }
    try {
      if ((JSON.parse(m.data) as { type?: string }).type === "exit") exit();
    } catch {
      return;
    }
  };
  ws.onclose = () => closed();
  return {
    send: (text) => push(JSON.stringify({ type: "input", data: text })),
    resize: (cols, rows) => push(JSON.stringify({ type: "resize", cols, rows })),
    onData: (handler) => {
      data = handler;
    },
    onExit: (handler) => {
      exit = handler;
    },
    onClose: (handler) => {
      closed = handler;
    },
    close: () => {
      ws.onclose = null;
      ws.onmessage = null;
      ws.close();
    },
  };
}

function adoptTerminalStyles(root: HTMLElement): void {
  let seq = 0;
  const panes = new Map<HTMLElement, Pane>();
  const sync = (p: Pane): void => {
    const raw = new CSSStyleSheet();
    try {
      raw.replaceSync(p.styles.map((s) => s.textContent ?? "").join("\n"));
    } catch {
      return;
    }
    p.sheet.replaceSync([...raw.cssRules].map((r) => (r instanceof CSSStyleRule ? `${p.scope} { ${r.cssText} }` : r.cssText)).join("\n"));
  };
  const take = (style: HTMLStyleElement): void => {
    const host = style.closest<HTMLElement>(".ft-term");
    if (!host) return;
    let p = panes.get(host);
    if (!p) {
      const id = String(++seq);
      host.setAttribute("data-fm-term", id);
      const made: Pane = { sheet: new CSSStyleSheet(), styles: [], scope: `.ft-term[data-fm-term="${id}"]`, watch: new MutationObserver(() => sync(made)) };
      document.adoptedStyleSheets = [...document.adoptedStyleSheets, made.sheet];
      panes.set(host, made);
      p = made;
    }
    if (p.styles.includes(style)) return;
    p.styles.push(style);
    style.remove();
    p.watch.observe(style, { childList: true, characterData: true, subtree: true });
    sync(p);
  };
  new MutationObserver((records) => {
    for (const m of records) {
      for (const n of m.addedNodes) {
        if (n instanceof HTMLStyleElement) take(n);
        else if (n instanceof HTMLElement) for (const s of n.querySelectorAll<HTMLStyleElement>("style")) take(s);
      }
      for (const n of m.removedNodes) {
        const p = n instanceof HTMLElement ? panes.get(n) : undefined;
        if (!p || !(n instanceof HTMLElement)) continue;
        p.watch.disconnect();
        document.adoptedStyleSheets = document.adoptedStyleSheets.filter((x) => x !== p.sheet);
        panes.delete(n);
      }
    }
  }).observe(root, { childList: true, subtree: true });
}

function closeReserved(): void {
  const html = document.documentElement;
  if (html.getAttribute("data-ft-dock") === "open") html.setAttribute("data-ft-dock", "closed");
}

export async function mountDock(engine: Engine): Promise<AgentDock> {
  const fetchFn: typeof fetch = (input, init) => fetch(input, init);
  let avail: Availability;
  try {
    avail = await request<Availability>(fetchFn, "GET", "/api/dock");
  } catch (e) {
    closeReserved();
    throw e;
  }
  const kinds = KINDS.filter((k) => avail.kinds.includes(k));
  if (!kinds.length) {
    closeReserved();
    throw new Error(t(OFF));
  }
  const host = document.getElementById("app");
  if (!host) throw new Error("form.md: no #app");

  let dock: Dock | null = null;
  let fonts: Promise<unknown> | null = null;
  const ready = (): Promise<unknown> => {
    fonts ??= document.fonts.load(`13px ${monoFamily()}`).catch(() => []);
    return fonts;
  };
  const programs = async (): Promise<DockAgentStatus | null> => {
    try {
      const r = await request<Partial<Record<"claude" | "codex" | "cursor", DockProgramStatus>>>(fetchFn, "GET", "/api/dock/agents");
      const out: DockAgentStatus = {};
      for (const name of ["claude", "codex", "cursor"] as const) {
        const p = r[name];
        if (p && kinds.includes(name)) out[name] = { installed: p.installed === true, version: p.version ?? null, install: p.install ?? "", signedIn: p.signedIn ?? null };
      }
      return out;
    } catch {
      return null;
    }
  };

  const context = (): string => {
    const r = engine.state.get("route");
    if (r.name !== "page") return "form.md window: no page open";
    const p = engine.pages.get(r.id);
    return `form.md window: page id ${r.id}${p ? `, titled "${p.title || t("Untitled")}"` : ""}`;
  };
  const note = (): string => {
    const r = engine.state.get("route");
    if (r.name !== "page") return "";
    const p = engine.pages.get(r.id);
    return p ? t('with the page "{title}"', { title: p.title || t("Untitled") }) : "";
  };

  let handlers: DockChatHandlers | null = null;
  let seq = 0;
  let block: string | null = null;
  let totals: Usage = { ...NO_USAGE };
  let mdReady = false;
  const running = new Set<string>();
  void markdownReady().then((ok) => {
    mdReady = ok;
  });
  const emit = (e: DockChatEvent): void => handlers?.event(e);
  const meter = (u: Partial<Usage>, reply = -1): void => {
    totals = { ...totals, ...u };
    const parts: string[] = [];
    if (totals.context) parts.push(`${t("Memory")} ${tokens(totals.context)}${totals.window ? ` / ${tokens(totals.window)}` : ""}`);
    if (reply >= 0) parts.push(`${t("Reply")} ${tokens(reply)} ${t("tokens")}`);
    if (totals.input || totals.output) parts.push(`${t("Chat")} ${tokens(totals.input + totals.output)} ${t("tokens")}${totals.cost_usd ? ` · $${totals.cost_usd.toFixed(totals.cost_usd < 1 ? 3 : 2)}` : ""}`);
    emit({ type: "meter", text: parts.join(" · "), high: totals.window > 0 && totals.context / totals.window > 0.8 });
  };
  const delta = (text: string): void => {
    if (!text) return;
    if (!block) {
      block = `b${++seq}`;
      emit({ type: "block", id: block, kind: "text" });
    }
    emit({ type: "delta", id: block, text });
  };
  const tool = (id: string, name: string, input: string): void => {
    block = null;
    running.add(id);
    emit({ type: "tool", id, name, summary: input });
  };
  const toolDone = (id: string, error: boolean): void => {
    if (!running.delete(id)) return;
    emit({ type: "tool_result", id, text: "", error });
  };
  const part = (p: Part): void => {
    if (p.kind === "text") {
      delta(p.text ?? "");
      return;
    }
    const id = p.id || `t${++seq}`;
    tool(id, p.name ?? "", p.input ?? "");
    if (p.status && p.status !== "running") toolDone(id, p.status === "error");
  };
  const finish = (error: string, meta: Meta | undefined): void => {
    block = null;
    for (const id of [...running]) toolDone(id, false);
    let line = "";
    if (meta && meta.seconds !== undefined) {
      const used = (meta.input_tokens ?? 0) + (meta.output_tokens ?? 0);
      line = `${meta.seconds} s${meta.turns ? ` · ${meta.turns} ${t("steps")}` : ""}${used ? ` · ${tokens(used)} ${t("tokens")}` : ""}${meta.cost_usd ? ` · $${meta.cost_usd.toFixed(3)}` : ""}`;
    }
    emit({ type: "result", ok: !error, text: error, ...(line ? { meta: line } : {}) });
  };
  const user = (text: string, with_: string): void => {
    block = null;
    emit({ type: "user", text, ...(with_ ? { note: with_ } : {}) });
  };
  const state = (busy: boolean, session: string | null): void => emit({ type: "state", running: true, state: busy ? "busy" : "idle", session });
  const replay = (s: ChatState): void => {
    emit({ type: "reset" });
    block = null;
    running.clear();
    totals = { ...NO_USAGE };
    for (const m of s.messages ?? []) {
      if (m.role === "user") {
        user(m.text ?? "", m.note ?? "");
        continue;
      }
      block = null;
      for (const p of m.parts ?? []) part(p);
      finish(m.error ?? "", m.meta);
    }
    if (s.live) {
      block = null;
      for (const p of s.live.parts ?? []) part(p);
    }
    meter(s.usage ?? {});
    state(s.busy, s.session || null);
  };

  const chat: DockChatEngine = {
    modes: [],
    status: async () => {
      try {
        const [agents, now] = await Promise.all([programs(), request<ChatState>(fetchFn, "GET", "/api/claude/chat")]);
        const c = agents?.claude;
        return {
          installed: c ? c.installed : true,
          version: c?.version ?? null,
          signedIn: c?.signedIn ?? null,
          install: c?.install ?? "",
          running: true,
          state: now.busy ? "busy" : "idle",
          session: now.session || null,
          lead: t("Ask Claude Code about this workspace: write a page, build a database, tidy a section. It works through form.md's tools, every change shows here and in the window, and agent changes can be undone."),
        };
      } catch (e) {
        if (e instanceof ApiError && e.status > 0) return { installed: true, version: null, signedIn: null, install: "", running: false, state: "idle", session: null, refused: e.message };
        return null;
      }
    },
    send: (text) =>
      new Promise<void>((resolve, reject) => {
        let began = false;
        const go = (): void => {
          if (began) return;
          began = true;
          resolve();
        };
        stream(fetchFn, "/api/claude/chat", { text, context: context(), note: note() }, go).then(go, (e: unknown) => {
          if (!began) reject(e instanceof Error ? e : new Error(String(e)));
        });
      }),
    subscribe: (h) => {
      handlers = h;
      h.open();
      const source = new EventSource("/api/claude/chat/events");
      const on = (name: string, fn: (d: Record<string, unknown>) => void): void => {
        source.addEventListener(name, (ev) => {
          if (handlers !== h) return;
          try {
            fn(JSON.parse((ev as MessageEvent<string>).data) as Record<string, unknown>);
          } catch {
            return;
          }
        });
      };
      on("snapshot", (d) => replay(d as unknown as ChatState));
      on("start", (d) => {
        user(String(d.text ?? ""), String(d.note ?? ""));
        state(true, null);
      });
      on("delta", (d) => delta(String(d.text ?? "")));
      on("tool", (d) => tool(String(d.id ?? `t${++seq}`), String(d.name ?? ""), String(d.input ?? "")));
      on("toolDone", (d) => toolDone(String(d.id ?? ""), d.status === "error"));
      on("usage", (d) => meter({ context: Number(d.context ?? 0), window: Number(d.window ?? 0) || totals.window }, Number(d.output ?? 0)));
      on("done", (d) => {
        finish(String(d.error ?? ""), d.meta as Meta | undefined);
        if (d.usage) meter(d.usage as Usage);
        state(false, null);
      });
      source.onerror = () => {
        if (source.readyState !== EventSource.CLOSED || handlers !== h) return;
        handlers = null;
        h.close();
      };
      return () => {
        if (handlers === h) handlers = null;
        source.close();
      };
    },
    start: async (_mode, fresh) => {
      if (!fresh) return;
      await request(fetchFn, "POST", "/api/claude/chat/reset");
      block = null;
      running.clear();
      totals = { ...NO_USAGE };
      emit({ type: "meter", text: "" });
      state(false, null);
    },
    stop: async () => {
      await request(fetchFn, "POST", "/api/claude/chat/cancel");
    },
    renderText: (node, text) => {
      if (!node.classList.contains("fm-chat-md")) node.classList.add("md", "fm-chat-md");
      reconcile(node, render(text, mdReady));
    },
  };

  const legacy: DockLegacy = {
    keys: [OLD_HEIGHT_KEY],
    read: (store) => {
      const h = Number(store.getItem(OLD_HEIGHT_KEY));
      return h > 0 ? { h } : null;
    },
  };

  const adapter: DockAdapter = {
    words: { app: "form.md", noun: "page", lead: "Ask Claude Code about this workspace.", example: "add a table of this week's tasks" },
    strings: {
      terminal: "Terminal and Claude Code",
      resize: "Resize the panel",
      newTerminal: "New terminal",
      chatAbout: "Chat with Claude Code",
      chatWith: "Chat with Claude Code",
      claudeCode: "Claude Code in a terminal",
      hide: "Hide the panel",
      hideTitle: t("Hide the panel ({keys}); programs keep running", { keys: engine.keymap.chordFor("dock.toggle")?.join("+") ?? "" }),
      toRight: "Move the panel to the right",
      toBottom: "Move the panel to the bottom",
      askPlaceholder: "Ask Claude Code to work on this page",
      chatPlaceholder: "Ask Claude about this page (Enter to send)",
      closeTab: "Close terminal",
      handoff: "Open in Claude Code",
      newChat: "New chat",
      outsideAgents: "Connected apps",
      noAnswer: "This terminal did not connect. Close this tab and open it again.",
      offline: "form.md did not answer. Check that it is still running, then try again.",
      oldEngine: "This form.md cannot run Claude Code in a tab.",
      programMissing: "{program} is not installed on the computer form.md runs on yet: install it there with the command below, then sign in once in a terminal.",
      tourPanelTitle: "This is the terminal panel",
      tourPanelBody: "This panel holds terminals, Claude Code and a chat with Claude Code, all on the computer form.md runs on. It stays out of the way until you open it.",
      tourMissingTitle: "Claude Code is not installed yet",
      tourMissingBody: "For the bigger jobs, form.md uses a program called Claude Code. To add it, copy the line below, run it in a terminal on the computer form.md runs on, then run claude once to sign in.",
    },
    placement: {
      mode: "own",
      storage: { key: DOCK_KEY, legacy },
      sizes: { minHeight: 160, minWidth: 320, defaultHeight: 280, defaultWidth: 420, maxHeightShare: 0.85, maxWidthShare: 0.6 },
      canDockRight: () => window.innerWidth >= 768,
    },
    isDockChord: () => false,
    chordLabel: () => engine.keymap.chordFor("dock.toggle")?.join("+") ?? "",
    tabs: () => ({ builtin: kinds }),
    classes: { iconButton: "icon-btn", button: "btn btn--sm", primaryButton: "btn--primary", input: "input", select: "select", note: "muted" },
    t: (_key, english) => t(english),
    onTheme: (handler) => {
      new MutationObserver(handler).observe(document.documentElement, { attributes: true, attributeFilter: ["data-dark-theme", "data-light-theme", "data-oled", "data-contrast", "data-accent"] });
    },
    keyHeaders: () => ({}),
    errors: { message: (err) => (err instanceof ApiError ? err.message : null), kind: errorKind },
    terminal: {
      list: async () => {
        await ready();
        const r = await request<{ terminals: TermInfo[] }>(fetchFn, "GET", "/api/terminal");
        return r.terminals.map(record);
      },
      open: async (kind, o) => {
        await ready();
        if (kind === "claude" && o.prompt) {
          const tab = dock?.activeKind() === "claude" ? dock.activeTab() : null;
          return record(await request<TermInfo>(fetchFn, "POST", "/api/claude/open", { prompt: `${o.prompt} (${context()})`, cols: o.cols, rows: o.rows, ...(tab ? { tab } : {}) }));
        }
        return record(await request<TermInfo>(fetchFn, "POST", "/api/terminal", { kind, cols: o.cols, rows: o.rows, ...(o.resume ? { resume: o.resume } : {}) }));
      },
      restart: async (id) => {
        await request(fetchFn, "POST", `/api/terminal/${encodeURIComponent(id)}/restart`);
      },
      close: async (id) => {
        await request(fetchFn, "DELETE", `/api/terminal/${encodeURIComponent(id)}`);
      },
      channel,
      scrollback: 5000,
    },
    agentStatus: programs,
    ...(kinds.includes("chat") ? { chat } : {}),
    openSettings: () => {
      engine.commands.run("account.open");
    },
  };

  const made = new Dock(adapter);
  dock = made;
  made.root.setAttribute("data-own-keys", "dock.toggle");
  adoptTerminalStyles(made.root);
  made.mount(host);

  const adopt = async (): Promise<void> => {
    const r = await request<{ sessions: { id: string; title: string; modified: number; current: boolean }[] }>(fetchFn, "GET", "/api/claude/sessions");
    if (!r.sessions.length) throw new Error(t("No Claude Code conversation in form.md yet: open one with the Claude Code button first"));
    await made.show("chat");
    const entries: MenuEntry[] = [{ kind: "heading", label: t("Continue from Claude Code") }];
    for (const s of r.sessions) {
      entries.push({
        label: s.title || t("Untitled chat"),
        hint: formatRelative(s.modified * 1000),
        checked: s.current,
        run: () => void request<ChatState>(fetchFn, "POST", "/api/claude/adopt", { session: s.id }).then(replay, (e: unknown) => engine.reportError(e)),
      });
    }
    openMenu(made.root.querySelector<HTMLElement>(".ft-dock-head") ?? made.root, entries, { side: "top", align: "end", label: t("Continue from Claude Code") });
  };

  return {
    toggle: () => made.toggle(),
    newTerminal: async () => {
      if (!made.isOpen()) made.toggle(true, "none");
      await made.newTerminal();
    },
    claude: async () => {
      await made.claude();
    },
    chat: () => made.show("chat"),
    adopt,
    isOpen: () => made.isOpen(),
  };
}
