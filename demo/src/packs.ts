declare global {
  interface Window {
    __formmdPacks?: Record<string, string>;
  }
}

const loading = new Map<string, Promise<string | null>>();

export function loadPack(name: string, file: string): Promise<string | null> {
  const known = window.__formmdPacks?.[name];
  if (known !== undefined) return Promise.resolve(known);
  const pending = loading.get(file);
  if (pending) return pending;
  const p = new Promise<string | null>((resolve) => {
    const s = document.createElement("script");
    s.src = file;
    s.async = true;
    s.onload = () => {
      s.remove();
      resolve(window.__formmdPacks?.[name] ?? null);
    };
    s.onerror = () => {
      s.remove();
      loading.delete(file);
      resolve(null);
    };
    document.head.append(s);
  });
  loading.set(file, p);
  return p;
}
