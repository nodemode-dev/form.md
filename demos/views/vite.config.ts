import type { Plugin } from "vite";
import { resolve } from "node:path";
import { iconSvg, isIconName } from "../../web/src/ui/icons.ts";

const here = import.meta.dirname;
const src = resolve(here, "src");
const web = resolve(here, "../../web");
const shim = resolve(src, "pageicon-shim.ts");
const realIcon = resolve(web, "src/ui/pageicon.ts");

const swapFileUrl: Plugin = {
  name: "views-demo-file-url",
  enforce: "pre",
  async resolveId(id, importer, options) {
    if (!importer || importer === shim || !/pageicon(\.ts)?$/.test(id)) return null;
    const r = await this.resolve(id, importer, { ...options, skipSelf: true });
    return r && r.id === realIcon ? shim : null;
  },
};

const inlineIcons: Plugin = {
  name: "views-demo-inline-icons",
  transformIndexHtml(html) {
    return html.replace(/<i data-icon="([a-z0-9-]+)"(?: class="([^"]*)")?><\/i>/g, (tag, name: string, cls?: string) => {
      if (!isIconName(name)) throw new Error(`views index.html: unknown icon ${tag}`);
      return iconSvg(name, cls ?? "");
    });
  },
};

const classicScript: Plugin = {
  name: "views-demo-classic-script",
  transformIndexHtml: {
    order: "post",
    handler(html) {
      return html.replace(/<script type="module" crossorigin src=/g, "<script defer src=").replace(/<link rel="stylesheet" crossorigin /g, '<link rel="stylesheet" ');
    },
  },
};

export default {
  root: src,
  base: "./",
  publicDir: false,
  plugins: [swapFileUrl, inlineIcons, classicScript],
  resolve: {
    alias: {
      "virtual:i18n": resolve(here, "../../demo/src/i18n-empty.ts"),
    },
  },
  server: { port: 5770, host: false, strictPort: true },
  build: {
    outDir: here,
    emptyOutDir: false,
    sourcemap: false,
    target: "es2022",
    cssCodeSplit: false,
    modulePreload: false,
    assetsInlineLimit: 120000,
    rolldownOptions: {
      output: {
        format: "iife",
        inlineDynamicImports: true,
        entryFileNames: "assets/views.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
  },
};
