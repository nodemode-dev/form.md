import type MarkdownItType from "markdown-it";
import { DOMParser as PMDOMParser, type Fragment, type Mark, type Node as PMNode, type ResolvedPos, Slice } from "prosemirror-model";
import { schema } from "../../engine/doc/schema";

const MARK_ORDER = ["link", "strong", "em", "strike", "underline", "code"];

const ESCAPE = /[\\*_`[\]<~]/g;

function escapeText(s: string, lineStart: boolean, inTable: boolean): string {
  let out = s.replace(ESCAPE, (c) => `\\${c}`);
  if (inTable) out = out.replace(/\|/g, "\\|");
  if (lineStart) {
    if (/^[#>\-+=]/.test(out)) out = `\\${out}`;
    else out = out.replace(/^(\d+)([.)])/, "$1\\$2");
  }
  return out;
}

function fenceFor(s: string): string {
  let longest = 0;
  for (const m of s.matchAll(/`+/g)) longest = Math.max(longest, m[0].length);
  return "`".repeat(longest + 1);
}

function linkDest(href: string): string {
  return /[\s()<>]/.test(href) ? `<${href.replace(/</g, "%3C").replace(/>/g, "%3E")}>` : href;
}

const fileUrl = (ref: string) => (ref.startsWith("f:") ? `/files/${ref.slice(2)}` : ref);

interface Run {
  text: string;
  marks: readonly Mark[];
  raw?: boolean;
}

export class MarkdownWriter {
  constructor(private readonly title: (id: string) => string = () => "") {}

  doc(content: Fragment): string {
    const lines = this.blocks(nodesOf(content), true);
    const md = lines.join("\n").replace(/\n+$/, "");
    return md ? `${md}\n` : "";
  }

  private blocks(nodes: PMNode[], loose: boolean): string[] {
    const out: string[] = [];
    for (const n of nodes) {
      const lines = this.block(n);
      if (!lines.length) continue;
      if (out.length && loose) out.push("");
      out.push(...lines);
    }
    return out;
  }

