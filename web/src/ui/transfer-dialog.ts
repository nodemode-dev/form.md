import type { Engine } from "../engine";
import { t } from "../engine/i18n";
import { simpleDialog } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { button, choices, select, toggle } from "./form";
import { iconNode } from "./icons";

interface ImportJob {
  id: string;
  state: "queued" | "reading" | "writing" | "done" | "failed" | "cancelled";
  format: string;
  done: number;
  total: number;
  pages: number;
  rows: number;
  files: number;
  root_ids: string[];
  warnings: string[];
  error?: string;
}

export interface ImportTarget {
  workspaceId: string;
  parentId?: string;
}

const FORMATS: readonly (readonly [string, string])[] = [
  ["auto", "Detect automatically"],
  ["notion", "Notion export (.zip)"],
  ["evernote", "Evernote (.enex)"],
  ["markdown", "Markdown files or a zip"],
  ["html", "HTML files or a zip"],
  ["confluence", "Confluence space export (.zip)"],
  ["docx", "Word (.docx)"],
  ["xlsx", "Excel (.xlsx)"],
  ["csv", "CSV or Asana export"],
  ["trello", "Trello board (.json)"],
  ["opml", "OPML or WorkFlowy"],
  ["pdf", "PDF text"],
  ["text", "Plain text"],
];

const ACCEPT = ".zip,.md,.markdown,.txt,.html,.htm,.enex,.xml,.docx,.xlsx,.csv,.tsv,.json,.opml,.pdf";

