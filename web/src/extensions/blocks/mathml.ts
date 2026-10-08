const NS = "http://www.w3.org/1998/Math/MathML";

const TAGS = new Set([
  "math", "semantics", "annotation", "mrow", "mi", "mn", "mo", "ms", "mtext", "mspace", "msub", "msup", "msubsup", "mfrac",
  "msqrt", "mroot", "mstyle", "mtable", "mtr", "mtd", "mlabeledtr", "mover", "munder", "munderover", "mpadded", "mphantom",
  "menclose", "merror", "mmultiscripts", "mprescripts", "none",
]);

const ATTRS = new Set([
  "mathvariant", "displaystyle", "scriptlevel", "stretchy", "fence", "separator", "lspace", "rspace", "minsize", "maxsize",
  "movablelimits", "accent", "accentunder", "columnalign", "rowalign", "columnspacing", "rowspacing", "columnlines", "rowlines",
  "frame", "framespacing", "linethickness", "width", "height", "depth", "voffset", "notation", "encoding", "display", "mathcolor",
  "mathbackground", "symmetric", "largeop", "form", "columnspan", "rowspan", "align", "bevelled",
]);

function copy(src: Element, root: boolean): Element | null {
  const name = src.localName;
  if (!TAGS.has(name) || (root && name !== "math") || (!root && name === "math")) return null;
  if (name === "annotation" && src.getAttribute("encoding") !== "application/x-tex") return null;
  const el = document.createElementNS(NS, name);
  for (const a of Array.from(src.attributes)) {
    if (a.namespaceURI || !ATTRS.has(a.localName) || (a.localName === "display" && name !== "math")) continue;
    el.setAttribute(a.localName, a.value);
  }
  for (const c of Array.from(src.childNodes)) {
    if (c.nodeType === Node.TEXT_NODE) el.appendChild(document.createTextNode(c.textContent ?? ""));
    else if (c.nodeType === Node.ELEMENT_NODE) {
      const k = copy(c as Element, false);
      if (k) el.appendChild(k);
    }
  }
  return el;
}

export function mathElement(markup: string): Element | null {
  if (!markup.trimStart().startsWith("<math") || markup.length > 200000) return null;
  const doc = new DOMParser().parseFromString(markup, "application/xml");
  if (doc.getElementsByTagName("parsererror").length) return null;
  return copy(doc.documentElement, true);
}
