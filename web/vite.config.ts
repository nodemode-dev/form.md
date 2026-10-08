import { defineConfig, type Plugin } from "vite";
import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { iconSvg, isIconName } from "./src/ui/icons.ts";

const FRONTEND_PORT = 5750;
const BACKEND = "http://127.0.0.1:8750";

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

export default defineConfig({
  root: ".",
  publicDir: "public",
  plugins: [inlineIcons, keepDistPlaceholder],

  server: {
    port: FRONTEND_PORT,
    host: false,
    strictPort: true,
    proxy: {
      "/api": BACKEND,
      "/healthz": BACKEND,
      "/notices": BACKEND,
      "/events": BACKEND,
      "/mcp": BACKEND,
      "/files": BACKEND,
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
  },
});
