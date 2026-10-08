import { h } from "../../ui/dom";

export interface Drawing {
  url: string;
  w: number;
  h: number;
}

interface Reply {
  type: "ready" | "done" | "error";
  id?: string;
  svg?: string;
  w?: number;
  h?: number;
  message?: string;
}

let frame: HTMLIFrameElement | null = null;
let booting: Promise<Window> | null = null;
let seq = 0;
const waiting = new Map<string, { resolve: (d: Drawing) => void; reject: (e: Error) => void }>();
const cache = new Map<string, Promise<Drawing>>();

function boot(): Promise<Window> {
  if (booting) return booting;
  booting = new Promise<Window>((resolve, reject) => {
    const el = h("iframe", { class: "diagram-renderer", attrs: { src: "/sandbox.html", sandbox: "allow-scripts", "aria-hidden": "true", tabindex: "-1", title: "Diagram renderer" } });
    frame = el;
    const timer = setTimeout(() => reject(new Error("the diagram renderer did not start")), 30000);
    addEventListener("message", (e: MessageEvent) => {
      if (!frame || e.source !== frame.contentWindow) return;
      const d = e.data as Reply;
      if (d?.type === "ready" && frame.contentWindow) {
        clearTimeout(timer);
        resolve(frame.contentWindow);
        return;
      }
      if (!d?.id) return;
      const w = waiting.get(d.id);
      waiting.delete(d.id);
      if (!w) return;
      if (d.type === "done" && typeof d.svg === "string") w.resolve({ url: toDataUrl(d.svg), w: Number(d.w) || 600, h: Number(d.h) || 400 });
      else w.reject(new Error(String(d.message ?? "the diagram could not be drawn")));
    });
    document.body.append(el);
  });
  booting.catch(() => {
    booting = null;
    frame?.remove();
    frame = null;
  });
  return booting;
}

function toDataUrl(svg: string): string {
  const bytes = new TextEncoder().encode(svg);
  let bin = "";
  for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return `data:image/svg+xml;base64,${btoa(bin)}`;
}

export function drawDiagram(code: string): Promise<Drawing> {
  const hit = cache.get(code);
  if (hit) return hit;
  const job = boot().then(
    (win) =>
      new Promise<Drawing>((resolve, reject) => {
        const id = String(++seq);
        waiting.set(id, { resolve, reject });
        win.postMessage({ type: "render", id, code }, "*");
      }),
  );
  cache.set(code, job);
  job.catch(() => cache.delete(code));
  if (cache.size > 40) cache.delete(cache.keys().next().value as string);
  return job;
}

export function sourceSum(code: string): string {
  let x = 0x811c9dc5;
  for (let i = 0; i < code.length; i++) {
    x ^= code.charCodeAt(i);
    x = Math.imul(x, 0x01000193);
  }
  return (x >>> 0).toString(36);
}
