import type { Engine, Route } from "../engine";
import { t } from "../engine/i18n";
import { ref, setAttr, setText } from "./dom";

export function routeTitle(engine: Engine, r: Route): string {
  switch (r.name) {
    case "page": {
      const p = engine.pages.get(r.id) ?? engine.state.get("detail");
      return p && p.id === r.id ? p.title || t("Untitled") : "";
    }
    case "library":
      return t("Library");
    case "trash":
      return t("Trash");
    default:
      return engine.workspace()?.name ?? "";
  }
}

export function mountHome(engine: Engine): void {
  const eyebrow = ref("home-eyebrow");
  const add = ref<HTMLButtonElement>("home-new");
  const paint = () => {
    setText(eyebrow, engine.workspace()?.name ?? t("Welcome"));
    add.hidden = !engine.canWrite();
  };
  engine.state.watch("workspaceId", paint);
  engine.state.watch("me", paint);
  paint();
}

export function mountRoutes(engine: Engine): void {
  const { state } = engine;
  const root = document.documentElement;
  const scroller = ref("scroller");
  const instance = engine.boot.instance_name || "form.md";
  const positions = new Map<string, number>();
  let current = "";

  const paintTitle = () => {
    const title = routeTitle(engine, state.get("route"));
    const next = title ? `${title} · ${instance}` : instance;
    if (document.title !== next) document.title = next;
  };

  state.watch(
    "route",
    (r) => {
      const key = r.name === "page" ? `p:${r.id}` : r.name;
      if (current) positions.set(current, scroller.scrollTop);
      current = key;
      setAttr(root, "data-route", r.name);
      const top = positions.get(key) ?? 0;
      if (scroller.scrollTop !== top) scroller.scrollTop = top;
      paintTitle();
    },
    { immediate: true },
  );
  state.watch("detail", paintTitle);
  engine.pages.subscribe((c) => {
    if (c.kind === "upsert") {
      const r = state.get("route");
      if (r.name === "page" && r.id === c.page.id) paintTitle();
    }
  });
}
