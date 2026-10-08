import { type DOMOutputSpec, type MarkSpec, type Node as PMNode, type NodeSpec, Schema } from "prosemirror-model";
import { tableNodes } from "prosemirror-tables";

export const ROOT = "prosemirror";

type Attrs = Record<string, string>;

const blockAttr = (n: PMNode): Attrs => (n.attrs.id ? { "data-block": String(n.attrs.id) } : {});
const withId = { id: { default: null } };

const ALPHABET = "abcdefghijklmnopqrstuvwxyz234567";

export function newBlockId(): string {
  const bytes = new Uint8Array(6);
  crypto.getRandomValues(bytes);
  let bits = 0;
  let value = 0;
  let out = "";
  for (const b of bytes) {
    value = (value << 8) | b;
    bits += 8;
    while (bits >= 5) {
      out += ALPHABET[(value >>> (bits - 5)) & 31];
      bits -= 5;
    }
  }
  if (bits > 0) out += ALPHABET[(value << (5 - bits)) & 31];
  return out;
}

export const TONES = ["", "violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"] as const;

const fileSrc = (ref: string): string => (ref.startsWith("f:") ? `/files/${ref.slice(2)}` : ref);
const refFromSrc = (src: string | null): string => {
  if (!src) return "";
  const m = /^\/files\/([a-z2-7]{26})$/.exec(src);
  return m ? `f:${m[1]}` : src;
};

