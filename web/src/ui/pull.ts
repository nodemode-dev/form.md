import type { Engine } from "../engine";
import { syncNow } from "../extensions/shell";
import { h, ref, setAttr } from "./dom";
import { iconNode } from "./icons";

const TRIGGER = 72;

export function mountPullToRefresh(engine: Engine): void {
  const scroller = ref("scroller");
  const marker = h("div", { class: "pull", attrs: { "aria-hidden": "true" } }, iconNode("refresh"));
  document.getElementById("overlay-root")?.append(marker);
  let startY = -1;
  let distance = 0;
  let busy = false;
  const show = (d: number) => {
    marker.style.setProperty("--pull", `${Math.min(d, TRIGGER * 1.5)}px`);
    setAttr(marker, "data-state", d <= 0 ? null : d >= TRIGGER ? "ready" : "pulling");
  };
  scroller.addEventListener(
    "touchstart",
    (e) => {
      startY = scroller.scrollTop <= 0 && !busy && e.touches.length === 1 ? (e.touches[0]?.clientY ?? -1) : -1;
      distance = 0;
    },
    { passive: true },
  );
  scroller.addEventListener(
    "touchmove",
    (e) => {
      if (startY < 0) return;
      distance = (e.touches[0]?.clientY ?? startY) - startY;
      if (distance <= 0 || scroller.scrollTop > 0) {
        startY = -1;
        show(0);
        return;
      }
      show(distance / 2);
    },
    { passive: true },
  );
  scroller.addEventListener("touchend", () => {
    if (startY < 0) return;
    startY = -1;
    if (distance / 2 < TRIGGER) {
      show(0);
      return;
    }
    busy = true;
    setAttr(marker, "data-state", "busy");
    void syncNow(engine).finally(() => {
      busy = false;
      show(0);
    });
  });
}