  private block(n: PMNode): string[] {
    switch (n.type.name) {
      case "paragraph": {
        const s = this.inline(n, false);
        return s ? s.split("\n") : n.childCount === 0 ? ["<br>"] : [];
      }
      case "heading":
        return [`${"#".repeat(Math.min(6, Math.max(1, Number(n.attrs.level) || 1)))} ${this.inline(n, false).replace(/\\\n/g, " ")}`];
      case "code_block": {
        const code = n.textContent;
        let fence = "```";
        while (code.includes(fence)) fence += "`";
        return [`${fence}${String(n.attrs.language ?? "")}`, ...(code ? code.split("\n") : []), fence];
      }
      case "horizontal_rule":
        return ["---"];
      case "blockquote":
        return prefix(this.blocks(nodesOf(n.content), true), "> ", "> ");
      case "callout": {
        const kind = ({ green: "TIP", violet: "IMPORTANT", amber: "WARNING", orange: "WARNING", rose: "CAUTION", pink: "CAUTION" } as Record<string, string>)[String(n.attrs.tone)] ?? "NOTE";
        return prefix([`[!${kind}]`, ...this.blocks(nodesOf(n.content), true)], "> ", "> ");
      }
      case "toggle": {
        let summary = "";
        const body: PMNode[] = [];
        n.forEach((c) => {
          if (c.type.name === "toggle_title") summary = this.inline(c, false).replace(/\\\n/g, " ");
          else body.push(c);
        });
        const lvl = Number(n.attrs.level) || 0;
        if (lvl > 0) summary = `<h${lvl}>${summary}</h${lvl}>`;
        return ["<details>", `<summary>${summary}</summary>`, "", ...this.blocks(body, true), "", "</details>"];
      }
      case "bullet_list":
      case "ordered_list":
      case "task_list":
        return this.list(n);
      case "table":
        return this.table(n);
      case "image": {
        const caption = String(n.attrs.caption ?? "");
        return [`![${escapeText(String(n.attrs.alt ?? ""), false, false)}](${linkDest(fileUrl(String(n.attrs.src)))}${caption ? ` "${caption.replace(/"/g, '\\"')}"` : ""})`];
      }
      case "video":
      case "audio":
        return [`![${escapeText(String(n.attrs.caption || n.type.name), false, false)}](${linkDest(fileUrl(String(n.attrs.src)))})`];
      case "file":
        return [`[${escapeText(String(n.attrs.name || "file"), false, false)}](${linkDest(fileUrl(String(n.attrs.src)))})`];
      case "bookmark":
      case "embed":
        return [`<${String(n.attrs.url)}>`];
      case "toc":
        return ["<!-- toc -->"];
      case "button": {
        let actions: unknown = [];
        try {
          actions = JSON.parse(String(n.attrs.actions));
        } catch {
          actions = [];
        }
        const bytes = new TextEncoder().encode(JSON.stringify({ label: String(n.attrs.label), actions }));
        let bin = "";
        for (const b of bytes) bin += String.fromCharCode(b);
        return [`<!-- button:${btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")} -->`];
      }
      case "board": {
        let items: unknown = [];
        try {
          items = JSON.parse(String(n.attrs.items));
        } catch {
          items = [];
        }
        const bytes = new TextEncoder().encode(JSON.stringify({ height: Number(n.attrs.height) || 600, items }));
        let bin = "";
        for (const b of bytes) bin += String.fromCharCode(b);
        return [`<!-- board:${btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "")} -->`];
      }
      case "breadcrumb":
        return ["<!-- breadcrumb -->"];
      case "synced":
        return [`<!-- synced:${String(n.attrs.page)}/${String(n.attrs.key)} -->`];
      case "math_block":
        return ["$$", ...String(n.attrs.tex).split("\n"), "$$"];
      case "database":
        return [`<!-- database:${String(n.attrs.collection)} -->`];
      case "columns": {
        const out: string[] = [];
        n.forEach((col) => {
          const lines = this.blocks(nodesOf(col.content), true);
          if (!lines.length) return;
          if (out.length) out.push("");
          out.push(...lines);
        });
        return out;
      }
    }
    return this.blocks(nodesOf(n.content), true);
  }

  private list(n: PMNode): string[] {
    const out: string[] = [];
    let num = Number(n.attrs.start) || 1;
    let loose = false;
    n.forEach((item) => {
      if (item.childCount > 1) loose = true;
    });
    n.forEach((item) => {
      let marker = "- ";
      if (n.type.name === "ordered_list") marker = `${num++}. `;
      else if (n.type.name === "task_list") marker = item.attrs.checked ? "- [x] " : "- [ ] ";
      const indent = n.type.name === "task_list" ? "  " : " ".repeat(marker.length);
      const lines = this.blocks(nodesOf(item.content), loose);
      if (!lines.length) lines.push("");
      if (lines[0] === "<br>") lines[0] = "";
      if (loose && out.length) out.push("");
      out.push(...prefix(lines, marker, indent));
    });
    return out;
  }

  private table(n: PMNode): string[] {
    const rows: string[][] = [];
    let width = 0;
    n.forEach((row) => {
      const cells: string[] = [];
      row.forEach((cell) => {
        const parts: string[] = [];
        cell.forEach((p) => parts.push(this.inline(p, true).replace(/\\\n/g, "<br>").replace(/\n/g, " ")));
        cells.push(parts.join("<br>"));
      });
      width = Math.max(width, cells.length);
      rows.push(cells);
    });
    if (!rows.length || !width) return [];
    const line = (cells: string[]) => `| ${[...cells, ...Array(Math.max(0, width - cells.length)).fill("")].join(" | ")} |`;
    return [line(rows[0] ?? []), line(Array(width).fill("---")), ...rows.slice(1).map(line)];
  }

  inline(n: PMNode, inTable: boolean): string {
    const runs: Run[] = [];
    n.forEach((c) => {
      if (c.isText) runs.push({ text: c.text ?? "", marks: c.marks });
      else if (c.type.name === "hard_break") runs.push({ text: "\n", marks: [] });
      else if (c.type.name === "mention") {
        const kind = String(c.attrs.kind);
        const ref = String(c.attrs.ref);
        if (kind === "page") {
          const label = this.title(ref) || String(c.attrs.label) || "Untitled";
          runs.push({ text: label, marks: [schema.marks.link!.create({ href: `/p/${ref}` })] });
        } else if (kind === "user") runs.push({ text: `@${String(c.attrs.label)}`, marks: [] });
        else runs.push({ text: ref, marks: [] });
      } else if (c.type.name === "math_inline") runs.push({ text: `$${String(c.attrs.tex)}$`, marks: [], raw: true });
    });
    let out = "";
    let open: string[] = [];
    let openHref = "";
    let fence = "";
    const closeTo = (k: number) => {
      while (open.length > k) {
        const m = open.pop() as string;
        if (m === "link") out += `](${linkDest(openHref)})`;
        else if (m === "code") out += fence;
        else if (m === "underline") out += "</u>";
        else out += delim(m);
      }
    };
    let lineStart = true;
    for (const r of runs) {
      if (r.text === "\n") {
        closeTo(0);
        out += "\\\n";
        lineStart = true;
        continue;
      }
      if (r.raw) {
        closeTo(0);
        out += r.text;
        lineStart = false;
        continue;
      }
      const names = new Set(r.marks.map((m) => m.type.name));
      const want = MARK_ORDER.filter((m) => names.has(m));
      const href = String(r.marks.find((m) => m.type.name === "link")?.attrs.href ?? "");
      let keep = 0;
      while (keep < open.length && keep < want.length && open[keep] === want[keep] && (open[keep] !== "link" || href === openHref)) keep++;
      const lead = /^\s*/.exec(r.text)?.[0] ?? "";
      const trail = /\s*$/.exec(r.text.slice(lead.length))?.[0] ?? "";
      const core = r.text.slice(lead.length, r.text.length - trail.length);
      closeTo(keep);
      out += lead;
      if (!core) {
        out += trail;
        continue;
      }
      for (const m of want.slice(keep)) {
        if (m === "link") {
          openHref = href;
          out += "[";
        } else if (m === "code") {
          fence = fenceFor(core);
          out += fence;
        } else if (m === "underline") out += "<u>";
        else out += delim(m);
        open.push(m);
      }
      if (open.at(-1) === "code") out += core.startsWith("`") || core.endsWith("`") ? ` ${core} ` : core;
      else out += escapeText(core, lineStart, inTable);
      lineStart = false;
      if (trail) {
        closeTo(0);
        out += trail;
      }
    }
    closeTo(0);
    open = [];
    return out;
  }
}

