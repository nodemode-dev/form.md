import type { Engine, Scope } from "../engine";
import { byId, h } from "./dom";
import { iconNode } from "./icons";
import { openLayer } from "./layers";

const LISTED_SCOPES: ReadonlySet<Scope> = new Set(["global", "view", "editor"]);

interface Parts {
  scrim: HTMLElement;
  dialog: HTMLElement;
  body: HTMLElement;
}

export function mountShortcutsSheet(engine: Engine): void {
  let parts: Parts | null = null;
  let closeLayer: (() => void) | null = null;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;

  const build = (): Parts => {
    const close = h(
      "button",
      { class: "icon-btn", attrs: { type: "button", "aria-label": "Close", "data-tip": "Close", "data-command": "layers.closeTop" } },
      iconNode("close"),
    );
    const head = h("header", { class: "dialog__head" }, h("h2", { class: "dialog__title", attrs: { id: "shortcuts-title" }, text: "Keyboard shortcuts" }), close);
    const body = h("div", { class: "dialog__body" });
    const dialog = h("div", { class: "dialog", attrs: { role: "dialog", "aria-modal": "true", "aria-labelledby": "shortcuts-title", tabindex: "-1" } }, head, body);
    const scrim = h("div", { class: "scrim", attrs: { "aria-hidden": "true" } });
    scrim.hidden = true;
    dialog.hidden = true;
    byId("overlay-root").append(scrim, dialog);
    return { scrim, dialog, body };
  };

  const fill = (body: HTMLElement) => {
    const groups = new Map<string, HTMLElement>();
    const sections: HTMLElement[] = [];
    for (const item of engine.keymap.list(LISTED_SCOPES)) {
      let list = groups.get(item.group);
      if (!list) {
        list = h("div", { class: "keys-group__rows" });
        groups.set(item.group, list);
        sections.push(h("section", { class: "keys-group" }, h("h3", { class: "keys-group__title", text: item.group }), list));
      }
      const chords = h("span", { class: "keys-row__chords" });
      item.chords.forEach((chord, i) => {
        if (i > 0) chords.append("or");
        chords.append(h("span", { class: "keys-row__chord" }, ...chord.map((k) => h("kbd", { text: k }))));
      });
      list.append(h("div", { class: "keys-row" }, h("span", { text: item.label }), chords));
    }
    body.replaceChildren(...sections);
  };

  const open = () => {
    parts ??= build();
    const p = parts;
    clearTimeout(hideTimer);
    if (p.dialog.hidden) fill(p.body);
    p.scrim.hidden = false;
    p.dialog.hidden = false;
    requestAnimationFrame(() => {
      p.scrim.classList.add("is-visible");
      p.dialog.classList.add("is-visible");
    });
    if (!closeLayer) {
      closeLayer = openLayer({
        el: p.dialog,
        modal: true,
        dismissOnOutside: true,
        initialFocus: p.dialog,
        onClose: () => {
          closeLayer = null;
          engine.state.set("overlay", null);
        },
      });
    }
  };

  const close = () => {
    if (closeLayer) {
      const c = closeLayer;
      closeLayer = null;
      c();
    }
    if (!parts || parts.dialog.hidden) return;
    const p = parts;
    p.scrim.classList.remove("is-visible");
    p.dialog.classList.remove("is-visible");
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => {
      p.scrim.hidden = true;
      p.dialog.hidden = true;
    }, 220);
  };

  engine.state.watch("overlay", (v) => (v === "shortcuts" ? open() : close()));
}
