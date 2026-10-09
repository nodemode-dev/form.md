import type { Node as PMNode } from "prosemirror-model";
import { DOMSerializer } from "prosemirror-model";
import { yXmlFragmentToProseMirrorRootNode } from "y-prosemirror";
import * as Y from "yjs";
import type { Engine, PageVersion } from "../engine";
import { schema } from "../engine/doc/schema";
import { formatDateTime, formatRelative } from "../engine/format";
import { plural, t } from "../engine/i18n";
import { editorHost } from "../extensions/blocks";
import { confirmDialog, simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { busy, button } from "./form";
import { reconcile } from "./list";
import { toast } from "./toast";

function decode(b64: string): Uint8Array {
  const raw = atob(b64);
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

function versionDoc(state: string): PMNode {
  const ydoc = new Y.Doc();
  try {
    Y.applyUpdate(ydoc, decode(state));
    return yXmlFragmentToProseMirrorRootNode(ydoc.getXmlFragment("prosemirror"), schema);
  } finally {
    ydoc.destroy();
  }
}

type Diff = "same" | "changed" | "gone";

function compare(version: PMNode, now: PMNode | null): { marks: Diff[]; removed: number } {
  if (!now) return { marks: version.content.content.map(() => "same"), removed: 0 };
  const byId = new Map<string, PMNode>();
  now.forEach((n) => {
    const id = String(n.attrs.id ?? "");
    if (id) byId.set(id, n);
  });
  const seen = new Set<string>();
  const marks: Diff[] = [];
  version.forEach((n, _off, i) => {
    const id = String(n.attrs.id ?? "");
    const other = id ? byId.get(id) : now.maybeChild(i);
    if (id) seen.add(id);
    marks.push(!other ? "gone" : other.eq(n) ? "same" : "changed");
  });
  let removed = 0;
  now.forEach((n) => {
    const id = String(n.attrs.id ?? "");
    if (id && !seen.has(id)) removed++;
  });
  return { marks, removed };
}

const kindLabel = (v: PageVersion) => (v.kind === "manual" ? t("Saved by hand") : v.kind === "restore" ? t("Before a restore") : t("Saved while editing"));

export function openHistory(engine: Engine, pageId: string): void {
  const host = editorHost();
  const canEdit = () => !!host && host.current() === pageId && host.canEdit();
  simpleDialog(
    t("Page history"),
    (body, close) => {
      const list = h("nav", { class: "dialog__nav hv-list", attrs: { role: "listbox", "aria-label": t("Versions") } });
      const heading = h("h3", { class: "dialog__panel-title" });
      const meta = h("p", { class: "muted hv-meta" });
      const summary = h("p", { class: "hv-summary" });
      const preview = h("div", { class: "doc-body hv-preview", attrs: { "aria-label": t("Version preview") } });
      const restore = button(t("Restore this version"), "primary", "restore");
      const save = button(t("Save the current version"), "ghost");
      const empty = h("p", { class: "muted", text: t("No versions yet. A version is kept every ten minutes while someone edits, and when editing stops.") });
      const panel = h("section", { class: "dialog__panel hv-panel" }, h("div", { class: "hv-head" }, h("div", {}, heading, meta), h("div", { class: "hv-actions" }, save, restore)), summary, preview, empty);
      body.append(list, h("div", { class: "dialog__panels" }, panel));
      let versions: PageVersion[] = [];
      let selected = "";
      let current: PMNode | null = null;
      let seq = 0;

      const show = async (id: string) => {
        selected = id;
        for (const el of Array.from(list.children) as HTMLElement[]) setAttr(el, "aria-selected", el.dataset.key === id ? "true" : "false");
        const v = versions.find((x) => x.id === id);
        if (!v) return;
        setText(heading, formatDateTime(v.created_at));
        setText(meta, [kindLabel(v), v.names.length ? v.names.join(", ") : "", plural(v.words, "{count} word", "{count} words")].filter(Boolean).join(" · "));
        const mine = ++seq;
        try {
          const r = await engine.api.history.get(id);
          if (mine !== seq) return;
          const doc = versionDoc(r.state);
          current = doc;
          const live = host?.current() === pageId ? (host.view()?.state.doc ?? null) : null;
          const diff = compare(doc, live);
          const frag = DOMSerializer.fromSchema(schema).serializeFragment(doc.content);
          const els = Array.from(frag.childNodes);
          els.forEach((el, i) => {
            if (el instanceof HTMLElement) setAttr(el, "data-diff", diff.marks[i] === "same" ? null : (diff.marks[i] ?? null));
          });
          preview.replaceChildren(frag);
          const changed = diff.marks.filter((m) => m !== "same").length;
          setText(
            summary,
            !live ? "" : changed === 0 && diff.removed === 0 ? t("Same as the page now.") : [changed ? plural(changed, "{count} block differs from now", "{count} blocks differ from now") : "", diff.removed ? plural(diff.removed, "{count} block added since would go", "{count} blocks added since would go") : ""].filter(Boolean).join(" · "),
          );
          summary.hidden = !summary.textContent;
          restore.disabled = !canEdit();
        } catch (e) {
          preview.replaceChildren(h("p", { class: "muted", text: errorMessage(e) }));
        }
      };

      const paint = () => {
        reconcile(
          list,
          versions,
          (v) => v.id,
          () => h("button", { class: "dialog__tab hv-item", attrs: { type: "button", role: "option" } }, h("span", { class: "hv-item__when" }), h("span", { class: "hv-item__who" })),
          (el, v) => {
            setText(el.firstElementChild as HTMLElement, formatRelative(v.created_at));
            setAttr(el, "title", formatDateTime(v.created_at));
            setText(el.lastElementChild as HTMLElement, v.names.join(", ") || kindLabel(v));
            setAttr(el, "aria-selected", v.id === selected ? "true" : "false");
          },
        );
        empty.hidden = versions.length > 0;
        restore.hidden = versions.length === 0 || !canEdit();
        save.hidden = !canEdit();
      };

      const load = async (pick?: string) => {
        try {
          versions = (await engine.api.history.list(pageId)).versions;
          paint();
          const first = pick ?? selected ?? versions[0]?.id;
          if (first && versions.some((v) => v.id === first)) await show(first);
          else if (versions[0]) await show(versions[0].id);
        } catch (e) {
          toast({ tone: "danger", message: errorMessage(e) });
        }
      };

      list.addEventListener("click", (e) => {
        const el = (e.target as Element).closest<HTMLElement>(".hv-item");
        if (el?.dataset.key) void show(el.dataset.key);
      });
      list.addEventListener("keydown", (e) => {
        const i = versions.findIndex((v) => v.id === selected);
        const next = e.key === "ArrowDown" ? versions[i + 1] : e.key === "ArrowUp" ? versions[i - 1] : undefined;
        if (!next) return;
        e.preventDefault();
        void show(next.id).then(() => list.querySelector<HTMLElement>(`[data-key="${next.id}"]`)?.focus());
      });
      save.addEventListener("click", async () => {
        try {
          const r = await busy(save, () => engine.api.history.save(pageId));
          toast({ tone: "success", message: t("Version saved.") });
          await load(r.id);
        } catch (e) {
          toast({ tone: "danger", message: errorMessage(e) });
        }
      });
      restore.addEventListener("click", async () => {
        const v = versions.find((x) => x.id === selected);
        const doc = current;
        if (!v || !doc || !canEdit()) return;
        const ok = await confirmDialog({
          title: t("Restore this version?"),
          body: t("The page goes back to how it was at {when}. The current version is saved first, and blocks someone else is editing right now stay as they are.", { when: formatDateTime(v.created_at) }),
          confirm: t("Restore"),
        });
        if (!ok) return;
        try {
          const kept = await busy(restore, async () => {
            await engine.api.history.save(pageId, true);
            const done = editorHost()?.restoreDoc(doc);
            if (!done) throw new Error(t("Open the page in Editing mode to restore."));
            if (v.title) engine.pages.edit(pageId, { title: v.title }, false);
            await engine.api.history.restored(v.id);
            return done.kept;
          });
          close();
          const when = formatDateTime(v.created_at);
          toast({ tone: "success", message: kept ? plural(kept, "Restored the version from {when}. {count} block someone is editing was kept.", "Restored the version from {when}. {count} blocks people are editing were kept.", { when }) : t("Restored the version from {when}.", { when }) });
        } catch (e) {
          toast({ tone: "danger", message: errorMessage(e) });
        }
      });
      void load();
      return list;
    },
    undefined,
    "lg",
  );
}
