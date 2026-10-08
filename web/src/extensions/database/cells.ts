import type { Engine } from "../../engine";
import type { QuerySession } from "../../engine/collections";
import { formatDateTime } from "../../engine/format";
import { t } from "../../engine/i18n";
import type { Collection, ComputedValue, Property, PropOption, RowData } from "../../engine/types";
import { avatar } from "../../ui/avatar";
import { h, setAttr, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { fileUrl, pageIcon } from "../../ui/pageicon";
import { asList, computedText, type DateValue, formatNumber, formatRange } from "./props";

export interface DbCtx {
  engine: Engine;
  colId: string;
  col(): Collection;
  session: QuerySession;
  canWrite(): boolean;
  canDesign(): boolean;
  workspaceId(): string;
}

export type CellMode = "table" | "card" | "list" | "panel";

export function optionChip(o: PropOption | undefined, status = false): HTMLElement {
  const el = h("span", { class: status ? "opt opt--status" : "opt" });
  if (status) el.append(h("span", { class: "opt__dot" }));
  el.append(h("span", { class: "opt__name", text: o?.name ?? "" }));
  setAttr(el, "data-color", o?.color ?? "default");
  return el;
}

function personNode(ctx: DbCtx, id: string, withName: boolean): HTMLElement {
  const m = ctx.engine.collections.person(ctx.workspaceId(), id);
  const name = m?.name ?? (id ? t("Someone") : "");
  const el = h("span", { class: "person" }, avatar(name, m?.color ?? "violet", false, m?.avatar ?? ""));
  if (withName) el.append(h("span", { class: "person__name", text: name }));
  return el;
}

function pageChip(ctx: DbCtx, id: string, title: string, icon: string): HTMLElement {
  const a = h("a", { class: "ref", attrs: { href: `/p/${id}`, "data-id": id } }, pageIcon(icon, "page"), h("span", { class: "ref__title", text: title || t("Untitled") }));
  void ctx;
  return a;
}

function numberNode(n: number, p: Property): HTMLElement {
  const text = formatNumber(n, p);
  if (p.display === "bar" || p.display === "ring") {
    const max = p.max && p.max > 0 ? p.max : 100;
    const ratio = Math.max(0, Math.min(1, n / max));
    if (p.display === "bar") {
      const fill = h("span", { class: "nbar__fill" });
      fill.style.width = `${Math.round(ratio * 100)}%`;
      const bar = h("span", { class: "nbar" }, fill);
      setAttr(bar, "data-color", p.color ?? "default");
      return h("span", { class: "num num--bar" }, h("span", { class: "num__text", text }), bar);
    }
    const ns = "http://www.w3.org/2000/svg";
    const svg = document.createElementNS(ns, "svg");
    svg.setAttribute("viewBox", "0 0 20 20");
    svg.setAttribute("class", "nring");
    const bg = document.createElementNS(ns, "circle");
    bg.setAttribute("cx", "10");
    bg.setAttribute("cy", "10");
    bg.setAttribute("r", "8");
    bg.setAttribute("class", "nring__bg");
    const fg = document.createElementNS(ns, "circle");
    fg.setAttribute("cx", "10");
    fg.setAttribute("cy", "10");
    fg.setAttribute("r", "8");
    fg.setAttribute("class", "nring__fg");
    const c = 2 * Math.PI * 8;
    fg.setAttribute("stroke-dasharray", `${(ratio * c).toFixed(2)} ${c.toFixed(2)}`);
    svg.append(bg, fg);
    const wrap = h("span", { class: "num num--ring" });
    setAttr(wrap, "data-color", p.color ?? "default");
    wrap.append(svg, h("span", { class: "num__text", text }));
    return wrap;
  }
  return h("span", { class: "num", text });
}

function computedNode(ctx: DbCtx, v: ComputedValue | null | undefined, p: Property): HTMLElement | string {
  if (!v) return "";
  switch (v.type) {
    case "number":
      return numberNode(v.value as number, p);
    case "boolean":
      return checkNode(!!v.value);
    case "text": {
      const span = h("span", { class: "ftext", text: String(v.value ?? "") });
      if (v.style) for (const s of v.style.split(",")) span.classList.add(`fs-${s.replace(/_/g, "-")}`);
      if (v.href && /^(https?:|mailto:|\/)/.test(v.href)) {
        const a = h("a", { class: "ftext ftext--link", attrs: { href: v.href, target: "_blank", rel: "noopener noreferrer" }, text: String(v.value ?? "") });
        return a;
      }
      return span;
    }
    case "list": {
      const wrap = h("span", { class: "chips" });
      for (const item of v.value as (ComputedValue | null)[]) {
        if (!item) continue;
        if (item.type === "page") {
          const pg = item.value as { id: string; title: string };
          const ref = ctx.session.refs.get(pg.id);
          wrap.append(pageChip(ctx, pg.id, pg.title, ref?.icon ?? ""));
        } else if (item.type === "person") wrap.append(personNode(ctx, (item.value as { id: string }).id, true));
        else wrap.append(h("span", { class: "chip", text: computedText(item, p) }));
      }
      return wrap;
    }
    case "person":
      return personNode(ctx, (v.value as { id: string }).id, true);
    case "page": {
      const pg = v.value as { id: string; title: string };
      return pageChip(ctx, pg.id, pg.title, "");
    }
    case "error":
      return h("span", { class: "cell-error", text: String(v.value) });
  }
  return h("span", { text: computedText(v, p) });
}

function checkNode(on: boolean): HTMLElement {
  const box = h("span", { class: "cbx", attrs: { role: "checkbox", "aria-checked": on ? "true" : "false" } });
  if (on) box.append(iconNode("check"));
  return box;
}

function sigOf(ctx: DbCtx, p: Property, row: RowData, mode: CellMode): string {
  const v = p.type === "title" ? [row.title, row.icon] : row.values[p.id];
  let extra = "";
  if (p.type === "relation") extra = asList(v).map((id) => ctx.session.refs.get(id)?.title ?? "").join("\u0001");
  if (p.type === "person" || p.type === "created_by" || p.type === "last_edited_by") extra = asList(v).map((id) => ctx.engine.collections.person(ctx.workspaceId(), id)?.name ?? "").join("\u0001");
  return JSON.stringify([p.type, p.format, p.display, p.max, p.color, p.precision, p.prefix, p.options?.length, mode, v, extra]);
}

export function paintCell(el: HTMLElement, p: Property, row: RowData, ctx: DbCtx, mode: CellMode = "table"): void {
  const sig = sigOf(ctx, p, row, mode);
  if (el.dataset.sig === sig) return;
  el.dataset.sig = sig;
  const node = renderValue(ctx, p, row, mode);
  if (typeof node === "string") {
    if (el.firstElementChild) el.replaceChildren();
    setText(el, node);
  } else el.replaceChildren(node);
  setAttr(el, "data-empty", el.textContent === "" && !el.querySelector(".cbx,.nring,img") ? "true" : null);
}

export function renderValue(ctx: DbCtx, p: Property, row: RowData, mode: CellMode): HTMLElement | string {
  const v = row.values[p.id];
  switch (p.type) {
    case "title": {
      const wrap = h("span", { class: "cell-title" });
      if (mode !== "table" || row.icon) wrap.append(pageIcon(row.icon, "page"));
      wrap.append(h("span", { class: "cell-title__text", text: row.title || (mode === "panel" ? "" : t("Untitled")) }));
      if (!row.title) wrap.dataset.untitled = "true";
      return wrap;
    }
    case "text":
      return typeof v === "string" ? v : "";
    case "number":
      return typeof v === "number" ? numberNode(v, p) : "";
    case "select":
    case "status": {
      const id = typeof v === "string" ? v : "";
      if (!id) return "";
      const o = p.options?.find((x) => x.id === id);
      return o ? optionChip(o, p.type === "status") : "";
    }
    case "multi_select": {
      const ids = asList(v);
      if (!ids.length) return "";
      const wrap = h("span", { class: "chips" });
      for (const id of ids) {
        const o = p.options?.find((x) => x.id === id);
        if (o) wrap.append(optionChip(o));
      }
      return wrap;
    }
    case "date": {
      const d = v as DateValue | undefined;
      if (!d?.start) return "";
      const span = h("span", { class: "date", text: formatRange(d, p) });
      if (d.remind) span.append(iconNode("clock", "date__remind"));
      return span;
    }
    case "person": {
      const ids = asList(v);
      if (!ids.length) return "";
      const wrap = h("span", { class: "people" });
      for (const id of ids) wrap.append(personNode(ctx, id, ids.length === 1 || mode === "panel"));
      return wrap;
    }
    case "files": {
      const files = Array.isArray(v) ? (v as { id?: string; url?: string; name: string; mime?: string }[]) : [];
      if (!files.length) return "";
      const wrap = h("span", { class: "chips" });
      for (const f of files) {
        const href = f.id ? fileUrl(f.id) : (f.url ?? "");
        const a = h("a", { class: "file-chip", attrs: { href, target: "_blank", rel: "noopener noreferrer" } }, iconNode("attachment"), h("span", { text: f.name }));
        wrap.append(a);
      }
      return wrap;
    }
    case "checkbox":
      return checkNode(v === true);
    case "url":
    case "email":
    case "phone": {
      const s = typeof v === "string" ? v : "";
      if (!s) return "";
      const href = p.type === "email" ? `mailto:${s}` : p.type === "phone" ? `tel:${s.replace(/[^\d+]/g, "")}` : /^https?:\/\//i.test(s) ? s : `https://${s}`;
      return h("a", { class: "link-cell", attrs: { href, target: "_blank", rel: "noopener noreferrer" }, text: p.type === "url" ? s.replace(/^https?:\/\//i, "") : s });
    }
    case "relation": {
      const ids = asList(v);
      if (!ids.length) return "";
      const wrap = h("span", { class: "chips" });
      for (const id of ids) {
        const ref = ctx.session.refs.get(id);
        wrap.append(pageChip(ctx, id, ref?.title ?? "", ref?.icon ?? ""));
      }
      return wrap;
    }
    case "rollup":
    case "formula":
      return computedNode(ctx, v as ComputedValue | null, p);
    case "created_time":
    case "last_edited_time":
      return typeof v === "number" && v ? formatDateTime(v) : "";
    case "created_by":
    case "last_edited_by":
      return typeof v === "string" && v ? personNode(ctx, v, true) : "";
    case "unique_id":
      return typeof v === "number" ? (p.prefix ? `${p.prefix}-${v}` : String(v)) : "";
    case "place": {
      const pl = v as { name?: string; address?: string; lat: number; lng: number } | undefined;
      if (!pl) return "";
      return h("span", { class: "place" }, iconNode("pin"), h("span", { text: pl.name || pl.address || `${pl.lat.toFixed(4)}, ${pl.lng.toFixed(4)}` }));
    }
    case "button":
      return h("span", { class: "btn-cell", text: p.name });
  }
  return "";
}

export function cardCover(ctx: DbCtx, row: RowData, source: string): string {
  if (source === "none" || !source) return "";
  if (source === "cover") {
    const c = row.cover ?? "";
    if (c.startsWith("f:")) return fileUrl(c.slice(2));
    return "";
  }
  if (source === "content") {
    if (row.thumb) return row.thumb.startsWith("f:") ? fileUrl(row.thumb.slice(2)) : row.thumb.startsWith("/") ? row.thumb : "";
    const c = row.cover ?? "";
    if (c.startsWith("f:")) return fileUrl(c.slice(2));
    return "";
  }
  const files = row.values[source];
  if (Array.isArray(files)) {
    for (const f of files as { id?: string; url?: string; name: string; mime?: string }[]) {
      if (f.id && (!f.mime || f.mime.startsWith("image/") || /\.(png|jpe?g|gif|webp|svg)$/i.test(f.name))) return fileUrl(f.id);
    }
  }
  void ctx;
  return "";
}
