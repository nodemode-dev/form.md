import "./site.css";

function toggle(btn: HTMLElement): void {
  const box = btn.closest<HTMLElement>(".toggle");
  if (!box) return;
  const open = !box.classList.contains("is-open");
  box.classList.toggle("is-open", open);
  btn.setAttribute("aria-expanded", open ? "true" : "false");
}

function reveal(id: string): boolean {
  const el = document.querySelector<HTMLElement>(`[data-block="${CSS.escape(id)}"]`);
  if (!el) return false;
  for (let t = el.closest<HTMLElement>(".toggle:not(.is-open)"); t; t = t.parentElement?.closest<HTMLElement>(".toggle:not(.is-open)") ?? null) {
    const btn = t.querySelector<HTMLElement>(":scope > .toggle__btn");
    if (btn) toggle(btn);
  }
  el.scrollIntoView({ block: "start" });
  return true;
}

document.addEventListener("click", (e) => {
  const target = e.target as Element;
  const btn = target.closest<HTMLElement>(".toggle__btn");
  if (btn) {
    e.preventDefault();
    toggle(btn);
    return;
  }
  const anchor = target.closest<HTMLAnchorElement>('a[href^="#"]');
  if (anchor && reveal(decodeURIComponent(anchor.getAttribute("href")?.slice(1) ?? ""))) {
    e.preventDefault();
    history.replaceState(null, "", anchor.getAttribute("href"));
  }
});

if (location.hash.length > 1) reveal(decodeURIComponent(location.hash.slice(1)));
