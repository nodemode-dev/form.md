import type { Node as PMNode } from "prosemirror-model";
import { prosemirrorToYXmlFragment } from "y-prosemirror";
import { ROOT, schema } from "../../web/src/engine/doc/schema";
import { confirmDialog, simpleDialog } from "../../web/src/ui/dialog";
import { h, ref, setAttr, setText } from "../../web/src/ui/dom";
import { actions, button, field, input, select, textarea } from "../../web/src/ui/form";
import { type IconName, iconNode } from "../../web/src/ui/icons";
import { closeTop, hasLayers } from "../../web/src/ui/layers";
import { reconcile } from "../../web/src/ui/list";
import { type MenuEntry, openMenu } from "../../web/src/ui/menu";
import { fallbackGlyph, pageIcon, paintIcon } from "../../web/src/ui/pageicon";
import { toast } from "../../web/src/ui/toast";
import { buildTable, cellText, detectDelimiter, parseDelimited, type Table } from "./csv";
import { DbView } from "./dbview";
import { DemoEditor, FORMAT_TOOLS } from "./editor";
import { download, kindOf, readText } from "./files";
import { bodyMarkdown, addCollection, addPage, baseName, docFromMarkdown, docOf, fullMarkdown, get, list, removeItem, seed, splitTitle, subscribe, type DataItem, type Item, type PageItem } from "./store";

const THEME_KEY = "formmd.demo.theme";
const SIDEBAR_KEY = "formmd.demo.sidebar";

function remember(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    return;
  }
}

function hasText(item: Item, q: string): boolean {
  if (item.title.toLowerCase().includes(q)) return true;
  if (item.kind === "page") return bodyMarkdown(item).toLowerCase().includes(q);
  return item.rows.some((r) => r.title.toLowerCase().includes(q) || item.schema.some((p) => cellText(p, r).toLowerCase().includes(q)));
}

function tableFromDoc(doc: PMNode): Table | null {
  let found: PMNode | null = null;
  doc.descendants((node) => {
    if (found) return false;
    if (node.type.name === "table") found = node;
    return !found;
  });
  if (!found) return null;
  const grid: string[][] = [];
  (found as PMNode).forEach((row) => {
    const cells: string[] = [];
    row.forEach((cell) => cells.push(cell.textContent.trim()));
    grid.push(cells);
  });
  return grid.length >= 2 ? buildTable(grid) : null;
}

export class App {
  private readonly root = document.documentElement;
  private readonly nav = ref("nav");
  private readonly crumbs = ref("crumbs");
  private readonly titleEl = ref<HTMLTextAreaElement>("title");
  private readonly stats = ref("stats");
  private readonly tools = ref("tools");
  private readonly iconHost = ref("page-icon");
  private readonly viewPage = ref("view-page");
  private readonly coverImg = ref<HTMLImageElement>("cover-img");
  private readonly dbHost = ref("db-host");
  private readonly editorHost = ref("editor");
  private readonly searchEl = ref<HTMLInputElement>("search");
  private readonly fileInput = ref<HTMLInputElement>("file-input");
  private readonly veil = ref("veil");
  private readonly editor = new DemoEditor(this.editorHost);
  private readonly db: DbView;
  private readonly bigIcon = pageIcon("", "page", "page-icon--lg");
  private readonly list = h("ul", { class: "tree-group", attrs: { role: "tree" } });
  private readonly empty = h("p", { class: "sb-empty", text: "Nothing matches." });
  private readonly formatButtons = new Map<string, HTMLButtonElement>();
  private currentId = "";
  private statsFrame = 0;
  private dragDepth = 0;
  private welcomeId = "";

  constructor() {
    this.db = new DbView(this.dbHost, {
      insertMarkdown: (md) => this.pickPageForInsert(md),
      download,
      changed: () => this.renderTree(),
    });
    this.iconHost.append(this.bigIcon);
    this.nav.append(this.list, this.empty);
    this.empty.hidden = true;
    this.buildTools();
    this.bind();
    subscribe(() => this.renderTree());
    this.editor.onChange(() => {
      cancelAnimationFrame(this.statsFrame);
      this.statsFrame = requestAnimationFrame(() => {
        this.paintStats();
        this.paintFormat();
      });
    });
  }

