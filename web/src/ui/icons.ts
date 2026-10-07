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
  "chevron-down": '<path d="m7 10 5 5 5-5"/>',
  "chevron-right": '<path d="m10 7 5 5-5 5"/>',
  more: '<path d="M6 12h.01M12 12h.01M18 12h.01"/>',
  check: '<path d="M5 12.5l4.5 4.5L19 7.5"/>',
  plus: '<path d="M12 5.5v13M5.5 12h13"/>',
  user: '<circle cx="12" cy="8.5" r="3.75"/><path d="M4.75 20a7.25 7.25 0 0 1 14.5 0"/>',
  users:
    '<circle cx="9" cy="8.75" r="3.25"/><path d="M3 19.5a6 6 0 0 1 12 0"/>' +
    '<path d="M15.5 5.75a3.25 3.25 0 0 1 0 6"/><path d="M17.5 14a6 6 0 0 1 3.5 5.5"/>',
  shield: '<path d="M12 3.5 5 6.25v5.5c0 4.25 2.9 7.6 7 8.75 4.1-1.15 7-4.5 7-8.75v-5.5Z"/><path d="m9.25 12 2 2 3.5-4"/>',
  key: '<circle cx="8" cy="15" r="3.75"/><path d="m10.75 12.25 8.25-8.25M15.5 7.5l2.5 2.5M17.75 5.25l1.75 1.75"/>',
  logout: '<path d="M14 4.5H7a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h7"/><path d="M10 12h10.5M17 8.5l3.5 3.5-3.5 3.5"/>',
  settings:
    '<circle cx="12" cy="12" r="3"/>' +
    '<path d="M12 2.75v2.5M12 18.75v2.5M2.75 12h2.5M18.75 12h2.5M5.46 5.46l1.77 1.77M16.77 16.77l1.77 1.77M5.46 18.54l1.77-1.77M16.77 7.23l1.77-1.77"/>',
  globe: '<circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.4 2.6 3.5 5.4 3.5 8.5s-1.1 5.9-3.5 8.5c-2.4-2.6-3.5-5.4-3.5-8.5s1.1-5.9 3.5-8.5Z"/>',
  mail: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="m4 7 8 6 8-6"/>',
  copy: '<rect x="8.5" y="8.5" width="11" height="11" rx="2"/><path d="M15.5 8.5V6.5a2 2 0 0 0-2-2h-7a2 2 0 0 0-2 2v7a2 2 0 0 0 2 2h2"/>',
  trash: '<path d="M4.5 7h15M9.5 7V5a1.5 1.5 0 0 1 1.5-1.5h2A1.5 1.5 0 0 1 14.5 5v2M6.5 7l.75 12a1.5 1.5 0 0 0 1.5 1.4h6.5a1.5 1.5 0 0 0 1.5-1.4L17.5 7"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1"/><path d="M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  lock: '<rect x="5" y="10.5" width="14" height="10" rx="2"/><path d="M8.5 10.5V7.5a3.5 3.5 0 0 1 7 0v3"/>',
  building: '<path d="M5 20.5V5.5a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v15M16 9.5h2a2 2 0 0 1 2 2v9M3.5 20.5h17"/><path d="M9 7.5h3M9 11h3M9 14.5h3"/>',
  bot: '<rect x="4.5" y="8" width="15" height="11" rx="3"/><path d="M12 8V4.5M9.5 13h.01M14.5 13h.01M2.5 13v2M21.5 13v2"/>',
  alert: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.75v5M12 16.25h.01"/>',
  download: '<path d="M12 4v11M7.5 10.5 12 15l4.5-4.5M4.5 19.5h15"/>',
  search: '<circle cx="11" cy="11" r="6.5"/><path d="m16 16 4 4"/>',
  external: '<path d="M13.5 4.5h6v6M19.5 4.5l-8 8M10 5.5H6.5a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V14"/>',
  library: '<path d="M5 4.5v15M9 4.5v15"/><path d="m13 5.25 3.4-.9 3.6 13.5-3.4.9Z"/><path d="M3.5 19.5h17"/>',
  star: '<path d="m12 3.5 2.6 5.3 5.9.9-4.25 4.1 1 5.8L12 16.9l-5.25 2.7 1-5.8L3.5 9.7l5.9-.9Z"/>',
  "star-fill": '<path d="m12 3.5 2.6 5.3 5.9.9-4.25 4.1 1 5.8L12 16.9l-5.25 2.7 1-5.8L3.5 9.7l5.9-.9Z" fill="currentColor"/>',
  tag: '<path d="M3.5 12.1V5A1.5 1.5 0 0 1 5 3.5h7.1a1.5 1.5 0 0 1 1.06.44l7.4 7.4a1.5 1.5 0 0 1 0 2.12l-7.1 7.1a1.5 1.5 0 0 1-2.12 0l-7.4-7.4a1.5 1.5 0 0 1-.44-1.06Z"/><path d="M8 8h.01"/>',
  restore: '<path d="M4.5 12a7.5 7.5 0 1 0 2.2-5.3"/><path d="M4.5 4.5v4.25h4.25"/>',
  "move-to": '<path d="M4.5 19.5v-6a4 4 0 0 1 4-4h11"/><path d="m15.5 5.5 4 4-4 4"/>',
  "move-up": '<path d="M19.5 19.5h-6a4 4 0 0 1-4-4v-11"/><path d="m5.5 8.5 4-4 4 4"/>',
  image: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M9 9.5h.01"/><path d="m3.5 17 5-5 4 4 2.5-2.5 5.5 5.5"/>',
  "align-left": '<path d="M4.5 6.5h15M4.5 11h10M4.5 15.5h15M4.5 20h8"/>',
  upload: '<path d="M12 15.5V4M7.5 8.5 12 4l4.5 4.5M4.5 19.5h15"/>',
  shape: '<circle cx="8" cy="8" r="3.5"/><rect x="13" y="13" width="7" height="7" rx="1.5"/><path d="m16.5 4 3.5 6h-7Z"/>',
  "new-tab": '<rect x="3.5" y="5" width="17" height="14" rx="2"/><path d="M3.5 9h17M12 12v4.5M9.75 14.25h4.5"/>',
  filter: '<path d="M4 5.5h16l-6.25 7.5v5.5l-3.5 1.5V13Z"/>',
  "list-tree": '<path d="M4.5 5.5h6M8.5 12h6M8.5 18.5h6M6 5.5v13h2.5M6 12h2.5"/><path d="M16.5 5.5h3M18.5 12h1M18.5 18.5h1"/>',
  cards: '<rect x="3.5" y="4.5" width="17" height="6" rx="1.5"/><rect x="3.5" y="13.5" width="17" height="6" rx="1.5"/>',
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
