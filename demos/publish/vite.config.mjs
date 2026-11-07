import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { iconSvg, isIconName } from "../../web/src/ui/icons.ts";

const here = import.meta.dirname;
const web = resolve(here, "../../web");
const src = resolve(here, "src");
const webEntry = resolve(web, "src/main.ts");

const bareFromWeb = {
  name: "publish-demo-bare-from-web",
  enforce: "pre",
  async resolveId(id, importer, options) {
    if (!importer || !importer.startsWith(src)) return null;
    if (id.startsWith(".") || id.startsWith("/") || id.startsWith("\0") || id.startsWith("virtual:")) return null;
    return this.resolve(id, webEntry, { ...options, skipSelf: true });
  },
};

const offlineUrls = {
  name: "publish-demo-offline-urls",
  enforce: "pre",
  transform(code, id) {
    if (!id.includes("/web/src/") || !/\.ts$/.test(id)) return null;
    let out = code;
    out = out.replace(/`\/files\/\$\{([^}]*)\}`/g, "globalThis.__fileHref($1)");
    out = out.replace(/\blocation\.origin\b/g, "(globalThis.__demoOrigin ?? location.origin)");
    if (id.endsWith("/ui/publish-dialog.ts")) out = out.replace(/\blocation\.host\b/g, "(globalThis.__demoHost ?? location.host)");
    return out === code ? null : { code: out, map: null };
  },
};

const shellFromWeb = {
  name: "publish-demo-shell-from-web",
  transformIndexHtml: {
    order: "pre",
    handler(html) {
      const page = readFileSync(resolve(web, "index.html"), "utf8");
      const body = /<body>([\s\S]*)<\/body>/.exec(page)?.[1] ?? "";
      const shell = body
        .replace(/<script[\s\S]*?<\/script>/g, "")
        .replace(/<!--slot:[a-z-]+-->/g, "")
        .replace(/<!--html:[a-z-]+-->/g, "")
        .replace('class="view view--page"', 'class="view view--page has-cover"');
      const links = `<nav class="demo-links" aria-label="Demos"><span class="demo-links__label">Demos</span><a class="demo-links__item" href="../../demo/index.html" title="Editor and databases">1 Editor</a><a class="demo-links__item" href="../views/index.html" title="Database views">2 Views</a><a class="demo-links__item" href="./index.html" title="Write and publish" aria-current="page">3 Publish</a></nav>`;
      return html.replace("<!--shell-->", shell.replace(/<\/footer>(\s*<\/div>\s*<div class="sidebar__resize")/, `</footer>${links}$1`));
    },
  },
};

const inlineIcons = {
  name: "publish-demo-inline-icons",
  transformIndexHtml(html) {
    return html.replace(/<i data-icon="([a-z0-9-]+)"(?: class="([^"]*)")?><\/i>/g, (tag, name, cls) => {
      if (!isIconName(name)) throw new Error(`shell: unknown icon ${tag}`);
      return iconSvg(name, cls ?? "");
    });
  },
};

const classicScript = {
  name: "publish-demo-classic-script",
  transformIndexHtml: {
    order: "post",
    handler(html) {
      return html
        .replace(/<script type="module" crossorigin src=/g, "<script defer src=")
        .replace(/<link rel="stylesheet" crossorigin /g, '<link rel="stylesheet" ');
    },
  },
};

export default {
  root: src,
  base: "./",
  publicDir: false,
  plugins: [bareFromWeb, offlineUrls, shellFromWeb, inlineIcons, classicScript],
  resolve: {
    alias: {
      "virtual:i18n": resolve(src, "i18n-empty.ts"),
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
    assetsInlineLimit: 200000,
    chunkSizeWarningLimit: 6000,
    rolldownOptions: {
      output: {
        format: "iife",
        inlineDynamicImports: true,
        entryFileNames: "assets/publish.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
  },
};
