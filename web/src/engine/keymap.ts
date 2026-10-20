import type { Commands } from "./commands";
import type { Unsubscribe } from "./state";

export type Scope = "global" | "view" | "editor" | "modal";

const RANK: Record<Scope, number> = { global: 0, view: 1, editor: 2, modal: 3 };

export interface Binding {
  id: string;
  command: string;
  chords: readonly string[];
  scope: Scope;
  whileTyping?: boolean;
  when?: () => boolean;
}

export interface KeyInput {
  key: string;
  code: string;
  ctrlKey: boolean;
  metaKey: boolean;
  altKey: boolean;
  shiftKey: boolean;
  isComposing: boolean;
}

export interface KeyContext {
  typing: boolean;
  scopes: ReadonlySet<Scope>;
  only?: ReadonlySet<string> | undefined;
}

export interface ListedShortcut {
  command: string;
  label: string;
  group: string;
  chords: string[][];
}

type Kind = "code" | "named" | "symbol";

interface Chord {
  mod: boolean;
  shift: boolean;
  alt: boolean;
  key: string;
  kind: Kind;
}

const NAMED = new Set([
  "escape", "enter", "tab", "space", "backspace", "delete", "home", "end", "pageup", "pagedown",
  "arrowup", "arrowdown", "arrowleft", "arrowright",
  "f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
]);

const NAMED_LABEL: Record<string, string> = {
  escape: "Esc", enter: "Enter", tab: "Tab", space: "Space", backspace: "Backspace", delete: "Delete",
  home: "Home", end: "End", pageup: "PgUp", pagedown: "PgDn",
  arrowup: "↑", arrowdown: "↓", arrowleft: "←", arrowright: "→",
};

function parseChord(text: string): Chord {
  const parts = text.split("+");
  const raw = parts.pop() ?? "";
  const mods = new Set(parts.map((p) => p.toLowerCase()));
  const lower = raw.toLowerCase();
  let kind: Kind = "symbol";
  let key = raw;
  if (/^[a-z]$/.test(lower)) {
    kind = "code";
    key = `Key${lower.toUpperCase()}`;
  } else if (/^[0-9]$/.test(lower)) {
    kind = "code";
    key = `Digit${lower}`;
  } else if (NAMED.has(lower)) {
    kind = "named";
    key = lower;
  }
  return { mod: mods.has("mod"), shift: mods.has("shift"), alt: mods.has("alt"), key, kind };
}

export class Keymap {
  private bindings = new Map<string, { binding: Binding; seq: number }>();
  private parsed = new Map<string, Chord>();
  private custom = new Map<string, string[]>();
  private bound = new Map<string, number>();
  private seq = 0;

  constructor(
    private readonly commands: Commands,
    private readonly mac: boolean,
  ) {}

  register(binding: Binding): Unsubscribe {
    if (this.bindings.has(binding.id)) throw new Error(`duplicate binding: ${binding.id}`);
    for (const c of binding.chords) this.chord(c);
    this.bindings.set(binding.id, { binding, seq: ++this.seq });
    this.bound.set(binding.command, (this.bound.get(binding.command) ?? 0) + 1);
    return () => {
      if (!this.bindings.delete(binding.id)) return;
      const n = (this.bound.get(binding.command) ?? 1) - 1;
      if (n > 0) this.bound.set(binding.command, n);
      else this.bound.delete(binding.command);
    };
  }

  isMac(): boolean {
    return this.mac;
  }

  text(chord: string): string {
    return this.format(chord).join(this.mac ? "" : "+");
  }

  defaultChords(command: string): string[] {
    const out: string[] = [];
    for (const { binding } of this.bindings.values()) {
      if (binding.command !== command || (binding.when && !binding.when())) continue;
      for (const c of binding.chords) if (!out.some((x) => this.same(x, c))) out.push(c);
    }
    return out;
  }

  chordsFor(command: string): string[] {
    return this.custom.get(command)?.slice() ?? this.defaultChords(command);
  }

