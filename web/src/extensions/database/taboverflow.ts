import { t } from "../../engine/i18n";
import { h, setAttr, setText } from "../../ui/dom";
import { type IconName, iconNode } from "../../ui/icons";
import { openMenu } from "../../ui/menu";

export interface TabEntry {
  id: string;
  name: string;
  icon: IconName;
}

export interface TabOverflow {
  more: HTMLButtonElement;
  update(activeId: string): void;
  destroy(): void;
}

export function tabOverflow(tabs: HTMLElement, entries: () => TabEntry[], pick: (id: string) => void): TabOverflow {
  const count = h("span", { class: "db__more-count" });
  const more = h("button", { class: "db__tab db__tab--more", attrs: { type: "button", "aria-label": t("More views"), title: t("More views"), "aria-haspopup": "menu" } }, count, iconNode("chevron-down"));
  more.hidden = true;
  let revealed = "";
  let frame = 0;

  const tabOf = (id: string) => tabs.querySelector<HTMLElement>(`[data-view="${CSS.escape(id)}"]`);
  const clipped = (): string[] => {
    const box = tabs.getBoundingClientRect();
    const out: string[] = [];
    for (const el of Array.from(tabs.querySelectorAll<HTMLElement>("[data-view]"))) {
      const r = el.getBoundingClientRect();
      if (r.left < box.left - 1 || r.right > box.right + 1) out.push(el.dataset.view ?? "");
    }
    return out;
  };
  const paintEdges = () => {
    const over = tabs.scrollWidth > tabs.clientWidth + 1;
    const start = over && tabs.scrollLeft > 1;
    const end = over && tabs.scrollLeft + tabs.clientWidth < tabs.scrollWidth - 1;
    setAttr(tabs, "data-edge", start && end ? "both" : start ? "start" : end ? "end" : null);
    const n = over ? clipped().length : 0;
    setText(count, n ? String(n) : "");
    return over;
  };
  const reveal = (id: string) => {
    const el = tabOf(id);
    if (!el) return;
    const box = tabs.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    if (r.left < box.left) tabs.scrollLeft -= box.left - r.left + 8;
    else if (r.right > box.right) tabs.scrollLeft += r.right - box.right + 8;
  };
  const measure = () => {
    if (!tabs.isConnected) return;
    if (!more.hidden) more.hidden = true;
    const over = tabs.scrollWidth > tabs.clientWidth + 1;
    if (over) more.hidden = false;
    paintEdges();
  };

  const ro = new ResizeObserver(() => {
    measure();
    if (revealed) reveal(revealed);
    paintEdges();
  });
  ro.observe(tabs);
  tabs.addEventListener(
    "scroll",
    () => {
      if (frame) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        paintEdges();
      });
    },
    { passive: true },
  );
  more.addEventListener("click", () => {
    const hidden = new Set(clipped());
    const list = entries().filter((e) => hidden.has(e.id));
    openMenu(
      more,
      list.map((e) => ({ label: e.name, icon: e.icon, run: () => pick(e.id) })),
      { side: "bottom", align: "start", label: t("More views") },
    );
  });

  return {
    more,
    update(activeId: string) {
      measure();
      if (activeId && activeId !== revealed) {
        revealed = activeId;
        reveal(activeId);
      }
      paintEdges();
    },
    destroy() {
      ro.disconnect();
      cancelAnimationFrame(frame);
    },
  };
}
