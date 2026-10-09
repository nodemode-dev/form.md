import type { CommentListing, CommentThread, Engine, Member, PageComment, ReactionIcon, Suggestion } from "../engine";
import { formatDateTime, formatRelative } from "../engine/format";
import { t } from "../engine/i18n";
import { type ListItem, ListPopover } from "../extensions/blocks/popover";
import { avatar, updateAvatar } from "./avatar";
import { confirmDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { type Rect } from "./floating";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { type MenuEntry, openMenu } from "./menu";
import { fallbackGlyph, pageIcon } from "./pageicon";
import { toast } from "./toast";

export const REACTIONS: readonly (readonly [ReactionIcon, () => string])[] = [
  ["heart", () => t("Love")],
  ["star", () => t("Star")],
  ["sparkle", () => t("Brilliant")],
  ["eye", () => t("Looking into it")],
  ["rocket", () => t("Ship it")],
  ["lightbulb", () => t("Good idea")],
];

const MENTION = /@\[([^\]\n]{1,200})\]\((user|page):([a-z2-7]{26})\)/g;
const memberCache = new Map<string, { at: number; list: Member[] }>();

function chip(engine: Engine, kind: string, id: string, label: string): HTMLElement {
  if (kind === "page") {
    const p = engine.pages.get(id);
    return h("a", { class: "mention", attrs: { "data-kind": "page", "data-ref": id, "data-label": label, href: `/p/${id}`, contenteditable: "false" } }, pageIcon(p?.icon ?? "", fallbackGlyph(p?.kind ?? "doc"), "mention__icon"), h("span", { text: p?.title || label }));
  }
  return h("span", { class: "mention", attrs: { "data-kind": "user", "data-ref": id, "data-label": label, contenteditable: "false" }, text: `@${label}` });
}

export function renderBody(engine: Engine, body: string): DocumentFragment {
  const frag = document.createDocumentFragment();
  const text = (s: string) => {
    s.split("\n").forEach((line, i) => {
      if (i > 0) frag.append(h("br"));
      if (line) frag.append(document.createTextNode(line));
    });
  };
  let last = 0;
  for (const m of body.matchAll(MENTION)) {
    text(body.slice(last, m.index));
    frag.append(chip(engine, m[2] ?? "user", m[3] ?? "", m[1] ?? ""));
    last = (m.index ?? 0) + m[0].length;
  }
  text(body.slice(last));
  return frag;
}

function serialize(root: Node): string {
  let out = "";
  for (const n of Array.from(root.childNodes)) {
    if (n.nodeType === Node.TEXT_NODE) out += (n.textContent ?? "").replace(/ /g, " ");
    else if (n instanceof HTMLElement) {
      if (n.dataset.ref && n.dataset.kind) out += `@[${(n.dataset.label ?? "").replace(/[\]\n]/g, "")}](${n.dataset.kind}:${n.dataset.ref})`;
      else if (n.tagName === "BR") out += "\n";
      else {
        if ((n.tagName === "DIV" || n.tagName === "P") && out && !out.endsWith("\n")) out += "\n";
        out += serialize(n);
      }
    }
  }
  return out;
}

function caretRect(): Rect | null {
  const sel = window.getSelection();
  if (!sel || sel.rangeCount === 0) return null;
  const r = sel.getRangeAt(0).cloneRange();
  r.collapse(true);
  const box = r.getClientRects()[0] ?? (r.startContainer instanceof Element ? r.startContainer.getBoundingClientRect() : r.startContainer.parentElement?.getBoundingClientRect());
  return box ? { x: box.left, y: box.top, width: 1, height: box.height || 18 } : null;
}

function fuzzy(title: string, q: string): number {
  const a = title.toLowerCase();
  if (!q) return 3;
  if (a.startsWith(q)) return 0;
  if (a.split(/\s+/).some((w) => w.startsWith(q))) return 1;
  return a.includes(q) ? 2 : -1;
}

export interface ComposerOptions {
  workspaceId: () => string;
  placeholder: string;
  submitLabel: string;
  onSubmit: (body: string) => Promise<void>;
  onCancel?: () => void;
  value?: string;
}

export interface Composer {
  el: HTMLElement;
  focus(): void;
  clear(): void;
  isEmpty(): boolean;
}

