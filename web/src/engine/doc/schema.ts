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

interface Provider {
  id: string;
  re: RegExp;
  src: (m: RegExpExecArray, url: string) => string;
  height: number;
}

const q = encodeURIComponent;

const PROVIDERS: Provider[] = [
  { id: "youtube", re: /^https:\/\/(?:www\.)?(?:youtube\.com\/watch\?(?:.*&)?v=|youtu\.be\/|youtube\.com\/shorts\/)([A-Za-z0-9_-]{6,20})/, src: (m) => `https://www.youtube-nocookie.com/embed/${m[1]}`, height: 0 },
  { id: "vimeo", re: /^https:\/\/(?:www\.)?vimeo\.com\/(\d{4,12})/, src: (m) => `https://player.vimeo.com/video/${m[1]}?dnt=1`, height: 0 },
  { id: "loom", re: /^https:\/\/(?:www\.)?loom\.com\/(?:share|embed)\/([a-f0-9]{16,40})/, src: (m) => `https://www.loom.com/embed/${m[1]}`, height: 0 },
  { id: "dailymotion", re: /^https:\/\/(?:www\.)?dailymotion\.com\/video\/([A-Za-z0-9]{5,12})/, src: (m) => `https://www.dailymotion.com/embed/video/${m[1]}`, height: 0 },
  { id: "tiktok", re: /^https:\/\/(?:www\.)?tiktok\.com\/@[\w.-]+\/video\/(\d{6,25})/, src: (m) => `https://www.tiktok.com/embed/v2/${m[1]}`, height: 740 },
  { id: "figma", re: /^https:\/\/(?:www\.)?figma\.com\/(?:file|design|proto|board|slides)\/[A-Za-z0-9]{10,40}/, src: (_m, url) => `https://www.figma.com/embed?embed_host=formmd&url=${q(url)}`, height: 450 },
  { id: "codepen", re: /^https:\/\/codepen\.io\/([\w-]{1,40})\/(?:pen|full|details)\/([A-Za-z0-9]{4,12})/, src: (m) => `https://codepen.io/${m[1]}/embed/${m[2]}?default-tab=result`, height: 400 },
  { id: "codesandbox", re: /^https:\/\/codesandbox\.io\/(?:s|embed|p\/sandbox)\/([\w-]{3,80})/, src: (m) => `https://codesandbox.io/embed/${m[1]}`, height: 500 },
  { id: "spotify", re: /^https:\/\/open\.spotify\.com\/(track|album|playlist|episode|show|artist)\/([A-Za-z0-9]{10,30})/, src: (m) => `https://open.spotify.com/embed/${m[1]}/${m[2]}`, height: 352 },
  { id: "soundcloud", re: /^https:\/\/(?:www\.)?soundcloud\.com\/[\w-]{1,60}\/[\w-]{1,120}/, src: (_m, url) => `https://w.soundcloud.com/player/?url=${q(url)}`, height: 166 },
  { id: "gdocs", re: /^https:\/\/docs\.google\.com\/(document|spreadsheets|presentation)\/d\/([\w-]{20,80})/, src: (m) => `https://docs.google.com/${m[1]}/d/${m[2]}/preview`, height: 600 },
  { id: "gforms", re: /^https:\/\/docs\.google\.com\/forms\/d\/e\/([\w-]{20,80})/, src: (m) => `https://docs.google.com/forms/d/e/${m[1]}/viewform?embedded=true`, height: 600 },
  { id: "gdrive", re: /^https:\/\/drive\.google\.com\/file\/d\/([\w-]{20,80})/, src: (m) => `https://drive.google.com/file/d/${m[1]}/preview`, height: 480 },
  { id: "gmaps", re: /^https:\/\/(?:www\.|maps\.)?google\.[a-z.]{2,6}\/maps(?:\/place\/([^/?#]+)|.*[?&]q=([^&#]+))/, src: (m) => `https://www.google.com/maps?output=embed&q=${m[1] || m[2] || ""}`, height: 400 },
  { id: "miro", re: /^https:\/\/miro\.com\/app\/board\/([\w=-]{6,40})/, src: (m) => `https://miro.com/app/live-embed/${m[1]}/`, height: 450 },
  { id: "airtable", re: /^https:\/\/airtable\.com\/(?:embed\/)?((?:app\w{10,20}\/)?shr\w{10,20})/, src: (m) => `https://airtable.com/embed/${m[1]}`, height: 533 },
  { id: "typeform", re: /^https:\/\/[\w-]{1,60}\.typeform\.com\/to\/[A-Za-z0-9]{4,20}/, src: (m) => m[0], height: 500 },
  { id: "excalidraw", re: /^https:\/\/excalidraw\.com\//, src: (_m, url) => url, height: 500 },
];

const WEB_HEIGHT = 480;

export function embedSrc(url: string, kind = ""): { src: string; height: number } {
  for (const p of PROVIDERS) {
    const m = p.re.exec(url);
    if (m) return { src: p.src(m, url), height: p.height };
  }
  if (kind === "web") {
    try {
      const u = new URL(url);
      if (u.protocol === "https:" && u.host && !u.username && !u.password && !/["<> ]/.test(url) && u.origin !== globalThis.location?.origin) return { src: url, height: WEB_HEIGHT };
    } catch {
      return { src: "", height: 0 };
    }
  }
  return { src: "", height: 0 };
}

export function embedProvider(url: string): string {
  return PROVIDERS.find((p) => p.re.test(url))?.id ?? "";
}

export const EMBED_SANDBOX = "allow-scripts allow-same-origin allow-presentation allow-popups allow-popups-to-escape-sandbox allow-forms";
export const EMBED_ALLOW = "encrypted-media; picture-in-picture; fullscreen; clipboard-write";

export function isRemote(src: string): boolean {
  return /^https?:\/\//i.test(src);
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
  if (isRemote(src)) return ["figure", { class: `media media--${kind} is-remote`, "data-src": src, ...blockAttr(n) }, ["div", { class: "media__pending" }, ["span", { class: "media__pending-text" }, src]], caption];
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
      src: refFromSrc(el?.getAttribute("src") ?? dom.dataset.src ?? ""),
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
  ai_block: {
    content: "block+",
    group: "block",
    defining: true,
    attrs: { ...withId, prompt: { default: "" } },
    parseDOM: [
      {
        tag: "div.ai-block",
        contentElement: ".ai-block__body",
        getAttrs: (dom: HTMLElement) => ({ prompt: dom.dataset.prompt ?? "" }),
      },
    ],
    toDOM: (n) => [
      "div",
      { class: "ai-block", ...blockAttr(n), "data-prompt": String(n.attrs.prompt ?? "") },
      ["div", { class: "ai-block__head", contenteditable: "false" }, ["span", { class: "ai-block__prompt" }, String(n.attrs.prompt ?? "")]],
      ["div", { class: "ai-block__body" }, 0],
    ],
  },
  meeting: {
    content: "block+",
    group: "block",
    defining: true,
    attrs: { ...withId, title: { default: "" }, started: { default: 0 }, duration: { default: 0 }, state: { default: "" } },
    parseDOM: [
      {
        tag: "div.meeting",
        contentElement: ".meeting__body",
        getAttrs: (dom: HTMLElement) => ({ title: dom.dataset.title ?? "", started: Number(dom.dataset.started) || 0, duration: Number(dom.dataset.duration) || 0, state: dom.dataset.state ?? "" }),
      },
    ],
    toDOM: (n) => [
      "div",
      { class: "meeting", ...blockAttr(n), "data-title": String(n.attrs.title ?? ""), "data-started": String(n.attrs.started ?? 0), "data-duration": String(n.attrs.duration ?? 0), "data-state": String(n.attrs.state ?? "") },
      ["div", { class: "meeting__head", contenteditable: "false" }, ["span", { class: "meeting__title" }, String(n.attrs.title || "Meeting notes")]],
      ["div", { class: "meeting__body" }, 0],
    ],
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
    attrs: { ...withId, language: { default: "" }, view: { default: null }, diagram: { default: "" }, dw: { default: null }, dh: { default: null }, dsum: { default: "" } },
    parseDOM: [
      {
        tag: "div.code",
        contentElement: "code",
        preserveWhitespace: "full",
        getAttrs: (dom: HTMLElement) => {
          const img = dom.querySelector(".diagram__img");
          const src = img?.getAttribute("src") ?? "";
          const ok = src.startsWith("data:image/svg+xml;base64,");
          return {
            language: dom.dataset.language ?? "",
            view: dom.dataset.view ?? null,
            diagram: ok ? src : "",
            dw: ok ? Number(img?.getAttribute("width")) || null : null,
            dh: ok ? Number(img?.getAttribute("height")) || null : null,
            dsum: ok ? (dom.dataset.sum ?? "") : "",
          };
        },
      },
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
      if (n.attrs.view) a["data-view"] = String(n.attrs.view);
      const diagram = String(n.attrs.diagram ?? "");
      if (n.attrs.language !== "mermaid" || !diagram.startsWith("data:image/svg+xml;base64,"))
        return ["div", a, ["div", { class: "code__bar", contenteditable: "false" }], ["pre", { class: "code__pre", spellcheck: "false" }, ["code", 0]]];
      a["data-sum"] = String(n.attrs.dsum ?? "");
      const img: Attrs = { class: "diagram__img", src: diagram, alt: "" };
      if (n.attrs.dw) img.width = String(n.attrs.dw);
      if (n.attrs.dh) img.height = String(n.attrs.dh);
      return ["div", a, ["div", { class: "code__bar", contenteditable: "false" }], ["pre", { class: "code__pre", spellcheck: "false" }, ["code", 0]], ["div", { class: "diagram", contenteditable: "false" }, ["img", img]]];
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
    attrs: { ...withId, url: { default: "" }, title: { default: "" }, description: { default: "" }, image: { default: "" }, site: { default: "" }, icon: { default: "" }, fetched: { default: false } },
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
      const icon = String(n.attrs.icon ?? "");
      const line: DOMOutputSpec = icon.startsWith("f:")
        ? ["span", { class: "bookmark__url" }, ["img", { class: "bookmark__icon", src: fileSrc(icon), alt: "", width: "16", height: "16" }], url]
        : ["span", { class: "bookmark__url" }, url];
      const inner: DOMOutputSpec[] = [["span", { class: "bookmark__text" }, ["span", { class: "bookmark__title" }, title], ["span", { class: "bookmark__desc" }, String(n.attrs.description ?? "")], line]];
      const img = String(n.attrs.image ?? "");
      if (img.startsWith("f:")) inner.push(["img", { class: "bookmark__image", src: fileSrc(img), alt: "", loading: "lazy" }]);
      return ["div", { class: "bookmark", ...blockAttr(n) }, ["a", { class: "bookmark__link", href: url, target: "_blank", rel: "noopener noreferrer nofollow" }, ...inner]];
    },
  },
  embed: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, url: { default: "" }, provider: { default: "" }, height: { default: null } },
    parseDOM: [
      {
        tag: "figure.embed",
        getAttrs: (dom: HTMLElement) => ({ url: dom.querySelector("iframe")?.getAttribute("title") ?? "", provider: dom.dataset.provider ?? "", height: Number(dom.querySelector("iframe")?.getAttribute("height")) || null }),
      },
    ],
    toDOM: (n) => {
      const url = String(n.attrs.url ?? "");
      const a: Attrs = { class: "embed", ...blockAttr(n) };
      if (n.attrs.provider) a["data-provider"] = String(n.attrs.provider);
      const { src, height } = embedSrc(url, String(n.attrs.provider ?? ""));
      if (!src) return ["figure", a];
      const frame: Attrs = { src, title: url, loading: "lazy", allow: EMBED_ALLOW, referrerpolicy: "strict-origin-when-cross-origin", sandbox: EMBED_SANDBOX };
      const hh = Number(n.attrs.height) || height;
      if (hh) frame.height = String(hh);
      return ["figure", a, ["iframe", frame]];
    },
  },
  button: {
    group: "block",
    atom: true,
    selectable: true,
    attrs: { ...withId, label: { default: "" }, actions: { default: "[]" } },
    parseDOM: [{ tag: "div.docbtn", getAttrs: (d) => ({ id: (d as HTMLElement).dataset.block ?? null, label: (d as HTMLElement).dataset.label ?? "", actions: (d as HTMLElement).dataset.actions || "[]" }) }],
    toDOM: (n) => ["div", { class: "docbtn", ...blockAttr(n), "data-label": String(n.attrs.label), "data-actions": String(n.attrs.actions), contenteditable: "false" }, ["span", { class: "docbtn__label" }, String(n.attrs.label || "Button")]],
  },
  toc: {
    group: "block",
    atom: true,
    attrs: withId,
    parseDOM: [{ tag: "nav.toc" }],
    toDOM: (n) => ["nav", { class: "toc", ...blockAttr(n), contenteditable: "false" }],
  },
  math_block: {
    group: "block",
    atom: true,
    selectable: true,
    attrs: { ...withId, tex: { default: "" }, html: { default: "" } },
    parseDOM: [{ tag: "div.math-block[data-tex]", getAttrs: (d) => ({ id: (d as HTMLElement).dataset.block ?? null, tex: (d as HTMLElement).dataset.tex ?? "", html: "" }) }],
    toDOM: (n) => ["div", { class: "math-block", ...blockAttr(n), "data-tex": String(n.attrs.tex), contenteditable: "false" }],
  },
  breadcrumb: {
    group: "block",
    atom: true,
    attrs: withId,
    parseDOM: [{ tag: "nav.crumbline" }],
    toDOM: (n) => ["nav", { class: "crumbline", ...blockAttr(n), contenteditable: "false" }],
  },
  synced: {
    group: "block",
    atom: true,
    draggable: true,
    attrs: { ...withId, page: { default: "" }, key: { default: "" } },
    parseDOM: [{ tag: "div.synced[data-key]", getAttrs: (dom: HTMLElement) => ({ page: dom.dataset.page ?? "", key: dom.dataset.key ?? "" }) }],
    toDOM: (n) => ["div", { class: "synced", ...blockAttr(n), "data-page": String(n.attrs.page), "data-key": String(n.attrs.key), contenteditable: "false" }, ["div", { class: "synced__body" }]],
  },
  database: {
    group: "block",
    atom: true,
    selectable: true,
    draggable: false,
    attrs: { ...withId, collection: { default: "" }, view: { default: "" } },
    parseDOM: [{ tag: "div.dbblock[data-collection]", getAttrs: (d) => ({ id: (d as HTMLElement).dataset.block ?? null, collection: (d as HTMLElement).dataset.collection ?? "", view: (d as HTMLElement).dataset.view ?? "" }) }],
    toDOM: (n) => ["div", { class: "dbblock", ...blockAttr(n), "data-collection": String(n.attrs.collection), "data-view": String(n.attrs.view), contenteditable: "false" }],
  },
  columns: {
    content: "column{2,}",
    group: "block",
    isolating: true,
    attrs: withId,
    parseDOM: [{ tag: "div.columns" }],
    toDOM: (n) => ["div", { class: "columns", ...blockAttr(n), "data-count": String(n.childCount) }, 0],
  },
  column: {
    content: "block+",
    isolating: true,
    attrs: { width: { default: null } },
    parseDOM: [{ tag: "div.column", getAttrs: (d) => ({ width: Number((d as HTMLElement).dataset.w) || null }) }],
    toDOM: (n) => ["div", { class: "column", ...(n.attrs.width ? { "data-w": String(n.attrs.width) } : {}) }, 0],
  },
  text: { group: "inline" },
  hard_break: {
    inline: true,
    group: "inline",
    selectable: false,
    parseDOM: [{ tag: "br" }],
    toDOM: () => ["br"],
  },
  math_inline: {
    inline: true,
    group: "inline",
    atom: true,
    selectable: true,
    attrs: { tex: { default: "" }, html: { default: "" } },
    parseDOM: [{ tag: "span.math-inline[data-tex]", getAttrs: (d) => ({ tex: (d as HTMLElement).dataset.tex ?? "", html: "" }) }],
    toDOM: (n) => ["span", { class: "math-inline", "data-tex": String(n.attrs.tex), contenteditable: "false" }, String(n.attrs.tex)],
  },
  mention: {
    inline: true,
    group: "inline",
    atom: true,
    attrs: { kind: { default: "page" }, ref: { default: "" }, label: { default: "" }, remind: { default: "" }, at: { default: 0 } },
    parseDOM: [
      {
        tag: ".mention[data-kind]",
        getAttrs: (dom: HTMLElement) => ({
          kind: dom.dataset.kind ?? "page",
          ref: dom.dataset.ref ?? "",
          label: dom.querySelector(".mention__label")?.textContent ?? dom.textContent?.replace(/^@/, "") ?? "",
          remind: dom.dataset.remind ?? "",
          at: Number(dom.dataset.at ?? 0) || 0,
        }),
      },
    ],
    toDOM: (n) => {
      const kind = String(n.attrs.kind);
      const ref = String(n.attrs.ref);
      const label = String(n.attrs.label ?? "");
      if (kind === "page") return ["a", { class: "mention", "data-kind": "page", "data-ref": ref, href: `/p/${ref}`, contenteditable: "false" }, ["span", { class: "mention__icon" }], ["span", { class: "mention__label" }, label || "Untitled"]];
      if (kind === "user") return ["span", { class: "mention", "data-kind": "user", "data-ref": ref, contenteditable: "false" }, `@${label}`];
      if (n.attrs.remind) return ["span", { class: "mention", "data-kind": "date", "data-ref": ref, "data-remind": String(n.attrs.remind), "data-at": String(n.attrs.at ?? 0), contenteditable: "false" }, ref];
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
