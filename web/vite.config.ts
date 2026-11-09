import { defineConfig, type Plugin } from "vite";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { GLYPH_NAMES, glyphSvg } from "./src/ui/glyphs.ts";
import { ICON_NAMES, iconSvg, isIconName } from "./src/ui/icons.ts";

const FRONTEND_PORT = 5750;
const BACKEND = "http://127.0.0.1:8750";

const toBackend = {
  target: BACKEND,
  changeOrigin: true,
  configure: (proxy: { on(event: "proxyReq", fn: (req: { getHeader(name: string): unknown; setHeader(name: string, value: string): void }) => void): void }) => {
    proxy.on("proxyReq", (req) => {
      if (req.getHeader("origin")) req.setHeader("origin", BACKEND);
    });
  },
};

const keepDistPlaceholder: Plugin = {
  name: "form-md-keep-dist-placeholder",
  apply: "build",
  closeBundle() {
    writeFileSync(resolve(import.meta.dirname, "dist/.gitkeep"), "");
  },
};

const inlineIcons: Plugin = {
  name: "form-md-inline-icons",
  transformIndexHtml(html) {
    return html.replace(/<i data-icon="([a-z-]+)"(?: class="([^"]*)")?><\/i>/g, (tag, name: string, cls?: string) => {
      if (!isIconName(name)) throw new Error(`index.html: unknown icon ${tag}`);
      return iconSvg(name, cls ?? "");
    });
  },
};

const iconTable: Plugin = {
  name: "form-md-icon-table",
  apply: "build",
  generateBundle() {
    const glyphs = Object.fromEntries(GLYPH_NAMES.map((n) => [n, glyphSvg(n)]));
    const icons = Object.fromEntries(ICON_NAMES.map((n) => [n, iconSvg(n)]));
    this.emitFile({ type: "asset", fileName: ".vite/icons.json", source: JSON.stringify({ glyphs, icons }) });
  },
};

const REPO = resolve(import.meta.dirname, "..");
const DEMO_DIRS = ["demo", "demos/views", "demos/publish"];
const demoFiles = (dir: string): string[] => {
  const base = join(REPO, dir);
  const walk = (rel: string): string[] =>
    readdirSync(join(base, rel)).flatMap((n) => {
      const r = rel ? `${rel}/${n}` : n;
      return statSync(join(base, r)).isDirectory() ? walk(r) : [r];
    });
  return ["index.html", ...walk("assets")];
};
const DEMO_TYPES: Record<string, string> = { ".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".css": "text/css; charset=utf-8", ".webp": "image/webp" };
const demoBundles: Plugin = {
  name: "form-md-demo-bundles",
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      const url = (req.url ?? "").split("?")[0] ?? "";
      const dir = DEMO_DIRS.find((d) => url.startsWith(`/${d}/`));
      if (!dir) return next();
      const rel = decodeURIComponent(url.slice(dir.length + 2));
      if (!demoFiles(dir).includes(rel)) return next();
      res.setHeader("Content-Type", DEMO_TYPES[rel.slice(rel.lastIndexOf("."))] ?? "application/octet-stream");
      res.end(readFileSync(join(REPO, dir, rel)));
    });
  },
  generateBundle() {
    for (const dir of DEMO_DIRS) {
      for (const rel of demoFiles(dir)) this.emitFile({ type: "asset", fileName: `${dir}/${rel}`, source: readFileSync(join(REPO, dir, rel)) });
    }
  },
};

const I18N_DIR = resolve(import.meta.dirname, "src/i18n");
const I18N_ID = "virtual:i18n";
const i18nCatalogs: Plugin = {
  name: "form-md-i18n-catalogs",
  resolveId(id) {
    return id === I18N_ID || id.startsWith(`${I18N_ID}/`) ? `\0${id}` : null;
  },
  load(id) {
    if (!id.startsWith(`\0${I18N_ID}`)) return null;
    const langs = readdirSync(I18N_DIR).filter((d) => statSync(join(I18N_DIR, d)).isDirectory()).sort();
    if (id === `\0${I18N_ID}`) {
      return `export const loaders = {${langs.map((l) => `${JSON.stringify(l)}: () => import(${JSON.stringify(`${I18N_ID}/${l}`)})`).join(",")}};`;
    }
    const lang = id.slice(`\0${I18N_ID}/`.length);
    if (!langs.includes(lang)) throw new Error(`i18n: no catalog folder for ${lang}`);
    const dir = join(I18N_DIR, lang);
    const files = readdirSync(dir).filter((f) => f.endsWith(".json")).sort();
    for (const f of files) this.addWatchFile(join(dir, f));
    const merged = Object.assign({}, ...files.map((f) => JSON.parse(readFileSync(join(dir, f), "utf8")) as Record<string, string>));
    return `export default ${JSON.stringify(merged)};`;
  },
};

export default defineConfig({
  root: ".",
  publicDir: "public",
  plugins: [i18nCatalogs, inlineIcons, iconTable, demoBundles, keepDistPlaceholder],

  server: {
    port: FRONTEND_PORT,
    host: false,
    strictPort: true,
    proxy: {
      "/api/terminal": {
        target: BACKEND,
        ws: true,
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on("proxyReqWs", (req) => req.setHeader("origin", BACKEND));
          proxy.on("proxyReq", (req) => {
            if (req.getHeader("origin")) req.setHeader("origin", BACKEND);
          });
        },
      },
      "/api": toBackend,
      "/healthz": toBackend,
      "/notices": toBackend,
      "/events": toBackend,
      "/mcp": toBackend,
      "/files": toBackend,
      "/oauth": toBackend,
      "/.well-known": toBackend,
      "/sync": {
        target: BACKEND,
        ws: true,
        configure: (proxy) => {
          proxy.on("proxyReqWs", (req) => req.setHeader("origin", BACKEND));
        },
      },
    },
  },

  preview: {
    port: FRONTEND_PORT + 1,
    host: false,
    strictPort: true,
  },

  build: {
    outDir: "dist",
    emptyOutDir: true,
    sourcemap: false,
    target: "es2022",
    assetsInlineLimit: 0,
    manifest: true,
    rolldownOptions: {
      input: {
        main: resolve(import.meta.dirname, "index.html"),
        sandbox: resolve(import.meta.dirname, "sandbox.html"),
        site: resolve(import.meta.dirname, "src/site/main.ts"),
      },
    },
  },
});
