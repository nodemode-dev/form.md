import type { Engine } from "../engine";
import { byId, ref, setAttr } from "./dom";
import { openLayer } from "./layers";

export const NARROW_QUERY = "(max-width: 767px)";

export function mountSidebar(engine: Engine): void {
  const root = document.documentElement;
  const sidebar = byId("sidebar");
  const sheet = byId("sheet");
  const topbar = ref("topbar");
  const scroller = ref("scroller");
  let closeDrawer: (() => void) | null = null;

  engine.state.watch("sidebar", (v) => setAttr(root, "data-sidebar", v), { immediate: true });

  engine.state.watch("drawer", (open) => {
    setAttr(root, "data-drawer", open ? "open" : "closed");
    if (sheet.inert !== open) sheet.inert = open;
    if (open && !closeDrawer) {
      closeDrawer = openLayer({
        el: sidebar,
        modal: false,
        dismissOnOutside: true,
        initialFocus: sidebar.querySelector<HTMLElement>("a, button") ?? sidebar,
        returnFocus: document.querySelector<HTMLElement>(".topbar__menu"),
        onClose: () => {
          closeDrawer = null;
          engine.state.set("drawer", false);
        },
      });
    } else if (!open && closeDrawer) {
      const c = closeDrawer;
      closeDrawer = null;
      c();
    }
  });

  matchMedia(NARROW_QUERY).addEventListener("change", (e) => {
    if (!e.matches) engine.state.set("drawer", false);
  });

  scroller.addEventListener(
    "scroll",
    () => setAttr(topbar, "data-scrolled", scroller.scrollTop > 0 ? "true" : null),
    { passive: true },
  );
}
