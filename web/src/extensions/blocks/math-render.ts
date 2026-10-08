import katex from "katex";
import "katex/contrib/mhchem";

export function renderTex(tex: string, display: boolean): { mathml: string } | { error: string } {
  try {
    const out = katex.renderToString(tex, { output: "mathml", displayMode: display, throwOnError: true, strict: "ignore", trust: false, maxSize: 50, maxExpand: 2000 });
    const start = out.indexOf("<math");
    const end = out.lastIndexOf("</math>");
    if (start < 0 || end < 0) return { error: "no output" };
    return { mathml: out.slice(start, end + "</math>".length) };
  } catch (e) {
    return { error: e instanceof Error ? e.message.replace(/^KaTeX parse error:\s*/, "") : String(e) };
  }
}
