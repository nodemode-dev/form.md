import hljs from "highlight.js/lib/core";
import bash from "highlight.js/lib/languages/bash";
import c from "highlight.js/lib/languages/c";
import cpp from "highlight.js/lib/languages/cpp";
import csharp from "highlight.js/lib/languages/csharp";
import css from "highlight.js/lib/languages/css";
import diff from "highlight.js/lib/languages/diff";
import dockerfile from "highlight.js/lib/languages/dockerfile";
import go from "highlight.js/lib/languages/go";
import graphql from "highlight.js/lib/languages/graphql";
import ini from "highlight.js/lib/languages/ini";
import java from "highlight.js/lib/languages/java";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import kotlin from "highlight.js/lib/languages/kotlin";
import lua from "highlight.js/lib/languages/lua";
import makefile from "highlight.js/lib/languages/makefile";
import markdown from "highlight.js/lib/languages/markdown";
import php from "highlight.js/lib/languages/php";
import python from "highlight.js/lib/languages/python";
import ruby from "highlight.js/lib/languages/ruby";
import rust from "highlight.js/lib/languages/rust";
import scss from "highlight.js/lib/languages/scss";
import shell from "highlight.js/lib/languages/shell";
import sql from "highlight.js/lib/languages/sql";
import swift from "highlight.js/lib/languages/swift";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";

const table = { bash, c, cpp, csharp, css, diff, dockerfile, go, graphql, ini, java, javascript, json, kotlin, lua, makefile, markdown, php, python, ruby, rust, scss, shell, sql, swift, typescript, xml, yaml };

for (const [name, lang] of Object.entries(table)) hljs.registerLanguage(name, lang);
hljs.registerAliases(["html", "svg"], { languageName: "xml" });

export interface Token {
  from: number;
  to: number;
  cls: string;
}

export function tokens(code: string, language: string): Token[] {
  if (!language || !hljs.getLanguage(language)) return [];
  const html = hljs.highlight(code, { language, ignoreIllegals: true }).value;
  const tpl = document.createElement("template");
  tpl.innerHTML = html;
  const out: Token[] = [];
  let offset = 0;
  const walk = (node: Node, classes: string[]) => {
    for (const child of Array.from(node.childNodes)) {
      if (child.nodeType === Node.TEXT_NODE) {
        const len = (child.textContent ?? "").length;
        if (classes.length && len) out.push({ from: offset, to: offset + len, cls: classes.join(" ") });
        offset += len;
      } else if (child instanceof HTMLElement) {
        const own = Array.from(child.classList).filter((c) => c.startsWith("hljs-"));
        walk(child, [...classes, ...own]);
      }
    }
  };
  walk(tpl.content, []);
  return out;
}