function sizeText(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1 << 20) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1 << 20)).toFixed(1)} MB`;
}

function send(form: FormData, onProgress: (f: number) => void): Promise<ImportJob> {
  return new Promise((resolve, reject) => {
    const x = new XMLHttpRequest();
    x.open("POST", "/api/imports");
    x.setRequestHeader("Accept", "application/json");
    x.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    };
    x.onload = () => {
      type Reply = { job?: ImportJob; error?: { message?: string } };
      let body: Reply | null;
      try {
        body = JSON.parse(x.responseText) as Reply;
      } catch {
        body = null;
      }
      if (x.status === 202 && body?.job) resolve(body.job);
      else reject(new Error(body?.error?.message ?? t("The upload failed")));
    };
    x.onerror = () => reject(new Error(t("The server could not be reached")));
    x.send(form);
  });
}

export function openImport(engine: Engine, target: ImportTarget): void {
  const parent = target.parentId ? engine.pages.get(target.parentId) : undefined;
  const ws = engine.workspaceById(target.workspaceId);
  let stop = () => undefined as void;
  simpleDialog(
    t("Import"),
    (body, close) => {
      const where = h("p", { class: "muted", text: parent ? t("Pages are added inside {title}.", { title: parent.title || t("Untitled") }) : t("Pages are added to {name}.", { name: ws?.name ?? "" }) });
      const picker = h("input", { attrs: { type: "file", multiple: "", accept: ACCEPT, "aria-label": t("Choose files") } });
      picker.className = "xfer__file";
      const zone = h(
        "label",
        { class: "xfer__drop" },
        iconNode("upload"),
        h("span", { class: "xfer__droptitle", text: t("Drop files here or choose them") }),
        h("span", { class: "xfer__drophint", text: t("Notion and Confluence exports, Evernote, Markdown, HTML, Word, Excel, CSV, Asana, Trello, OPML and PDF") }),
        picker,
      );
      const list = h("ul", { class: "xfer__list", attrs: { role: "list" } });
      const fmt = select(FORMATS.map(([v, l]) => [v, t(l)] as const), "auto");
      const fmtField = h("label", { class: "field" }, h("span", { class: "field__label", text: t("Format") }), fmt);
      const bar = h("div", { class: "xfer__bar", attrs: { role: "progressbar", "aria-valuemin": "0", "aria-valuemax": "100", "aria-valuenow": "0" } }, h("span", { class: "xfer__fill" }));
      const line = h("p", { class: "xfer__line", attrs: { "aria-live": "polite" } });
      const warnBox = h("details", { class: "xfer__warn" }, h("summary", {}), h("ul", { attrs: { role: "list" } }));
      warnBox.hidden = true;
      const opens = h("div", { class: "xfer__opens" });
      const go = button(t("Import"), "primary", "upload");
      const cancel = button(t("Cancel"), "ghost");
      const actionsRow = h("div", { class: "form-actions" }, cancel, go);
      const state = h("div", { class: "xfer__state" }, bar, line, h("div", { class: "xfer__after" }, opens, warnBox));
      setText(line, t("Choose files, then Import."));
      body.append(h("div", { class: "xfer" }, where, zone, list, fmtField, state, actionsRow));
      let files: File[] = [];
      let job: ImportJob | null = null;
      let timer = 0;
      let running = false;
      let fresh = false;
      stop = () => window.clearTimeout(timer);
      const fill = bar.firstElementChild as HTMLElement;
      const setBar = (f: number) => {
        const pct = Math.max(0, Math.min(100, Math.round(f * 100)));
        fill.style.inlineSize = `${pct}%`;
        bar.setAttribute("aria-valuenow", String(pct));
      };
      const paintFiles = () => {
        list.replaceChildren(...files.map((f) => h("li", { class: "xfer__item" }, iconNode("attachment"), h("span", { class: "xfer__name", text: f.name }), h("span", { class: "xfer__size", text: sizeText(f.size) }))));
        go.disabled = files.length === 0 || running || fresh;
        setAttr(zone, "data-has", files.length ? "true" : null);
      };
      const add = (incoming: FileList | null) => {
        if (!incoming || !incoming.length || running) return;
        if (fresh) {
          fresh = false;
          files = [];
        }
        files = [...files, ...Array.from(incoming)].slice(0, 100);
        paintFiles();
      };
      picker.addEventListener("change", () => {
        add(picker.files);
        picker.value = "";
      });
      zone.addEventListener("dragover", (e) => {
        e.preventDefault();
        setAttr(zone, "data-over", "true");
      });
      zone.addEventListener("dragleave", () => setAttr(zone, "data-over", null));
      zone.addEventListener("drop", (e) => {
        e.preventDefault();
        setAttr(zone, "data-over", null);
        add(e.dataTransfer?.files ?? null);
      });
      const finish = (j: ImportJob) => {
        running = false;
        setAttr(state, "data-state", j.state);
        if (j.state === "done") {
          setBar(1);
          setText(line, t("Imported {pages} pages, {rows} rows and {files} files.", { pages: j.pages, rows: j.rows, files: j.files }));
          const roots = j.root_ids.slice(0, 3);
          opens.replaceChildren(
            ...roots.map((id) => {
              const b = button(engine.pages.get(id)?.title || t("Open"), "default", "page");
              b.addEventListener("click", () => {
                close();
                engine.openPage(id);
              });
              return b;
            }),
          );
        } else {
          setText(line, j.error ? t("The import stopped: {reason}", { reason: j.error }) : t("The import stopped."));
        }
        if (j.warnings.length) {
          warnBox.hidden = false;
          setText(warnBox.firstElementChild as HTMLElement, t("{n} notes from the import", { n: j.warnings.length }));
          (warnBox.lastElementChild as HTMLElement).replaceChildren(...j.warnings.map((w) => h("li", { text: w })));
        }
        fresh = true;
        go.disabled = true;
        setText(cancel.lastElementChild as HTMLElement, t("Close"));
      };
      const poll = async () => {
        if (!job) return;
        try {
          const r = await engine.api.http.get<{ job: ImportJob }>(`/api/imports/${encodeURIComponent(job.id)}`);
          job = r.job;
        } catch (e) {
          engine.reportError(e);
          return;
        }
        if (job.state === "done" || job.state === "failed" || job.state === "cancelled") return finish(job);
        if (job.state === "writing" && job.total > 0) {
          setBar(0.1 + 0.9 * (job.done / job.total));
          setText(line, t("Creating pages: {done} of {total}", { done: job.done, total: job.total }));
        } else {
          setText(line, job.state === "queued" ? t("Waiting for another import to finish") : t("Reading the files"));
        }
        timer = window.setTimeout(() => void poll(), 600);
      };
      go.addEventListener("click", async () => {
        if (!files.length || running || fresh) return;
        running = true;
        go.disabled = true;
        opens.replaceChildren();
        warnBox.hidden = true;
        setAttr(state, "data-state", "running");
        setBar(0);
        setText(line, t("Uploading"));
        const form = new FormData();
        form.set("workspace_id", target.workspaceId);
        if (target.parentId) form.set("parent_id", target.parentId);
        form.set("format", fmt.value);
        for (const f of files) form.append("file", f, f.name);
        try {
          job = await send(form, (f) => setBar(f * 0.1));
          void poll();
        } catch (e) {
          running = false;
          setAttr(state, "data-state", "failed");
          setText(line, e instanceof Error ? e.message : String(e));
          paintFiles();
        }
      });
      cancel.addEventListener("click", () => {
        if (running && job) void engine.api.http.del(`/api/imports/${encodeURIComponent(job.id)}`).catch(() => undefined);
        close();
      });
      paintFiles();
      return go;
    },
    () => stop(),
    "md",
  );
}

export interface ExportTarget {
  pageId?: string;
  workspaceId?: string;
}

function download(url: string): void {
  const a = h("a", { attrs: { href: url, download: "" } });
  a.hidden = true;
  document.body.append(a);
  a.click();
  a.remove();
}

export function openExport(engine: Engine, target: ExportTarget): void {
  const page = target.pageId ? engine.pages.get(target.pageId) : undefined;
  const hasKids = page ? engine.pages.children(page.workspace_id, page.id).length > 0 || page.kind === "collection" : true;
  simpleDialog(
    target.pageId ? t("Export page") : t("Export workspace"),
    (body, close) => {
      const opts: (readonly [string, string])[] = [
        ["markdown", t("Markdown and CSV")],
        ["html", t("HTML")],
      ];
      if (target.pageId) opts.push(["pdf", t("PDF")]);
      const fmt = choices("xfer-format", opts, "markdown");
      const subs = toggle(t("Include pages inside"), hasKids, t("Subpages and database rows go in as well, with links between them kept."));
      subs.el.hidden = !target.pageId;
      const note = h("p", { class: "muted" });
      const paint = () => {
        const v = fmt.value();
        setText(
          note,
          v === "markdown"
            ? t("A zip of Markdown files, with databases as CSV and attachments beside them.")
            : v === "html"
              ? t("Pages styled like a published site. One page comes as a single file; more come as a zip.")
              : t("A PDF document, one section per page, with pictures included."),
        );
      };
      fmt.el.addEventListener("change", paint);
      paint();
      const go = button(t("Export"), "primary", "download");
      const cancel = button(t("Cancel"), "ghost");
      go.addEventListener("click", () => {
        const v = fmt.value();
        const url = target.pageId
          ? `/api/pages/${encodeURIComponent(target.pageId)}/export?format=${v}&subpages=${subs.input.checked ? "1" : "0"}`
          : `/api/workspaces/${encodeURIComponent(target.workspaceId ?? "")}/export?format=${v}`;
        download(url);
        close();
      });
      cancel.addEventListener("click", close);
      body.append(h("div", { class: "xfer" }, page ? h("p", { class: "muted", text: page.title || t("Untitled") }) : h("span"), field2(t("Format"), fmt.el), subs.el, note, h("div", { class: "form-actions" }, cancel, go)));
      return go;
    },
    undefined,
    "sm",
  );
}

function field2(label: string, control: HTMLElement): HTMLElement {
  return h("div", { class: "field" }, h("span", { class: "field__label", text: label }), control);
}
