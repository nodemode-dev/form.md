import type { Engine } from "../../engine";
import { plural, t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import { h, setText } from "../../ui/dom";
import { editorHost } from "../blocks";
import { type Align, setAlign } from "../blocks/align";
import { SLASH_KEY } from "../blocks/triggers";
import type { Extension } from "../registry";

const COUNT_KEY = "formmd.wordCount";
const FOCUS_KEY = "formmd.focus";
const WPM = 230;

let teardown: Unsubscribe[] = [];

function read(key: string): boolean {
  try {
    return localStorage.getItem(key) === "1";
  } catch {
    return false;
  }
}

function write(key: string, on: boolean): void {
  try {
    if (on) localStorage.setItem(key, "1");
    else localStorage.removeItem(key);
  } catch {
    return;
  }
}

const segmenter = typeof Intl !== "undefined" && "Segmenter" in Intl ? new Intl.Segmenter(undefined, { granularity: "word" }) : null;

export function countWords(text: string): number {
  if (!text.trim()) return 0;
  if (segmenter) {
    let n = 0;
    for (const s of segmenter.segment(text)) if (s.isWordLike) n++;
    return n;
  }
  return text.trim().split(/\s+/).length;
}

export interface Counts { words: number; characters: number; minutes: number; scope: "page" | "selection" | "paragraph" }

export function countsFor(scope: "page" | "selection" | "paragraph"): Counts | null {
  const v = editorHost()?.view();
  if (!v) return null;
  const { state } = v;
  let text: string;
  if (scope === "selection") text = state.doc.textBetween(state.selection.from, state.selection.to, "\n", " ");
  else if (scope === "paragraph") text = state.selection.$from.parent.textContent;
  else text = state.doc.textBetween(0, state.doc.content.size, "\n", " ");
  const words = countWords(text);
  return { words, characters: [...text.replace(/\n/g, "")].length, minutes: Math.max(words ? 1 : 0, Math.round(words / WPM)), scope };
}

function setup(engine: Engine): void {
  const { commands, keymap, state } = engine;
  const root = document.documentElement;
  const meter = h("div", { class: "wcount", attrs: { role: "status", "aria-live": "off" } });
  meter.hidden = true;
  document.body.appendChild(meter);

  let frame = 0;
  const onPage = () => {
    const r = state.get("route");
    if (r.name !== "page") return false;
    const p = engine.pages.get(r.id);
    return !!p && p.kind !== "collection";
  };
  const paint = () => {
    frame = 0;
    const show = (read(COUNT_KEY) || root.hasAttribute("data-focus")) && onPage();
    if (meter.hidden === show) meter.hidden = !show;
    if (!show) return;
    const v = editorHost()?.view();
    const sel = v && !v.state.selection.empty ? countsFor("selection") : null;
    const all = countsFor("page");
    if (!all) return;
    const para = v ? countsFor("paragraph") : null;
    const parts = sel
      ? [t("Selection"), plural(sel.words, "{count} word", "{count} words"), plural(sel.characters, "{count} character", "{count} characters")]
      : [plural(all.words, "{count} word", "{count} words"), plural(all.characters, "{count} character", "{count} characters"), t("{count} min read", { count: all.minutes })];
    if (!sel && para && para.words) parts.push(t("This paragraph: {count}", { count: para.words }));
    setText(meter, parts.join(" · "));
  };
  const schedule = () => {
    if (!frame) frame = requestAnimationFrame(paint);
  };
  const onInput = (e: Event) => {
    if ((e.target as HTMLElement | null)?.closest?.(".doc-body")) schedule();
  };
  document.addEventListener("input", onInput, true);
  document.addEventListener("selectionchange", schedule);
  const offShown = editorHost()?.onShown(() => schedule()) ?? (() => undefined);

  const setFocus = (on: boolean) => {
    if (on === root.hasAttribute("data-focus")) return;
    root.toggleAttribute("data-focus", on);
    write(FOCUS_KEY, on);
    schedule();
  };
  if (read(FOCUS_KEY)) root.toggleAttribute("data-focus", true);

  const group = () => t("Writing");
  teardown = [
    commands.register({
      id: "view.focus",
      label: () => t("Focus mode"),
      group,
      description: "Hides the sidebar, top bar, tabs and page controls so only the text remains, and shows the word count. Run again to leave.",
      params: { on: { type: "boolean" } },
      run: (p) => setFocus(typeof p?.on === "boolean" ? p.on : !root.hasAttribute("data-focus")),
    }),
    commands.register({
      id: "view.wordCount",
      label: () => t("Show or hide the word count"),
      group,
      description: "Words, characters and reading time for the page, the selection or the paragraph under the cursor.",
      run: () => {
        write(COUNT_KEY, !read(COUNT_KEY));
        schedule();
      },
    }),
    commands.register({
      id: "editor.counts",
      label: () => t("Count words"),
      group,
      description: "Returns word, character and reading-time counts for the open page, the current selection or the paragraph under the cursor.",
      params: { scope: { type: "string", enum: ["page", "selection", "paragraph"] } },
      run: (p) => countsFor((p?.scope as Counts["scope"]) ?? "page") ?? { error: "no document is open" },
    }),
    commands.register({
      id: "editor.align",
      label: () => t("Text alignment"),
      group,
      description: "Aligns the paragraphs and headings in the selection, or the one under the cursor; left removes the alignment.",
      params: { align: { type: "string", required: true, enum: ["left", "center", "right", "justify"] } },
      mutates: true,
      run: (p) => {
        const v = editorHost()?.view();
        if (!v || !v.editable) throw new Error("no editable document is open");
        const a = String(p?.align ?? "left");
        return { changed: setAlign(v, a === "left" ? "" : (a as Align)) };
      },
    }),
    commands.register({
      id: "editor.slashAtStart",
      label: () => t("Open the slash menu only at the start of a line"),
      group,
      description: "When on, typing a slash in the middle of a sentence stays a slash; the block menu opens only from an empty line or the start of a block.",
      run: () => write(SLASH_KEY, !read(SLASH_KEY)),
    }),
    keymap.register({ id: "view.focus", command: "view.focus", chords: ["mod+shift+f"], scope: "global", whileTyping: true }),
    keymap.register({ id: "view.focus.exit", command: "view.focus", chords: ["escape"], scope: "global", when: () => root.hasAttribute("data-focus") }),
    state.watch("route", schedule),
    () => {
      document.removeEventListener("input", onInput, true);
      document.removeEventListener("selectionchange", schedule);
      offShown();
      meter.remove();
    },
  ];
  schedule();
}

const writing: Extension = {
  id: "writing",
  setup,
  dispose() {
    for (const fn of teardown) fn();
    teardown = [];
  },
};

export default writing;
