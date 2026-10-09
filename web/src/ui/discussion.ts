import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { type Composer, composer, reconcileThreads, type ThreadView } from "./comment-view";
import { h } from "./dom";

export interface Discussion {
  el: HTMLElement;
  show(pageId: string): void;
  openComposer(): void;
  hasThreads(): boolean;
}

export function mountDiscussion(engine: Engine, onChange: () => void): Discussion {
  const list = h("div", { class: "page-discussion__list" });
  const fresh = h("div", { class: "page-discussion__new" });
  const el = h("section", { class: "page-discussion", attrs: { "aria-label": t("Comments on this page") } }, list, fresh);
  el.hidden = true;
  const views = new Map<string, ThreadView>();
  let pageId = "";
  let unwatch: (() => void) | null = null;
  let open = false;
  let box: Composer | null = null;

  const workspaceId = () => engine.pages.get(pageId)?.workspace_id ?? engine.state.get("workspaceId");

  const pageThreads = () => engine.comments.get(pageId).threads.filter((x) => x.anchor === "page" && !x.resolved_at);

  const render = () => {
    const listing = engine.comments.get(pageId);
    const threads = pageThreads();
    reconcileThreads(engine, pageId, list, threads, listing, { workspaceId }, views);
    const showNew = listing.can_comment && (open || threads.length > 0);
    if (showNew && !box) {
      box = composer(engine, {
        workspaceId,
        placeholder: t("Add a comment"),
        submitLabel: t("Comment"),
        onSubmit: async (body) => {
          await engine.comments.run(pageId, () => engine.api.comments.create(pageId, { anchor: "page", body }));
        },
        onCancel: () => {
          if (pageThreads().length === 0 && box?.isEmpty()) {
            open = false;
            render();
            onChange();
          }
        },
      });
      fresh.replaceChildren(box.el);
    } else if (!showNew && box) {
      box = null;
      fresh.replaceChildren();
    }
    el.hidden = threads.length === 0 && !showNew;
  };

  engine.comments.subscribe((id) => {
    if (id !== pageId) return;
    render();
    onChange();
  });

  return {
    el,
    show(id) {
      if (id === pageId) return;
      unwatch?.();
      pageId = id;
      open = false;
      box = null;
      fresh.replaceChildren();
      unwatch = engine.comments.watch(id);
      render();
      if (!engine.comments.has(id)) void engine.comments.load(id).catch(() => undefined);
    },
    openComposer() {
      open = true;
      render();
      onChange();
      box?.focus();
    },
    hasThreads: () => pageThreads().length > 0,
  };
}