  setCustom(command: string, chords: string[] | null): void {
    if (chords === null) this.custom.delete(command);
    else {
      for (const c of chords) this.chord(c);
      this.custom.set(command, chords.slice());
    }
  }

  customs(): Record<string, string[]> {
    return Object.fromEntries([...this.custom].map(([k, v]) => [k, v.slice()]));
  }

  loadCustoms(record: unknown): void {
    this.custom.clear();
    if (!record || typeof record !== "object") return;
    for (const [k, v] of Object.entries(record as Record<string, unknown>)) {
      if (Array.isArray(v) && v.every((c) => typeof c === "string" && c.length > 0 && c.length < 40)) this.setCustom(k, v as string[]);
    }
  }

  scopesOf(command: string): Scope[] {
    const out = new Set<Scope>();
    for (const { binding } of this.bindings.values()) if (binding.command === command) out.add(binding.scope);
    if (!out.size) out.add("global");
    return [...out];
  }

  conflicts(): Map<string, { chord: string; with: string[] }[]> {
    const ids = new Set<string>([...this.commands.list().map((c) => c.id), ...this.custom.keys()]);
    const scopes = new Map<string, Scope[]>();
    const byChord = new Map<string, string[]>();
    const chordsOf = new Map<string, string[]>();
    for (const id of ids) {
      scopes.set(id, this.scopesOf(id));
      const chords = this.chordsFor(id);
      chordsOf.set(id, chords);
      for (const c of chords) {
        const k = this.canon(c);
        const list = byChord.get(k) ?? [];
        if (!list.includes(id)) list.push(id);
        byChord.set(k, list);
      }
    }
    const out = new Map<string, { chord: string; with: string[] }[]>();
    for (const [id, chords] of chordsOf) {
      const mine = scopes.get(id) ?? [];
      for (const chord of chords) {
        const hit = (byChord.get(this.canon(chord)) ?? []).filter((o) => o !== id && (scopes.get(o) ?? []).some((s) => mine.includes(s)));
        if (!hit.length) continue;
        const list = out.get(id) ?? [];
        list.push({ chord, with: hit });
        out.set(id, list);
      }
    }
    return out;
  }

  private canon(text: string): string {
    const c = this.chord(text);
    return `${c.mod ? "m" : ""}${c.alt ? "a" : ""}${c.shift && c.kind !== "symbol" ? "s" : ""}|${c.kind}|${c.key}`;
  }

  fromEvent(e: KeyInput): string | null {
    if (e.isComposing || (this.mac ? e.ctrlKey : e.metaKey)) return null;
    const lower = e.key.toLowerCase();
    if (["shift", "control", "alt", "meta", "altgraph", "capslock", "os", "fn", "dead", "process", "unidentified"].includes(lower)) return null;
    let key = "";
    let symbol = false;
    if (/^Key[A-Z]$/.test(e.code)) key = e.code.slice(3).toLowerCase();
    else if (/^Digit[0-9]$/.test(e.code)) key = e.code.slice(5);
    else if (e.key === " ") key = "space";
    else if (NAMED.has(lower)) key = lower;
    else if (e.key.length === 1 && e.key !== "+") {
      key = e.key;
      symbol = true;
    } else return null;
    const mods: string[] = [];
    if (this.mac ? e.metaKey : e.ctrlKey) mods.push("mod");
    if (e.altKey) mods.push("alt");
    if (e.shiftKey && !symbol) mods.push("shift");
    return [...mods, key].join("+");
  }

  private same(a: string, b: string): boolean {
    return this.canon(a) === this.canon(b);
  }

  private chordsOf(b: Binding): readonly string[] {
    return this.custom.get(b.command) ?? b.chords;
  }

