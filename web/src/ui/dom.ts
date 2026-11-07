export function ref<T extends HTMLElement = HTMLElement>(name: string, root: ParentNode = document): T {
  const el = root.querySelector<T>(`[data-ref="${name}"]`);
  if (!el) throw new Error(`missing [data-ref="${name}"]`);
  return el;
}

export function byId<T extends HTMLElement = HTMLElement>(id: string): T {
  const el = document.getElementById(id) as T | null;
  if (!el) throw new Error(`missing #${id}`);
  return el;
}

export function setText(el: Element, text: string): void {
  if (el.textContent !== text) el.textContent = text;
}

export function setAttr(el: Element, name: string, value: string | null): void {
  if (value === null) {
    if (el.hasAttribute(name)) el.removeAttribute(name);
  } else if (el.getAttribute(name) !== value) {
    el.setAttribute(name, value);
  }
}

type Child = Node | string;

interface Props {
  class?: string;
  attrs?: Record<string, string>;
  text?: string;
}

export function h<K extends keyof HTMLElementTagNameMap>(tag: K, props: Props = {}, ...children: Child[]): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  if (props.class) el.className = props.class;
  if (props.attrs) for (const [k, v] of Object.entries(props.attrs)) el.setAttribute(k, v);
  if (props.text !== undefined) el.textContent = props.text;
  for (const c of children) el.append(c);
  return el;
}

export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  const tag = target.tagName;
  if (tag === "TEXTAREA" || tag === "SELECT") return true;
  if (tag !== "INPUT") return false;
  const type = (target as HTMLInputElement).type;
  return !["button", "checkbox", "radio", "submit", "reset", "range", "color", "file"].includes(type);
}

export function nextFrame(): Promise<void> {
  return new Promise((resolve) => requestAnimationFrame(() => resolve()));
}

let revealed = false;

export function revealApp(): void {
  const tauri = (window as unknown as { __TAURI_INTERNALS__?: { invoke(cmd: string): Promise<unknown> } }).__TAURI_INTERNALS__;
  if (revealed || !tauri) return;
  revealed = true;
  requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => void tauri.invoke("app_ready").catch(() => undefined))));
}

export function delayedIndicator(show: () => void, hide: () => void, delay = 300, minVisible = 250): () => void {
  let shownAt = 0;
  const timer = setTimeout(() => {
    shownAt = performance.now();
    show();
  }, delay);
  return () => {
    if (shownAt === 0) {
      clearTimeout(timer);
      return;
    }
    const remaining = Math.max(0, minVisible - (performance.now() - shownAt));
    setTimeout(hide, remaining);
  };
}