export function humanSize(n: number): string {
  if (!n || n <= 0) return "";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

const YOUTUBE = /^https:\/\/(?:www\.)?(?:youtube\.com\/watch\?(?:.*&)?v=|youtu\.be\/|youtube\.com\/shorts\/)([A-Za-z0-9_-]{6,20})/;
const VIMEO = /^https:\/\/(?:www\.)?vimeo\.com\/(\d{4,12})/;

export function embedSrc(url: string): string {
  const y = YOUTUBE.exec(url);
  if (y) return `https://www.youtube-nocookie.com/embed/${y[1]}`;
  const v = VIMEO.exec(url);
  if (v) return `https://player.vimeo.com/video/${v[1]}?dnt=1`;
  return "";
}

export function embedProvider(url: string): string {
  if (YOUTUBE.test(url)) return "youtube";
  if (VIMEO.test(url)) return "vimeo";
  return "";
}

export function safeHref(h: string): string {
  const v = h.trim();
  const l = v.toLowerCase();
  if (["http://", "https://", "mailto:", "/p/", "/files/", "#"].some((p) => l.startsWith(p))) return v;
  return l.includes(":") ? "" : v;
}

const media = (kind: "image" | "video" | "audio", n: PMNode): DOMOutputSpec => {
  const src = fileSrc(String(n.attrs.src ?? ""));
  const caption: DOMOutputSpec = ["figcaption", { class: "media__caption" }, String(n.attrs.caption ?? "")];
  const dims = (): Attrs => {
    const o: Attrs = {};
    if (n.attrs.w) o.width = String(n.attrs.w);
    if (n.attrs.h) o.height = String(n.attrs.h);
    return o;
  };
  if (kind === "image")
    return ["figure", { class: "media media--image", ...blockAttr(n) }, ["div", { class: "media__frame" }, ["img", { src, alt: String(n.attrs.alt ?? ""), ...dims(), loading: "lazy", decoding: "async", draggable: "false" }]], caption];
  if (kind === "video") return ["figure", { class: "media media--video", ...blockAttr(n) }, ["div", { class: "media__frame" }, ["video", { src, ...dims(), controls: "controls", preload: "metadata" }]], caption];
  return ["figure", { class: "media media--audio", ...blockAttr(n) }, ["audio", { src, controls: "controls", preload: "metadata" }], caption];
};

const mediaParse = (kind: string) => ({
  tag: `figure.media--${kind}`,
  getAttrs: (dom: HTMLElement) => {
    const el = dom.querySelector(kind === "image" ? "img" : kind);
    return {
      src: refFromSrc(el?.getAttribute("src") ?? ""),
      alt: el?.getAttribute("alt") ?? "",
      caption: dom.querySelector("figcaption")?.textContent ?? "",
      w: Number(el?.getAttribute("width")) || null,
      h: Number(el?.getAttribute("height")) || null,
    };
  },
});

const nodes: Record<string, NodeSpec> = {
  doc: { content: "block+" },
  paragraph: {
    content: "inline*",
    group: "block",
    attrs: withId,
    parseDOM: [{ tag: "p" }],
    toDOM: (n) => ["p", blockAttr(n), 0],
  },
  heading: {
    content: "inline*",
    group: "block",
    defining: true,
    attrs: { ...withId, level: { default: 1 } },
    parseDOM: [1, 2, 3, 4, 5, 6].map((level) => ({ tag: `h${level}`, attrs: { level } })),
    toDOM: (n) => [`h${n.attrs.level as number}`, blockAttr(n), 0],
  },
  blockquote: {
    content: "block+",
    group: "block",
    defining: true,
    attrs: withId,
    parseDOM: [{ tag: "blockquote" }],
    toDOM: (n) => ["blockquote", blockAttr(n), 0],
  },
  callout: {
    content: "block+",
    group: "block",
    defining: true,
    attrs: { ...withId, icon: { default: "" }, tone: { default: "" } },
    parseDOM: [
      {
        tag: "div.doc-callout",
        contentElement: ".callout__body",
        getAttrs: (dom: HTMLElement) => ({ icon: dom.dataset.icon ?? "", tone: dom.dataset.tone ?? "" }),
      },
    ],
    toDOM: (n) => {
      const a: Attrs = { class: "doc-callout", ...blockAttr(n) };
      if (n.attrs.tone) a["data-tone"] = String(n.attrs.tone);
      if (n.attrs.icon) a["data-icon"] = String(n.attrs.icon);
      return ["div", a, ["span", { class: "callout__icon", contenteditable: "false" }], ["div", { class: "callout__body" }, 0]];
    },
  },
  toggle: {
    content: "toggle_title block*",
    group: "block",
    defining: true,
    attrs: { ...withId, level: { default: 0 } },
    parseDOM: [
      {
        tag: "div.toggle",
        contentElement: ".toggle__content",
        getAttrs: (dom: HTMLElement) => ({ level: Number(dom.dataset.level) || 0 }),
      },
      {
        tag: "details",
        getAttrs: (dom: HTMLElement) => {
          const h = dom.querySelector(":scope > summary h1, :scope > summary h2, :scope > summary h3");
          return { level: h ? Number(h.tagName.slice(1)) : 0 };
        },
      },
    ],
    toDOM: (n) => {
      const a: Attrs = { class: "toggle", ...blockAttr(n) };
      if (n.attrs.level) a["data-level"] = String(n.attrs.level);
      return ["div", a, ["button", { class: "toggle__btn", type: "button", contenteditable: "false", "aria-expanded": "false" }], ["div", { class: "toggle__content" }, 0]];
    },
  },
  toggle_title: {
    content: "inline*",
    defining: true,
    parseDOM: [{ tag: "div.toggle__title" }, { tag: "summary" }],
    toDOM: () => ["div", { class: "toggle__title" }, 0],
  },
  bullet_list: {
    content: "list_item+",
    group: "block",
    attrs: withId,
    parseDOM: [{ tag: "ul:not(.tasks)", getAttrs: (dom: HTMLElement) => (dom.querySelector(":scope > li > input[type=checkbox], :scope > li > p > input[type=checkbox]") ? false : null) }],
    toDOM: (n) => ["ul", blockAttr(n), 0],
  },
  ordered_list: {
    content: "list_item+",
    group: "block",
    attrs: { ...withId, start: { default: 1 } },
    parseDOM: [{ tag: "ol", getAttrs: (dom: HTMLElement) => ({ start: dom.hasAttribute("start") ? Number(dom.getAttribute("start")) || 1 : 1 }) }],
    toDOM: (n) => {
      const a: Attrs = blockAttr(n);
      if (n.attrs.start !== 1) a.start = String(n.attrs.start);
      return ["ol", a, 0];
    },
  },
  task_list: {
    content: "task_item+",
    group: "block",
    attrs: withId,
    parseDOM: [{ tag: "ul.tasks" }, { tag: "ul", getAttrs: (dom: HTMLElement) => (dom.querySelector(":scope > li > input[type=checkbox], :scope > li > p > input[type=checkbox]") ? null : false), priority: 60 }],
    toDOM: (n) => ["ul", { class: "tasks", ...blockAttr(n) }, 0],
  },
  list_item: {
    content: "paragraph block*",
    defining: true,
    attrs: withId,
    parseDOM: [{ tag: "li:not(.task)", getAttrs: (dom: HTMLElement) => (dom.parentElement?.classList.contains("tasks") ? false : null) }],
    toDOM: (n) => ["li", blockAttr(n), 0],
  },
  task_item: {
    content: "paragraph block*",
    defining: true,
    attrs: { ...withId, checked: { default: false } },
    parseDOM: [
      { tag: "li.task", contentElement: ".task__body", getAttrs: (dom: HTMLElement) => ({ checked: dom.dataset.checked === "true" }) },
      { tag: "li", getAttrs: (dom: HTMLElement) => ({ checked: !!dom.querySelector(":scope > input[type=checkbox]:checked, :scope > p > input[type=checkbox]:checked") }) },
    ],
    toDOM: (n) => [
      "li",
      { class: "task", ...blockAttr(n), "data-checked": n.attrs.checked ? "true" : "false" },
      ["span", { class: "task__check", contenteditable: "false" }, ["input", n.attrs.checked ? { type: "checkbox", checked: "checked" } : { type: "checkbox" }]],
      ["div", { class: "task__body" }, 0],
    ],
  },
  code_block: {
    content: "text*",
    marks: "",
    group: "block",
    code: true,
    defining: true,
    attrs: { ...withId, language: { default: "" } },
    parseDOM: [
      { tag: "div.code", contentElement: "code", preserveWhitespace: "full", getAttrs: (dom: HTMLElement) => ({ language: dom.dataset.language ?? "" }) },
      {
        tag: "pre",
        preserveWhitespace: "full",
        getAttrs: (dom: HTMLElement) => {
          const cls = dom.querySelector("code")?.className ?? dom.className;
          const m = /language-([\w+#-]+)/.exec(cls);
          return { language: m ? m[1]!.toLowerCase() : "" };
        },
      },
    ],
    toDOM: (n) => {
      const a: Attrs = { class: "code", ...blockAttr(n) };
      if (n.attrs.language) a["data-language"] = String(n.attrs.language);
      return ["div", a, ["div", { class: "code__bar", contenteditable: "false" }], ["pre", { class: "code__pre", spellcheck: "false" }, ["code", 0]]];
    },
  },
  horizontal_rule: {
    group: "block",
    attrs: withId,
    parseDOM: [{ tag: "hr" }],
    toDOM: (n) => ["hr", blockAttr(n)],
  },
  image: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, src: { default: "" }, alt: { default: "" }, caption: { default: "" }, w: { default: null }, h: { default: null } },
    parseDOM: [
      mediaParse("image"),
      {
        tag: "img[src]",
        getAttrs: (dom: HTMLElement) => ({
          src: refFromSrc(dom.getAttribute("src")),
          alt: dom.getAttribute("alt") ?? "",
          w: Number(dom.getAttribute("width")) || null,
          h: Number(dom.getAttribute("height")) || null,
        }),
      },
    ],
    toDOM: (n) => media("image", n),
  },
  video: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, src: { default: "" }, caption: { default: "" }, w: { default: null }, h: { default: null } },
    parseDOM: [mediaParse("video")],
    toDOM: (n) => media("video", n),
  },
  audio: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, src: { default: "" }, caption: { default: "" } },
    parseDOM: [mediaParse("audio")],
    toDOM: (n) => media("audio", n),
  },
  file: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, src: { default: "" }, name: { default: "" }, size: { default: 0 }, mime: { default: "" } },
    parseDOM: [
      {
        tag: "div.file",
        getAttrs: (dom: HTMLElement) => {
          const a = dom.querySelector("a");
          return { src: refFromSrc(a?.getAttribute("href") ?? ""), name: a?.textContent ?? "", mime: dom.dataset.mime ?? "" };
        },
      },
    ],
    toDOM: (n) => {
      const src = fileSrc(String(n.attrs.src ?? ""));
      const mime = String(n.attrs.mime ?? "");
      const a: Attrs = { class: "file", ...blockAttr(n) };
      if (mime) a["data-mime"] = mime;
      const card: DOMOutputSpec = [
        "div",
        { class: "file__card" },
        ["span", { class: "file__icon" }],
        ["a", { class: "file__name", href: src, target: "_blank", rel: "noopener" }, String(n.attrs.name ?? "")],
        ["span", { class: "file__size" }, humanSize(Number(n.attrs.size) || 0)],
      ];
      if (mime === "application/pdf") return ["div", a, card, ["iframe", { class: "file__pdf", src, title: String(n.attrs.name ?? ""), loading: "lazy" }]];
      return ["div", a, card];
    },
  },
  bookmark: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, url: { default: "" }, title: { default: "" }, description: { default: "" }, image: { default: "" }, site: { default: "" } },
    parseDOM: [
      {
        tag: "div.bookmark",
        getAttrs: (dom: HTMLElement) => ({
          url: dom.querySelector("a")?.getAttribute("href") ?? "",
          title: dom.querySelector(".bookmark__title")?.textContent ?? "",
          description: dom.querySelector(".bookmark__desc")?.textContent ?? "",
        }),
      },
    ],
    toDOM: (n) => {
      const url = safeHref(String(n.attrs.url ?? ""));
      const title = String(n.attrs.title || url);
      const inner: DOMOutputSpec[] = [
        ["span", { class: "bookmark__text" }, ["span", { class: "bookmark__title" }, title], ["span", { class: "bookmark__desc" }, String(n.attrs.description ?? "")], ["span", { class: "bookmark__url" }, url]],
      ];
      const img = String(n.attrs.image ?? "");
      if (img.startsWith("f:")) inner.push(["img", { class: "bookmark__image", src: fileSrc(img), alt: "", loading: "lazy" }]);
      return ["div", { class: "bookmark", ...blockAttr(n) }, ["a", { class: "bookmark__link", href: url, target: "_blank", rel: "noopener noreferrer nofollow" }, ...inner]];
    },
  },
  embed: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, url: { default: "" }, provider: { default: "" } },
    parseDOM: [{ tag: "figure.embed", getAttrs: (dom: HTMLElement) => ({ url: dom.querySelector("iframe")?.getAttribute("title") ?? "", provider: dom.dataset.provider ?? "" }) }],
    toDOM: (n) => {
      const url = String(n.attrs.url ?? "");
      const a: Attrs = { class: "embed", ...blockAttr(n) };
      if (n.attrs.provider) a["data-provider"] = String(n.attrs.provider);
      const src = embedSrc(url);
      if (!src) return ["figure", a];
      return [
        "figure",
        a,
        [
          "iframe",
          {
            src,
            title: url,
            loading: "lazy",
            allow: "encrypted-media; picture-in-picture; fullscreen",
            referrerpolicy: "strict-origin-when-cross-origin",
            sandbox: "allow-scripts allow-same-origin allow-presentation allow-popups",
          },
        ],
      ];
    },
  },
  toc: {
    group: "block",
    atom: true,
    attrs: withId,
    parseDOM: [{ tag: "nav.toc" }],
    toDOM: (n) => ["nav", { class: "toc", ...blockAttr(n), contenteditable: "false" }],
  },
  columns: {
    content: "column{2,3}",
    group: "block",
    isolating: true,
    attrs: withId,
    parseDOM: [{ tag: "div.columns" }],
    toDOM: (n) => ["div", { class: "columns", ...blockAttr(n), "data-count": String(n.childCount) }, 0],
  },
  column: {
    content: "block+",
    isolating: true,
    parseDOM: [{ tag: "div.column" }],
    toDOM: () => ["div", { class: "column" }, 0],
  },
  text: { group: "inline" },
  hard_break: {
    inline: true,
    group: "inline",
    selectable: false,
    parseDOM: [{ tag: "br" }],
    toDOM: () => ["br"],
  },
  mention: {
    inline: true,
    group: "inline",
    atom: true,
    attrs: { kind: { default: "page" }, ref: { default: "" }, label: { default: "" } },
    parseDOM: [
      {
        tag: ".mention[data-kind]",
        getAttrs: (dom: HTMLElement) => ({ kind: dom.dataset.kind ?? "page", ref: dom.dataset.ref ?? "", label: dom.querySelector(".mention__label")?.textContent ?? dom.textContent?.replace(/^@/, "") ?? "" }),
      },
    ],
    toDOM: (n) => {
      const kind = String(n.attrs.kind);
      const ref = String(n.attrs.ref);
      const label = String(n.attrs.label ?? "");
      if (kind === "page") return ["a", { class: "mention", "data-kind": "page", "data-ref": ref, href: `/p/${ref}`, contenteditable: "false" }, ["span", { class: "mention__icon" }], ["span", { class: "mention__label" }, label || "Untitled"]];
      if (kind === "user") return ["span", { class: "mention", "data-kind": "user", "data-ref": ref, contenteditable: "false" }, `@${label}`];
      return ["span", { class: "mention", "data-kind": "date", "data-ref": ref, contenteditable: "false" }, ref];
    },
  },
};

