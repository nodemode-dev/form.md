import type { NodeType } from "prosemirror-model";
import { type EditorState, Plugin, PluginKey, TextSelection } from "prosemirror-state";
import type { EditorView } from "prosemirror-view";
import type { Member, Page } from "../../engine";
import { schema } from "../../engine/doc/schema";
import { formatDate, formatDateTime } from "../../engine/format";
import { t } from "../../engine/i18n";
import { avatar } from "../../ui/avatar";
import { fallbackGlyph, pageIcon } from "../../ui/pageicon";
import { CATALOG, filterCatalog } from "./catalog";
import { caretRect, type EditorCtx } from "./context";
import { type ListItem, ListPopover } from "./popover";

export type TriggerKind = "slash" | "mention" | "link";

export interface TriggerState {
  active: boolean;
  kind: TriggerKind;
  from: number;
  query: string;
}

const INACTIVE: TriggerState = { active: false, kind: "slash", from: 0, query: "" };
const MARK: Record<TriggerKind, string> = { slash: "/", mention: "@", link: "[[" };

export const triggerKey = new PluginKey<TriggerState>("formmd-trigger");

export const SLASH_KEY = "formmd.slashAtStart";

function slashAtStartOnly(): boolean {
  try {
    return localStorage.getItem(SLASH_KEY) === "1";
  } catch {
    return false;
  }
}

function boundaryBefore(state: EditorState, pos: number): boolean {
  const $pos = state.doc.resolve(pos);
  if (!$pos.parent.isTextblock || $pos.parent.type.spec.code) return false;
  if ($pos.parentOffset === 0) return true;
  const before = state.doc.textBetween(pos - 1, pos, "\n", "￼");
  return /\s|￼/.test(before) || before === "(";
}

export function triggerPlugin(ctx: EditorCtx): Plugin<TriggerState> {
  return new Plugin<TriggerState>({
    key: triggerKey,
    state: {
      init: () => INACTIVE,
      apply(tr, prev, _old, next) {
        const meta = tr.getMeta(triggerKey) as TriggerState | undefined;
        if (meta) return meta;
        if (!prev.active || (!tr.docChanged && !tr.selectionSet)) return prev;
        const mapped = tr.mapping.mapResult(prev.from, 1);
        if (mapped.deleted) return INACTIVE;
        const from = mapped.pos;
        const sel = next.selection;
        if (!sel.empty || sel.head <= from || sel.$head.start() !== next.doc.resolve(from).start()) return INACTIVE;
        const text = next.doc.textBetween(from, sel.head, "\n", "￼");
        const mark = MARK[prev.kind];
        if (!text.startsWith(mark)) return INACTIVE;
        const query = text.slice(mark.length);
        if (query.length > 60 || query.includes("￼") || query.includes("\n") || /\s{2}/.test(query) || /^\s/.test(query)) return INACTIVE;
        if (prev.kind === "link" && query.includes("]]")) return INACTIVE;
        return { ...prev, from, query };
      },
    },
    props: {
      handleTextInput(view, from, to, text) {
        const st = triggerKey.getState(view.state);
        if (st?.active || !view.editable) return false;
        let kind: TriggerKind | null = null;
        let start = from;
        if (text === "/" && boundaryBefore(view.state, from) && (!slashAtStartOnly() || view.state.doc.resolve(from).parentOffset === 0)) kind = "slash";
        else if (text === "@" && boundaryBefore(view.state, from)) kind = "mention";
        else if (text === "[" && from > 0 && view.state.doc.textBetween(from - 1, from) === "[" && boundaryBefore(view.state, from - 1)) {
          kind = "link";
          start = from - 1;
        }
        if (!kind) return false;
        view.dispatch(view.state.tr.insertText(text, from, to).setMeta(triggerKey, { active: true, kind, from: start, query: "" }));
        return true;
      },
      handleKeyDown(view, e) {
        const st = triggerKey.getState(view.state);
        const tv = views.get(view);
        if (!st?.active || !tv?.popover?.open) return false;
        if (e.key === "ArrowDown" || (e.key === "n" && e.ctrlKey)) {
          tv.popover.move(1);
          return true;
        }
        if (e.key === "ArrowUp" || (e.key === "p" && e.ctrlKey)) {
          tv.popover.move(-1);
          return true;
        }
        if (e.key === "Enter" || e.key === "Tab") {
          if (tv.popover.count === 0) {
            tv.dismiss();
            return e.key === "Tab";
          }
          return tv.popover.pick();
        }
        return false;
      },
    },
    view(view) {
      const tv = new TriggerView(view, ctx);
      views.set(view, tv);
      return {
        update: () => tv.update(),
        destroy: () => {
          tv.destroy();
          views.delete(view);
        },
      };
    },
  });
}

