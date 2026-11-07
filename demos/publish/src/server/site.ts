import { DOMSerializer } from "prosemirror-model";
import { yXmlFragmentToProseMirrorRootNode } from "y-prosemirror";
import { ROOT, schema } from "../../../../web/src/engine/doc/schema";
import type { Page, Site } from "../../../../web/src/engine/types";
import { glyphSvg, parseIcon } from "../../../../web/src/ui/glyphs";
import siteCss from "../../../../web/src/site/site.css?inline";
import { pages, serverDoc } from "./db";
import { FILES } from "./files";
import { pictureUri } from "./packs";

export interface View {
  theme: Site["theme"];
  font: Site["font"];
  indexing: boolean;
  title: string;
  description: string;
}

export interface Ctx {
  mode: "site" | "export";
  root: string;
  nodes: Page[];
  view: View;
  href(id: string): string;
  src(path: string): string;
  origin: string;
}

const themeScript = `(function(){var d=document.documentElement;if(d.getAttribute("data-theme")==="auto")d.setAttribute("data-theme",matchMedia("(prefers-color-scheme: light)").matches?"light":"dark")})()`;
const toggleScript = `document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest(".toggle__btn");if(!b)return;e.preventDefault();var t=b.closest(".toggle");if(!t)return;var o=!t.classList.contains("is-open");t.classList.toggle("is-open",o);b.setAttribute("aria-expanded",o?"true":"false")});`;
const siteScript = `document.addEventListener("click",function(e){var a=e.target.closest&&e.target.closest("a[data-page]");if(!a)return;e.preventDefault();window.parent.__pubGo&&window.parent.__pubGo(a.getAttribute("data-page"))});`;
const gateScript = `document.addEventListener("submit",function(e){var f=e.target.closest&&e.target.closest("form.site-gate__card");if(!f)return;e.preventDefault();var ok=window.parent.__pubUnlock&&window.parent.__pubUnlock(f.querySelector("input[type=password]").value);if(!ok){var p=f.querySelector(".site-gate__error");if(p)p.hidden=false}});`;

export const esc = (s: string): string => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");

export function collect(rootId: string, subpages: boolean): Page[] {
  const root = pages.find((p) => p.id === rootId);
  if (!root) return [];
  const out = [root];
  if (!subpages) return out;
  const walk = (id: string) => {
    for (const p of pages.filter((x) => x.parent_id === id && !x.trashed_at && x.visibility === "workspace").sort((a, b) => a.sort_key.localeCompare(b.sort_key))) {
      out.push(p);
      walk(p.id);
    }
  };
  walk(rootId);
  return out;
}

function iconSpan(ref: string, fallback: string, cls: string, ctx: Ctx): string {
  const icon = parseIcon(ref);
  if (icon.kind === "file") return `<span class="${cls}" data-kind="picture"><img src="${esc(ctx.src(FILES.get(icon.id)?.path ?? ""))}" alt="" decoding="async"></span>`;
  if (icon.kind === "glyph") return `<span class="${cls}" data-kind="glyph"${icon.tint ? ` data-color="${esc(icon.tint)}"` : ""}>${glyphSvg(icon.name)}</span>`;
  return `<span class="${cls}" data-kind="default">${glyphSvg(fallback)}</span>`;
}

function formatDate(ms: number): string {
  return new Date(ms).toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
}

export function body(id: string, ctx: Ctx): string {
  const doc = yXmlFragmentToProseMirrorRootNode(serverDoc(id).getXmlFragment(ROOT), schema);
  const holder = document.createElement("div");
  holder.append(DOMSerializer.fromSchema(schema).serializeFragment(doc.content));
  const heads: { id: string; level: number; text: string }[] = [];
  doc.descendants((n) => {
    if (n.type.name === "heading") heads.push({ id: String(n.attrs.id ?? ""), level: Number(n.attrs.level) || 1, text: n.textContent });
    return !n.isTextblock;
  });
  for (const nav of Array.from(holder.querySelectorAll<HTMLElement>("nav.toc"))) {
    nav.setAttribute("aria-label", "Contents");
    nav.removeAttribute("contenteditable");
    nav.innerHTML = heads.map((x) => `<a class="toc__item" data-level="${x.level}" href="#${esc(x.id)}">${esc(x.text)}</a>`).join("");
  }
  for (const a of Array.from(holder.querySelectorAll<HTMLAnchorElement>("a.mention[data-kind=page]"))) {
    const ref = a.dataset.ref ?? "";
    const page = ctx.nodes.find((n) => n.id === ref);
    const span = document.createElement("span");
    if (!page) {
      span.className = "mention";
      span.dataset.kind = "page";
      span.dataset.missing = "true";
      span.innerHTML = `${iconSpan("", "lock", "mention__icon", ctx)}<span class="mention__label">Private page</span>`;
      a.replaceWith(span);
      continue;
    }
    a.setAttribute("href", ctx.href(ref));
    a.setAttribute("data-page", ref);
    a.removeAttribute("contenteditable");
    a.innerHTML = `${iconSpan(page.icon, "page", "mention__icon", ctx)}<span class="mention__label">${esc(page.title || "Untitled")}</span>`;
  }
  for (const img of Array.from(holder.querySelectorAll<HTMLImageElement>("img[src]"))) img.setAttribute("src", ctx.src(img.getAttribute("src") ?? ""));
  for (const el of Array.from(holder.querySelectorAll("[contenteditable]"))) el.removeAttribute("contenteditable");
  for (const a of Array.from(holder.querySelectorAll<HTMLAnchorElement>("a.bookmark__link, a[href^='http']"))) a.setAttribute("rel", "noopener noreferrer nofollow");
  return holder.innerHTML;
}

