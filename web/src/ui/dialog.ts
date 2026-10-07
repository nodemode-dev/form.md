import type { Engine, Overlay } from "../engine";
import { t } from "../engine/i18n";
import { byId, h, setAttr, setText } from "./dom";
import { actions, button, field, input, type Variant } from "./form";
import { type IconName, iconNode } from "./icons";
import { openLayer } from "./layers";

export interface Tab {
  id: string;
  label: string;
  icon: IconName;
  visible?: () => boolean;
  render: (panel: HTMLElement) => void | (() => void);
}

export interface DialogHandle {
  open(tab?: string): void;
  close(): void;
  isOpen(): boolean;
  title(text: string): void;
  body: HTMLElement;
}

interface Parts {
  scrim: HTMLElement;
  dialog: HTMLElement;
  title: HTMLElement;
  body: HTMLElement;
}

function shell(size: "sm" | "md" | "lg", label: string): Parts {
  const id = `dlg-${Math.random().toString(36).slice(2, 9)}`;
  const title = h("h2", { class: "dialog__title", attrs: { id }, text: label });
  const close = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Close"), "data-tip": t("Close"), "data-command": "layers.closeTop" } }, iconNode("close"));
  const body = h("div", { class: size === "lg" ? "dialog__layout" : "dialog__body" });
  const dialog = h(
    "div",
    { class: `dialog dialog--${size}`, attrs: { role: "dialog", "aria-modal": "true", "aria-labelledby": id, tabindex: "-1" } },
    h("header", { class: "dialog__head" }, title, close),
    body,
  );
  const scrim = h("div", { class: "scrim", attrs: { "aria-hidden": "true" } });
  scrim.hidden = true;
  dialog.hidden = true;
  byId("overlay-root").append(scrim, dialog);
  return { scrim, dialog, title, body };
}

function controller(parts: Parts, onOpen: (tab?: string) => void, onClose?: () => void, initialFocus?: () => HTMLElement | null): DialogHandle {
  let closeLayer: (() => void) | null = null;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  const handle: DialogHandle = {
    body: parts.body,
    isOpen: () => closeLayer !== null,
    title: (text) => setText(parts.title, text),
    open(tab) {
      clearTimeout(hideTimer);
      onOpen(tab);
      if (closeLayer) return;
      parts.scrim.hidden = false;
      parts.dialog.hidden = false;
      requestAnimationFrame(() => {
        parts.scrim.classList.add("is-visible");
        parts.dialog.classList.add("is-visible");
      });
      closeLayer = openLayer({
        el: parts.dialog,
        modal: true,
        dismissOnOutside: true,
        initialFocus: initialFocus?.() ?? parts.dialog,
        onClose: () => {
          closeLayer = null;
          parts.scrim.classList.remove("is-visible");
          parts.dialog.classList.remove("is-visible");
          hideTimer = setTimeout(() => {
            parts.scrim.hidden = true;
            parts.dialog.hidden = true;
          }, 220);
          onClose?.();
        },
      });
    },
    close() {
      closeLayer?.();
    },
  };
  return handle;
}

export function tabbedDialog(label: string, tabs: Tab[], onClose?: () => void): DialogHandle {
  let parts: Parts | null = null;
  const panels = new Map<string, { tab: HTMLButtonElement; panel: HTMLElement; refresh?: () => void; rendered: boolean }>();
  let active = "";

  const build = (): Parts => {
    const p = shell("lg", label);
    const nav = h("nav", { class: "dialog__nav", attrs: { role: "tablist", "aria-orientation": "vertical" } });
    const holder = h("div", { class: "dialog__panels" });
    for (const tab of tabs) {
      const btn = h("button", { class: "dialog__tab", attrs: { type: "button", role: "tab", "aria-selected": "false", id: `tab-${tab.id}` } }, iconNode(tab.icon), h("span", { text: tab.label }));
      const panel = h("section", { class: "dialog__panel", attrs: { role: "tabpanel", "aria-labelledby": `tab-${tab.id}` } });
      panel.hidden = true;
      btn.addEventListener("click", () => select(tab.id));
      nav.append(btn);
      holder.append(panel);
      panels.set(tab.id, { tab: btn, panel, rendered: false });
    }
    nav.addEventListener("keydown", (e) => {
      const list = tabs.filter((tb) => !panels.get(tb.id)?.tab.hidden).map((tb) => tb.id);
      const i = list.indexOf(active);
      const next = e.key === "ArrowDown" ? list[i + 1] : e.key === "ArrowUp" ? list[i - 1] : undefined;
      if (next) {
        e.preventDefault();
        select(next);
        panels.get(next)?.tab.focus();
      }
    });
    p.body.append(nav, holder);
    return p;
  };

  const select = (id: string) => {
    const def = tabs.find((tb) => tb.id === id && (tb.visible?.() ?? true)) ?? tabs.find((tb) => tb.visible?.() ?? true);
    if (!def) return;
    for (const [tid, x] of panels) {
      const on = tid === def.id;
      setAttr(x.tab, "aria-selected", on ? "true" : "false");
      x.panel.hidden = !on;
    }
    const entry = panels.get(def.id);
    if (entry && !entry.rendered) {
      entry.rendered = true;
      const refresh = def.render(entry.panel);
      if (refresh) entry.refresh = refresh;
    } else {
      entry?.refresh?.();
    }
    active = def.id;
  };

  let inner: DialogHandle | null = null;
  const ensure = (): DialogHandle => {
    if (inner) return inner;
    parts = build();
    inner = controller(
      parts,
      (tab) => {
        for (const tb of tabs) {
          const x = panels.get(tb.id);
          if (x) x.tab.hidden = !(tb.visible?.() ?? true);
        }
        select(tab || active || tabs[0]?.id || "");
      },
      onClose,
    );
    return inner;
  };
  return {
    open: (tab) => ensure().open(tab),
    close: () => inner?.close(),
    isOpen: () => inner?.isOpen() ?? false,
    title: (text) => ensure().title(text),
    get body() {
      return ensure().body;
    },
  };
}

