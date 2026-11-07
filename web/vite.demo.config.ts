import { defineConfig, type Plugin } from "vite";
import { resolve } from "node:path";
import { iconSvg, isIconName } from "./src/ui/icons.ts";

const web = import.meta.dirname;
const here = resolve(web, "../demo");
const src = resolve(here, "src");
const webEntry = resolve(web, "src/main.ts");

const bareFromWeb: Plugin = {
  name: "form-md-demo-bare-from-web",
  enforce: "pre",
  async resolveId(id, importer, options) {
    if (!importer || !importer.startsWith(src)) return null;
    if (id.startsWith(".") || id.startsWith("/") || id.startsWith("\0") || id.startsWith("virtual:")) return null;
    return this.resolve(id, webEntry, { ...options, skipSelf: true });
  },
};

const inlineIcons: Plugin = {
  name: "form-md-demo-inline-icons",
  transformIndexHtml(html) {
    return html.replace(/<i data-icon="([a-z0-9-]+)"(?: class="([^"]*)")?><\/i>/g, (tag, name: string, cls?: string) => {
      if (!isIconName(name)) throw new Error(`demo index.html: unknown icon ${tag}`);
      return iconSvg(name, cls ?? "");
    });
  },
};

const classicScript: Plugin = {
  name: "form-md-demo-classic-script",
  transformIndexHtml: {
    order: "post",
    handler(html) {
      return html
        .replace(/<script type="module" crossorigin src=/g, "<script defer src=")
        .replace(/<link rel="stylesheet" crossorigin /g, '<link rel="stylesheet" ');
    },
  },
};

export default defineConfig({
  root: src,
  base: "./",
  publicDir: false,
  plugins: [bareFromWeb, inlineIcons, classicScript],
  resolve: {
    alias: {
      "virtual:i18n": resolve(src, "i18n-empty.ts"),
    },
  },
  server: {
    port: 5760,
    host: false,
    strictPort: true,
  },
  preview: {
    port: 5761,
    host: false,
    strictPort: true,
  },
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
        entryFileNames: "assets/demo.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
  },
});