export function exportSrc(path: string): string {
  const file = [...FILES.values()].find((f) => f.path === path);
  return file ? pictureUri(file.name) || path : path;
}

function head(ctx: Ctx, title: string, desc: string, canonical: string): string {
  const v = ctx.view;
  let h = `<!doctype html><html lang="en" data-theme="${esc(v.theme)}" data-font="${esc(v.font)}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="dark light"><title>${esc(title)}</title>`;
  if (desc) h += `<meta name="description" content="${esc(desc)}">`;
  if (!v.indexing) h += `<meta name="robots" content="noindex, nofollow">`;
  if (canonical) h += `<link rel="canonical" href="${esc(canonical)}"><meta property="og:url" content="${esc(canonical)}">`;
  h += `<meta property="og:type" content="article"><meta property="og:title" content="${esc(title)}">`;
  if (desc) h += `<meta property="og:description" content="${esc(desc)}">`;
  h += `<meta name="twitter:card" content="summary"><script>${themeScript}</script><style>${siteCss.replace(/<\/style/g, "<\\/style")}</style></head>`;
  return h;
}

export function renderPage(id: string, ctx: Ctx): string {
  const n = ctx.nodes.find((x) => x.id === id) as Page;
  const title = n.title || "Untitled";
  const root = ctx.nodes.find((x) => x.id === ctx.root) as Page;
  let pageTitle = title;
  let desc = n.description || n.snippet;
  if (id === ctx.root) {
    if (ctx.view.title) pageTitle = ctx.view.title;
    if (ctx.view.description) desc = ctx.view.description;
  } else {
    const name = ctx.view.title || root.title;
    if (name) pageTitle = `${title} · ${name}`;
  }
  const chain: Page[] = [];
  for (let c: Page | undefined = n; c && chain.length < 64; c = ctx.nodes.find((x) => x.id === c!.parent_id)) {
    chain.unshift(c);
    if (c.id === ctx.root) break;
  }
  let out = head(ctx, pageTitle, desc.slice(0, 200), ctx.mode === "site" ? `${ctx.origin}${ctx.href(id)}` : "");
  out += `<body class="site${n.cover ? " has-cover" : ""}">`;
  out += `<header class="site-bar"><nav class="site-crumbs" aria-label="Breadcrumb">`;
  chain.forEach((c, i) => {
    if (i > 0) out += `<span class="site-sep" aria-hidden="true">/</span>`;
    const label = c.title || "Untitled";
    if (i === chain.length - 1) out += `<span class="site-crumb" aria-current="page">${iconSpan(c.icon, "page", "site-icon", ctx)}<span>${esc(label)}</span></span>`;
    else out += `<a class="site-crumb" href="${esc(ctx.href(c.id))}" data-page="${c.id}">${iconSpan(c.icon, "page", "site-icon", ctx)}<span>${esc(label)}</span></a>`;
  });
  out += `</nav><div class="site-bar__end"></div></header>`;
  const cover = parseCover(n.cover);
  if (cover.kind === "gradient") out += `<div class="site-cover" data-cover="${esc(cover.value)}"></div>`;
  else if (cover.kind === "file") out += `<div class="site-cover"><img src="${esc(ctx.src(FILES.get(cover.value)?.path ?? ""))}" alt=""></div>`;
  out += `<main class="site-main"><header class="site-head">`;
  if (n.icon) out += iconSpan(n.icon, "page", "site-icon", ctx);
  out += `<h1 class="site-title">${esc(title)}</h1>`;
  if (n.description) out += `<p class="site-desc">${esc(n.description)}</p>`;
  out += `</header><article class="doc-body">${body(id, ctx)}</article>`;
  const kids = ctx.nodes.filter((x) => x.parent_id === id);
  if (kids.length) {
    out += `<nav class="site-children" aria-label="Pages"><div class="site-children__head">Pages</div>`;
    for (const k of kids) out += `<a class="site-child" href="${esc(ctx.href(k.id))}" data-page="${k.id}">${iconSpan(k.icon, "page", "site-icon", ctx)}<span>${esc(k.title || "Untitled")}</span></a>`;
    out += `</nav>`;
  }
  out += `</main><footer class="site-foot">Updated ${esc(formatDate(n.updated_at))} · ${ctx.mode === "site" ? "Published with form.md" : "Exported from form.md"}</footer>`;
  out += `<script>${toggleScript}</script>${ctx.mode === "site" ? `<script>${siteScript}</script>` : ""}</body></html>`;
  return out;
}

function parseCover(ref: string): { kind: "none" | "gradient" | "file"; value: string } {
  if (ref.startsWith("g:")) return { kind: "gradient", value: ref.slice(2) };
  if (ref.startsWith("f:")) return { kind: "file", value: ref.slice(2) };
  return { kind: "none", value: "" };
}

export function renderGate(ctx: Ctx, failed: boolean): string {
  return `${head(ctx, "Password required", "", "")}<body class="site"><main class="site-gate"><form class="site-gate__card"><h1 class="site-gate__title">This page is protected</h1><p>Enter the password you were given to read it.</p><p class="site-gate__error" role="alert"${failed ? "" : " hidden"}>That password is not right.</p><input type="password" name="password" autocomplete="current-password" aria-label="Password" required autofocus><button type="submit">Continue</button></form></main><script>${gateScript}</script></body></html>`;
}

export function renderMissing(ctx: Ctx): string {
  return `${head(ctx, "Page not found", "", "")}<body class="site"><main class="site-missing"><h1 class="site-title">Nothing here</h1><p class="site-desc">This page is not published, or the link has ended.</p></main></body></html>`;
}
