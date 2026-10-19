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
  private seq = 0;

  constructor(
    private readonly commands: Commands,
    private readonly mac: boolean,
  ) {}

  register(binding: Binding): Unsubscribe {
    if (this.bindings.has(binding.id)) throw new Error(`duplicate binding: ${binding.id}`);
    for (const c of binding.chords) this.chord(c);
    this.bindings.set(binding.id, { binding, seq: ++this.seq });
    return () => {
      this.bindings.delete(binding.id);
    };
  }

  handle(e: KeyInput, ctx: KeyContext): boolean {
    if (e.isComposing || e.key === "Process") return false;
    const hits: { binding: Binding; seq: number }[] = [];
    for (const entry of this.bindings.values()) {
      const b = entry.binding;
      if (!ctx.scopes.has(b.scope)) continue;
      if (ctx.typing && !b.whileTyping) continue;
      if (ctx.only && !ctx.only.has(b.command)) continue;
      if (!b.chords.some((c) => this.matches(this.chord(c), e))) continue;
      if (b.when && !b.when()) continue;
      hits.push(entry);
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
      for (const c of binding.chords) item.chords.push(this.format(c));
    }
    return [...byCommand.values()];
  }

  chordFor(commandId: string): string[] | undefined {
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