const views = new WeakMap<EditorView, TriggerView>();
const memberCache = new Map<string, { at: number; list: Member[] }>();

function fuzzy(title: string, q: string): number {
  const a = title.toLowerCase();
  if (!q) return 3;
  if (a.startsWith(q)) return 0;
  if (a.split(/\s+/).some((w) => w.startsWith(q))) return 1;
  if (a.includes(q)) return 2;
  return -1;
}

function localStamp(ms: number): string {
  const d = new Date(ms);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

function reminderTimes(q: string): number[] {
  const typed = /^(?:remind\s+)?(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{1,2}):(\d{2}))?$/.exec(q);
  if (typed) {
    const [, y, mo, d, hh, mm] = typed;
    const at = new Date(Number(y), Number(mo) - 1, Number(d), hh ? Number(hh) : 9, mm ? Number(mm) : 0).getTime();
    return Number.isFinite(at) && at > Date.now() ? [at] : [];
  }
  if (q && !"remind".startsWith(q.split(/\s/)[0] ?? "") && !q.startsWith("remind")) return [];
  const morning = (days: number) => {
    const d = new Date();
    d.setDate(d.getDate() + days);
    d.setHours(9, 0, 0, 0);
    return d.getTime();
  };
  const monday = (() => {
    const d = new Date();
    const add = ((8 - d.getDay()) % 7) || 7;
    return morning(add);
  })();
  const hour = Math.ceil((Date.now() + 3600_000) / 60_000) * 60_000;
  return [hour, morning(1), monday].filter((v, i, all) => all.indexOf(v) === i);
}

function isoDay(offset: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offset);
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

class TriggerView {
  popover: ListPopover | null = null;
  private kind: TriggerKind | null = null;
  private lastQuery = "\u0000";
  private loadingMembers = "";

  constructor(
    private readonly view: EditorView,
    private readonly ctx: EditorCtx,
  ) {}

  update(): void {
    const st = triggerKey.getState(this.view.state);
    if (!st?.active) {
      this.closePopover();
      return;
    }
    if (!this.popover || this.kind !== st.kind) {
      this.closePopover();
      this.kind = st.kind;
      this.lastQuery = "\u0000";
      const label = st.kind === "slash" ? t("Blocks") : st.kind === "mention" ? t("Mention") : t("Link to a page");
      this.popover = new ListPopover({
        label,
        empty: t("Nothing matches"),
        rect: () => {
          const s = triggerKey.getState(this.view.state);
          return s?.active ? caretRect(this.view, Math.min(s.from, this.view.state.doc.content.size)) : null;
        },
        onPick: (item) => this.pick(item),
        onClose: () => {
          this.popover = null;
          const s = triggerKey.getState(this.view.state);
          if (s?.active) this.view.dispatch(this.view.state.tr.setMeta(triggerKey, INACTIVE));
        },
      });
    }
    if (st.query !== this.lastQuery) {
      this.lastQuery = st.query;
      const items = this.items(st);
      if (!items.length && st.kind === "slash" && /\s$/.test(st.query)) {
        this.dismiss();
        return;
      }
      this.popover.set(items);
    } else this.popover.reposition();
  }

  dismiss(): void {
    this.view.dispatch(this.view.state.tr.setMeta(triggerKey, INACTIVE));
  }

  private closePopover(): void {
    const p = this.popover;
    this.popover = null;
    this.kind = null;
    p?.close();
  }

