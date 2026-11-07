import type { Engine } from "../../../web/src/engine";
import { parseRoute } from "../../../web/src/engine/router";
import { h, ref, setAttr } from "../../../web/src/ui/dom";
import { iconNode } from "../../../web/src/ui/icons";
import { toast } from "../../../web/src/ui/toast";
import { importFiles } from "./importer";
import { openPreview, openPreviewBySlug } from "./preview";
import { sites } from "./server/db";
import { exportHtml, exportMarkdown, saveBlob } from "./server/export";

const ORIGIN = "https://fielddesk.example";

function pickFiles(engine: Engine): void {
  const input = h("input", { attrs: { type: "file", accept: ".md,.markdown,.txt,text/markdown,text/plain", multiple: "multiple" } });
  input.hidden = true;
  input.addEventListener("change", () => {
    const files = Array.from(input.files ?? []);
    input.remove();
    void importFiles(engine, files);
  });
  document.body.append(input);
  input.click();
}

async function runExport(url: URL): Promise<void> {
  const id = /^\/api\/pages\/([^/]+)\/export$/.exec(url.pathname)?.[1] ?? "";
  const format = url.searchParams.get("format") ?? "markdown";
  const subpages = url.searchParams.get("subpages") === "1";
  if (format === "pdf") {
    toast({ message: "PDF export needs the form.md server and is not part of this offline demo." });
    return;
  }
  const out = format === "html" ? await exportHtml(id, subpages) : await exportMarkdown(id, subpages);
  saveBlob(out.name, out.blob);
  toast({ tone: "success", message: `Exported ${out.name}` });
}

function previewButton(engine: Engine): void {
  const share = ref<HTMLElement>("act-share");
  const btn = h("button", { class: "btn btn--ghost topbar__published", attrs: { type: "button", "data-ref": "act-published" } }, iconNode("globe"), h("span", { text: "Published view" }));
  btn.hidden = true;
  share.before(btn);
  const sync = () => {
    const r = engine.state.get("route");
    const on = r.name === "page" && sites.has(r.id);
    btn.hidden = !on;
    setAttr(btn, "aria-hidden", on ? null : "true");
  };
  btn.addEventListener("click", () => {
    const r = engine.state.get("route");
    if (r.name === "page") openPreview(r.id);
  });
  engine.state.watch("route", sync, { immediate: true });
  window.addEventListener("demo:site", sync);
}

export function installGuards(engine: Engine): void {
  document.addEventListener(
    "click",
    (e) => {
      const target = e.target as Element;
      const a = target.closest<HTMLAnchorElement>("a[href]");
      if (a?.hasAttribute("download") && a.getAttribute("href")?.startsWith("/api/")) {
        e.preventDefault();
        e.stopPropagation();
        void runExport(new URL(a.getAttribute("href") ?? "", ORIGIN)).catch(engine.reportError);
        return;
      }
      if (target.closest("[data-ref='sb-import'], [data-ref='home-choose']")) {
        e.preventDefault();
        e.stopImmediatePropagation();
        pickFiles(engine);
        return;
      }
      if (!a || a.closest(".demo-links")) return;
      const href = a.getAttribute("href") ?? "";
      if (/^https?:/i.test(href)) {
        e.preventDefault();
        toast({ id: "offline-link", message: "This demo is offline, so links to other sites are switched off." });
        return;
      }
      if (href.startsWith("/")) {
        e.preventDefault();
        const q = href.indexOf("?");
        engine.navigate(parseRoute(q < 0 ? href : href.slice(0, q), q < 0 ? "" : href.slice(q)));
      }
    },
    true,
  );
  document.addEventListener(
    "drop",
    (e) => {
      const files = Array.from(e.dataTransfer?.files ?? []);
      if (!files.length) return;
      e.preventDefault();
      e.stopPropagation();
      void importFiles(engine, files);
    },
    true,
  );
  document.addEventListener(
    "dragover",
    (e) => {
      if (e.dataTransfer?.types.includes("Files")) e.preventDefault();
    },
    true,
  );
  const open = window.open.bind(window);
  window.open = (url?: string | URL, target?: string, features?: string) => {
    const raw = typeof url === "string" ? url : (url?.href ?? "");
    const slug = raw.startsWith(`${ORIGIN}/s/`) ? raw.slice(`${ORIGIN}/s/`.length).split(/[/?#]/)[0] : "";
    if (slug && openPreviewBySlug(slug)) return null;
    if (/^https?:/i.test(raw)) {
      toast({ id: "offline-link", message: "This demo is offline, so links to other sites are switched off." });
      return null;
    }
    return open(url, target, features);
  };
  previewButton(engine);
}
