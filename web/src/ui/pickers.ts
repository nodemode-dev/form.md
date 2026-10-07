import type { Engine, FilePurpose } from "../engine";
import { t } from "../engine/i18n";
import { byId, h, setAttr } from "./dom";
import { errorMessage } from "./errors";
import { anchorTo } from "./floating";
import { GLYPH_NAMES, glyphNode, glyphWords, iconRef, parseIcon, TINTS } from "./glyphs";
import { iconNode } from "./icons";
import { openLayer } from "./layers";
import { toast } from "./toast";

export const COVERS = ["dawn", "lagoon", "meadow", "ember", "dusk", "glacier", "orchard", "slate", "aurora", "sand", "tide", "plum"] as const;

const COVER_LABELS: Record<(typeof COVERS)[number], () => string> = {
  dawn: () => t("Dawn"),
  lagoon: () => t("Lagoon"),
  meadow: () => t("Meadow"),
  ember: () => t("Ember"),
  dusk: () => t("Dusk"),
  glacier: () => t("Glacier"),
  orchard: () => t("Orchard"),
  slate: () => t("Slate"),
  aurora: () => t("Aurora"),
  sand: () => t("Sand"),
  tide: () => t("Tide"),
  plum: () => t("Plum"),
};

export interface UploadTarget {
  purpose: FilePurpose;
  workspace_id?: string;
  page_id?: string;
}

export function pickFile(accept: string): Promise<File | null> {
  return new Promise((resolve) => {
    const input = h("input", { attrs: { type: "file", accept } });
    input.addEventListener("change", () => resolve(input.files?.[0] ?? null), { once: true });
    input.addEventListener("cancel", () => resolve(null), { once: true });
    input.click();
  });
}

export async function uploadPicture(engine: Engine, target: UploadTarget, accept = "image/png,image/jpeg,image/gif,image/webp"): Promise<string | null> {
  const file = await pickFile(accept);
  if (!file) return null;
  const id = `upload-${target.purpose}`;
  toast({ id, message: t("Uploading {name}", { name: file.name }), durationMs: 0 });
  try {
    const body: { workspace_id?: string; page_id?: string } = {};
    if (target.workspace_id) body.workspace_id = target.workspace_id;
    if (target.page_id) body.page_id = target.page_id;
    const r = await engine.api.files.upload(file, target.purpose, body);
    toast({ id, tone: "success", message: t("Uploaded {name}", { name: file.name }) });
    return r.file.ref;
  } catch (e) {
    toast({ id, tone: "danger", message: errorMessage(e) });
    return null;
  }
}

function popover(anchor: HTMLElement, label: string, body: HTMLElement[], initial: () => HTMLElement | null): () => void {
  const panel = h("div", { class: "popover picker", attrs: { role: "dialog", "aria-label": label } }, ...body);
  byId("overlay-root").append(panel);
  const release = anchorTo(panel, anchor, { side: "bottom", align: "start", offset: 6 });
  requestAnimationFrame(() => panel.classList.add("is-visible"));
  setAttr(anchor, "aria-expanded", "true");
  let done = false;
  const close = openLayer({
    el: panel,
    modal: false,
    dismissOnOutside: true,
    returnFocus: anchor,
    initialFocus: initial() ?? panel,
    onClose: () => {
      if (done) return;
      done = true;
      release();
      setAttr(anchor, "aria-expanded", "false");
      panel.remove();
    },
  });
  return close;
}

export interface IconPickerOptions {
  current: string;
  upload: UploadTarget;
  allowRemove: boolean;
  onPick: (ref: string) => void;
  onRemove: () => void;
}

