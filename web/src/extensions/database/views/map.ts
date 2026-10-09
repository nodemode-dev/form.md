import { t } from "../../../engine/i18n";
import type { Property, RowData } from "../../../engine/types";
import { h, setAttr, setText } from "../../../ui/dom";
import { iconNode } from "../../../ui/icons";
import { type MenuEntry, openMenu } from "../../../ui/menu";
import { paintCell } from "../cells";
import type { ViewHost, ViewRenderer } from "../surface";
import { saveConfig } from "../toolbar";
import { configKey } from "./shared";

const NS = "http://www.w3.org/2000/svg";
const W = 1000;
const MAXLAT = 85.0511;
const PIN = "M0 0C-1.6-4.2-7-7.2-7-12.5a7 7 0 1 1 14 0c0 5.3-5.4 8.3-7 12.5z";

interface Basemap {
  w: number;
  countries: { n: string; d: string }[];
}

let basemap: Promise<Basemap> | null = null;

function loadBasemap(): Promise<Basemap> {
  basemap ??= fetch("/maps/world-110m.json", { credentials: "same-origin" })
    .then((r) => {
      if (!r.ok) throw new Error(`basemap: HTTP ${r.status}`);
      return r.json() as Promise<Basemap>;
    })
    .catch((e: unknown) => {
      basemap = null;
      throw e;
    });
  return basemap;
}

export function project(lat: number, lng: number): [number, number] {
  const la = Math.max(-MAXLAT, Math.min(MAXLAT, lat));
  const r = (la * Math.PI) / 180;
  return [((lng + 180) / 360) * W, ((1 - Math.log(Math.tan(r) + 1 / Math.cos(r)) / Math.PI) / 2) * W];
}

interface Place {
  lat: number;
  lng: number;
  name?: string;
  address?: string;
}

function placeOf(row: RowData, propId: string): Place | null {
  const v = row.values[propId] as Place | undefined;
  if (!v || typeof v.lat !== "number" || typeof v.lng !== "number" || !Number.isFinite(v.lat) || !Number.isFinite(v.lng)) return null;
  return v;
}

function el<K extends keyof SVGElementTagNameMap>(tag: K, cls = ""): SVGElementTagNameMap[K] {
  const e = document.createElementNS(NS, tag);
  if (cls) e.setAttribute("class", cls);
  return e;
}

