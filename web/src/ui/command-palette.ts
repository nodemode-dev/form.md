import type { Engine } from "../engine";
import type { Command } from "../engine/commands";
import { t } from "../engine/i18n";
import { byId, h, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { openLayer } from "./layers";

const ROWS = 12;
const RECENT_KEY = "formmd.palette.recent";
const RECENT_MAX = 5;
const HIDDEN = new Set(["palette.open", "layers.closeTop"]);

type Row = { kind: "head"; label: string } | { kind: "cmd"; command: Command };

function readRecent(): string[] {
  try {
    const v: unknown = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string").slice(0, RECENT_MAX) : [];
  } catch {
    return [];
  }
}

function remember(id: string): void {
  const next = [id, ...readRecent().filter((x) => x !== id)].slice(0, RECENT_MAX);
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    return;
  }
}

class Palette {
  private readonly scrim: HTMLElement;
  private readonly dialog: HTMLElement;
  private readonly input: HTMLInputElement;
  private readonly rows: HTMLElement[] = [];
  private readonly none: HTMLElement;
  private shown: Row[] = [];
  private all: Command[] = [];
  private at = -1;
  private closeLayer: (() => void) | null = null;
  private hideTimer: ReturnType<typeof setTimeout> | undefined;

  constructor(private readonly engine: Engine) {
    this.input = h("input", {
      class: "cpal__input",
      attrs: { type: "search", role: "combobox", "aria-expanded": "true", "aria-controls": "cpal-list", "aria-autocomplete": "list", autocomplete: "off", spellcheck: "false", placeholder: t("Type what you want to do"), "aria-label": t("Find a command") },
    });
    const head = h("div", { class: "cpal__head" }, h("span", { class: "cpal__glass" }, iconNode("search")), this.input);
    const list = h("div", { class: "cpal__list", attrs: { id: "cpal-list", role: "listbox", "aria-label": t("Commands") } });
    this.none = h("p", { class: "cpal__none", text: t("No command matches. Try another word.") });
    this.none.hidden = true;
    list.append(this.none);
    for (let i = 0; i < ROWS; i++) {
      const row = h("div", { class: "cpal__row", attrs: { id: `cpal-opt-${i}`, role: "option", "aria-selected": "false" } }, h("span", { class: "cpal__label" }), h("span", { class: "cpal__group" }), h("kbd", { class: "cpal__keys" }));
      row.hidden = true;
      row.addEventListener("pointerdown", (e) => {
        e.preventDefault();
        this.pick(i);
      });
      row.addEventListener("pointermove", () => this.highlight(i));
      list.append(row);
      this.rows.push(row);
    }
    this.dialog = h("div", { class: "dialog cpal", attrs: { role: "dialog", "aria-modal": "true", "aria-label": t("Find a command"), tabindex: "-1" } }, head, list);
    this.scrim = h("div", { class: "scrim", attrs: { "aria-hidden": "true" } });
    this.scrim.hidden = true;
    this.dialog.hidden = true;
    byId("overlay-root").append(this.scrim, this.dialog);
    this.input.addEventListener("input", () => this.filter());
    this.input.addEventListener("keydown", (e) => {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        this.move(e.key === "ArrowDown" ? 1 : -1);
      } else if (e.key === "Enter") {
        e.preventDefault();
        this.pick(this.at);
      }
    });
  }

  show(): void {
    this.all = this.engine.commands.list().filter((c) => !HIDDEN.has(c.id) && !Object.values(c.params ?? {}).some((p) => p.required));
    clearTimeout(this.hideTimer);
    this.input.value = "";
    this.filter();
    this.scrim.hidden = false;
    this.dialog.hidden = false;
    requestAnimationFrame(() => {
      this.scrim.classList.add("is-visible");
      this.dialog.classList.add("is-visible");
    });
    if (!this.closeLayer) {
      this.closeLayer = openLayer({
        el: this.dialog,
        modal: true,
        dismissOnOutside: true,
        initialFocus: this.input,
        onClose: () => {
          this.closeLayer = null;
          this.hide();
        },
      });
    } else this.input.focus();
  }

  private close(): void {
    const c = this.closeLayer;
    this.closeLayer = null;
    c?.();
    this.hide();
  }

  private hide(): void {
    this.scrim.classList.remove("is-visible");
    this.dialog.classList.remove("is-visible");
    clearTimeout(this.hideTimer);
    this.hideTimer = setTimeout(() => {
      this.scrim.hidden = true;
      this.dialog.hidden = true;
    }, 180);
  }

  private filter(): void {
    const words = this.input.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
    const score = (c: Command): number => {
      const label = c.label().toLowerCase();
      const hay = `${c.id.toLowerCase()} ${label} ${c.group().toLowerCase()}`;
      if (!words.every((w) => hay.includes(w))) return -1;
      return (words.length && label.startsWith(words[0] ?? "") ? 4 : 0) + (words.every((w) => label.includes(w)) ? 2 : 0) + (c.enabled?.() !== false ? 1 : 0);
    };
    const ranked = this.all
      .map((c) => [c, score(c)] as const)
      .filter(([, s]) => s >= 0)
      .sort((a, b) => b[1] - a[1] || a[0].label().localeCompare(b[0].label()))
      .map(([c]) => c);
    const rows: Row[] = [];
    if (!words.length) {
      const recent = readRecent()
        .map((id) => this.all.find((c) => c.id === id))
        .filter((c): c is Command => !!c);
      if (recent.length) {
        rows.push({ kind: "head", label: t("Recently used") }, ...recent.map((command) => ({ kind: "cmd" as const, command })));
        rows.push({ kind: "head", label: t("All commands") });
        for (const command of ranked) if (!recent.includes(command)) rows.push({ kind: "cmd", command });
      }
    }
    if (!rows.length) rows.push(...ranked.map((command) => ({ kind: "cmd" as const, command })));
    this.shown = rows.slice(0, ROWS);
    if (this.shown.at(-1)?.kind === "head") this.shown.pop();
    this.rows.forEach((row, i) => {
      const r = this.shown[i];
      if (row.hidden !== !r) row.hidden = !r;
      if (!r) return;
      const head = r.kind === "head";
      row.classList.toggle("is-head", head);
      setAttr(row, "role", head ? "presentation" : "option");
      setText(row.children[0] as HTMLElement, head ? r.label : r.command.label());
      setText(row.children[1] as HTMLElement, head ? "" : r.command.group());
      const chord = head ? undefined : this.engine.keymap.chordsFor(r.command.id)[0];
      setText(row.children[2] as HTMLElement, chord ? this.engine.keymap.text(chord) : "");
      const off = !head && r.command.enabled?.() === false;
      row.classList.toggle("is-off", off);
      setAttr(row, "aria-disabled", off ? "true" : null);
    });
    if (this.none.hidden !== this.shown.length > 0) this.none.hidden = this.shown.length > 0;
    this.highlight(this.next(-1, 1));
  }

  private next(from: number, dir: number): number {
    for (let i = from + dir; i >= 0 && i < this.shown.length; i += dir) if (this.shown[i]?.kind === "cmd") return i;
    return -1;
  }

  private move(dir: number): void {
    const n = this.next(this.at, dir);
    if (n >= 0) this.highlight(n);
  }

  private highlight(i: number): void {
    if (i >= 0 && this.shown[i]?.kind !== "cmd") return;
    this.at = i;
    this.rows.forEach((row, n) => {
      const on = n === i && !row.hidden;
      row.classList.toggle("is-on", on);
      setAttr(row, "aria-selected", on ? "true" : "false");
    });
    setAttr(this.input, "aria-activedescendant", i >= 0 ? `cpal-opt-${i}` : null);
  }

  private pick(i: number): void {
    const r = this.shown[i];
    if (!r || r.kind !== "cmd" || r.command.enabled?.() === false) return;
    const id = r.command.id;
    remember(id);
    this.close();
    setTimeout(() => this.engine.commands.run(id), 0);
  }
}

let palette: Palette | null = null;

export function openPalette(engine: Engine): void {
  palette ??= new Palette(engine);
  palette.show();
}
