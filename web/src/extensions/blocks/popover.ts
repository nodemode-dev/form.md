import { byId, h, setAttr, setText } from "../../ui/dom";
import { anchorToRect, type Rect } from "../../ui/floating";
import { type IconName, iconNode, isIconName } from "../../ui/icons";
import { openLayer } from "../../ui/layers";

export interface ListItem {
  key: string;
  label: string;
  icon?: IconName | (() => Node) | undefined;
  hint?: string | undefined;
  section?: string | undefined;
  detail?: string | undefined;
}

export interface ListOptions {
  label: string;
  rect: () => Rect | null;
  onPick: (item: ListItem) => void;
  onClose: () => void;
  empty?: string;
}

let seq = 0;

export class ListPopover {
  readonly el: HTMLElement;
  private items: ListItem[] = [];
  private sel = 0;
  private closeLayer: () => void;
  private place: { update: () => void; release: () => void };
  private emptyEl: HTMLElement;
  private closed = false;
  private readonly id = `lp${++seq}`;

  constructor(private readonly opts: ListOptions) {
    this.el = h("div", { class: "popover listbox", attrs: { role: "listbox", "aria-label": opts.label, id: this.id } });
    this.emptyEl = h("div", { class: "listbox__empty", text: opts.empty ?? "" });
    this.el.append(this.emptyEl);
    this.el.addEventListener("pointerdown", (e) => e.preventDefault());
    this.el.addEventListener("click", (e) => {
      const row = (e.target as Element).closest<HTMLElement>("[data-key]");
      const item = row ? this.items.find((x) => x.key === row.dataset.key) : undefined;
      if (item) opts.onPick(item);
    });
    this.el.addEventListener("pointermove", (e) => {
      const row = (e.target as Element).closest<HTMLElement>("[data-key]");
      if (!row) return;
      const i = this.items.findIndex((x) => x.key === row.dataset.key);
      if (i >= 0 && i !== this.sel) {
        this.sel = i;
        this.paintSelection(false);
      }
    });
    byId("overlay-root").append(this.el);
    this.place = anchorToRect(this.el, opts.rect, { side: "bottom", align: "start", offset: 6 });
    requestAnimationFrame(() => this.el.classList.add("is-visible"));
    this.closeLayer = openLayer({
      el: this.el,
      modal: false,
      dismissOnOutside: true,
      returnFocus: null,
      onClose: () => this.teardown(),
    });
  }

  get open(): boolean {
    return !this.closed;
  }

  set(items: ListItem[]): void {
    this.items = items;
    this.sel = Math.min(this.sel, Math.max(0, items.length - 1));
    const existing = new Map<string, HTMLElement>();
    for (const c of Array.from(this.el.querySelectorAll<HTMLElement>(":scope > [data-key], :scope > .listbox__section"))) existing.set(c.dataset.key ?? `s:${c.dataset.section}`, c);
    let cursor: Element | null = this.emptyEl.nextElementSibling;
    let section = "";
    const place = (el: HTMLElement) => {
      if (el !== cursor) this.el.insertBefore(el, cursor);
      else cursor = cursor.nextElementSibling;
    };
    for (const it of items) {
      if (it.section && it.section !== section) {
        section = it.section;
        const k = `s:${section}`;
        let s = existing.get(k);
        if (s) existing.delete(k);
        else {
          s = h("div", { class: "listbox__section", attrs: { role: "presentation" } });
          s.dataset.section = section;
        }
        setText(s, section);
        place(s);
      }
      let row = existing.get(it.key);
      if (row) existing.delete(it.key);
      else {
        row = h("div", { class: "listbox__item", attrs: { role: "option", id: `${this.id}-${it.key}` } }, h("span", { class: "listbox__icon" }), h("span", { class: "listbox__label" }), h("span", { class: "listbox__hint" }));
        row.dataset.key = it.key;
      }
      const icon = row.children[0] as HTMLElement;
      const want = typeof it.icon === "string" ? it.icon : it.icon ? "custom" : "";
      if (icon.dataset.icon !== want || want === "custom") {
        icon.dataset.icon = want;
        icon.replaceChildren();
        if (typeof it.icon === "string" && isIconName(it.icon)) icon.append(iconNode(it.icon));
        else if (typeof it.icon === "function") icon.append(it.icon());
      }
      setText(row.children[1] as HTMLElement, it.label);
      setText(row.children[2] as HTMLElement, it.hint ?? it.detail ?? "");
      place(row);
    }
    for (const el of existing.values()) el.remove();
    this.emptyEl.hidden = items.length > 0 || !this.opts.empty;
    this.paintSelection(true);
    this.place.update();
  }

  move(d: 1 | -1): void {
    const n = this.items.length;
    if (!n) return;
    this.sel = (this.sel + d + n) % n;
    this.paintSelection(true);
  }

  pick(): boolean {
    const item = this.items[this.sel];
    if (!item) return false;
    this.opts.onPick(item);
    return true;
  }

  get count(): number {
    return this.items.length;
  }

  reposition(): void {
    this.place.update();
  }

  close(): void {
    this.closeLayer();
  }

  private paintSelection(scroll: boolean): void {
    const rows = this.el.querySelectorAll<HTMLElement>(":scope > [data-key]");
    rows.forEach((r) => setAttr(r, "aria-selected", r.dataset.key === this.items[this.sel]?.key ? "true" : "false"));
    const active = this.items[this.sel];
    setAttr(this.el, "aria-activedescendant", active ? `${this.id}-${active.key}` : null);
    if (scroll && active) this.el.querySelector<HTMLElement>(`:scope > [data-key="${CSS.escape(active.key)}"]`)?.scrollIntoView({ block: "nearest" });
  }

  private teardown(): void {
    if (this.closed) return;
    this.closed = true;
    this.place.release();
    this.el.remove();
    this.opts.onClose();
  }
}

export interface PromptOptions {
  label: string;
  placeholder: string;
  value?: string;
  action: string;
  allowEmpty?: boolean;
  rect: () => Rect | null;
  onSubmit: (value: string) => void;
  onClose?: () => void;
}

export function openPrompt(opts: PromptOptions): () => void {
  const input = h("input", { class: "prompt__input", attrs: { type: "text", placeholder: opts.placeholder, "aria-label": opts.label, spellcheck: "false", autocomplete: "off" } });
  input.value = opts.value ?? "";
  const submit = h("button", { class: "btn btn--primary prompt__btn", attrs: { type: "submit" }, text: opts.action });
  const form = h("form", { class: "popover prompt", attrs: { "aria-label": opts.label } }, input, submit);
  byId("overlay-root").append(form);
  const place = anchorToRect(form, opts.rect, { side: "bottom", align: "start", offset: 6 });
  requestAnimationFrame(() => form.classList.add("is-visible"));
  let closed = false;
  const close = openLayer({
    el: form,
    modal: false,
    dismissOnOutside: true,
    initialFocus: input,
    returnFocus: null,
    onClose: () => {
      if (closed) return;
      closed = true;
      place.release();
      form.remove();
      opts.onClose?.();
    },
  });
  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const v = input.value.trim();
    if (!v && !opts.allowEmpty) return;
    close();
    opts.onSubmit(v);
  });
  return close;
}