export function overlayDialog(engine: Engine, name: Exclude<Overlay, null>, label: string, tabs: Tab[]): DialogHandle {
  const { state } = engine;
  const dlg = tabbedDialog(label, tabs, () => {
    if (state.get("overlay") === name) state.set("overlay", null);
  });
  state.watch("overlay", (v) => {
    if (v === name) dlg.open(state.get("overlayTab") || undefined);
    else if (dlg.isOpen()) dlg.close();
  });
  state.watch("overlayTab", (tab) => {
    if (state.get("overlay") === name && tab) dlg.open(tab);
  });
  return dlg;
}

export interface ConfirmOptions {
  title: string;
  body: string | HTMLElement;
  confirm: string;
  variant?: Variant;
  cancel?: string;
}

export function confirmDialog(o: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    const parts = shell("sm", o.title);
    parts.dialog.classList.add("dialog--confirm");
    const msg = typeof o.body === "string" ? h("p", { class: "section__lead", text: o.body }) : o.body;
    const ok = button(o.confirm, o.variant ?? "primary");
    const cancel = button(o.cancel ?? t("Cancel"), "ghost");
    parts.body.append(msg);
    parts.dialog.append(h("footer", { class: "dialog__foot" }, cancel, ok));
    let result = false;
    const handle = controller(
      parts,
      () => undefined,
      () => {
        resolve(result);
        setTimeout(() => {
          parts.scrim.remove();
          parts.dialog.remove();
        }, 260);
      },
      () => cancel,
    );
    ok.addEventListener("click", () => {
      result = true;
      handle.close();
    });
    cancel.addEventListener("click", () => handle.close());
    handle.open();
  });
}

export interface TypedConfirmOptions {
  title: string;
  body: string;
  expect: string;
  confirm: string;
}

export function confirmTyped(o: TypedConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    let ok = false;
    simpleDialog(
      o.title,
      (body, close) => {
        const typed = input({ autocomplete: "off" });
        const go = button(o.confirm, "danger", undefined, "submit");
        go.disabled = true;
        typed.addEventListener("input", () => {
          go.disabled = typed.value.trim() !== o.expect.trim();
        });
        const cancel = button(t("Cancel"), "ghost");
        cancel.addEventListener("click", close);
        const form = h(
          "form",
          { class: "form-grid", attrs: { novalidate: "" } },
          h("p", { class: "section__lead", text: o.body }),
          field(t("Type {name} to confirm", { name: o.expect }), typed),
          actions(cancel, go),
        );
        form.addEventListener("submit", (e) => {
          e.preventDefault();
          if (go.disabled) return;
          ok = true;
          close();
        });
        body.append(form);
        return typed;
      },
      () => resolve(ok),
    );
  });
}

export function simpleDialog(title: string, build: (body: HTMLElement, close: () => void) => HTMLElement | void, onClose?: () => void): DialogHandle {
  const parts = shell("sm", title);
  let handle: DialogHandle | null = null;
  const initial = build(parts.body, () => handle?.close());
  handle = controller(
    parts,
    () => undefined,
    () => {
      onClose?.();
      setTimeout(() => {
        parts.scrim.remove();
        parts.dialog.remove();
      }, 260);
    },
    () => initial ?? null,
  );
  handle.open();
  return handle;
}