export function mapView(host: ViewHost): ViewRenderer {
  const session = host.makeSession({ limit: 1000, noGroups: true, search: host.search() });
  const ctx = host.ctx(session);
  const svg = el("svg", "mapv__svg");
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", t("Map"));
  const sea = el("rect", "mapv__sea");
  const land = el("g", "mapv__land");
  const pins = el("g", "mapv__pins");
  svg.append(sea, land, pins);
  const zoomIn = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Zoom in"), "data-tip": t("Zoom in") } }, iconNode("plus"));
  const zoomOut = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Zoom out"), "data-tip": t("Zoom out") } }, h("span", { class: "mapv__minus", attrs: { "aria-hidden": "true" } }));
  const fitBtn = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Show all pins"), "data-tip": t("Show all pins") } }, iconNode("expand"));
  const controls = h("div", { class: "mapv__controls" }, zoomIn, zoomOut, fitBtn);
  const card = h("div", { class: "mapv__card", attrs: { role: "dialog", "aria-label": t("Page on the map") } });
  card.hidden = true;
  const frame = h("div", { class: "mapv__frame" });
  frame.append(svg, controls, card);
  const foot = h("div", { class: "mapv__foot" });
  const credit = h("span", { class: "mapv__credit", text: t("Basemap: Natural Earth (public domain)") });
  const el2 = h("div", { class: "mapv", attrs: { "data-inline": host.inline ? "true" : "false" } }, frame, h("div", { class: "mapv__bar" }, foot, credit));
  let destroyed = false;
  let cx = W / 2;
  let cy = W * 0.46;
  let scale = 0;
  let fitted = false;
  let needFit = false;
  let cardRow = "";

  const propId = (): string => {
    const want = String(host.view().config.layout?.place ?? "");
    const schema = host.col().schema;
    if (want && schema.some((p) => p.id === want && p.type === "place")) return want;
    return schema.find((p) => p.type === "place")?.id ?? "";
  };
  const size = () => [frame.clientWidth || 1, frame.clientHeight || 1] as const;
  const minScale = () => {
    const [w] = size();
    return w / (W * 1.05);
  };
  const apply = () => {
    const [w, hh] = size();
    scale = Math.max(minScale(), Math.min(64, scale || minScale()));
    const vw = w / scale;
    const vh = hh / scale;
    cy = Math.max(vh / 2 - 40, Math.min(W - vh / 2 + 40, cy));
    svg.setAttribute("viewBox", `${(cx - vw / 2).toFixed(2)} ${(cy - vh / 2).toFixed(2)} ${vw.toFixed(2)} ${vh.toFixed(2)}`);
    const k = (1 / scale).toFixed(4);
    for (const g of Array.from(pins.children) as SVGGElement[]) {
      const x = g.dataset.x ?? "0";
      const y = g.dataset.y ?? "0";
      g.setAttribute("transform", `translate(${x} ${y}) scale(${k})`);
    }
    placeCard();
  };

  const visibleRows = (): { row: RowData; place: Place }[] => {
    const pid = propId();
    if (!pid) return [];
    const out: { row: RowData; place: Place }[] = [];
    for (const id of session.ordered()) {
      const row = session.rows.get(id);
      const pl = row ? placeOf(row, pid) : null;
      if (row && pl) out.push({ row, place: pl });
    }
    return out;
  };

  const fit = () => {
    if (!frame.clientWidth || !frame.clientHeight) {
      needFit = true;
      return;
    }
    needFit = false;
    svg.setAttribute("data-ready", "true");
    const list = visibleRows();
    const [w, hh] = size();
    if (!list.length) {
      cx = W / 2;
      cy = W * 0.46;
      scale = minScale();
      return apply();
    }
    let x0 = Infinity;
    let y0 = Infinity;
    let x1 = -Infinity;
    let y1 = -Infinity;
    for (const { place } of list) {
      const [x, y] = project(place.lat, place.lng);
      x0 = Math.min(x0, x);
      y0 = Math.min(y0, y);
      x1 = Math.max(x1, x);
      y1 = Math.max(y1, y);
    }
    const spanX = Math.max(x1 - x0, 6);
    const spanY = Math.max(y1 - y0, 6);
    cx = (x0 + x1) / 2;
    cy = (y0 + y1) / 2;
    scale = Math.min((w - 80) / spanX, (hh - 80) / spanY);
    apply();
  };

  const paint = () => {
    if (destroyed) return;
    const pid = propId();
    const list = visibleRows();
    const existing = new Map<string, SVGGElement>();
    for (const g of Array.from(pins.children) as SVGGElement[]) existing.set(g.dataset.row ?? "", g);
    const k = (1 / (scale || 1)).toFixed(4);
    for (const { row, place } of list) {
      let g = existing.get(row.id);
      if (g) existing.delete(row.id);
      else {
        g = el("g", "mapv__pin");
        g.dataset.row = row.id;
        g.setAttribute("tabindex", "0");
        g.setAttribute("role", "button");
        const shape = el("path", "mapv__pinshape");
        shape.setAttribute("d", PIN);
        const dot = el("circle", "mapv__pindot");
        dot.setAttribute("cy", "-12.5");
        dot.setAttribute("r", "2.6");
        g.append(el("title"), shape, dot);
        pins.append(g);
      }
      const [x, y] = project(place.lat, place.lng);
      g.dataset.x = x.toFixed(2);
      g.dataset.y = y.toFixed(2);
      g.setAttribute("transform", `translate(${g.dataset.x} ${g.dataset.y}) scale(${k})`);
      const label = `${row.title || t("Untitled")}${place.name ? ` · ${place.name}` : ""}`;
      setText(g.firstElementChild as Element as HTMLElement, label);
      g.setAttribute("aria-label", label);
      setAttr(g as unknown as HTMLElement, "data-open", row.id === cardRow ? "true" : null);
    }
    for (const g of existing.values()) g.remove();
    const missing = session.ordered().length - list.length;
    if (!pid) setText(foot, t("Add a place property to put pages on the map."));
    else setText(foot, missing > 0 ? t("{n} pages on the map, {m} without a place", { n: list.length, m: missing }) : t("{n} pages on the map", { n: list.length }));
    if (cardRow && !list.some((x) => x.row.id === cardRow)) closeCard();
    else if (cardRow) fillCard();
  };

  const fillCard = () => {
    const row = session.rows.get(cardRow);
    if (!row) return;
    const props = host
      .col()
      .schema.filter((p: Property) => p.type !== "title" && p.type !== "button" && row.values[p.id] !== undefined && row.values[p.id] !== null && row.values[p.id] !== "")
      .slice(0, 4);
    let title = card.querySelector<HTMLElement>(".mapv__cardtitle");
    if (!title) {
      title = h("button", { class: "mapv__cardtitle", attrs: { type: "button" } });
      const list = h("div", { class: "mapv__cardprops" });
      const close = h("button", { class: "icon-btn icon-btn--sm mapv__cardclose", attrs: { type: "button", "aria-label": t("Close") } }, iconNode("close"));
      card.append(h("div", { class: "mapv__cardhead" }, title, close), list);
      close.addEventListener("click", closeCard);
      title.addEventListener("click", (e) => host.openRow(cardRow, e));
    }
    setText(title, row.title || t("Untitled"));
    const list = card.querySelector(".mapv__cardprops") as HTMLElement;
    const cells = Array.from(list.children) as HTMLElement[];
    props.forEach((p, i) => {
      let line = cells[i];
      if (!line) {
        line = h("div", { class: "mapv__cardprop" }, h("span", { class: "mapv__cardname" }), h("span", { class: "mapv__cardval" }));
        list.append(line);
      }
      setText(line.firstElementChild as HTMLElement, p.name);
      paintCell(line.lastElementChild as HTMLElement, p, row, ctx, "list");
    });
    for (const extra of cells.slice(props.length)) extra.remove();
  };
  const placeCard = () => {
    if (!cardRow || card.hidden) return;
    const g = pins.querySelector<SVGGElement>(`[data-row="${cardRow}"]`);
    const ctm = svg.getScreenCTM();
    if (!g || !ctm) return;
    const pt = svg.createSVGPoint();
    pt.x = Number(g.dataset.x);
    pt.y = Number(g.dataset.y);
    const s = pt.matrixTransform(ctm);
    const box = frame.getBoundingClientRect();
    const x = Math.max(8, Math.min(box.width - 248, s.x - box.left - 120));
    const y = Math.max(8, s.y - box.top - 40);
    card.style.transform = `translate(${Math.round(x)}px, ${Math.round(Math.min(y, box.height - 8))}px) translateY(-100%)`;
  };
  const openCard = (id: string) => {
    cardRow = id;
    for (const g of Array.from(pins.children) as SVGGElement[]) g.toggleAttribute("data-open", g.dataset.row === id);
    fillCard();
    card.hidden = false;
    placeCard();
  };
  const closeCard = () => {
    cardRow = "";
    card.hidden = true;
    for (const g of Array.from(pins.children) as SVGGElement[]) g.removeAttribute("data-open");
  };

  const zoomAt = (factor: number, sx?: number, sy?: number) => {
    const [w, hh] = size();
    const px = sx ?? w / 2;
    const py = sy ?? hh / 2;
    const wx = cx + (px - w / 2) / scale;
    const wy = cy + (py - hh / 2) / scale;
    const next = Math.max(minScale(), Math.min(64, scale * factor));
    cx = wx - (px - w / 2) / next;
    cy = wy - (py - hh / 2) / next;
    scale = next;
    apply();
  };
  frame.addEventListener(
    "wheel",
    (e) => {
      if (!e.ctrlKey && !e.metaKey && Math.abs(e.deltaY) < 1) return;
      e.preventDefault();
      const r = frame.getBoundingClientRect();
      zoomAt(Math.exp(-e.deltaY * 0.0022), e.clientX - r.left, e.clientY - r.top);
    },
    { passive: false },
  );
  let drag: { x: number; y: number; cx: number; cy: number; moved: boolean; id: number } | null = null;
  svg.addEventListener("pointerdown", (e) => {
    if (e.button !== 0) return;
    drag = { x: e.clientX, y: e.clientY, cx, cy, moved: false, id: e.pointerId };
  });
  svg.addEventListener("pointermove", (e) => {
    if (!drag || e.pointerId !== drag.id) return;
    const dx = e.clientX - drag.x;
    const dy = e.clientY - drag.y;
    if (!drag.moved && Math.hypot(dx, dy) < 4) return;
    if (!drag.moved) svg.setPointerCapture(e.pointerId);
    drag.moved = true;
    cx = drag.cx - dx / scale;
    cy = drag.cy - dy / scale;
    apply();
  });
  const endDrag = (e: PointerEvent) => {
    if (!drag || e.pointerId !== drag.id) return;
    const moved = drag.moved;
    drag = null;
    if (moved) return;
    const g = (e.target as Element).closest<SVGGElement>(".mapv__pin");
    if (g?.dataset.row) openCard(g.dataset.row);
    else closeCard();
  };
  svg.addEventListener("pointerup", endDrag);
  svg.addEventListener("pointercancel", () => (drag = null));
  pins.addEventListener("keydown", (e) => {
    const g = (e.target as Element).closest<SVGGElement>(".mapv__pin");
    if (!g?.dataset.row) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      if (cardRow === g.dataset.row) host.openRow(g.dataset.row, e);
      else openCard(g.dataset.row);
    }
  });
  frame.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && cardRow) {
      e.stopPropagation();
      closeCard();
    }
  });
  zoomIn.addEventListener("click", () => zoomAt(1.6));
  zoomOut.addEventListener("click", () => zoomAt(1 / 1.6));
  fitBtn.addEventListener("click", fit);

  let raf = 0;
  const ro = new ResizeObserver(() => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(() => {
      if (needFit) fit();
      else if (fitted && frame.clientWidth) apply();
    });
  });
  ro.observe(frame);

  session.on((c) => {
    if (c.kind === "rows" || c.kind === "reset") paint();
  });
  const ready = Promise.all([
    loadBasemap().then((b) => {
      if (destroyed) return;
      sea.setAttribute("x", "-2000");
      sea.setAttribute("y", "-2000");
      sea.setAttribute("width", String(b.w + 4000));
      sea.setAttribute("height", String(b.w + 4000));
      const frag = document.createDocumentFragment();
      for (const c of b.countries) {
        const p = el("path", "mapv__country");
        p.setAttribute("d", c.d);
        const tt = el("title");
        tt.textContent = c.n;
        p.append(tt);
        frag.append(p);
      }
      land.append(frag);
    }),
    session.load(),
  ]).then(() => {
    if (destroyed) return;
    paint();
    fit();
    fitted = true;
  });

  return {
    el: el2,
    ready,
    ctx,
    configChanged(prev, next) {
      if (configKey(prev.config) !== configKey(next.config)) void session.load().catch(host.engine.reportError);
      else if (String(prev.config.layout?.place ?? "") !== String(next.config.layout?.place ?? "")) {
        paint();
        fit();
      } else paint();
    },
    schemaChanged: paint,
    setSearch(q) {
      session.options.search = q;
      void session.load().catch(host.engine.reportError);
    },
    settings(anchor) {
      const places = host.col().schema.filter((p) => p.type === "place");
      const cur = propId();
      const entries: MenuEntry[] = [{ kind: "heading", label: t("Place property") }];
      if (!places.length) entries.push({ kind: "heading", label: t("No place property yet") });
      for (const p of places) entries.push({ label: p.name, checked: p.id === cur, run: () => saveConfig(ctx, host.viewId, (cfg) => ({ ...cfg, layout: { ...(cfg.layout ?? {}), place: p.id } }), 0) });
      entries.push({ kind: "separator" }, { label: t("Show all pins"), icon: "expand", run: fit });
      openMenu(anchor, entries, { side: "bottom", align: "end", label: t("Map") });
    },
    destroy() {
      destroyed = true;
      ro.disconnect();
      cancelAnimationFrame(raf);
      session.close();
      el2.remove();
    },
  };
}