  private items(st: TriggerState): ListItem[] {
    const q = st.query.trim().toLowerCase();
    if (st.kind === "slash") {
      return filterCatalog(q).map((e) => ({ key: e.id, label: e.label(), icon: e.icon, hint: e.hint, section: q ? undefined : e.section() }));
    }
    const out: ListItem[] = [];
    const ws = this.ctx.workspaceId();
    const pages: Page[] = ws ? this.ctx.engine.pages.all(ws) : [];
    const hits = pages
      .filter((p) => p.id !== this.ctx.pageId() && !p.is_template)
      .map((p) => ({ p, s: fuzzy(p.title || t("Untitled"), q) }))
      .filter((x) => x.s >= 0)
      .sort((a, b) => a.s - b.s || b.p.updated_at - a.p.updated_at)
      .slice(0, st.kind === "link" ? 8 : 6);
    for (const { p } of hits) {
      out.push({ key: `p:${p.id}`, label: p.title || t("Untitled"), icon: () => pageIcon(p.icon, fallbackGlyph(p.kind)), section: t("Pages") });
    }
    if (st.kind === "link" && q && this.ctx.writable()) {
      out.push({ key: "new", label: t("Create “{title}” inside this page", { title: st.query.trim() }), icon: "plus", section: t("New") });
    }
    if (st.kind === "mention") {
      const members = this.members(ws);
      for (const m of members.filter((m) => !m.disabled && fuzzy(m.name, q) >= 0).slice(0, 5)) {
        out.push({ key: `u:${m.user_id}`, label: m.name, icon: () => avatar(m.name, m.color, false, m.avatar), section: t("People") });
      }
      const days: [string, string][] = [
        [t("Today"), isoDay(0)],
        [t("Tomorrow"), isoDay(1)],
        [t("Yesterday"), isoDay(-1)],
      ];
      const typed = /^\d{4}-\d{2}-\d{2}$/.test(q) ? q : "";
      if (typed) days.unshift([formatDate(new Date(`${typed}T12:00:00`).getTime()), typed]);
      for (const [label, iso] of days) {
        if (typed || fuzzy(label, q) >= 0) out.push({ key: `d:${iso}`, label, icon: "calendar", detail: formatDate(new Date(`${iso}T12:00:00`).getTime()), section: t("Dates") });
      }
      for (const at of reminderTimes(q)) out.push({ key: `r:${at}`, label: t("Remind me"), icon: "bell", detail: formatDateTime(at), section: t("Reminders") });
    }
    return out;
  }

  private members(ws: string): Member[] {
    if (!ws) return [];
    const hit = memberCache.get(ws);
    if (hit && Date.now() - hit.at < 60_000) return hit.list;
    if (this.loadingMembers !== ws) {
      this.loadingMembers = ws;
      void this.ctx.engine.api.workspaces
        .members(ws)
        .then((r) => {
          memberCache.set(ws, { at: Date.now(), list: r.members });
          this.lastQuery = "\u0000";
          this.update();
        })
        .catch(() => undefined)
        .finally(() => {
          this.loadingMembers = "";
        });
    }
    return hit?.list ?? [];
  }

  private pick(item: ListItem): void {
    const st = triggerKey.getState(this.view.state);
    if (!st?.active) return;
    const view = this.view;
    const from = st.from;
    const to = view.state.selection.head;
    const tr = view.state.tr.delete(from, to).setMeta(triggerKey, INACTIVE);
    tr.setSelection(TextSelection.create(tr.doc, from));
    view.dispatch(tr);
    const mention = schema.nodes.mention as NodeType;
    const insertMention = (attrs: Record<string, string | number>) => {
      const node = mention.create(attrs);
      const tr2 = view.state.tr.replaceSelectionWith(node, false).insertText(" ");
      view.dispatch(tr2.scrollIntoView());
      view.focus();
    };
    if (st.kind === "slash") {
      const entry = CATALOG.find((e) => e.id === item.key);
      entry?.run(view, this.ctx);
      view.focus();
      return;
    }
    if (item.key.startsWith("p:")) {
      const p = this.ctx.engine.pages.get(item.key.slice(2));
      insertMention({ kind: "page", ref: item.key.slice(2), label: p?.title ?? "" });
    } else if (item.key.startsWith("u:")) {
      insertMention({ kind: "user", ref: item.key.slice(2), label: item.label });
    } else if (item.key.startsWith("d:")) {
      insertMention({ kind: "date", ref: item.key.slice(2), label: "" });
    } else if (item.key.startsWith("r:")) {
      const at = Number(item.key.slice(2));
      insertMention({ kind: "date", ref: localStamp(at), label: "", remind: this.ctx.engine.state.get("me")?.user.id ?? "", at });
    } else if (item.key === "new") {
      const title = st.query.trim();
      const ws = this.ctx.workspaceId();
      const parent = this.ctx.pageId();
      if (!ws || !parent) return;
      void this.ctx.engine.pages
        .create(ws, parent)
        .then((p) => {
          this.ctx.engine.pages.edit(p.id, { title }, false);
          insertMention({ kind: "page", ref: p.id, label: title });
        })
        .catch(this.ctx.engine.reportError);
    }
  }

  destroy(): void {
    this.closePopover();
  }
}
