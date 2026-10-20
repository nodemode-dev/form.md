type Mermaid = (typeof import("mermaid"))["default"];

interface RenderRequest {
  type: "render";
  id: string;
  code: string;
  theme?: string;
}

const stage = document.getElementById("stage") as HTMLElement;
let engine: Promise<Mermaid> | null = null;
let queue: Promise<unknown> = Promise.resolve();
let seq = 0;

const PALETTE = {
  dark: ["#a78bfa", "#60a5fa", "#2dd4bf", "#4ade80", "#fbbf24", "#fb923c", "#fb7185", "#f472b6", "#22d3ee", "#a3e635", "#9a8cff", "#94a3b8"],
  neutral: ["#7c3aed", "#2563eb", "#0f766e", "#15803d", "#b45309", "#c2410c", "#be123c", "#be185d", "#0e7490", "#4d7c0f", "#5a46e0", "#64748b"],
};

function settings(theme: "dark" | "neutral") {
  const pal = PALETTE[theme];
  const pie = Object.fromEntries(pal.map((c, i) => [`pie${i + 1}`, c]));
  return {
    startOnLoad: false,
    securityLevel: "strict",
    theme,
    themeVariables: { ...pie, pieStrokeColor: theme === "dark" ? "#111113" : "#ffffff", xyChart: { plotColorPalette: pal.join(", ") } },
    deterministicIds: true,
    fontFamily: "Arial, Helvetica, sans-serif",
    htmlLabels: false,
    flowchart: { htmlLabels: false },
  } as const;
}

function load(): Promise<Mermaid> {
  engine ??= import("mermaid").then((m) => m.default);
  return engine;
}

function fit(svg: string): { svg: string; w: number; h: number } {
  const doc = new DOMParser().parseFromString(svg, "image/svg+xml");
  const root = doc.documentElement;
  const box = (root.getAttribute("viewBox") ?? "").split(/[\s,]+/).map(Number);
  const w = Math.ceil(box[2] ?? 0) || 600;
  const h = Math.ceil(box[3] ?? 0) || 400;
  root.setAttribute("width", String(w));
  root.setAttribute("height", String(h));
  root.removeAttribute("style");
  root.setAttribute("xmlns", "http://www.w3.org/2000/svg");
  return { svg: new XMLSerializer().serializeToString(root), w, h };
}

async function render(req: RenderRequest): Promise<void> {
  const id = `diagram${++seq}`;
  try {
    const mermaid = await load();
    mermaid.initialize(settings(req.theme === "dark" ? "dark" : "neutral"));
    const { svg } = await mermaid.render(id, req.code, stage);
    parent.postMessage({ type: "done", id: req.id, ...fit(svg) }, "*");
  } catch (e) {
    const message = e instanceof Error ? e.message : String(e);
    parent.postMessage({ type: "error", id: req.id, message: message.slice(0, 600) }, "*");
  } finally {
    document.getElementById(id)?.remove();
    document.getElementById(`d${id}`)?.remove();
    stage.replaceChildren();
  }
}

addEventListener("message", (e: MessageEvent) => {
  if (e.source !== parent) return;
  const d = e.data as RenderRequest | null;
  if (!d || d.type !== "render" || typeof d.id !== "string" || typeof d.code !== "string" || d.code.length > 50_000) return;
  queue = queue.then(() => render(d));
});

parent.postMessage({ type: "ready" }, "*");
