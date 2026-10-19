import { h } from "./dom";
import { iconNode } from "./icons";

export interface Disclosure {
  el: HTMLDetailsElement;
  summary: HTMLElement;
  body: HTMLElement;
}

export function disclosure(label: string, ...content: HTMLElement[]): Disclosure {
  const body = h("div", { class: "disclosure__body" }, ...content);
  const summary = h("summary", { class: "disclosure__summary" }, iconNode("chevron-right"), h("span", { text: label }));
  const el = h("details", { class: "disclosure" }, summary, body);
  return { el, summary, body };
}