export function openIconPicker(engine: Engine, anchor: HTMLElement, o: IconPickerOptions): void {
  const cur = parseIcon(o.current);
  let tint = cur.kind === "glyph" ? cur.tint : "";
  const search = h("input", { class: "input picker__search", attrs: { type: "search", placeholder: t("Search icons"), "aria-label": t("Search icons"), autocomplete: "off", spellcheck: "false" } });
  const tints = h("div", { class: "picker__tints", attrs: { role: "radiogroup", "aria-label": t("Colour") } });
  for (const name of ["", ...TINTS]) {
    const b = h("button", { class: "picker__tint", attrs: { type: "button", role: "radio", "data-tint": name || "none", "aria-label": name ? name : t("No colour"), "aria-checked": name === tint ? "true" : "false" } });
    if (name) b.dataset.color = name;
    tints.append(b);
  }
  const grid = h("div", { class: "picker__grid", attrs: { role: "listbox", "aria-label": t("Icons") } });
  if (tint) grid.dataset.color = tint;
  for (const name of GLYPH_NAMES) {
    const b = h("button", { class: "picker__glyph", attrs: { type: "button", role: "option", tabindex: "-1", "data-name": name, "aria-label": name.replace(/-/g, " "), "data-tip": name.replace(/-/g, " "), "aria-selected": cur.kind === "glyph" && cur.name === name ? "true" : "false" } }, glyphNode(name));
    grid.append(b);
  }
  const none = h("p", { class: "picker__empty", text: t("No icon matches.") });
  none.hidden = true;
  const upload = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("upload"), h("span", { text: t("Upload picture") }));
  const remove = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("trash"), h("span", { text: t("Remove icon") }));
  remove.hidden = !o.allowRemove;
  const foot = h("div", { class: "picker__foot" }, upload, remove);

  const visible = () => [...grid.querySelectorAll<HTMLButtonElement>(".picker__glyph")].filter((b) => !b.hidden);
  const focusAt = (i: number) => {
    const list = visible();
    if (!list.length) return;
    const b = list[Math.max(0, Math.min(list.length - 1, i))];
    for (const x of list) x.tabIndex = -1;
    if (b) {
      b.tabIndex = 0;
      b.focus();
    }
  };

  const close = popover(anchor, t("Choose an icon"), [h("div", { class: "picker__head" }, search), tints, grid, none, foot], () => search);

  search.addEventListener("input", () => {
    const q = search.value.trim().toLowerCase();
    let shown = 0;
    for (const b of grid.querySelectorAll<HTMLButtonElement>(".picker__glyph")) {
      const hit = !q || glyphWords(b.dataset.name ?? "").includes(q);
      if (b.hidden === hit) b.hidden = !hit;
      if (hit) shown++;
    }
    none.hidden = shown > 0;
  });
  search.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      focusAt(0);
    } else if (e.key === "Enter") {
      e.preventDefault();
      const first = visible()[0];
      if (first) pick(first.dataset.name ?? "");
    }
  });
  tints.addEventListener("click", (e) => {
    const b = (e.target as HTMLElement).closest<HTMLElement>(".picker__tint");
    if (!b) return;
    tint = b.dataset.tint === "none" ? "" : (b.dataset.tint ?? "");
    for (const x of tints.children) setAttr(x, "aria-checked", x === b ? "true" : "false");
    setAttr(grid, "data-color", tint || null);
    if (cur.kind === "glyph") o.onPick(iconRef(cur.name, tint));
  });
  const pick = (name: string) => {
    if (!name) return;
    close();
    o.onPick(iconRef(name, tint));
  };
  grid.addEventListener("click", (e) => {
    const b = (e.target as HTMLElement).closest<HTMLElement>(".picker__glyph");
    if (b) pick(b.dataset.name ?? "");
  });
  grid.addEventListener("keydown", (e) => {
    const list = visible();
    const i = list.indexOf(document.activeElement as HTMLButtonElement);
    const cols = 8;
    const moves: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: cols, ArrowUp: -cols };
    const d = moves[e.key];
    if (d !== undefined) {
      e.preventDefault();
      if (e.key === "ArrowUp" && i < cols) search.focus();
      else focusAt(i + d);
    } else if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      pick(list[i]?.dataset.name ?? "");
    }
  });
  upload.addEventListener("click", () => {
    close();
    void uploadPicture(engine, o.upload, o.upload.purpose === "icon" || o.upload.purpose === "workspace" ? "image/png,image/jpeg,image/gif,image/webp,image/svg+xml" : undefined).then((ref) => ref && o.onPick(ref));
  });
  remove.addEventListener("click", () => {
    close();
    o.onRemove();
  });
}

export interface CoverPickerOptions {
  current: string;
  pageId: string;
  onPick: (ref: string) => void;
  onRemove: () => void;
}

export function openCoverPicker(engine: Engine, anchor: HTMLElement, o: CoverPickerOptions): void {
  const grid = h("div", { class: "picker__covers", attrs: { role: "listbox", "aria-label": t("Gradients") } });
  for (const name of COVERS) {
    grid.append(h("button", { class: "picker__cover", attrs: { type: "button", role: "option", "data-cover": name, "aria-label": COVER_LABELS[name](), "data-tip": COVER_LABELS[name](), "aria-selected": o.current === `g:${name}` ? "true" : "false" } }));
  }
  const upload = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("upload"), h("span", { text: t("Upload image") }));
  const remove = h("button", { class: "btn btn--ghost", attrs: { type: "button" } }, iconNode("trash"), h("span", { text: t("Remove cover") }));
  remove.hidden = !o.current;
  const close = popover(anchor, t("Choose a cover"), [h("p", { class: "picker__title", text: t("Gradients") }), grid, h("div", { class: "picker__foot" }, upload, remove)], () => grid.querySelector<HTMLElement>('[aria-selected="true"]') ?? grid.querySelector<HTMLElement>("button"));
  grid.addEventListener("click", (e) => {
    const b = (e.target as HTMLElement).closest<HTMLElement>(".picker__cover");
    if (!b) return;
    close();
    o.onPick(`g:${b.dataset.cover ?? ""}`);
  });
  grid.addEventListener("keydown", (e) => {
    const list = [...grid.querySelectorAll<HTMLButtonElement>(".picker__cover")];
    const i = list.indexOf(document.activeElement as HTMLButtonElement);
    const moves: Record<string, number> = { ArrowRight: 1, ArrowLeft: -1, ArrowDown: 4, ArrowUp: -4 };
    const d = moves[e.key];
    if (d === undefined) return;
    e.preventDefault();
    list[Math.max(0, Math.min(list.length - 1, i + d))]?.focus();
  });
  upload.addEventListener("click", () => {
    close();
    void uploadPicture(engine, { purpose: "cover", page_id: o.pageId }).then((ref) => ref && o.onPick(ref));
  });
  remove.addEventListener("click", () => {
    close();
    o.onRemove();
  });
}