function delim(m: string): string {
  return m === "strong" ? "**" : m === "em" ? "*" : m === "strike" ? "~~" : "";
}

function nodesOf(f: Fragment): PMNode[] {
  const out: PMNode[] = [];
  f.forEach((n) => out.push(n));
  return out;
}

function prefix(lines: string[], first: string, rest: string): string[] {
  return lines.map((l, i) => {
    const p = i === 0 ? first : rest;
    if (!l) return p.trimEnd();
    return p + l;
  });
}

let MarkdownIt: typeof MarkdownItType | null = null;
let loading: Promise<void> | null = null;

export function preloadMarkdown(): void {
  loading ??= import("markdown-it").then(
    (m) => {
      MarkdownIt = m.default;
    },
    () => {
      loading = null;
    },
  );
}

export async function markdownReady(): Promise<boolean> {
  preloadMarkdown();
  await loading;
  return MarkdownIt !== null;
}

const makeParser = () => new (MarkdownIt as typeof MarkdownItType)({ html: true, linkify: true, breaks: false });
let md: ReturnType<typeof makeParser> | null = null;

function parser(): ReturnType<typeof makeParser> {
  md ??= makeParser();
  return md;
}

const ALERT = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/;
const TONES: Record<string, string> = { NOTE: "", TIP: "green", IMPORTANT: "violet", WARNING: "amber", CAUTION: "rose" };

