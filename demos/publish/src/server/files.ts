import { pid } from "./ids";

export interface FileRec {
  id: string;
  name: string;
  ext: string;
  mime: string;
  path: string;
  w: number;
  h: number;
  size: number;
}

const list: [string, number, number, number][] = [
  ["cover-telescope", 1600, 480, 161468],
  ["cover-milkyway", 1600, 480, 177874],
  ["cover-earthset", 1600, 480, 26106],
  ["cover-tripod", 1600, 480, 190324],
  ["cover-observatory", 1600, 480, 76850],
  ["galileo-moon-sketch-1", 500, 485, 69088],
  ["galileo-moon-sketch-2", 500, 480, 60626],
  ["galileo-jupiter-moons", 500, 420, 19472],
  ["moon-full-mono", 1100, 1033, 173892],
  ["moon-full-warm", 855, 1100, 68356],
  ["apollo-8-earthrise", 1100, 1100, 27264],
  ["apollo-11-bootprint", 1100, 1100, 248508],
  ["apollo-11-saturn-v", 887, 1000, 78366],
  ["artemis-ii-earthset", 1600, 1067, 59496],
  ["telescope-milky-way", 1600, 1067, 365090],
];

export const FILES = new Map<string, FileRec>();
const byName = new Map<string, FileRec>();

for (const [name, w, h, size] of list) {
  const rec: FileRec = { id: pid(`file:${name}`), name, ext: ".webp", mime: "image/webp", path: `assets/media/${name}.webp`, w, h, size };
  FILES.set(rec.id, rec);
  byName.set(name, rec);
}

export function fileId(name: string): string {
  const rec = byName.get(name);
  if (!rec) throw new Error(`no demo file ${name}`);
  return rec.id;
}

export function fileHref(id: string): string {
  return FILES.get(id)?.path ?? "";
}

declare global {
  interface Window {
    __fileHref?: (id: string) => string;
    __formmdPacks?: Record<string, string>;
  }
}

window.__fileHref = (id) => fileHref(id);
