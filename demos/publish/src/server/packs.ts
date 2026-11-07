let loading: Promise<boolean> | null = null;

export function loadPictures(): Promise<boolean> {
  if (window.__formmdPacks?.["img/cover-telescope"]) return Promise.resolve(true);
  loading ??= new Promise<boolean>((resolve) => {
    const s = document.createElement("script");
    s.src = "assets/media/pictures.js";
    s.async = true;
    s.onload = () => {
      s.remove();
      resolve(true);
    };
    s.onerror = () => {
      s.remove();
      loading = null;
      resolve(false);
    };
    document.head.append(s);
  });
  return loading;
}

export function pictureUri(name: string): string {
  return window.__formmdPacks?.[`img/${name}`] ?? "";
}

export function pictureBytes(name: string): Uint8Array {
  const uri = pictureUri(name);
  const raw = atob(uri.slice(uri.indexOf(",") + 1));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}
