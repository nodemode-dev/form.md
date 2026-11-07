import "./main.css";
import { mountDatabase, type DatabaseHandle } from "../../../web/src/extensions/database/surface";
import { mountRowProps } from "../../../web/src/extensions/database/rowprops";
import { simpleDialog } from "../../../web/src/ui/dialog";
import { h, ref, setAttr, setText } from "../../../web/src/ui/dom";
import { iconNode } from "../../../web/src/ui/icons";
import { closeTop, hasLayers } from "../../../web/src/ui/layers";
import { toast } from "../../../web/src/ui/toast";
import { buildTable, parseDelimited } from "../../../demo/src/csv";
import { download, readText } from "../../../demo/src/files";
import { COL_ASTRONAUTS, COL_MISSIONS, COL_PROGRAMS } from "./data";
import { createDemo } from "./engine";
import type { SeedCollection } from "./seed";
import { fileUrl } from "./pageicon-shim";

const THEME_KEY = "formmd.demo.theme";
const SIDEBAR_KEY = "formmd.demo.sidebar";
const DEMO_DAY = Date.UTC(1969, 6, 20, 12, 0, 0);

function remember(key: string, value: string): void {
  try {
    localStorage.setItem(key, value);
  } catch {
    return;
  }
}

function installClock(): void {
  const offset = DEMO_DAY - Date.now();
  const Real = Date;
  class DemoDate extends Real {
    constructor(...args: unknown[]) {
      if (args.length === 0) super(Real.now() + offset);
      else super(...(args as [number]));
    }
  }
  globalThis.Date = DemoDate as unknown as DateConstructor;
}

const BLURBS: Record<string, { title: string; lead: string; icon: string }> = {
  [COL_MISSIONS]: { title: "Crewed missions", lead: "Every US crewed flight from Mercury to Apollo 17. Switch views, then filter, sort, group and edit: relations, rollups and formulas recompute as you go.", icon: "rocket" },
  [COL_ASTRONAUTS]: { title: "Astronauts", lead: "Everyone who flew, linked to their missions. Flight counts, hours and career span are rollups and formulas over the Flights relation.", icon: "users" },
  [COL_PROGRAMS]: { title: "Programs", lead: "Mercury, Gemini and Apollo. Each program rolls up its missions: flight hours, longest flight, crew seats and first and last dates.", icon: "database" },
};

installClock();
const demo = createDemo();
const { engine, backend } = demo;
const root = document.documentElement;
const host = ref("db-host");
const viewPage = ref("view-page");
const titleEl = ref("title");
const leadEl = ref("lead");
const iconHost = ref("page-icon");
const crumbs = ref("crumbs");
const navEl = ref("nav");
const fileInput = ref<HTMLInputElement>("file-input");
const tree = navEl.querySelector(".tree-group") as HTMLElement;

let handle: DatabaseHandle | null = null;
let current = COL_MISSIONS;
const custom = new Map<string, { title: string }>();

function paintCrumbs(title: string): void {
  crumbs.replaceChildren(
    h("span", { class: "crumbs__item" }, h("span", { class: "crumbs__label", text: "Spaceflight" })),
    h("span", { class: "crumbs__item", attrs: { "aria-current": "page" } }, h("span", { class: "crumbs__sep", attrs: { "aria-hidden": "true" } }, iconNode("chevron-right")), h("span", { class: "crumbs__label", text: title })),
  );
}

function paintNav(): void {
  for (const row of Array.from(tree.querySelectorAll<HTMLElement>(".tree-row"))) setAttr(row, "aria-current", row.dataset.id === current ? "page" : null);
}

function mount(colId: string): void {
  handle?.destroy();
  current = colId;
  const info = BLURBS[colId] ?? { title: custom.get(colId)?.title ?? "Database", lead: "A database made from your CSV file. Every view, filter, sort and group works on it.", icon: "database" };
  setText(titleEl, info.title);
  setText(leadEl, info.lead);
  const glyph = iconHost.querySelector(".page-icon");
  if (glyph) glyph.replaceChildren(iconNode(info.icon as Parameters<typeof iconNode>[0]));
  paintCrumbs(info.title);
  paintNav();
  host.replaceChildren();
  handle = mountDatabase(engine, host, colId, { inline: false });
}

demo.onOpen((id) => void peek(id));

async function peek(id: string): Promise<void> {
  let detail;
  try {
    detail = await engine.api.collections.row(id);
  } catch (e) {
    engine.reportError(e);
    return;
  }
  const row = detail.row;
  if (!row) return;
  const cover = detail.collection.schema.find((p) => p.type === "files");
  const files = cover ? (row.values[cover.id] as { id?: string }[] | undefined) : undefined;
  const photo = files?.[0]?.id;
  const props = mountRowProps(engine, h("div"));
  simpleDialog(
    row.title || "Untitled",
    (body) => {
      const wrap = h("div", { class: "demo-peek" });
      if (photo) wrap.append(h("img", { class: "demo-peek__img", attrs: { src: fileUrl(photo), alt: row.title, decoding: "async" } }));
      body.append(wrap);
      const rp = mountRowProps(engine, wrap);
      void rp.show(id, detail);
      return undefined;
    },
    () => props.hide(),
    "md",
  );
}