  handle(e: KeyInput, ctx: KeyContext): boolean {
    if (e.isComposing || e.key === "Process") return false;
    const hits: { binding: Binding; seq: number }[] = [];
    for (const entry of this.bindings.values()) {
      const b = entry.binding;
      if (!ctx.scopes.has(b.scope)) continue;
      if (ctx.typing && !b.whileTyping) continue;
      if (ctx.only && !ctx.only.has(b.command)) continue;
      if (!this.chordsOf(b).some((c) => this.matches(this.chord(c), e))) continue;
      if (b.when && !b.when()) continue;
      hits.push(entry);
    }
    if (ctx.scopes.has("global")) {
      for (const [command, chords] of this.custom) {
        if (this.bound.get(command) || (ctx.only && !ctx.only.has(command))) continue;
        const c = chords.map((x) => this.chord(x)).find((x) => this.matches(x, e));
        if (!c || (ctx.typing && !c.mod && !c.alt && !/^f\d/.test(c.key))) continue;
        hits.push({ binding: { id: `custom:${command}`, command, chords, scope: "global", whileTyping: true }, seq: 0 });
      }
    }
    hits.sort((a, b) => RANK[b.binding.scope] - RANK[a.binding.scope] || b.seq - a.seq);
    for (const { binding } of hits) {
      if (this.commands.run(binding.command)) return true;
    }
    return false;
  }

  list(scopes: ReadonlySet<Scope>): ListedShortcut[] {
    const byCommand = new Map<string, ListedShortcut>();
    for (const { binding } of this.bindings.values()) {
      if (!scopes.has(binding.scope) || (binding.when && !binding.when())) continue;
      const command = this.commands.get(binding.command);
      if (!command) continue;
      let item = byCommand.get(command.id);
      if (!item) {
        item = { command: command.id, label: command.label(), group: command.group(), chords: [] };
        byCommand.set(command.id, item);
      }
      for (const c of this.chordsOf(binding)) {
        const f = this.format(c);
        if (!item.chords.some((x) => x.join("+") === f.join("+"))) item.chords.push(f);
      }
    }
    if (scopes.has("global")) {
      for (const [id, chords] of this.custom) {
        const command = this.bound.get(id) ? undefined : this.commands.get(id);
        if (command && chords.length) byCommand.set(id, { command: id, label: command.label(), group: command.group(), chords: chords.map((c) => this.format(c)) });
      }
    }
    for (const [id, item] of byCommand) if (!item.chords.length) byCommand.delete(id);
    return [...byCommand.values()];
  }

  chordFor(commandId: string): string[] | undefined {
    const custom = this.custom.get(commandId);
    if (custom) return custom[0] ? this.format(custom[0]) : undefined;
    for (const { binding } of this.bindings.values()) {
      if (binding.command === commandId && binding.chords[0]) return this.format(binding.chords[0]);
    }
    return undefined;
  }

  format(text: string): string[] {
    const c = this.chord(text);
    const out: string[] = [];
    if (c.mod) out.push(this.mac ? "⌘" : "Ctrl");
    if (c.alt) out.push(this.mac ? "⌥" : "Alt");
    if (c.shift) out.push(this.mac ? "⇧" : "Shift");
    if (c.kind === "code") out.push(c.key.replace(/^(Key|Digit)/, ""));
    else if (c.kind === "named") out.push(NAMED_LABEL[c.key] ?? c.key.toUpperCase());
    else out.push(c.key);
    return out;
  }

  private chord(text: string): Chord {
    let c = this.parsed.get(text);
    if (!c) {
      c = parseChord(text);
      this.parsed.set(text, c);
    }
    return c;
  }

  private matches(c: Chord, e: KeyInput): boolean {
    const mod = this.mac ? e.metaKey : e.ctrlKey;
    const foreign = this.mac ? e.ctrlKey : e.metaKey;
    if (foreign || mod !== c.mod || e.altKey !== c.alt) return false;
    switch (c.kind) {
      case "code":
        return e.code === c.key && e.shiftKey === c.shift;
      case "named":
        return e.key.toLowerCase() === (c.key === "space" ? " " : c.key) && e.shiftKey === c.shift;
      case "symbol":
        return e.key === c.key;
    }
  }
}