export function composer(engine: Engine, o: ComposerOptions): Composer {
  const input = h("div", { class: "composer__input", attrs: { contenteditable: "true", role: "textbox", "aria-multiline": "true", "aria-label": o.placeholder, "data-placeholder": o.placeholder, spellcheck: "true" } });
  const send = h("button", { class: "icon-btn composer__send", attrs: { type: "button", "aria-label": o.submitLabel, "data-tip": o.submitLabel } }, iconNode("arrow-up"));
  const actions = h("div", { class: "composer__actions" });
  if (o.onCancel) {
    const cancel = h("button", { class: "link-btn", attrs: { type: "button" }, text: t("Cancel") });
    cancel.addEventListener("click", () => o.onCancel?.());
    actions.append(cancel);
  }
  actions.append(send);
  const el = h("div", { class: "composer" }, input, actions);
  if (o.value) input.append(renderBody(engine, o.value));
  let popover: ListPopover | null = null;
  let busy = false;

  const paintEmpty = () => {
    const empty = !serialize(input).trim();
    setAttr(input, "data-empty", empty ? "true" : null);
    send.disabled = empty || busy;
  };
  paintEmpty();

  const members = (ws: string): Member[] => {
    const hit = memberCache.get(ws);
    if (hit && Date.now() - hit.at < 60_000) return hit.list;
    if (!hit && engine.canWrite(ws)) {
      memberCache.set(ws, { at: 0, list: [] });
      void engine.api.workspaces
        .members(ws)
        .then((r) => {
          memberCache.set(ws, { at: Date.now(), list: r.members });
          refreshPopover();
        })
        .catch(() => memberCache.delete(ws));
    }
    return hit?.list ?? [];
  };

  const query = (): { node: Text; start: number; q: string } | null => {
    const sel = window.getSelection();
    if (!sel || !sel.isCollapsed || !sel.anchorNode || sel.anchorNode.nodeType !== Node.TEXT_NODE || !input.contains(sel.anchorNode)) return null;
    const node = sel.anchorNode as Text;
    const before = (node.textContent ?? "").slice(0, sel.anchorOffset);
    const m = /(?:^|[\s(])@([^\s@]{0,30})$/.exec(before);
    if (!m) return null;
    return { node, start: sel.anchorOffset - (m[1]?.length ?? 0) - 1, q: (m[1] ?? "").toLowerCase() };
  };

  const items = (q: string): ListItem[] => {
    const ws = o.workspaceId();
    const out: ListItem[] = [];
    for (const m of members(ws).filter((m) => !m.disabled && fuzzy(m.name, q) >= 0).slice(0, 6)) {
      out.push({ key: `user:${m.user_id}:${m.name}`, label: m.name, icon: () => avatar(m.name, m.color, false, m.avatar), section: t("People") });
    }
    const pages = ws ? engine.pages.all(ws) : [];
    for (const { p } of pages
      .filter((p) => !p.is_template)
      .map((p) => ({ p, s: fuzzy(p.title || t("Untitled"), q) }))
      .filter((x) => x.s >= 0)
      .sort((a, b) => a.s - b.s || b.p.updated_at - a.p.updated_at)
      .slice(0, 5)) {
      out.push({ key: `page:${p.id}:${p.title || t("Untitled")}`, label: p.title || t("Untitled"), icon: () => pageIcon(p.icon, fallbackGlyph(p.kind)), section: t("Pages") });
    }
    return out;
  };

  const pick = (item: ListItem) => {
    const at = query();
    popover?.close();
    if (!at) return;
    const [kind, id, ...rest] = item.key.split(":");
    const node = chip(engine, kind ?? "user", id ?? "", rest.join(":"));
    const sel = window.getSelection();
    const range = document.createRange();
    range.setStart(at.node, at.start);
    range.setEnd(at.node, sel?.anchorOffset ?? at.start);
    range.deleteContents();
    const space = document.createTextNode(" ");
    range.insertNode(space);
    range.insertNode(node);
    range.setStartAfter(space);
    range.collapse(true);
    sel?.removeAllRanges();
    sel?.addRange(range);
    paintEmpty();
  };

  const refreshPopover = () => {
    const at = query();
    if (!at) {
      popover?.close();
      return;
    }
    if (!popover?.open) {
      popover = new ListPopover({ label: t("Mention"), empty: t("Nothing matches"), rect: caretRect, onPick: pick, onClose: () => (popover = null) });
    }
    popover.set(items(at.q));
  };

  const submit = async () => {
    const body = serialize(input).trim();
    if (!body || busy) return;
    busy = true;
    paintEmpty();
    try {
      await o.onSubmit(body);
      input.replaceChildren();
    } catch (e) {
      toast({ tone: "danger", message: errorMessage(e) });
    } finally {
      busy = false;
      paintEmpty();
    }
  };

  input.addEventListener("input", () => {
    paintEmpty();
    refreshPopover();
  });
  input.addEventListener("keydown", (e) => {
    if (popover?.open && popover.count > 0) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        popover.move(e.key === "ArrowDown" ? 1 : -1);
        e.preventDefault();
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        popover.pick();
        e.preventDefault();
        return;
      }
    }
    if (e.key === "Escape") {
      if (popover?.open) popover.close();
      else o.onCancel?.();
      e.preventDefault();
      e.stopPropagation();
      return;
    }
    if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
      e.preventDefault();
      void submit();
    }
  });
  input.addEventListener("paste", (e) => {
    e.preventDefault();
    const text = e.clipboardData?.getData("text/plain") ?? "";
    document.execCommand("insertText", false, text);
  });
  input.addEventListener("click", (e) => {
    if ((e.target as Element).closest("a.mention")) e.preventDefault();
  });
  send.addEventListener("click", () => void submit());
  return {
    el,
    focus: () => {
      input.focus();
      const sel = window.getSelection();
      if (sel) {
        sel.selectAllChildren(input);
        sel.collapseToEnd();
      }
    },
    clear: () => {
      input.replaceChildren();
      paintEmpty();
    },
    isEmpty: () => !serialize(input).trim(),
  };
}