  async start(): Promise<void> {
    this.welcomeId = await seed();
    this.paintHome();
    if (!location.hash) history.replaceState(null, "", `#${this.welcomeId}`);
    window.addEventListener("hashchange", () => this.route());
    this.route();
  }

  private bind(): void {
    ref("collapse").addEventListener("click", () => this.sidebar(false));
    ref("expand").addEventListener("click", () => this.sidebar(true));
    ref("sb-theme").addEventListener("click", () => this.toggleTheme());
    ref("sb-open").addEventListener("click", () => this.fileInput.click());
    ref("home-choose").addEventListener("click", () => this.fileInput.click());
    ref("home-paste").addEventListener("click", () => this.pasteDialog());
    ref("home-new").addEventListener("click", () => void this.newPage());
    ref("sb-reset").addEventListener("click", () => this.resetDialog());
    ref("sb-new").addEventListener("click", (e) => this.newMenu(e.currentTarget as HTMLElement));
    this.fileInput.addEventListener("change", () => {
      const files = Array.from(this.fileInput.files ?? []);
      this.fileInput.value = "";
      void this.importFiles(files);
    });
    this.searchEl.addEventListener("input", () => this.renderTree());
    this.editorHost.addEventListener("click", (e) => {
      const img = (e.target as HTMLElement).closest<HTMLImageElement>(".media--image img");
      if (img) this.zoom(img);
    });
    document.addEventListener("click", (e) => {
      if ((e.target as HTMLElement).closest('[data-command="layers.closeTop"]')) closeTop();
    });
    document.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && hasLayers()) {
        e.preventDefault();
        closeTop();
      }
    });
    this.list.addEventListener("click", (e) => {
      const row = (e.target as HTMLElement).closest<HTMLElement>(".tree-row");
      if (!row) return;
      if ((e.target as HTMLElement).closest("[data-action=delete]")) {
        this.deleteItem(row.dataset.id ?? "");
        return;
      }
      location.hash = row.dataset.id ?? "";
    });
    this.list.addEventListener("keydown", (e) => {
      if (e.key !== "Enter" && e.key !== " ") return;
      const row = (e.target as HTMLElement).closest<HTMLElement>(".tree-row");
      if (row) {
        e.preventDefault();
        location.hash = row.dataset.id ?? "";
      }
    });
    this.titleEl.addEventListener("input", () => {
      const item = get(this.currentId);
      if (!item) return;
      item.title = this.titleEl.value;
      this.renderTree();
      this.paintCrumbs(item);
    });
    this.titleEl.addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        this.editor.focus();
      }
    });
    document.addEventListener("dragenter", (e) => {
      if (!e.dataTransfer?.types.includes("Files")) return;
      this.dragDepth++;
      this.veil.hidden = false;
    });
    document.addEventListener("dragleave", () => {
      this.dragDepth = Math.max(0, this.dragDepth - 1);
      if (this.dragDepth === 0) this.veil.hidden = true;
    });
    document.addEventListener("dragover", (e) => {
      if (e.dataTransfer?.types.includes("Files")) e.preventDefault();
    });
    document.addEventListener("drop", (e) => {
      this.dragDepth = 0;
      this.veil.hidden = true;
      const files = Array.from(e.dataTransfer?.files ?? []);
      if (!files.length) return;
      e.preventDefault();
      void this.importFiles(files);
    });
  }

  private zoom(img: HTMLImageElement): void {
    const caption = img.closest(".media")?.querySelector(".media__caption")?.textContent ?? "";
    simpleDialog(
      caption || img.alt || "Photo",
      (body) => {
        const big = h("img", { class: "demo-zoom", attrs: { src: img.currentSrc || img.src, alt: img.alt } });
        body.append(h("div", { class: "demo-zoomwrap" }, big));
        return big;
      },
      undefined,
      "lg",
    );
  }

  private sidebar(open: boolean): void {
    this.root.setAttribute("data-sidebar", open ? "expanded" : "collapsed");
    remember(SIDEBAR_KEY, open ? "expanded" : "collapsed");
  }

  private toggleTheme(): void {
    const next = this.root.getAttribute("data-theme") === "dark" ? "light" : "dark";
    this.root.setAttribute("data-theme", next);
    this.root.setAttribute("data-theme-pref", next);
    remember(THEME_KEY, next);
  }

  private route(): void {
    const id = decodeURIComponent(location.hash.slice(1));
    const item = id && id !== "home" ? get(id) : undefined;
    if (!item) {
      this.currentId = "";
      this.root.setAttribute("data-route", "home");
      this.paintCrumbs(null);
      this.renderTree();
      return;
    }
    this.currentId = item.id;
    this.root.setAttribute("data-route", "page");
    setAttr(this.viewPage, "data-kind", item.kind === "collection" ? "collection" : "page");
    paintIcon(this.bigIcon, item.icon, fallbackGlyph(item.kind));
    this.viewPage.classList.toggle("has-cover", !!item.cover);
    if (item.cover && this.coverImg.getAttribute("src") !== item.cover) this.coverImg.setAttribute("src", item.cover);
    if (this.titleEl.value !== item.title) this.titleEl.value = item.title;
    this.tools.hidden = item.kind !== "page";
    this.stats.hidden = item.kind !== "page";
    this.editorHost.hidden = item.kind !== "page";
    this.dbHost.hidden = item.kind !== "collection";
    if (item.kind === "page") {
      this.editor.open(item);
      this.paintStats();
      this.paintFormat();
    } else this.db.open(item);
    this.paintCrumbs(item);
    this.renderTree();
    if (this.root.clientWidth < 760) this.sidebar(false);
  }

  private paintCrumbs(item: Item | null): void {
    const entries = [{ key: "ws", label: "Demo workspace", icon: "" }, ...(item ? [{ key: item.id, label: item.title || "Untitled", icon: item.icon }] : [{ key: "home", label: "Home", icon: "" }])];
    reconcile(
      this.crumbs,
      entries,
      (x) => x.key,
      () => h("span", { class: "crumbs__item" }, h("span", { class: "crumbs__sep", attrs: { "aria-hidden": "true" } }, iconNode("chevron-right")), h("span", { class: "crumbs__label" })),
      (el, x) => {
        setText(el.querySelector(".crumbs__label") as HTMLElement, x.label);
        setAttr(el, "aria-current", x.key === "ws" ? null : "page");
      },
    );
  }

  private renderTree(): void {
    const q = this.searchEl.value.trim().toLowerCase();
    const items = q ? list().filter((i) => hasText(i, q)) : [...list()];
    this.empty.hidden = items.length > 0;
    reconcile(
      this.list,
      items,
      (i) => i.id,
      (i) =>
        h(
          "li",
          { class: "tree-node", attrs: { role: "none" } },
          h(
            "div",
            { class: "tree-row", attrs: { role: "treeitem", tabindex: "0", "data-id": i.id } },
            h("span", { class: "tree-chev", attrs: { "aria-hidden": "true" } }),
            pageIcon(i.icon, fallbackGlyph(i.kind)),
            h("span", { class: "tree-title" }),
            h("span", { class: "tree-actions" }, h("button", { class: "tree-act", attrs: { type: "button", tabindex: "-1", "data-action": "delete", "aria-label": "Delete" } }, iconNode("trash"))),
          ),
        ),
      (li, i) => {
        const row = li.firstElementChild as HTMLElement;
        setAttr(row, "aria-current", i.id === this.currentId ? "page" : null);
        const title = row.querySelector(".tree-title") as HTMLElement;
        setText(title, i.title || "Untitled");
        setAttr(title, "data-untitled", i.title ? null : "true");
        paintIcon(row.querySelector(".page-icon") as HTMLElement, i.icon, fallbackGlyph(i.kind));
      },
    );
    this.paintHome();
  }

  private paintHome(): void {
    const grid = ref("home-grid");
    reconcile(
      grid,
      [...list()],
      (i) => i.id,
      (i) => {
        const card = h("a", { class: "hcard", attrs: { href: `#${i.id}` } }, h("span", { class: "hcard__preview" }, h("span", { class: "hcard__glyph" }, pageIcon("", "page")), h("img", { class: "hcard__img", attrs: { alt: "", decoding: "async", draggable: "false" } })), h("span", { class: "hcard__body" }, h("span", { class: "hcard__name" }), h("span", { class: "hcard__meta" })));
        return card;
      },
      (card, i) => {
        card.setAttribute("href", `#${i.id}`);
        const shot = card.querySelector(".hcard__img") as HTMLImageElement;
        const glyph = card.querySelector(".hcard__glyph") as HTMLElement;
        shot.hidden = !i.cover;
        glyph.hidden = !!i.cover;
        if (i.cover && shot.getAttribute("src") !== i.cover) shot.setAttribute("src", i.cover);
        setText(card.querySelector(".hcard__name") as HTMLElement, i.title || "Untitled");
        setText(card.querySelector(".hcard__meta") as HTMLElement, i.kind === "collection" ? `Database, ${i.rows.length} rows` : "Page");
        paintIcon(card.querySelector(".page-icon") as HTMLElement, i.icon, fallbackGlyph(i.kind));
      },
    );
  }

  private buildTools(): void {
    const strip = h("div", { class: "demo-fmt", attrs: { role: "toolbar", "aria-label": "Format" } });
    for (const t of FORMAT_TOOLS) {
      if (t.id === "h1") strip.append(h("span", { class: "demo-fmt__sep" }));
      if (t.id === "bullet") strip.append(h("span", { class: "demo-fmt__sep" }));
      const b = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t.label, title: t.label, "aria-pressed": "false" } }, iconNode(t.icon as IconName));
      b.addEventListener("mousedown", (e) => e.preventDefault());
      b.addEventListener("click", () => {
        this.editor.run(t);
        this.paintFormat();
      });
      this.formatButtons.set(t.id, b);
      strip.append(b);
    }
    const act = (label: string, icon: IconName, run: (btn: HTMLElement) => void) => {
      const b = h("button", { class: "btn btn--ghost btn--sm", attrs: { type: "button" } }, iconNode(icon), h("span", { text: label }));
      b.addEventListener("click", () => run(b));
      return b;
    };
    const row = h(
      "div",
      { class: "demo-actions" },
      act("Outline", "list-tree", (b) => this.outlineMenu(b)),
      act("Find and replace", "search", () => this.findDialog()),
      act("Export Markdown", "download", (b) => this.exportMenu(b)),
      act("Table to database", "table", () => this.tableToDatabase()),
    );
    this.tools.append(strip, row);
  }

  private paintFormat(): void {
    for (const t of FORMAT_TOOLS) {
      const b = this.formatButtons.get(t.id);
      if (b) setAttr(b, "aria-pressed", this.editor.active(t) ? "true" : "false");
    }
  }

  private paintStats(): void {
    const s = this.editor.stats();
    setText(this.stats, `${s.words.toLocaleString()} words · ${s.chars.toLocaleString()} characters · ${s.minutes} min read`);
  }

  private outlineMenu(anchor: HTMLElement): void {
    const heads = this.editor.outline();
    const entries: MenuEntry[] = heads.length ? heads.map((x) => ({ label: `${"  ".repeat(x.level - 1)}${x.text || "Untitled"}`, icon: (`h${Math.min(x.level, 3)}` as IconName), run: () => this.editor.scrollTo(x.pos) })) : [{ kind: "heading", label: "No headings on this page yet" }];
    openMenu(anchor, entries, { label: "Outline" });
  }

  private exportMenu(anchor: HTMLElement): void {
    const page = this.editor.current;
    if (!page) return;
    openMenu(
      anchor,
      [
        { label: "Download .md", icon: "download", run: () => download(`${page.title || "page"}.md`, fullMarkdown(page), "text/markdown") },
        { label: "Copy Markdown", icon: "copy", run: () => void navigator.clipboard?.writeText(fullMarkdown(page)).then(() => toast({ message: "Markdown copied", tone: "success" }), () => toast({ message: "Clipboard not available here", tone: "warning" })) },
        { label: "Show Markdown", icon: "code", run: () => this.showMarkdown(page) },
      ],
      { label: "Export" },
    );
  }

  private showMarkdown(page: PageItem): void {
    simpleDialog(
      "Markdown",
      (body, close) => {
        const area = textarea(fullMarkdown(page), 18);
        area.readOnly = true;
        area.spellcheck = false;
        area.classList.add("demo-mdout");
        const done = button("Close", "primary");
        done.addEventListener("click", close);
        body.append(area, actions(done));
        return area;
      },
      undefined,
      "lg",
    );
  }

  private findDialog(): void {
    simpleDialog("Find and replace", (body, close) => {
      const find = input({ placeholder: "Find", autocomplete: "off" });
      const repl = input({ placeholder: "Replace with", autocomplete: "off" });
      const caseBox = h("input", { attrs: { type: "checkbox" } });
      const note = h("p", { class: "field__hint" });
      const count = () => {
        const n = this.editor.find(find.value, caseBox.checked).length;
        setText(note, find.value ? `${n} match${n === 1 ? "" : "es"} on this page` : "Type to search this page");
      };
      find.addEventListener("input", count);
      caseBox.addEventListener("change", count);
      const go = button("Replace all", "primary");
      go.addEventListener("click", () => {
        const hits = this.editor.find(find.value, caseBox.checked);
        const n = this.editor.replace(hits, repl.value);
        toast({ message: n ? `Replaced ${n} match${n === 1 ? "" : "es"}` : "No matches", tone: n ? "success" : "info" });
        count();
      });
      const show = button("Jump to first", "ghost");
      show.addEventListener("click", () => {
        const first = this.editor.find(find.value, caseBox.checked)[0];
        if (!first) return;
        close();
        this.editor.scrollTo(first.from - 1);
      });
      const done = button("Close", "ghost");
      done.addEventListener("click", close);
      body.append(h("div", { class: "form-grid" }, field("Find", find), field("Replace with", repl), h("label", { class: "choice" }, caseBox, h("span", { text: "Match case" })), note, actions(show, done, go)));
      count();
      return find;
    });
  }

  private tableToDatabase(): void {
    const page = this.editor.current;
    if (!page) return;
    const table = tableFromDoc(docOf(page));
    if (!table) {
      toast({ message: "No table with a header and at least one row on this page", tone: "warning" });
      return;
    }
    const data = addCollection(`${page.title || "Page"} table`, table);
    toast({ message: `Database made from ${table.rows.length} rows`, tone: "success", action: { label: "Open", run: () => (location.hash = data.id) } });
    location.hash = data.id;
  }

  private pickPageForInsert(md: string): void {
    const pages = list().filter((i): i is PageItem => i.kind === "page");
    const anchor = ref("scroller");
    openMenu(
      anchor,
      [...pages.map((p): MenuEntry => ({ label: p.title || "Untitled", icon: "page", run: () => void this.insertInto(p, md) })), { kind: "separator" }, { label: "New page", icon: "plus", run: () => void this.insertInto(null, md) }],
      { label: "Insert into", side: "bottom", align: "center" },
    );
  }

  private async insertInto(page: PageItem | null, md: string): Promise<void> {
    const insert = await docFromMarkdown(md);
    if (!page) {
      const created = await addPage("Database export", md);
      location.hash = created.id;
      return;
    }
    const merged = schema.topNodeType.create(null, docOf(page).content.append(insert.content));
    page.ydoc.transact(() => {
      prosemirrorToYXmlFragment(merged, page.ydoc.getXmlFragment(ROOT));
    });
    toast({ message: `Inserted into ${page.title || "Untitled"}`, tone: "success" });
    location.hash = page.id;
  }

  private async newPage(): Promise<void> {
    const page = await addPage("", "");
    location.hash = page.id;
    requestAnimationFrame(() => this.titleEl.focus());
  }

  private newDatabase(): void {
    const grid = [["Name", "Status", "Due", "Done"], ["First task", "Open", "", "no"], ["Second task", "In review", "", "no"], ["Third task", "Closed", "", "yes"]];
    const table = buildTable(grid);
    const data = addCollection("Untitled database", table);
    location.hash = data.id;
  }

  private newMenu(anchor: HTMLElement): void {
    openMenu(
      anchor,
      [
        { label: "New page", icon: "page", run: () => void this.newPage() },
        { label: "New database", icon: "table", run: () => this.newDatabase() },
        { kind: "separator" },
        { label: "Open a file", icon: "upload", run: () => this.fileInput.click() },
        { label: "Paste text", icon: "text", run: () => this.pasteDialog() },
      ],
      { label: "New" },
    );
  }

  private deleteItem(id: string): void {
    const item = get(id);
    if (!item) return;
    void confirmDialog({ title: "Delete this item", body: `"${item.title || "Untitled"}" is removed from this tab. Samples return with Reset samples.`, confirm: "Delete", variant: "danger" }).then((ok) => {
      if (!ok) return;
      removeItem(id);
      if (this.currentId === id) location.hash = "home";
    });
  }

  private resetDialog(): void {
    void confirmDialog({ title: "Reset samples", body: "Everything in this tab, including files you opened, is replaced by the original samples.", confirm: "Reset", variant: "danger" }).then(async (ok) => {
      if (!ok) return;
      this.welcomeId = await seed();
      location.hash = this.welcomeId;
      this.route();
    });
  }

  private pasteDialog(): void {
    simpleDialog(
      "Paste text",
      (body, close) => {
        const area = textarea("", 12);
        area.placeholder = "Paste Markdown, or rows copied from a spreadsheet or CSV file";
        const mode = select<"auto" | "page" | "table">([["auto", "Detect"], ["page", "As a page"], ["table", "As a database"]], "auto");
        const go = button("Add", "primary");
        const cancel = button("Cancel", "ghost");
        cancel.addEventListener("click", close);
        go.addEventListener("click", () => {
          const text = area.value;
          if (!text.trim()) return;
          close();
          const asTable = mode.value === "table" || (mode.value === "auto" && this.looksTabular(text));
          void this.addText(text, "Pasted text", asTable);
        });
        body.append(h("div", { class: "form-grid" }, area, field("Treat as", mode), actions(cancel, go)));
        return area;
      },
      undefined,
      "lg",
    );
  }

  private looksTabular(text: string): boolean {
    const lines = text.split(/\r?\n/).filter((l) => l.trim());
    if (lines.length < 2 || lines[0]?.startsWith("#")) return false;
    const d = detectDelimiter(text);
    const n = (lines[0] as string).split(d).length;
    return n > 1 && lines.slice(0, 6).every((l) => l.split(d).length === n);
  }

  private async addText(text: string, name: string, asTable: boolean): Promise<void> {
    if (asTable) {
      const grid = parseDelimited(text);
      if (grid.length < 2) {
        toast({ message: "A database needs a header row and at least one row", tone: "warning" });
        return;
      }
      const data = addCollection(name, buildTable(grid));
      toast({ message: `${data.rows.length} rows, ${data.schema.length} columns`, tone: "success" });
      location.hash = data.id;
      return;
    }
    const { title, body } = splitTitle(text, name);
    const page = await addPage(title, body);
    location.hash = page.id;
  }

  async importFiles(files: File[]): Promise<void> {
    let last = "";
    for (const f of files) {
      const kind = kindOf(f);
      if (kind === "unknown") {
        toast({ message: `${f.name}: only Markdown, text, CSV and TSV files open in this demo`, tone: "warning" });
        continue;
      }
      try {
        const text = await readText(f);
        if (kind === "table") {
          const grid = parseDelimited(text);
          if (grid.length < 2) {
            toast({ message: `${f.name}: needs a header row and at least one row`, tone: "warning" });
            continue;
          }
          const data: DataItem = addCollection(baseName(f.name), buildTable(grid));
          last = data.id;
          toast({ message: `${f.name}: ${data.rows.length} rows, ${data.schema.length} columns`, tone: "success" });
        } else {
          const { title, body } = splitTitle(text, baseName(f.name));
          const page = await addPage(title, body);
          last = page.id;
          toast({ message: `${f.name} opened as a page`, tone: "success" });
        }
      } catch {
        toast({ message: `${f.name}: could not be read`, tone: "danger" });
      }
    }
    if (last) location.hash = last;
  }
}
