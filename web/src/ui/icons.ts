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
  text: '<path d="M5.5 6.5V5h13v1.5M12 5v14M9.5 19h5"/>',
  h1: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15.5 9.5 18 8v10"/>',
  h2: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15 9.75a2.25 2.25 0 1 1 4.1 1.3L15 18h4.5"/>',
  h3: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15 8.5h4l-2.25 3a2.5 2.5 0 1 1-2.1 3.9"/>',
  "list-bullet":
    '<path d="M9.5 6.5h10M9.5 12h10M9.5 17.5h10"/>' +
    '<circle cx="5" cy="6.5" r="1" fill="currentColor" stroke="none"/><circle cx="5" cy="12" r="1" fill="currentColor" stroke="none"/><circle cx="5" cy="17.5" r="1" fill="currentColor" stroke="none"/>',
  "list-number": '<path d="M10 6.5h9.5M10 12h9.5M10 17.5h9.5"/><path d="M4.5 5 6 4.25V9M4.25 14.25a1.5 1.5 0 1 1 2.4 1.2L4.25 19.5h2.75"/>',
  checkbox: '<rect x="4.5" y="4.5" width="15" height="15" rx="3"/><path d="m8.5 12.25 2.5 2.5 4.5-5"/>',
  quote: '<path d="M6 7.5h4v4.25c0 2.5-1.5 4-4 4.75M14 7.5h4v4.25c0 2.5-1.5 4-4 4.75"/>',
  code: '<path d="m8.5 7.5-4.5 4.5 4.5 4.5M15.5 7.5l4.5 4.5-4.5 4.5M13.5 5.5l-3 13"/>',
  divider: '<path d="M4 12h16"/><path d="M8 7.5h8M8 16.5h8" opacity=".45"/>',
  table: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M3.5 9.5h17M3.5 14.5h17M9.5 9.5v10M15 9.5v10"/>',
  video: '<rect x="3.5" y="6" width="12" height="12" rx="2"/><path d="m15.5 10.5 5-3v9l-5-3"/>',
  audio: '<path d="M9.5 17.5V6l10-2v11.5"/><circle cx="7" cy="17.5" r="2.5"/><circle cx="17" cy="15.5" r="2.5"/>',
  attachment: '<path d="m19 11.5-7.4 7.4a4.5 4.5 0 0 1-6.4-6.4l7.6-7.6a3 3 0 0 1 4.3 4.3l-7.5 7.5a1.5 1.5 0 0 1-2.1-2.1l6.9-6.9"/>',
  callout: '<rect x="3.5" y="5" width="17" height="14" rx="2.5"/><path d="M8 9.5h.01M11 9.5h5.5M8 14h8.5"/>',
  toc: '<path d="M4.5 6.5h15M8 11h11.5M8 15.5h11.5M11.5 20h8"/>',
  columns: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M12 4.5v15"/>',
  diagram: '<rect x="3.5" y="4" width="7" height="5" rx="1.25"/><rect x="13.5" y="15" width="7" height="5" rx="1.25"/><path d="M7 9v4.5a2 2 0 0 0 2 2h4.5"/>',
  grip:
    '<circle cx="9" cy="6.5" r="1.1" fill="currentColor" stroke="none"/><circle cx="15" cy="6.5" r="1.1" fill="currentColor" stroke="none"/>' +
    '<circle cx="9" cy="12" r="1.1" fill="currentColor" stroke="none"/><circle cx="15" cy="12" r="1.1" fill="currentColor" stroke="none"/>' +
    '<circle cx="9" cy="17.5" r="1.1" fill="currentColor" stroke="none"/><circle cx="15" cy="17.5" r="1.1" fill="currentColor" stroke="none"/>',
  bold: '<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z"/>',
  italic: '<path d="M10 5h7M7 19h7M14 5l-4 14"/>',
  underline: '<path d="M7 4.5v6.5a5 5 0 0 0 10 0V4.5M5.5 20h13"/>',
  strike: '<path d="M4.5 12h15M16 7.5c-.6-1.5-2.1-2.5-4-2.5-2.2 0-4 1.2-4 3 0 1.5 1.2 2.4 3 3M8 16.5c.6 1.6 2.2 2.5 4 2.5 2.4 0 4.25-1.3 4.25-3.25 0-.7-.2-1.3-.6-1.75"/>',
  "text-color": '<path d="m6.5 16 5.5-12 5.5 12M8.5 11.5h7M4.5 20h15"/>',
  "arrow-up": '<path d="M12 19V5M6.5 10.5 12 5l5.5 5.5"/>',
  "arrow-down": '<path d="M12 5v14M6.5 13.5 12 19l5.5-5.5"/>',
  turn: '<path d="M7 4.5 4 7.5l3 3M4 7.5h12.5a3 3 0 0 1 3 3M17 19.5l3-3-3-3M20 16.5H7.5a3 3 0 0 1-3-3"/>',
  at: '<circle cx="12" cy="12" r="3.5"/><path d="M15.5 12v1.5a2.5 2.5 0 0 0 5 0V12a8.5 8.5 0 1 0-3.4 6.8"/>',
  calendar: '<rect x="4" y="5.5" width="16" height="14.5" rx="2"/><path d="M4 10h16M8.5 3.5v4M15.5 3.5v4"/>',
  play: '<path d="M8 5.5v13l10-6.5Z"/>',
  database: '<ellipse cx="12" cy="6" rx="7" ry="2.5"/><path d="M5 6v12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5V6M5 12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5"/>',
  board: '<rect x="3.5" y="4.5" width="5" height="15" rx="1.5"/><rect x="9.5" y="4.5" width="5" height="10" rx="1.5"/><rect x="15.5" y="4.5" width="5" height="12.5" rx="1.5"/>',
  rows: '<rect x="3.5" y="4.5" width="17" height="4" rx="1.2"/><rect x="3.5" y="10" width="17" height="4" rx="1.2"/><rect x="3.5" y="15.5" width="17" height="4" rx="1.2"/>',
  gallery: '<rect x="3.5" y="3.5" width="7.5" height="7.5" rx="1.5"/><rect x="13" y="3.5" width="7.5" height="7.5" rx="1.5"/><rect x="3.5" y="13" width="7.5" height="7.5" rx="1.5"/><rect x="13" y="13" width="7.5" height="7.5" rx="1.5"/>',
  timeline: '<rect x="3.5" y="5" width="9" height="3" rx="1.5"/><rect x="7.5" y="10.5" width="10" height="3" rx="1.5"/><rect x="11" y="16" width="9.5" height="3" rx="1.5"/>',
  form: '<rect x="4.5" y="3.5" width="15" height="17" rx="2"/><path d="M8 8h8M8 12h8M8 16h4.5"/>',
  hash: '<path d="M9.5 4 8 20M16 4l-1.5 16M4.5 9h15M4 15h15"/>',
  select: '<circle cx="12" cy="12" r="8.5"/><path d="m8.75 10.75 3.25 3.25 3.25-3.25"/>',
  multi: '<rect x="3.5" y="5.5" width="8" height="5" rx="2.5"/><rect x="12.5" y="5.5" width="8" height="5" rx="2.5"/><rect x="3.5" y="13.5" width="10" height="5" rx="2.5"/>',
  status: '<circle cx="12" cy="12" r="8.5"/><path d="M12 3.5a8.5 8.5 0 0 1 0 17Z" fill="currentColor"/>',
  phone: '<path d="M6.5 3.5h3l1.5 4-2 1.25a9 9 0 0 0 6.25 6.25L16.5 13l4 1.5v3a2 2 0 0 1-2 2A15 15 0 0 1 4.5 5.5a2 2 0 0 1 2-2Z"/>',
  relation: '<path d="M7 17 17 7M9 7h8v8"/>',
  rollup: '<path d="M5 6.5h14M5 12h9M5 17.5h5M17 13v7M14 17l3 3 3-3"/>',
  formula: '<path d="M14.5 4.5h-1.75a2.5 2.5 0 0 0-2.47 2.1L8.72 17.4a2.5 2.5 0 0 1-2.47 2.1H5M7.5 10.5h7M14 13.5l5 5M19 13.5l-5 5"/>',
  id: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="M8 9.5v5M11.5 9.5v5h1.5a2.5 2.5 0 0 0 0-5Z"/>',
  pin: '<path d="M12 21s6.5-5.6 6.5-11a6.5 6.5 0 0 0-13 0c0 5.4 6.5 11 6.5 11Z"/><circle cx="12" cy="10" r="2.25"/>',
  button: '<rect x="3.5" y="7" width="17" height="10" rx="3"/><path d="M9 12h6"/>',
  sort: '<path d="M7 4.5v15M4 16.5l3 3 3-3M17 19.5v-15M14 7.5l3-3 3 3"/>',
  group: '<rect x="3.5" y="4" width="17" height="6.5" rx="1.5"/><rect x="3.5" y="13.5" width="17" height="6.5" rx="1.5"/>',
  eye: '<path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z"/><circle cx="12" cy="12" r="2.75"/>',
  "eye-off": '<path d="M4 4l16 16M10.6 6a9.6 9.6 0 0 1 1.4-.1C18 5.9 21.5 12 21.5 12a17 17 0 0 1-2.7 3.4M6.2 7.6A16 16 0 0 0 2.5 12S6 18.1 12 18.1a9 9 0 0 0 4-.9"/>',
  sigma: '<path d="M17.5 5.5h-11l6 6.5-6 6.5h11"/>',
  expand: '<path d="M14 4.5h5.5V10M19.5 4.5 13 11M10 19.5H4.5V14M4.5 19.5 11 13"/>',
  "chevron-left": '<path d="m14.5 6-6 6 6 6"/>',
  "chevron-up": '<path d="m6 14.5 6-6 6 6"/>',
  sliders: '<path d="M4 7h9M17 7h3M4 17h3M11 17h9M15 4.5v5M9 14.5v5"/>',
  dot: '<circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/>',
  comment: '<path d="M5.5 5h13a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-4.5 3.5V17h-1.5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/>',
  "comment-plus": '<path d="M5.5 5h13a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-4.5 3.5V17h-1.5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/><path d="M12 8.25v5.5M9.25 11h5.5"/>',
  pencil: '<path d="M15 5.5l3.5 3.5L9 18.5H5.5V15Z"/><path d="M12.75 7.75l3.5 3.5"/>',
  smile: '<circle cx="12" cy="12" r="8.5"/><path d="M8.75 14a4 4 0 0 0 6.5 0M9.25 9.75h.01M14.75 9.75h.01"/>',
  heart: '<path d="M12 19.5s-7.25-4.35-7.25-9.5A4 4 0 0 1 12 7.6a4 4 0 0 1 7.25 2.4c0 5.15-7.25 9.5-7.25 9.5Z"/>',
  sparkle: '<path d="M12 4.5l1.6 4.4 4.4 1.6-4.4 1.6-1.6 4.4-1.6-4.4-4.4-1.6 4.4-1.6Z"/><path d="M18.5 15.5l.6 1.6 1.6.6-1.6.6-.6 1.6-.6-1.6-1.6-.6 1.6-.6Z"/>',
  rocket: '<path d="M13.5 4.5c3.25 0 6 2.75 6 6l-6.5 6.5-6-6Z"/><path d="m7 11-2.5.5L3.5 14l3 .5M13 17l-.5 2.5L10 20.5l-.5-3"/><path d="M14.75 9.25h.01"/>',
  lightbulb: '<path d="M9.25 17.5h5.5M10 20.5h4"/><path d="M12 3.5a5.5 5.5 0 0 0-3.25 9.9c.6.45.95 1.15.95 1.9v.7h4.6v-.7c0-.75.35-1.45.95-1.9A5.5 5.5 0 0 0 12 3.5Z"/>',
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