function newViews(schema: ReturnType<typeof buildTable>["schema"], id: string): SeedCollection["views"] {
  const pick = schema.find((p) => p.type === "status") ?? schema.find((p) => p.type === "select");
  const views: SeedCollection["views"] = [{ id: `${id}_t`, name: "Table", type: "table", config: {} }];
  if (pick) views.push({ id: `${id}_b`, name: "Board", type: "board", config: { group: { prop: pick.id } } });
  views.push({ id: `${id}_c`, name: "Chart", type: "chart", config: {} });
  return views;
}

async function openCsv(file: File): Promise<void> {
  const text = await readText(file);
  const table = buildTable(parseDelimited(text));
  const id = `col_csv_${Date.now().toString(36)}`;
  const title = file.name.replace(/\.[^.]+$/, "") || "Imported";
  backend.addCollection({
    id,
    title,
    icon: "",
    schema: table.schema,
    views: newViews(table.schema, id),
    rows: table.rows.map((r) => ({ id: r.id, title: r.title, vals: r.values })),
  });
  custom.set(id, { title });
  tree.append(
    h(
      "li",
      { class: "tree-node", attrs: { role: "none" } },
      h("div", { class: "tree-row", attrs: { role: "treeitem", tabindex: "0", "data-id": id } }, h("span", { class: "tree-chev", attrs: { "aria-hidden": "true" } }), h("span", { class: "page-icon" }, iconNode("database")), h("span", { class: "tree-title", text: title })),
    ),
  );
  mount(id);
  toast({ id: "csv", tone: "success", message: `${title}: ${table.rows.length} rows, ${table.schema.length} typed columns` });
}

async function resetData(): Promise<void> {
  backend.reset();
  for (const row of Array.from(tree.querySelectorAll<HTMLElement>(".tree-row"))) if (custom.has(row.dataset.id ?? "")) row.closest("li")?.remove();
  custom.clear();
  await Promise.all([COL_MISSIONS, COL_ASTRONAUTS, COL_PROGRAMS].map((id) => engine.collections.load(id, true)));
  mount(BLURBS[current] ? current : COL_MISSIONS);
  toast({ id: "reset", tone: "success", message: "Data reset" });
}

tree.addEventListener("click", (e) => {
  const row = (e.target as HTMLElement).closest<HTMLElement>(".tree-row");
  if (row?.dataset.id && row.dataset.id !== current) mount(row.dataset.id);
});
tree.addEventListener("keydown", (e) => {
  if (e.key !== "Enter" && e.key !== " ") return;
  const row = (e.target as HTMLElement).closest<HTMLElement>(".tree-row");
  if (row?.dataset.id) {
    e.preventDefault();
    if (row.dataset.id !== current) mount(row.dataset.id);
  }
});

ref("collapse").addEventListener("click", () => {
  root.setAttribute("data-sidebar", "collapsed");
  remember(SIDEBAR_KEY, "collapsed");
});
ref("expand").addEventListener("click", () => {
  root.setAttribute("data-sidebar", "expanded");
  remember(SIDEBAR_KEY, "expanded");
});
ref("sb-theme").addEventListener("click", () => {
  const next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
  root.setAttribute("data-theme", next);
  root.setAttribute("data-theme-pref", next);
  remember(THEME_KEY, next);
});
ref("sb-open").addEventListener("click", () => fileInput.click());
ref("sb-reset").addEventListener("click", () => void resetData());
fileInput.addEventListener("change", () => {
  const file = fileInput.files?.[0];
  fileInput.value = "";
  if (file) void openCsv(file).catch(engine.reportError);
});

document.addEventListener("click", (e) => {
  const target = e.target as HTMLElement;
  if (target.closest('[data-command="layers.closeTop"]')) closeTop();
  const a = target.closest<HTMLAnchorElement>("a[download]");
  const href = a?.getAttribute("href") ?? "";
  if (a && /\/export\.csv$/.test(href)) {
    e.preventDefault();
    void backend
      .fetch(href)
      .then((r) => r.text())
      .then((text) => {
        const name = `${(BLURBS[current]?.title ?? custom.get(current)?.title ?? "database").toLowerCase().replace(/[^a-z0-9]+/g, "-")}.csv`;
        download(name, text, "text/csv");
      });
  }
});
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && hasLayers()) {
    e.preventDefault();
    closeTop();
  }
});
document.addEventListener("dragover", (e) => {
  if (e.dataTransfer?.types.includes("Files")) e.preventDefault();
});
document.addEventListener("drop", (e) => {
  const file = e.dataTransfer?.files[0];
  if (!file) return;
  e.preventDefault();
  if (/\.(csv|tsv)$/i.test(file.name)) void openCsv(file).catch(engine.reportError);
  else toast({ id: "drop", tone: "info", message: "Drop a CSV or TSV file to open it as a database" });
});

void engine.collections.load(COL_MISSIONS).then(() => {
  for (const id of [COL_ASTRONAUTS, COL_PROGRAMS]) void engine.collections.load(id).catch(engine.reportError);
  mount(COL_MISSIONS);
  viewPage.classList.add("has-cover");
});