const marks: Record<string, MarkSpec> = {
  link: {
    attrs: { href: { default: "" } },
    inclusive: false,
    parseDOM: [{ tag: "a[href]:not(.mention):not(.bookmark__link):not(.file__name)", getAttrs: (dom: HTMLElement) => ({ href: safeHref(dom.getAttribute("href") ?? "") }) }],
    toDOM: (m) => ["a", { href: safeHref(String(m.attrs.href)), rel: "noopener noreferrer nofollow" }, 0],
  },
  strong: {
    parseDOM: [
      { tag: "strong" },
      { tag: "b", getAttrs: (dom: HTMLElement) => (dom.style.fontWeight !== "normal" ? null : false) },
      { style: "font-weight=400", clearMark: (m) => m.type.name === "strong" },
      { style: "font-weight", getAttrs: (v: string) => (/^(bold(er)?|[5-9]\d{2,})$/.test(v) ? null : false) },
    ],
    toDOM: () => ["strong", 0],
  },
  em: {
    parseDOM: [{ tag: "i" }, { tag: "em" }, { style: "font-style=italic" }, { style: "font-style=normal", clearMark: (m) => m.type.name === "em" }],
    toDOM: () => ["em", 0],
  },
  strike: {
    parseDOM: [{ tag: "s" }, { tag: "del" }, { tag: "strike" }, { style: "text-decoration=line-through" }],
    toDOM: () => ["s", 0],
  },
  underline: {
    parseDOM: [{ tag: "u" }, { style: "text-decoration=underline" }],
    toDOM: () => ["u", 0],
  },
  highlight: {
    attrs: { value: { default: "amber" } },
    parseDOM: [{ tag: "mark", getAttrs: (dom: HTMLElement) => ({ value: dom.dataset.color ?? "amber" }) }],
    toDOM: (m) => ["mark", { "data-color": String(m.attrs.value) }, 0],
  },
  color: {
    attrs: { value: { default: "" } },
    parseDOM: [{ tag: "span.tc[data-color]", getAttrs: (dom: HTMLElement) => ({ value: dom.dataset.color ?? "" }) }],
    toDOM: (m) => ["span", { class: "tc", "data-color": String(m.attrs.value) }, 0],
  },
  code: {
    parseDOM: [{ tag: "code" }],
    toDOM: () => ["code", 0],
  },
};

const tables = tableNodes({ tableGroup: "block", cellContent: "paragraph+", cellAttributes: {} });
const tableSpec: NodeSpec = { ...tables.table, attrs: withId, toDOM: (n) => ["table", blockAttr(n), ["tbody", 0]] };

export const schema = new Schema({
  nodes: { ...nodes, table: tableSpec, table_row: tables.table_row, table_cell: tables.table_cell, table_header: tables.table_header },
  marks,
});

export const BLOCKS_WITH_ID = new Set(
  Object.entries(schema.nodes)
    .filter(([, t]) => "id" in (t.spec.attrs ?? {}))
    .map(([name]) => name),
);
