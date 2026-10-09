import type { AnalyticsDay } from "../engine";
import { formatDate } from "../engine/format";
import { plural } from "../engine/i18n";

const NS = "http://www.w3.org/2000/svg";
const W = 600;
const H = 140;
const PAD = 18;

function svg<K extends keyof SVGElementTagNameMap>(tag: K, attrs: Record<string, string>): SVGElementTagNameMap[K] {
  const el = document.createElementNS(NS, tag);
  for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
  return el;
}

function dayMs(day: string): number {
  return Date.parse(`${day}T12:00:00Z`);
}

export interface BarChart {
  el: SVGSVGElement;
  update(days: AnalyticsDay[]): void;
}

export function barChart(label: string): BarChart {
  const el = svg("svg", { class: "chart", viewBox: `0 0 ${W} ${H}`, role: "img", "aria-label": label });
  const bars = svg("g", { class: "chart__bars" });
  const base = svg("line", { class: "chart__axis", x1: "0", x2: String(W), y1: String(H - PAD), y2: String(H - PAD) });
  const first = svg("text", { class: "chart__label", x: "0", y: String(H - 4) });
  const last = svg("text", { class: "chart__label", x: String(W), y: String(H - 4), "text-anchor": "end" });
  const top = svg("text", { class: "chart__label", x: "0", y: "10" });
  el.append(bars, base, first, last, top);
  return {
    el,
    update(days) {
      const n = days.length;
      while (bars.childElementCount > n) bars.lastElementChild?.remove();
      while (bars.childElementCount < n) {
        const r = svg("rect", { class: "chart__bar", rx: "1.5" });
        r.append(svg("title", {}));
        bars.append(r);
      }
      const max = Math.max(1, ...days.map((d) => d.views));
      const slot = W / Math.max(1, n);
      const bw = Math.max(1, slot * 0.68);
      const span = H - PAD - 14;
      days.forEach((d, i) => {
        const r = bars.children[i] as SVGRectElement;
        const h = d.views ? Math.max(2, (d.views / max) * span) : 0;
        r.setAttribute("x", (i * slot + (slot - bw) / 2).toFixed(2));
        r.setAttribute("width", bw.toFixed(2));
        r.setAttribute("y", (H - PAD - h).toFixed(2));
        r.setAttribute("height", h.toFixed(2));
        const title = r.firstElementChild as SVGTitleElement;
        title.textContent = `${formatDate(dayMs(d.day))}: ${plural(d.views, "{count} view", "{count} views")}, ${plural(d.viewers, "{count} person", "{count} people")}`;
      });
      first.textContent = n ? formatDate(dayMs(days[0]?.day ?? "")) : "";
      last.textContent = n ? formatDate(dayMs(days[n - 1]?.day ?? "")) : "";
      top.textContent = plural(max, "{count} view", "{count} views");
    },
  };
}
