const PATHS = {
  logo:
    '<rect x="4" y="3.5" width="4.75" height="17" rx="1.6" fill="currentColor" stroke="none"/>' +
    '<rect x="10.25" y="3.5" width="9.75" height="4.75" rx="1.6" fill="currentColor" stroke="none"/>' +
    '<rect x="10.25" y="9.75" width="6.25" height="4.75" rx="1.6" fill="currentColor" stroke="none" opacity="0.62"/>',
  "panel-left": '<rect x="3.5" y="4.5" width="17" height="15" rx="2.5"/><path d="M9.5 4.5v15"/>',
  sun:
    '<circle cx="12" cy="12" r="4"/>' +
    '<path d="M12 2.5v2.2M12 19.3v2.2M2.5 12h2.2M19.3 12h2.2M5.3 5.3l1.6 1.6M17.1 17.1l1.6 1.6M5.3 18.7l1.6-1.6M17.1 6.9l1.6-1.6"/>',
  moon: '<path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5z"/>',
  monitor: '<rect x="3.5" y="5" width="17" height="11.5" rx="1.8"/><path d="M9 20h6M12 16.5V20"/>',
  keyboard:
    '<rect x="2.5" y="6" width="19" height="12" rx="2.5"/>' +
    '<path d="M6.5 10h.01M10 10h.01M14 10h.01M17.5 10h.01M6.5 14h.01M17.5 14h.01M9.5 14h5"/>',
  close: '<path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z" fill="currentColor" stroke="none"/>',
  info: '<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z" fill="currentColor" stroke="none"/>',
  success: '<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" fill="currentColor" stroke="none"/>',
  warning: '<path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z" fill="currentColor" stroke="none"/>',
  danger: '<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" fill="currentColor" stroke="none"/>',
  refresh: '<polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>',
  page: '<path d="M13.5 3.5h-6a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2V8.5Z"/><path d="M13.5 3.5v5h5"/>',
  "chevron-down": '<path d="M7.41 8.59L12 13.17l4.59-4.58L18 10l-6 6-6-6 1.41-1.41z" fill="currentColor" stroke="none"/>',
  "chevron-right": '<path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z" fill="currentColor" stroke="none"/>',
  more: '<circle cx="5.5" cy="12" r="1.7" fill="currentColor" stroke="none"/><circle cx="12" cy="12" r="1.7" fill="currentColor" stroke="none"/><circle cx="18.5" cy="12" r="1.7" fill="currentColor" stroke="none"/>',
  check: '<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" fill="currentColor" stroke="none"/>',
  plus: '<path d="M12 6v12M6 12h12"/>',
  user: '<circle cx="12" cy="8.5" r="3.75"/><path d="M4.75 20a7.25 7.25 0 0 1 14.5 0"/>',
  users:
    '<circle cx="9" cy="8" r="3.2"/><path d="M3 20c0.6-3.6 3-5.8 6-5.8s5.4 2.2 6 5.8"/>' +
    '<path d="M15.5 4.9a3.2 3.2 0 0 1 0 6.2M18 14.6c1.6 0.8 2.7 2.7 3 5.4"/>',
  shield: '<path d="M12 3.5 5 6.25v5.5c0 4.25 2.9 7.6 7 8.75 4.1-1.15 7-4.5 7-8.75v-5.5Z"/><path d="m9.25 12 2 2 3.5-4"/>',
  key: '<circle cx="8" cy="15" r="3.75"/><path d="m10.75 12.25 8.25-8.25M15.5 7.5l2.5 2.5M17.75 5.25l1.75 1.75"/>',
  logout: '<path d="M14 4.5H7a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h7"/><path d="M10 12h10.5M17 8.5l3.5 3.5-3.5 3.5"/>',
  settings: '<path d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z" fill="currentColor" stroke="none"/>',
  globe: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.7 2.6 4 5.6 4 9s-1.3 6.4-4 9c-2.7-2.6-4-5.6-4-9s1.3-6.4 4-9z"/>',
  mail: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="m4 7 8 6 8-6"/>',
  copy: '<rect x="8" y="8" width="12" height="12" rx="2"/><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>',
  trash: '<path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z" fill="currentColor" stroke="none"/>',
  link: '<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>',
  clock: '<circle cx="12" cy="12" r="8"/><path d="M12 8v4l3 2"/>',
  lock: '<path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z" fill="currentColor" stroke="none"/>',
  building: '<path d="M5 20.5V5.5a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v15M16 9.5h2a2 2 0 0 1 2 2v9M3.5 20.5h17"/><path d="M9 7.5h3M9 11h3M9 14.5h3"/>',
  bot: '<rect x="4" y="7.5" width="16" height="12" rx="4"/><path d="M12 7.5V4.5"/><circle cx="12" cy="3.6" r="1.1"/><circle cx="9.3" cy="13.3" r="1.35" fill="currentColor" stroke="none"/><circle cx="14.7" cy="13.3" r="1.35" fill="currentColor" stroke="none"/><path d="M2 12.2v2.8M22 12.2v2.8"/>',
  terminal: '<rect x="3" y="4.5" width="18" height="15" rx="2"/><path d="M7 9.5l3 2.5-3 2.5M12.5 15h4.5"/>',
  alert: '<path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" fill="currentColor" stroke="none"/>',
  download: '<path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z" fill="currentColor" stroke="none"/>',
  search: '<circle cx="10.5" cy="10.5" r="6"/><path d="M15 15l5 5"/><path d="M10.5 7.5v2M9.5 8.5h2" stroke-width="1.4"/>',
  external: '<path d="M14 4h6v6M20 4l-9 9"/><path d="M18 14v4.5A2.5 2.5 0 0 1 15.5 21h-10A2.5 2.5 0 0 1 3 18.5v-10A2.5 2.5 0 0 1 5.5 6H10"/>',
  library: '<path d="M5 4.5v15M9 4.5v15"/><path d="m13 5.25 3.4-.9 3.6 13.5-3.4.9Z"/><path d="M3.5 19.5h17"/>',
  star: '<path d="M12 3.5l2.6 5.3 5.9 0.9-4.25 4.1 1 5.8L12 16.9l-5.25 2.7 1-5.8L3.5 9.7l5.9-0.9z"/>',
  "star-fill": '<path d="M12 3.5l2.6 5.3 5.9 0.9-4.25 4.1 1 5.8L12 16.9l-5.25 2.7 1-5.8L3.5 9.7l5.9-0.9z" fill="currentColor" stroke="none"/>',
  tag: '<path d="M3 12V4a1 1 0 0 1 1-1h8l9 9-9 9z"/><circle cx="7.5" cy="7.5" r="1.4"/>',
  restore: '<path d="M3.5 12a8.5 8.5 0 1 0 2.5-6"/><path d="M3.5 3.5V8H8"/>',
  "move-to": '<path d="M4.5 19.5v-6a4 4 0 0 1 4-4h11"/><path d="m15.5 5.5 4 4-4 4"/>',
  "move-up": '<path d="M19.5 19.5h-6a4 4 0 0 1-4-4v-11"/><path d="m5.5 8.5 4-4 4 4"/>',
  image: '<path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z" fill="currentColor" stroke="none"/>',
  "align-left": '<path d="M4 6h16M4 10h10M4 14h16M4 18h10"/>',
  upload: '<path d="M9 16h6v-6h4l-7-7-7 7h4v6zm-4 2h14v2H5v-2z" fill="currentColor" stroke="none"/>',
  shape: '<circle cx="8" cy="8" r="3.5"/><rect x="13" y="13" width="7" height="7" rx="1.5"/><path d="m16.5 4 3.5 6h-7Z"/>',
  "new-tab": '<rect x="3.5" y="5" width="17" height="14" rx="2"/><path d="M3.5 9h17M12 12v4.5M9.75 14.25h4.5"/>',
  filter: '<path d="M3.5 4.5h17l-6.5 8v6l-4 2v-8z"/>',
  "list-tree": '<path d="M4.5 5.5h6M8.5 12h6M8.5 18.5h6M6 5.5v13h2.5M6 12h2.5"/><path d="M16.5 5.5h3M18.5 12h1M18.5 18.5h1"/>',
  cards: '<rect x="3.5" y="4.5" width="17" height="6" rx="1.5"/><rect x="3.5" y="13.5" width="17" height="6" rx="1.5"/>',
  text: '<path d="M5 6V4h14v2M12 4v16M9 20h6"/>',
  h1: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15.5 9.5 18 8v10"/>',
  h2: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15 9.75a2.25 2.25 0 1 1 4.1 1.3L15 18h4.5"/>',
  h3: '<path d="M4.5 6v12M11.5 6v12M4.5 12h7"/><path d="M15 8.5h4l-2.25 3a2.5 2.5 0 1 1-2.1 3.9"/>',
  "list-bullet":
    '<path d="M9 6.5h11M9 12h11M9 17.5h11"/>' +
    '<circle cx="5" cy="6.5" r="0.9" fill="currentColor" stroke="none"/><circle cx="5" cy="12" r="0.9" fill="currentColor" stroke="none"/><circle cx="5" cy="17.5" r="0.9" fill="currentColor" stroke="none"/>',
  "list-number": '<path d="M10 6.5h9.5M10 12h9.5M10 17.5h9.5"/><path d="M4.5 5 6 4.25V9M4.25 14.25a1.5 1.5 0 1 1 2.4 1.2L4.25 19.5h2.75"/>',
  checkbox: '<rect x="4.5" y="4.5" width="15" height="15" rx="3"/><path d="m8.5 12.25 2.5 2.5 4.5-5"/>',
  quote: '<path d="M6 7.5h4v4.25c0 2.5-1.5 4-4 4.75M14 7.5h4v4.25c0 2.5-1.5 4-4 4.75"/>',
  code: '<path d="m8.5 7.5-4.5 4.5 4.5 4.5M15.5 7.5l4.5 4.5-4.5 4.5M13.5 5.5l-3 13"/>',
  divider: '<path d="M4 12h16"/><path d="M8 7.5h8M8 16.5h8" opacity=".45"/>',
  table: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M3.5 9.5h17M3.5 14.5h17M9.5 9.5v10M15 9.5v10"/>',
  video: '<rect x="3.5" y="6" width="12" height="12" rx="2"/><path d="m15.5 10.5 5-3v9l-5-3"/>',
  audio: '<path d="M9 18V5l11-2v13"/><circle cx="6.5" cy="18" r="2.5"/><circle cx="17.5" cy="16" r="2.5"/>',
  attachment: '<path d="m19 11.5-7.4 7.4a4.5 4.5 0 0 1-6.4-6.4l7.6-7.6a3 3 0 0 1 4.3 4.3l-7.5 7.5a1.5 1.5 0 0 1-2.1-2.1l6.9-6.9"/>',
  callout: '<rect x="3.5" y="5" width="17" height="14" rx="2.5"/><path d="M8 9.5h.01M11 9.5h5.5M8 14h8.5"/>',
  toc: '<path d="M4.5 6.5h15M8 11h11.5M8 15.5h11.5M11.5 20h8"/>',
  columns: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M12 4.5v15"/>',
  diagram: '<rect x="3.5" y="4" width="7" height="5" rx="1.25"/><rect x="13.5" y="15" width="7" height="5" rx="1.25"/><path d="M7 9v4.5a2 2 0 0 0 2 2h4.5"/>',
  grip:
    '<circle cx="9" cy="6" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="6" r="1.6" fill="currentColor" stroke="none"/>' +
    '<circle cx="9" cy="12" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="12" r="1.6" fill="currentColor" stroke="none"/>' +
    '<circle cx="9" cy="18" r="1.6" fill="currentColor" stroke="none"/><circle cx="15" cy="18" r="1.6" fill="currentColor" stroke="none"/>',
  bold: '<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z"/>',
  italic: '<path d="M10 5h7M7 19h7M14 5l-4 14"/>',
  underline: '<path d="M7 4.5v6.5a5 5 0 0 0 10 0V4.5M5.5 20h13"/>',
  strike: '<path d="M4.5 12h15M16 7.5c-.6-1.5-2.1-2.5-4-2.5-2.2 0-4 1.2-4 3 0 1.5 1.2 2.4 3 3M8 16.5c.6 1.6 2.2 2.5 4 2.5 2.4 0 4.25-1.3 4.25-3.25 0-.7-.2-1.3-.6-1.75"/>',
  "text-color": '<path d="m6.5 16 5.5-12 5.5 12M8.5 11.5h7M4.5 20h15"/>',
  align: '<path d="M4 6h16M7 10h10M4 14h16M7 18h10"/>',
  "arrow-up": '<path d="M12 19V5M6.5 10.5 12 5l5.5 5.5"/>',
  "arrow-down": '<path d="M12 5v14M6.5 13.5 12 19l5.5-5.5"/>',
  turn: '<path d="M7 4.5 4 7.5l3 3M4 7.5h12.5a3 3 0 0 1 3 3M17 19.5l3-3-3-3M20 16.5H7.5a3 3 0 0 1-3-3"/>',
  at: '<circle cx="12" cy="12" r="3.5"/><path d="M15.5 12v1.5a2.5 2.5 0 0 0 5 0V12a8.5 8.5 0 1 0-3.4 6.8"/>',
  calendar: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>',
  play: '<path d="M8 5.5v13a1 1 0 0 0 1.52.85l10.4-6.5a1 1 0 0 0 0-1.7L9.52 4.65A1 1 0 0 0 8 5.5z" fill="currentColor" stroke="none"/>',
  database: '<ellipse cx="12" cy="5.5" rx="8" ry="2.8"/><path d="M4 5.5v13c0 1.5 3.6 2.8 8 2.8s8-1.3 8-2.8v-13"/><path d="M4 12c0 1.5 3.6 2.8 8 2.8s8-1.3 8-2.8"/>',
  board: '<rect x="3.5" y="4.5" width="5" height="15" rx="1.5"/><rect x="9.5" y="4.5" width="5" height="10" rx="1.5"/><rect x="15.5" y="4.5" width="5" height="12.5" rx="1.5"/>',
  rows: '<rect x="3.5" y="4.5" width="17" height="4" rx="1.2"/><rect x="3.5" y="10" width="17" height="4" rx="1.2"/><rect x="3.5" y="15.5" width="17" height="4" rx="1.2"/>',
  gallery: '<rect x="4" y="4" width="6.5" height="6.5" rx="1.2"/><rect x="13.5" y="4" width="6.5" height="6.5" rx="1.2"/><rect x="4" y="13.5" width="6.5" height="6.5" rx="1.2"/><rect x="13.5" y="13.5" width="6.5" height="6.5" rx="1.2"/>',
  timeline: '<rect x="3.5" y="5" width="9" height="3" rx="1.5"/><rect x="7.5" y="10.5" width="10" height="3" rx="1.5"/><rect x="11" y="16" width="9.5" height="3" rx="1.5"/>',
  form: '<rect x="4.5" y="3.5" width="15" height="17" rx="2"/><path d="M8 8h8M8 12h8M8 16h4.5"/>',
  hash: '<path d="M9.5 4 8 20M16 4l-1.5 16M4.5 9h15M4 15h15"/>',
  select: '<circle cx="12" cy="12" r="8.5"/><path d="m8.75 10.75 3.25 3.25 3.25-3.25"/>',
  multi: '<rect x="3.5" y="5.5" width="8" height="5" rx="2.5"/><rect x="12.5" y="5.5" width="8" height="5" rx="2.5"/><rect x="3.5" y="13.5" width="10" height="5" rx="2.5"/>',
  status: '<circle cx="12" cy="12" r="8.5"/><path d="M12 3.5a8.5 8.5 0 0 1 0 17z" fill="currentColor" stroke="none"/>',
  phone: '<path d="M6.5 3.5h3l1.5 4-2 1.25a9 9 0 0 0 6.25 6.25L16.5 13l4 1.5v3a2 2 0 0 1-2 2A15 15 0 0 1 4.5 5.5a2 2 0 0 1 2-2Z"/>',
  relation: '<path d="M7 17 17 7M9 7h8v8"/>',
  rollup: '<path d="M5 6.5h14M5 12h9M5 17.5h5M17 13v7M14 17l3 3 3-3"/>',
  formula: '<path d="M14.5 4.5h-1.75a2.5 2.5 0 0 0-2.47 2.1L8.72 17.4a2.5 2.5 0 0 1-2.47 2.1H5M7.5 10.5h7M14 13.5l5 5M19 13.5l-5 5"/>',
  id: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="M8 9.5v5M11.5 9.5v5h1.5a2.5 2.5 0 0 0 0-5Z"/>',
  pin: '<path d="M12 21s-6.5-6.1-6.5-11a6.5 6.5 0 0 1 13 0c0 4.9-6.5 11-6.5 11z"/><circle cx="12" cy="10" r="2.4"/>',
  button: '<rect x="3.5" y="7" width="17" height="10" rx="3"/><path d="M9 12h6"/>',
  sort: '<path d="M7 4v16M3.5 7.5L7 4l3.5 3.5"/><path d="M17 20V4M13.5 16.5L17 20l3.5-3.5"/>',
  group: '<rect x="3.5" y="4" width="17" height="6.5" rx="1.5"/><rect x="3.5" y="13.5" width="17" height="6.5" rx="1.5"/>',
  eye: '<path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z" fill="currentColor" stroke="none"/>',
  "eye-off": '<path d="M12 7c2.76 0 5 2.24 5 5 0 .65-.13 1.26-.36 1.83l2.92 2.92c1.51-1.26 2.7-2.89 3.43-4.75-1.73-4.39-6-7.5-11-7.5-1.4 0-2.74.25-3.98.7l2.16 2.16C10.74 7.13 11.35 7 12 7zM2 4.27l2.28 2.28.46.46C3.08 8.3 1.78 10.02 1 12c1.73 4.39 6 7.5 11 7.5 1.55 0 3.03-.3 4.38-.84l.42.42L19.73 22 21 20.73 3.27 3 2 4.27zM7.53 9.8l1.55 1.55c-.05.21-.08.43-.08.65 0 1.66 1.34 3 3 3 .22 0 .44-.03.65-.08l1.55 1.55c-.67.33-1.41.53-2.2.53-2.76 0-5-2.24-5-5 0-.79.2-1.53.53-2.2zm4.31-.78l3.15 3.15.02-.16c0-1.66-1.34-3-3-3l-.17.01z" fill="currentColor" stroke="none"/>',
  sigma: '<path d="M17.5 5.5h-11l6 6.5-6 6.5h11"/>',
  chart: '<path d="M5 9.2h3V19H5V9.2zM10.6 5h2.8v14h-2.8V5zm5.6 8H19v6h-2.8v-6z" fill="currentColor" stroke="none"/>',
  bolt: '<path d="M13 3.5 5.5 13.5h6l-1 7 7.5-10h-6Z"/>',
  dashboard: '<rect x="4" y="4" width="7" height="9" rx="1.5"/><rect x="13" y="4" width="7" height="5" rx="1.5"/><rect x="13" y="11" width="7" height="9" rx="1.5"/><rect x="4" y="15" width="7" height="5" rx="1.5"/>',
  feed: '<rect x="4.5" y="4" width="15" height="7" rx="1.5"/><rect x="4.5" y="13" width="15" height="7" rx="1.5"/><path d="M7.5 7.5h6M7.5 16.5h4"/>',
  graph: '<circle cx="6" cy="7" r="2.25"/><circle cx="17.5" cy="6" r="2.25"/><circle cx="12" cy="17.5" r="2.25"/><circle cx="18.5" cy="15" r="1.5"/><path d="M8.1 7.9l7.2-1.4M7.1 9l3.8 6.6M16.6 8.1l-3.6 7.3M14.2 17l2.9-1.4"/>',
  heatmap: '<rect x="4" y="4" width="4.5" height="4.5" rx="1"/><rect x="9.75" y="4" width="4.5" height="4.5" rx="1" fill="currentColor" fill-opacity=".45"/><rect x="15.5" y="4" width="4.5" height="4.5" rx="1"/><rect x="4" y="9.75" width="4.5" height="4.5" rx="1" fill="currentColor"/><rect x="9.75" y="9.75" width="4.5" height="4.5" rx="1"/><rect x="15.5" y="9.75" width="4.5" height="4.5" rx="1" fill="currentColor" fill-opacity=".45"/><rect x="4" y="15.5" width="4.5" height="4.5" rx="1"/><rect x="9.75" y="15.5" width="4.5" height="4.5" rx="1" fill="currentColor"/><rect x="15.5" y="15.5" width="4.5" height="4.5" rx="1"/>',
  expand: '<polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/>',
  "chevron-left": '<path d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z" fill="currentColor" stroke="none"/>',
  "chevron-up": '<path d="M7.41 15.41L12 10.83l4.59 4.58L18 14l-6-6-6 6 1.41 1.41z" fill="currentColor" stroke="none"/>',
  sliders: '<path d="M4 6h16M4 12h16M4 18h16"/><circle cx="9" cy="6" r="2" fill="currentColor"/><circle cx="15" cy="12" r="2" fill="currentColor"/><circle cx="7" cy="18" r="2" fill="currentColor"/>',
  dot: '<circle cx="12" cy="12" r="3" fill="currentColor" stroke="none"/>',
  comment: '<path d="M5.5 5h13a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-4.5 3.5V17h-1.5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/>',
  "comment-plus": '<path d="M5.5 5h13a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-7l-4.5 3.5V17h-1.5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2Z"/><path d="M12 8.25v5.5M9.25 11h5.5"/>',
  inbox: '<path d="M4 13.5 6.25 5.5a1.5 1.5 0 0 1 1.45-1h8.6a1.5 1.5 0 0 1 1.45 1L20 13.5V18a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18Z"/><path d="M4 13.5h4.5l1.25 2h4.5l1.25-2H20"/>',
  bell: '<path d="M6.5 16.5V11a5.5 5.5 0 0 1 11 0v5.5l1.5 1.5H5Z"/><path d="M10.25 20.25a1.9 1.9 0 0 0 3.5 0"/>',
  archive: '<rect x="3.75" y="4.5" width="16.5" height="4" rx="1"/><path d="M5.25 8.5v9.25a1.5 1.5 0 0 0 1.5 1.5h10.5a1.5 1.5 0 0 0 1.5-1.5V8.5M10 12.5h4"/>',
  pencil: '<path d="M15.5 4.5l4 4L8.5 19.5 3.5 20.5l1-5z"/><path d="M13.5 6.5l4 4"/><path d="M4.5 15.5l4 4"/>',
  smile: '<circle cx="12" cy="12" r="8.5"/><path d="M8.75 14a4 4 0 0 0 6.5 0M9.25 9.75h.01M14.75 9.75h.01"/>',
  heart: '<path d="M12 20s-7.5-4.5-7.5-10A4.3 4.3 0 0 1 12 7.2 4.3 4.3 0 0 1 19.5 10c0 5.5-7.5 10-7.5 10z"/>',
  agent: '<rect x="4" y="7.5" width="16" height="12" rx="4"/><path d="M12 7.5V4.5"/><circle cx="12" cy="3.6" r="1.1"/><circle cx="9.3" cy="13.3" r="1.35" fill="currentColor" stroke="none"/><circle cx="14.7" cy="13.3" r="1.35" fill="currentColor" stroke="none"/><path d="M2 12.2v2.8M22 12.2v2.8"/>',
  "agent-chat": '<path d="M6 4.5h12a3.5 3.5 0 0 1 3.5 3.5v6a3.5 3.5 0 0 1-3.5 3.5h-6.5L7 21v-3.5H6A3.5 3.5 0 0 1 2.5 14V8A3.5 3.5 0 0 1 6 4.5z"/><circle cx="9.2" cy="11" r="1.3" fill="currentColor" stroke="none"/><circle cx="14.8" cy="11" r="1.3" fill="currentColor" stroke="none"/>',
  send: '<line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/>',
  "ai-chip": '<rect x="6" y="6" width="12" height="12" rx="2.5"/><rect x="9.5" y="9.5" width="5" height="5" rx="1"/><path d="M9.5 3v3M14.5 3v3M9.5 18v3M14.5 18v3M3 9.5h3M3 14.5h3M18 9.5h3M18 14.5h3"/>',
  rocket: '<path d="M13.5 4.5c3.25 0 6 2.75 6 6l-6.5 6.5-6-6Z"/><path d="m7 11-2.5.5L3.5 14l3 .5M13 17l-.5 2.5L10 20.5l-.5-3"/><path d="M14.75 9.25h.01"/>',
  lightbulb: '<path d="M9 18h6M10 21h4"/><path d="M12 3a6 6 0 0 0-3.6 10.8c0.7 0.6 1.1 1.4 1.1 2.2h5c0-0.8 0.4-1.6 1.1-2.2A6 6 0 0 0 12 3z"/>',
  offline: '<path d="M7 18.5h10.25a3.75 3.75 0 0 0 .6-7.45A5.75 5.75 0 0 0 6.6 9.6 4.5 4.5 0 0 0 7 18.5Z"/><path d="m9.5 13.75 1.75 1.75 3.25-3.5"/>',
  home: '<path d="M4.5 10.25 12 4l7.5 6.25V19a1.5 1.5 0 0 1-1.5 1.5h-3.75v-5.75h-4.5v5.75H6A1.5 1.5 0 0 1 4.5 19Z"/>',
  compose: '<path d="M11 4.5H6.5a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2V13"/><path d="M17.25 3.75l3 3-7.75 7.75H9.5v-3Z"/>',
  "chevrons-left": '<path d="M11.5 6.5 6 12l5.5 5.5M18 6.5 12.5 12l5.5 5.5"/>',
  "arrow-left": '<path d="M19 12H5M10.5 6.5 5 12l5.5 5.5"/>',
  "arrow-right": '<path d="M5 12h14M13.5 6.5 19 12l-5.5 5.5"/>',
  help: '<circle cx="12" cy="12" r="8.5"/><path d="M9.6 9.6a2.45 2.45 0 0 1 4.75.85c0 1.65-2.35 2.15-2.35 3.75M12 17h.01"/>',
} as const;

export type IconName = keyof typeof PATHS;

export const ICON_NAMES = Object.keys(PATHS) as IconName[];

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
