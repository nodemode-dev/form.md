import { ApiError } from "../engine/api";
import { delayedIndicator, h, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { type IconName, iconNode } from "./icons";

export type Variant = "primary" | "danger" | "ghost" | "default";

export function button(label: string, variant: Variant = "default", icon?: IconName, type: "button" | "submit" = "button"): HTMLButtonElement {
  const cls = variant === "default" ? "btn" : `btn btn--${variant}`;
  const b = h("button", { class: cls, attrs: { type } });
  if (icon) b.append(iconNode(icon));
  b.append(h("span", { text: label }));
  return b;
}

export function iconButton(icon: IconName, label: string): HTMLButtonElement {
  return h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": label, "data-tip": label } }, iconNode(icon));
}

export interface InputOptions {
  type?: string;
  name?: string;
  value?: string;
  placeholder?: string;
  autocomplete?: string;
  maxlength?: number;
  inputmode?: string;
}

export function input(o: InputOptions = {}): HTMLInputElement {
  const el = h("input", { class: "input" });
  el.type = o.type ?? "text";
  if (o.name) el.name = o.name;
  if (o.value !== undefined) el.value = o.value;
  if (o.placeholder) el.placeholder = o.placeholder;
  if (o.autocomplete) el.autocomplete = o.autocomplete as AutoFill;
  if (o.maxlength) el.maxLength = o.maxlength;
  if (o.inputmode) el.inputMode = o.inputmode;
  if (o.type === "password" || o.name === "code") el.spellcheck = false;
  return el;
}

export function textarea(value = "", rows = 6): HTMLTextAreaElement {
  const el = h("textarea", { class: "textarea" });
  el.value = value;
  el.rows = rows;
  return el;
}

export function select<V extends string>(options: readonly (readonly [V, string])[], value?: V): HTMLSelectElement {
  const el = h("select", { class: "select" });
  for (const [v, label] of options) el.append(h("option", { attrs: { value: v }, text: label }));
  if (value !== undefined) el.value = value;
  return el;
}

export function field(label: string, control: HTMLElement, hint?: string): HTMLLabelElement {
  const el = h("label", { class: "field" }, h("span", { class: "field__label", text: label }), control);
  if (hint) el.append(h("span", { class: "field__hint", text: hint }));
  return el;
}

export function toggle(label: string, checked: boolean, hint?: string): { el: HTMLLabelElement; input: HTMLInputElement } {
  const box = h("input", { attrs: { type: "checkbox", role: "switch" } });
  box.checked = checked;
  const text = h("span", { class: "switch__text" }, h("span", { class: "switch__label", text: label }));
  if (hint) text.append(h("span", { class: "switch__hint", text: hint }));
  return { el: h("label", { class: "switch" }, text, box), input: box };
}

export function choices<V extends string>(name: string, options: readonly (readonly [V, string])[], value: V): { el: HTMLElement; value(): V; set(v: V): void } {
  const el = h("div", { class: "choice-row", attrs: { role: "radiogroup" } });
  const radios: HTMLInputElement[] = [];
  for (const [v, label] of options) {
    const r = h("input", { attrs: { type: "radio", name, value: v } });
    r.checked = v === value;
    radios.push(r);
    el.append(h("label", { class: "choice" }, r, h("span", { text: label })));
  }
  return {
    el,
    value: () => (radios.find((r) => r.checked)?.value ?? value) as V,
    set: (v) => {
      for (const r of radios) r.checked = r.value === v;
    },
  };
}

export function optionCards<V extends string>(
  name: string,
  options: readonly (readonly [V, string, string])[],
  value: V,
): { el: HTMLElement; value(): V; set(v: V): void; disable(off: boolean): void } {
  const el = h("div", { class: "option-cards", attrs: { role: "radiogroup" } });
  const radios: HTMLInputElement[] = [];
  for (const [v, title, hint] of options) {
    const r = h("input", { attrs: { type: "radio", name, value: v } });
    r.checked = v === value;
    radios.push(r);
    el.append(h("label", { class: "option-card" }, r, h("span", { class: "option-card__title", text: title }), h("span", { class: "option-card__hint", text: hint })));
  }
  return {
    el,
    value: () => (radios.find((r) => r.checked)?.value ?? value) as V,
    set: (v) => {
      for (const r of radios) r.checked = r.value === v;
    },
    disable: (off) => {
      for (const r of radios) r.disabled = off;
    },
  };
}

export interface Status {
  el: HTMLElement;
  error(e: unknown): void;
  say(text: string, tone?: "success" | "info" | "error"): void;
  clear(): void;
}

export function status(existing?: HTMLElement): Status {
  const el = existing ?? h("p", { class: "form-status", attrs: { role: "alert" } });
  return {
    el,
    error(e) {
      if (!(e instanceof ApiError)) console.error(e);
      setAttr(el, "data-tone", null);
      setText(el, errorMessage(e));
    },
    say(text, tone = "info") {
      setAttr(el, "data-tone", tone === "error" ? null : tone);
      setText(el, text);
    },
    clear() {
      setText(el, "");
    },
  };
}

export async function busy<T>(btn: HTMLButtonElement, work: () => Promise<T>): Promise<T> {
  if (btn.getAttribute("aria-busy") === "true") throw new Error("busy");
  btn.disabled = true;
  const stop = delayedIndicator(
    () => btn.setAttribute("aria-busy", "true"),
    () => btn.removeAttribute("aria-busy"),
  );
  try {
    return await work();
  } finally {
    stop();
    btn.disabled = false;
  }
}

export function section(title: string, lead?: string): HTMLElement {
  const el = h("section", { class: "section" }, h("h3", { class: "section__title", text: title }));
  if (lead) el.append(h("p", { class: "section__lead", text: lead }));
  return el;
}

export function actions(...buttons: HTMLElement[]): HTMLElement {
  return h("div", { class: "form-actions" }, ...buttons);
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
