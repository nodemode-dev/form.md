const PATHS = {
  logo:
    '<rect x="4" y="3.5" width="4.75" height="17" rx="1.6" fill="currentColor" stroke="none"/>' +
    '<rect x="10.25" y="3.5" width="9.75" height="4.75" rx="1.6" fill="currentColor" stroke="none"/>' +
    '<rect x="10.25" y="9.75" width="6.25" height="4.75" rx="1.6" fill="currentColor" stroke="none" opacity="0.62"/>',
  "panel-left": '<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"/><path d="M9.5 4.5v15"/>',
  sun:
    '<circle cx="12" cy="12" r="3.75"/>' +
    '<path d="M12 3v1.75M12 19.25V21M3 12h1.75M19.25 12H21M5.64 5.64l1.24 1.24M17.12 17.12l1.24 1.24M5.64 18.36l1.24-1.24M17.12 6.88l1.24-1.24"/>',
  moon: '<path d="M19.5 14.25A7.5 7.5 0 0 1 9.75 4.5a7.5 7.5 0 1 0 9.75 9.75Z"/>',
  monitor: '<rect x="3.5" y="4.5" width="17" height="11.5" rx="2"/><path d="M8.5 20h7M12 16v4"/>',
  keyboard:
    '<rect x="2.5" y="6" width="19" height="12" rx="2.5"/>' +
    '<path d="M6.5 10h.01M10 10h.01M14 10h.01M17.5 10h.01M6.5 14h.01M17.5 14h.01M9.5 14h5"/>',
  close: '<path d="M6.5 6.5l11 11M17.5 6.5l-11 11"/>',
  info: '<circle cx="12" cy="12" r="8.5"/><path d="M12 11v5.5M12 7.75h.01"/>',
  success: '<circle cx="12" cy="12" r="8.5"/><path d="m8.5 12.25 2.5 2.5 4.5-5"/>',
  warning:
    '<path d="M10.27 4.6 3.2 17a2 2 0 0 0 1.73 3h14.14a2 2 0 0 0 1.73-3L13.73 4.6a2 2 0 0 0-3.46 0Z"/>' +
    '<path d="M12 9.5v4.25M12 16.75h.01"/>',
  danger: '<circle cx="12" cy="12" r="8.5"/><path d="m9.25 9.25 5.5 5.5M14.75 9.25l-5.5 5.5"/>',
  refresh: '<path d="M19.5 12a7.5 7.5 0 1 1-2.2-5.3"/><path d="M19.5 4.5v4.25h-4.25"/>',
  page: '<path d="M13.5 3.5h-6a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2V8.5Z"/><path d="M13.5 3.5v5h5"/>',
} as const;

export type IconName = keyof typeof PATHS;

export function isIconName(name: string): name is IconName {
  return Object.hasOwn(PATHS, name);
}

export function iconSvg(name: IconName, extraClass = ""): string {
  const cls = extraClass ? `icon ${extraClass}` : "icon";
  return (
    `<svg class="${cls}" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" ` +
    `stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">` +
    `${PATHS[name]}</svg>`
  );
}

const cache = new Map<string, SVGSVGElement>();

export function iconNode(name: IconName, extraClass = ""): SVGSVGElement {
  const key = `${name}|${extraClass}`;
  let proto = cache.get(key);
  if (!proto) {
    const tpl = document.createElement("template");
    tpl.innerHTML = iconSvg(name, extraClass);
    proto = tpl.content.firstElementChild as SVGSVGElement;
    cache.set(key, proto);
  }
  return proto.cloneNode(true) as SVGSVGElement;
}
