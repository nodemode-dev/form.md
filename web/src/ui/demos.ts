import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { h, ref } from "./dom";
import { iconSvg, type IconName } from "./icons";
import { openLayer } from "./layers";

export interface DemoInfo {
  id: "editor" | "views" | "publish";
  path: string;
  title: () => string;
  line: () => string;
  icon: IconName;
}

export const DEMOS: DemoInfo[] = [
  {
    id: "editor",
    path: "/demo/index.html",
    title: () => t("Editor and databases"),
    line: () => t("Markdown pages in the real editor and a database of 66 real exoplanets, as a table and a board."),
    icon: "page",
  },
  {
    id: "views",
    path: "/demos/views/index.html",
    title: () => t("Database views"),
    line: () => t("Eight views, relations, rollups and formulas over US crewed spaceflight from Freedom 7 to Apollo 17."),
    icon: "board",
  },
  {
    id: "publish",
    path: "/demos/publish/index.html",
    title: () => t("Writing and publishing"),
    line: () => t("A long page with comments, page history, publishing to the web and export."),
    icon: "globe",
  },
];

export function insideDemo(): boolean {
  return location.protocol === "file:" || /^\/demos?\//.test(location.pathname);
}

let stage: HTMLElement | null = null;
let closeStage: (() => void) | null = null;

export function openDemo(id: string): boolean {
  const demo = DEMOS.find((d) => d.id === id);
  if (!demo || insideDemo()) return false;
  closeStage?.();
  const close = h("button", { class: "btn btn--ghost demo-stage__close", attrs: { type: "button", "data-demo-close": "", "aria-label": t("Close demo") } }, t("Close demo"));
  close.insertAdjacentHTML("afterbegin", iconSvg("close"));
  const frame = h("iframe", {
    class: "demo-stage__frame",
    attrs: { src: demo.path, title: demo.title(), sandbox: "allow-scripts allow-downloads", referrerpolicy: "no-referrer", "data-demo-frame": demo.id },
  });
  const bar = h(
    "div",
    { class: "demo-stage__bar" },
    h("span", { class: "demo-stage__badge", text: t("Demo") }),
    h("span", { class: "demo-stage__title", text: demo.title() }),
    h("span", { class: "demo-stage__note", text: t("A sample workspace in this window. Nothing you do here reaches your own pages.") }),
    close,
  );
  stage = h("div", { class: "demo-stage", attrs: { role: "dialog", "aria-modal": "true", "aria-label": demo.title(), "data-demo-stage": demo.id } }, bar, frame);
  document.body.append(stage);
  const el = stage;
  const off = openLayer({
    el,
    modal: true,
    initialFocus: close,
    onClose: () => {
      el.remove();
      if (stage === el) stage = null;
      closeStage = null;
    },
  });
  closeStage = off;
  close.addEventListener("click", () => off());
  return true;
}

export function mountDemos(engine: Engine): void {
  const section = document.querySelector<HTMLElement>('[data-ref="home-demos-sec"]');
  if (insideDemo()) {
    if (section) section.hidden = true;
    return;
  }
  for (const d of DEMOS) {
    engine.commands.register({
      id: `demo.${d.id}`,
      label: () => t("Open demo: {title}", { title: d.title() }),
      group: () => t("Demos"),
      description: `Opens the ${d.id} demo inside this window as a sample workspace in a sandboxed frame; it never reads or changes the person's own pages.`,
      mutates: false,
      run: () => openDemo(d.id),
    });
  }
  if (!section) return;
  ref("home-demos-grid", section).addEventListener("click", (e) => {
    const id = (e.target as HTMLElement).closest<HTMLElement>("[data-demo]")?.dataset.demo;
    if (id) openDemo(id);
  });
  section.dataset.ready = "";
}
