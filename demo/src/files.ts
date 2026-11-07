export type Kind = "markdown" | "table" | "unknown";

export function kindOf(file: File): Kind {
  const n = file.name.toLowerCase();
  if (/\.(md|markdown|mdown|txt)$/.test(n)) return "markdown";
  if (/\.(csv|tsv)$/.test(n) || file.type === "text/csv" || file.type === "text/tab-separated-values") return "table";
  if (file.type === "text/markdown" || file.type === "text/plain") return "markdown";
  return "unknown";
}

export function readText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result ?? ""));
    r.onerror = () => reject(r.error ?? new Error("read failed"));
    r.readAsText(file);
  });
}

export function download(name: string, text: string, mime: string): void {
  const url = URL.createObjectURL(new Blob([text], { type: `${mime};charset=utf-8` }));
  const a = document.createElement("a");
  a.href = url;
  a.download = name.replace(/[\\/:*?"<>|]+/g, "-");
  document.body.append(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 2000);
}
