export function reconcile<T, E extends HTMLElement>(
  container: HTMLElement,
  items: readonly T[],
  key: (item: T) => string,
  create: (item: T) => E,
  update: (el: E, item: T) => void,
): void {
  const existing = new Map<string, E>();
  for (const child of Array.from(container.children) as E[]) {
    const k = child.dataset.key;
    if (k !== undefined) existing.set(k, child);
  }
  let cursor: Element | null = container.firstElementChild;
  for (const item of items) {
    const k = key(item);
    let el = existing.get(k);
    if (el) {
      existing.delete(k);
    } else {
      el = create(item);
      el.dataset.key = k;
    }
    update(el, item);
    if (el !== cursor) container.insertBefore(el, cursor);
    else cursor = cursor.nextElementSibling;
  }
  for (const el of existing.values()) el.remove();
}