export interface ThreadOptions {
  workspaceId: () => string;
  onJump?: (thread: CommentThread) => void;
  showQuote?: boolean;
}

export interface ThreadView {
  el: HTMLElement;
  update(thread: CommentThread, listing: CommentListing): void;
  focusReply(): void;
}

export function threadView(engine: Engine, pageId: string, first: CommentThread, o: ThreadOptions): ThreadView {
  const me = () => engine.state.get("me")?.user.id ?? "";
  let thread = first;
  let listing: CommentListing = engine.comments.get(pageId);
  const run = (work: () => Promise<unknown>) => void engine.comments.run(pageId, work).catch((e: unknown) => toast({ tone: "danger", message: errorMessage(e) }));
  const quote = h("button", { class: "cthread__quote", attrs: { type: "button" } });
  quote.addEventListener("click", () => o.onJump?.(thread));
  const resolve = h("button", { class: "icon-btn cthread__resolve", attrs: { type: "button" } }, iconNode("check"));
  resolve.addEventListener("click", () => run(() => engine.api.comments.resolve(thread.id, !thread.resolved_at)));
  const state = h("span", { class: "cthread__state" });
  const head = h("div", { class: "cthread__head" }, quote, state, resolve);
  const list = h("div", { class: "cthread__list" });
  const reply = composer(engine, { workspaceId: o.workspaceId, placeholder: t("Reply"), submitLabel: t("Send reply"), onSubmit: async (body) => {
    await engine.comments.run(pageId, () => engine.api.comments.reply(thread.id, body));
  } });
  const el = h("article", { class: "cthread", attrs: { "data-thread": first.id } }, head, list, reply.el);
  const editing = new Map<string, Composer>();

  const commentRow = (c: PageComment): HTMLElement => {
    const face = avatar(c.author_name, c.author_color, false, c.author_avatar);
    const name = h("span", { class: "cmt__name" });
    const when = h("time", { class: "cmt__time" });
    const more = h("button", { class: "icon-btn cmt__more", attrs: { type: "button", "aria-label": t("Comment actions"), "data-tip": t("Comment actions") } }, iconNode("more"));
    const react = h("button", { class: "icon-btn cmt__react", attrs: { type: "button", "aria-label": t("Add a reaction"), "data-tip": t("Add a reaction") } }, iconNode("smile"));
    const body = h("div", { class: "cmt__body" });
    const reacts = h("div", { class: "cmt__reacts" });
    const row = h("div", { class: "cmt" }, h("div", { class: "cmt__head" }, face, name, when, h("span", { class: "cmt__tools" }, react, more)), body, reacts);
    react.addEventListener("click", () => {
      const cur = thread.comments.find((x) => x.id === row.dataset.key);
      const entries: MenuEntry[] = REACTIONS.map(([icon, label]) => ({ label: label(), icon, checked: !!cur?.reactions.find((r) => r.icon === icon)?.mine, run: () => toggleReaction(row.dataset.key ?? "", icon) }));
      openMenu(react, entries, { side: "bottom", align: "end", label: t("Reactions") });
    });
    more.addEventListener("click", () => {
      const cur = thread.comments.find((x) => x.id === row.dataset.key);
      if (!cur) return;
      const entries: MenuEntry[] = [];
      if (cur.author_id === me() && listing.can_comment) entries.push({ label: t("Edit"), icon: "pencil", run: () => startEdit(row, cur) });
      entries.push({ label: t("Copy text"), icon: "copy", run: () => void navigator.clipboard?.writeText(cur.body.replace(MENTION, "@$1")).catch(() => undefined) });
      if ((cur.author_id === me() && listing.can_comment) || listing.can_moderate) {
        entries.push({ kind: "separator" }, {
          label: t("Delete comment"),
          icon: "trash",
          danger: true,
          run: () =>
            void confirmDialog({ title: t("Delete this comment?"), body: thread.comments.length === 1 ? t("This is the only comment, so the discussion goes too.") : t("Replies stay."), confirm: t("Delete"), variant: "danger" }).then((ok) => {
              if (ok) run(() => engine.api.comments.remove(cur.id));
            }),
        });
      }
      openMenu(more, entries, { side: "bottom", align: "end", label: t("Comment actions") });
    });
    reacts.addEventListener("click", (e) => {
      const b = (e.target as Element).closest<HTMLElement>("[data-icon]");
      if (b?.dataset.icon && listing.can_comment) toggleReaction(row.dataset.key ?? "", b.dataset.icon as ReactionIcon);
    });
    return row;
  };

  const toggleReaction = (id: string, icon: ReactionIcon) => {
    const cur = thread.comments.find((x) => x.id === id);
    const on = !cur?.reactions.find((r) => r.icon === icon)?.mine;
    run(() => engine.api.comments.react(id, icon, on));
  };

  const startEdit = (row: HTMLElement, c: PageComment) => {
    if (editing.has(c.id)) return;
    const body = row.querySelector(".cmt__body") as HTMLElement;
    const ed = composer(engine, {
      workspaceId: o.workspaceId,
      placeholder: t("Edit comment"),
      submitLabel: t("Save"),
      value: c.body,
      onSubmit: async (next) => {
        await engine.comments.run(pageId, () => engine.api.comments.edit(c.id, next));
        stop();
      },
      onCancel: () => stop(),
    });
    const stop = () => {
      editing.delete(c.id);
      ed.el.remove();
      body.hidden = false;
    };
    editing.set(c.id, ed);
    body.hidden = true;
    body.after(ed.el);
    ed.focus();
  };

  const updateRow = (row: HTMLElement, c: PageComment) => {
    updateAvatar(row.querySelector<HTMLElement>(".avatar") as HTMLElement, c.author_name, c.author_color, c.author_avatar);
    setText(row.querySelector(".cmt__name") as HTMLElement, c.author_name || t("Someone"));
    const when = row.querySelector(".cmt__time") as HTMLElement;
    setText(when, formatRelative(c.created_at) + (c.edited_at ? ` · ${t("edited")}` : ""));
    setAttr(when, "title", formatDateTime(c.created_at));
    const body = row.querySelector(".cmt__body") as HTMLElement;
    if (body.dataset.src !== c.body) {
      body.dataset.src = c.body;
      body.replaceChildren(renderBody(engine, c.body));
    }
    const tools = row.querySelector(".cmt__tools") as HTMLElement;
    tools.hidden = !listing.can_comment;
    const reacts = row.querySelector(".cmt__reacts") as HTMLElement;
    reacts.hidden = c.reactions.length === 0;
    reconcile(
      reacts,
      c.reactions,
      (r) => r.icon,
      (r) => h("button", { class: "react", attrs: { type: "button", "data-icon": r.icon } }, iconNode(r.icon), h("span", { class: "react__n" })),
      (b, r) => {
        setText(b.querySelector(".react__n") as HTMLElement, String(r.count));
        setAttr(b, "aria-pressed", r.mine ? "true" : "false");
        const label = REACTIONS.find(([i]) => i === r.icon)?.[1]() ?? r.icon;
        setAttr(b, "title", `${label}: ${r.names.join(", ")}`);
        setAttr(b, "aria-label", `${label}, ${r.count}`);
        (b as HTMLButtonElement).disabled = !listing.can_comment;
      },
    );
  };

  const update = (next: CommentThread, l: CommentListing) => {
    thread = next;
    listing = l;
    const anchored = next.anchor !== "page";
    quote.hidden = !anchored || o.showQuote === false;
    setText(quote, next.quote || (next.anchor === "block" ? t("A block") : t("Selected text")));
    setAttr(el, "data-resolved", next.resolved_at ? "true" : null);
    setText(state, next.resolved_at ? t("Resolved") : "");
    const label = next.resolved_at ? t("Reopen") : t("Resolve");
    setAttr(resolve, "aria-label", label);
    setAttr(resolve, "data-tip", label);
    setAttr(resolve, "aria-pressed", next.resolved_at ? "true" : "false");
    resolve.hidden = !l.can_comment;
    reply.el.hidden = !l.can_comment || !!next.resolved_at;
    reconcile(list, next.comments, (c) => c.id, commentRow, updateRow);
  };
  update(first, listing);
  return { el, update, focusReply: () => reply.focus() };
}