export function markdownToDOM(text: string): HTMLElement {
  const html = parser().render(text);
  const doc = new DOMParser().parseFromString(`<body>${html}</body>`, "text/html");
  const body = doc.body;
  for (const li of Array.from(body.querySelectorAll("li"))) {
    const host = li.firstElementChild?.tagName === "P" ? (li.firstElementChild as HTMLElement) : li;
    const first = host.firstChild;
    if (first?.nodeType !== 3) continue;
    const m = /^\[([ xX])\]\s?/.exec(first.textContent ?? "");
    if (!m) continue;
    first.textContent = (first.textContent ?? "").slice(m[0].length);
    const box = doc.createElement("input");
    box.type = "checkbox";
    if (m[1] !== " ") box.checked = true;
    li.insertBefore(box, li.firstChild);
    li.classList.add("md-task");
  }
  for (const ul of Array.from(body.querySelectorAll("ul"))) {
    if (ul.querySelector(":scope > li.md-task")) ul.classList.add("tasks");
  }
  for (const li of Array.from(body.querySelectorAll<HTMLElement>("li.md-task"))) {
    const checked = (li.querySelector(":scope > input") as HTMLInputElement | null)?.checked;
    li.querySelector(":scope > input")?.remove();
    li.classList.remove("md-task");
    li.classList.add("task");
    li.dataset.checked = checked ? "true" : "false";
    const wrap = doc.createElement("div");
    wrap.className = "task__body";
    while (li.firstChild) wrap.appendChild(li.firstChild);
    li.appendChild(wrap);
  }
  for (const q of Array.from(body.querySelectorAll("blockquote"))) {
    const p = q.firstElementChild;
    const t = p?.tagName === "P" ? p.firstChild : null;
    const m = t?.nodeType === 3 ? ALERT.exec(t.textContent ?? "") : null;
    if (!m || !p || !t) continue;
    t.textContent = (t.textContent ?? "").slice(m[0].length);
    if (t.textContent === "" && p.firstChild === t && t.nextSibling?.nodeName === "BR") p.removeChild(t.nextSibling);
    if (!(p.textContent ?? "").trim() && !p.querySelector("img")) p.remove();
    const callout = doc.createElement("div");
    callout.className = "doc-callout";
    callout.dataset.tone = TONES[m[1] as string] ?? "";
    const inner = doc.createElement("div");
    inner.className = "callout__body";
    while (q.firstChild) inner.appendChild(q.firstChild);
    if (!inner.firstChild) inner.appendChild(doc.createElement("p"));
    callout.appendChild(inner);
    q.replaceWith(callout);
  }
  for (const a of Array.from(body.querySelectorAll("a[href]"))) {
    const m = /^\/p\/([a-z2-7]{26})$/.exec(a.getAttribute("href") ?? "");
    if (!m) continue;
    const span = doc.createElement("span");
    span.className = "mention";
    span.dataset.kind = "page";
    span.dataset.ref = m[1] as string;
    const label = doc.createElement("span");
    label.className = "mention__label";
    label.textContent = a.textContent;
    span.appendChild(label);
    a.replaceWith(span);
  }
  return body;
}

const MD_HINT = /(^|\n)\s{0,3}(#{1,6}\s|[-*+]\s|\d+[.)]\s|>\s?|```|~~~|\|.*\|)|\*\*[^*\n]+\*\*|`[^`\n]+`|\[[^\]\n]+\]\([^)\s]+\)/;

export function looksLikeMarkdown(text: string): boolean {
  return MarkdownIt !== null && MD_HINT.test(text);
}

export function parseMarkdownSlice(text: string, $context: ResolvedPos): Slice {
  const dom = markdownToDOM(text);
  return PMDOMParser.fromSchema(schema).parseSlice(dom, { preserveWhitespace: false, context: $context });
}

export function plainTextSlice(text: string): Slice {
  const paras = text.replace(/\r\n?/g, "\n").split(/\n{2,}/);
  const nodes = paras.map((p) => {
    const parts = p.split("\n");
    const inline: PMNode[] = [];
    parts.forEach((line, i) => {
      if (i > 0) inline.push(schema.nodes.hard_break!.create());
      if (line) inline.push(schema.text(line));
    });
    return schema.nodes.paragraph!.create(null, inline);
  });
  return new Slice(schema.nodes.doc!.create(null, nodes).content, 1, 1);
}