export interface SuggestionOptions {
  onDecide: (s: Suggestion, accept: boolean) => void;
  onJump?: (s: Suggestion) => void;
}

export interface SuggestionView {
  el: HTMLElement;
  update(s: Suggestion, listing: CommentListing): void;
}

export function suggestionView(engine: Engine, pageId: string, first: Suggestion, o: SuggestionOptions): SuggestionView {
  let cur = first;
  const face = avatar(first.author_name, first.author_color, false, first.author_avatar);
  const name = h("span", { class: "cmt__name" });
  const when = h("time", { class: "cmt__time" });
  const what = h("button", { class: "sugg__what", attrs: { type: "button" } });
  what.addEventListener("click", () => o.onJump?.(cur));
  const accept = h("button", { class: "btn btn--primary sugg__accept", attrs: { type: "button" } }, iconNode("check"), h("span", { text: t("Accept") }));
  const reject = h("button", { class: "btn btn--ghost sugg__reject", attrs: { type: "button" } }, iconNode("close"), h("span", { text: t("Reject") }));
  const state = h("span", { class: "sugg__state" });
  accept.addEventListener("click", () => o.onDecide(cur, true));
  reject.addEventListener("click", () => o.onDecide(cur, false));
  const el = h("article", { class: "sugg", attrs: { "data-suggestion": first.id } }, h("div", { class: "cmt__head" }, face, name, when, h("span")), what, h("div", { class: "sugg__actions" }, state, reject, accept));
  const quote = (s: string) => `“${s.length > 80 ? `${s.slice(0, 80)}…` : s}”`;
  const update = (s: Suggestion, l: CommentListing) => {
    cur = s;
    const me = engine.state.get("me")?.user.id ?? "";
    updateAvatar(face, s.author_name, s.author_color, s.author_avatar);
    setText(name, s.author_name || t("Someone"));
    setText(when, formatRelative(s.updated_at));
    setAttr(when, "title", formatDateTime(s.updated_at));
    const label = s.text && s.original ? t("Replace {old} with {new}", { old: quote(s.original), new: quote(s.text) }) : s.text ? t("Add {new}", { new: quote(s.text) }) : t("Delete {old}", { old: quote(s.original) });
    setText(what, label);
    setAttr(el, "data-status", s.status);
    const open = s.status === "open";
    accept.hidden = !open || !l.can_decide;
    reject.hidden = !open || !(l.can_decide || s.author_id === me);
    setText(reject.lastElementChild as HTMLElement, l.can_decide ? t("Reject") : t("Withdraw"));
    setText(state, s.status === "accepted" ? t("Accepted") : s.status === "rejected" ? t("Rejected") : "");
  };
  update(first, engine.comments.get(pageId));
  return { el, update };
}

export function reconcileThreads(engine: Engine, pageId: string, container: HTMLElement, threads: CommentThread[], listing: CommentListing, o: ThreadOptions, views: Map<string, ThreadView>): void {
  reconcile(
    container,
    threads,
    (x) => x.id,
    (x) => {
      const v = threadView(engine, pageId, x, o);
      views.set(x.id, v);
      return v.el;
    },
    (_el, x) => views.get(x.id)?.update(x, listing),
  );
  for (const id of [...views.keys()]) if (!threads.some((x) => x.id === id)) views.delete(id);
}
