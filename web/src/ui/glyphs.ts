const GLYPHS: Record<string, string> = {
  page: '<path d="M13.5 3.5h-6a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2V8.5Z"/><path d="M13.5 3.5v5h5"/>',
  note: '<path d="M5.5 4.5h13a1 1 0 0 1 1 1v9l-5 5h-9a1 1 0 0 1-1-1v-13a1 1 0 0 1 1-1Z"/><path d="M19.5 14.5h-4a1 1 0 0 0-1 1v4M8.5 9h7M8.5 12.5h4"/>',
  notebook: '<rect x="5.5" y="3.5" width="13" height="17" rx="2"/><path d="M9.5 3.5v17M3.5 8h4M3.5 12h4M3.5 16h4"/>',
  book: '<path d="M5 18V5.5a2 2 0 0 1 2-2h12v13H7a2 2 0 0 0-2 2Zm0 0a2 2 0 0 0 2 2h12v-3.5"/>',
  bookmark: '<path d="M7 3.5h10a1 1 0 0 1 1 1v16l-6-4-6 4v-16a1 1 0 0 1 1-1Z"/>',
  pen: '<path d="M12 3.5 6.5 9l2 9.5h7L17.5 9Z"/><path d="M12 3.5V12M8.5 20.5h7"/><circle cx="12" cy="13.25" r="1.25"/>',
  pencil: '<path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"/><path d="m14.5 5.5 3 3"/>',
  clipboard: '<rect x="5.5" y="4.5" width="13" height="16" rx="2"/><path d="M9 4.5V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v.5a1 1 0 0 1-1 1h-4a1 1 0 0 1-1-1ZM9 11h6M9 14.5h4"/>',
  list: '<path d="M9 6.5h11M9 12h11M9 17.5h11M4.5 6.5h.01M4.5 12h.01M4.5 17.5h.01"/>',
  checklist: '<path d="m3.5 6.5 1.5 1.5 3-3M3.5 15.5l1.5 1.5 3-3M11 7h9.5M11 16h9.5"/>',
  quote: '<path d="M10 7.5H6.5a1 1 0 0 0-1 1V12a1 1 0 0 0 1 1H10v1.5a3 3 0 0 1-3 3M18.5 7.5H15a1 1 0 0 0-1 1V12a1 1 0 0 0 1 1h3.5v1.5a3 3 0 0 1-3 3"/>',
  inbox: '<path d="M3.5 13.5 6 5.5a1.5 1.5 0 0 1 1.4-1h9.2a1.5 1.5 0 0 1 1.4 1l2.5 8"/><path d="M3.5 13.5V18a2 2 0 0 0 2 2h13a2 2 0 0 0 2-2v-4.5h-5l-1.5 2.5h-4l-1.5-2.5Z"/>',
  archive: '<rect x="3.5" y="4.5" width="17" height="4.5" rx="1"/><path d="M5 9v9.5A1.5 1.5 0 0 0 6.5 20h11a1.5 1.5 0 0 0 1.5-1.5V9M10 13h4"/>',
  folder: '<path d="M3.5 7.5a2 2 0 0 1 2-2h4l2 2.5h7a2 2 0 0 1 2 2v7.5a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2Z"/>',
  paperclip: '<path d="m19 11.5-7.3 7.3a4.5 4.5 0 0 1-6.4-6.4l7.6-7.6a3 3 0 0 1 4.2 4.2l-7.4 7.4a1.5 1.5 0 0 1-2.1-2.1l6.6-6.6"/>',
  tag: '<path d="M3.5 12.1V5A1.5 1.5 0 0 1 5 3.5h7.1a1.5 1.5 0 0 1 1.06.44l7.4 7.4a1.5 1.5 0 0 1 0 2.12l-7.1 7.1a1.5 1.5 0 0 1-2.12 0l-7.4-7.4a1.5 1.5 0 0 1-.44-1.06Z"/><circle cx="8" cy="8" r="1.25"/>',
  calendar: '<rect x="3.5" y="5" width="17" height="15.5" rx="2"/><path d="M3.5 10h17M8 3v4M16 3v4"/>',
  clock: '<circle cx="12" cy="12" r="8.5"/><path d="M12 7.5V12l3 2"/>',
  hourglass: '<path d="M6.5 3.5h11M6.5 20.5h11M7.5 3.5v3a4.5 4.5 0 0 0 9 0v-3M7.5 20.5v-3a4.5 4.5 0 0 1 9 0v3"/>',
  alarm: '<circle cx="12" cy="13" r="7"/><path d="M12 9.5V13l2 1.5M4.5 5.5l2.5-2M19.5 5.5 17 3.5"/>',
  briefcase: '<rect x="3.5" y="7" width="17" height="12.5" rx="2"/><path d="M9 7V5.5A1.5 1.5 0 0 1 10.5 4h3A1.5 1.5 0 0 1 15 5.5V7M3.5 12.5h17"/>',
  "chart-bar": '<path d="M4.5 20h15M7 20v-6M12 20V7M17 20v-9.5"/>',
  "chart-line": '<path d="M4 4v15.5a.5.5 0 0 0 .5.5H20"/><path d="m7.5 15 3.5-4 3 2.5 5-6"/>',
  "chart-pie": '<path d="M12 3.5a8.5 8.5 0 1 0 8.5 8.5H12Z"/><path d="M15 3.8A8.5 8.5 0 0 1 20.2 9H15Z"/>',
  target: '<circle cx="12" cy="12" r="8.5"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1.5"/>',
  flag: '<path d="M5.5 21V4.5"/><path d="M5.5 4.5h11l-2 4 2 4h-11"/>',
  rocket: '<path d="M12 3c3 2 4.5 5 4.5 8.5V16h-9v-4.5C7.5 8 9 5 12 3Z"/><path d="M7.5 13 5 15.5V19l2.5-1.5M16.5 13l2.5 2.5V19l-2.5-1.5M10 19.5c.5 1 1.2 1.5 2 1.5s1.5-.5 2-1.5"/><circle cx="12" cy="9.5" r="1.5"/>',
  lightbulb: '<path d="M9 17.5h6M10 20.5h4"/><path d="M12 3.5a6 6 0 0 0-3.5 10.9c.4.3.5.7.5 1.1v.5h6v-.5c0-.4.1-.8.5-1.1A6 6 0 0 0 12 3.5Z"/>',
  puzzle: '<path d="M4.5 9.5v-3a1 1 0 0 1 1-1h3.25a2.25 2.25 0 1 1 4.5 0h3.25a1 1 0 0 1 1 1v3.25a2.25 2.25 0 1 1 0 4.5v3.25a1 1 0 0 1-1 1h-3.25a2.25 2.25 0 1 0-4.5 0H5.5a1 1 0 0 1-1-1v-3.25a2.25 2.25 0 1 0 0-4.5Z"/>',
  layers: '<path d="m12 3.5 8.5 4.5-8.5 4.5L3.5 8Z"/><path d="m3.5 12 8.5 4.5 8.5-4.5M3.5 16l8.5 4.5 8.5-4.5"/>',
  kanban: '<rect x="3.5" y="4" width="17" height="16" rx="2"/><path d="M8 8v6M12 8v9M16 8v3.5"/>',
  code: '<path d="m8.5 7-5 5 5 5M15.5 7l5 5-5 5M13.5 4.5l-3 15"/>',
  terminal: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="m7.5 9.5 3 2.5-3 2.5M12.5 15h4"/>',
  bug: '<rect x="7.5" y="7.5" width="9" height="12" rx="4.5"/><path d="M9.5 7.5a2.5 2.5 0 0 1 5 0M12 11v8.5M3.5 13h4M16.5 13h4M4.5 8.5l3 2M19.5 8.5l-3 2M4.5 19l3-2M19.5 19l-3-2"/>',
  database: '<ellipse cx="12" cy="6" rx="7" ry="2.5"/><path d="M5 6v12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5V6M5 12c0 1.4 3.1 2.5 7 2.5s7-1.1 7-2.5"/>',
  server: '<rect x="4" y="4" width="16" height="6.5" rx="1.5"/><rect x="4" y="13.5" width="16" height="6.5" rx="1.5"/><path d="M7.5 7.25h.01M7.5 16.75h.01M11 7.25h5M11 16.75h5"/>',
  cloud: '<path d="M7 18.5a4 4 0 0 1-.6-7.95A5.5 5.5 0 0 1 17 9.5a4.5 4.5 0 0 1 0 9Z"/>',
  gear: '<path d="M10.3 3.5h3.4l.5 2.4 1.6.9 2.3-.8 1.7 2.9-1.8 1.6v1.9l1.8 1.6-1.7 2.9-2.3-.8-1.6.9-.5 2.4h-3.4l-.5-2.4-1.6-.9-2.3.8-1.7-2.9 1.8-1.6V11L4.2 9.4l1.7-2.9 2.3.8 1.6-.9Z"/><circle cx="12" cy="12" r="2.75"/>',
  wrench: '<path d="M14.7 5.3a4 4 0 0 1 4.6-.8l-2.6 2.6.2 2 2 .2 2.6-2.6a4 4 0 0 1-5.3 5.2l-7.8 7.8a1.9 1.9 0 0 1-2.7-2.7l7.8-7.8a4 4 0 0 1 1.2-3.9Z"/>',
  user: '<circle cx="12" cy="8.5" r="3.75"/><path d="M4.75 20a7.25 7.25 0 0 1 14.5 0"/>',
  users: '<circle cx="9" cy="8.75" r="3.25"/><path d="M3 19.5a6 6 0 0 1 12 0M15.5 5.75a3.25 3.25 0 0 1 0 6M17.5 14a6 6 0 0 1 3.5 5.5"/>',
  chat: '<path d="M20 12a7.5 7.5 0 0 1-11 6.6L4 20l1.4-4.6A7.5 7.5 0 1 1 20 12Z"/>',
  mail: '<rect x="3.5" y="5.5" width="17" height="13" rx="2"/><path d="m4 7 8 6 8-6"/>',
  phone: '<path d="M8.6 3.5H6.3a1.8 1.8 0 0 0-1.8 1.9 15.5 15.5 0 0 0 14.1 14.1 1.8 1.8 0 0 0 1.9-1.8v-2.3a1 1 0 0 0-.8-1l-3.2-.7a1 1 0 0 0-.9.3l-1.4 1.5a12 12 0 0 1-5.6-5.6L10 8.5a1 1 0 0 0 .3-.9l-.7-3.3a1 1 0 0 0-1-.8Z"/>',
  megaphone: '<path d="M4 10v4a1 1 0 0 0 1 1h2.5L15 19V5L7.5 9H5a1 1 0 0 0-1 1Z"/><path d="m7.5 15 1 4.5h2l-1-4M18 9.5a3.5 3.5 0 0 1 0 5"/>',
  heart: '<path d="M12 20s-7.5-4.6-7.5-10A4.5 4.5 0 0 1 12 7a4.5 4.5 0 0 1 7.5 3c0 5.4-7.5 10-7.5 10Z"/>',
  star: '<path d="m12 3.5 2.6 5.3 5.9.9-4.25 4.1 1 5.8L12 16.9l-5.25 2.7 1-5.8L3.5 9.7l5.9-.9Z"/>',
  home: '<path d="M4 10.5 12 4l8 6.5V19a1.5 1.5 0 0 1-1.5 1.5H15V15H9v5.5H5.5A1.5 1.5 0 0 1 4 19Z"/>',
  building: '<path d="M5 20.5V5.5a2 2 0 0 1 2-2h7a2 2 0 0 1 2 2v15M16 9.5h2a2 2 0 0 1 2 2v9M3.5 20.5h17M9 7.5h3M9 11h3M9 14.5h3"/>',
  map: '<path d="M9 4.5 3.5 6.5v13l5.5-2 6 2 5.5-2v-13l-5.5 2Z"/><path d="M9 4.5v13M15 6.5v13"/>',
  pin: '<path d="M12 20.5s-6.5-5.4-6.5-10.5a6.5 6.5 0 0 1 13 0c0 5.1-6.5 10.5-6.5 10.5Z"/><circle cx="12" cy="10" r="2.5"/>',
  globe: '<circle cx="12" cy="12" r="8.5"/><path d="M3.5 12h17M12 3.5c2.4 2.6 3.5 5.4 3.5 8.5s-1.1 5.9-3.5 8.5c-2.4-2.6-3.5-5.4-3.5-8.5s1.1-5.9 3.5-8.5Z"/>',
  compass: '<circle cx="12" cy="12" r="8.5"/><path d="m15.5 8.5-2 5-5 2 2-5Z"/>',
  leaf: '<path d="M5 19c0-8 5-14 15-14 0 9-5.5 14-12.5 14Z"/><path d="M5 19c3-4 6-6.5 9.5-8"/>',
  tree: '<path d="M12 21v-5M12 3 6.5 10h3L6 15h12l-3.5-5h3Z"/>',
  flower: '<circle cx="12" cy="6.5" r="2.5"/><circle cx="15.5" cy="10" r="2.5"/><circle cx="12" cy="13.5" r="2.5"/><circle cx="8.5" cy="10" r="2.5"/><path d="M12 16v4.5M12 19c-1.5-1.5-3-2-4.5-1.5M12 19c1.5-1.5 3-2 4.5-1.5"/>',
  sun: '<circle cx="12" cy="12" r="3.75"/><path d="M12 3v1.75M12 19.25V21M3 12h1.75M19.25 12H21M5.64 5.64l1.24 1.24M17.12 17.12l1.24 1.24M5.64 18.36l1.24-1.24M17.12 6.88l1.24-1.24"/>',
  moon: '<path d="M19.5 14.25A7.5 7.5 0 0 1 9.75 4.5a7.5 7.5 0 1 0 9.75 9.75Z"/>',
  mountain: '<path d="m3 19.5 6.5-11 4 6.5 2.5-3.5 5 8Z"/>',
  flame: '<path d="M12 20.5a6 6 0 0 1-6-6c0-4 3-6 4-9.5 2 1.5 3 3.5 3 5.5 1-.5 1.75-1.5 2-3 1.5 1.5 3 4 3 7a6 6 0 0 1-6 6Z"/>',
  drop: '<path d="M12 3.5s6 6.5 6 11a6 6 0 0 1-12 0c0-4.5 6-11 6-11Z"/>',
  snowflake: '<path d="M12 3v18M4.2 7.5l15.6 9M4.2 16.5l15.6-9M9.5 4.5l2.5 2 2.5-2M9.5 19.5l2.5-2 2.5 2"/>',
  wave: '<path d="M3 9c1.5 0 1.5-1.5 3-1.5S7.5 9 9 9s1.5-1.5 3-1.5S13.5 9 15 9s1.5-1.5 3-1.5S19.5 9 21 9M3 14c1.5 0 1.5-1.5 3-1.5S7.5 14 9 14s1.5-1.5 3-1.5 1.5 1.5 3 1.5 1.5-1.5 3-1.5 1.5 1.5 3 1.5M3 19c1.5 0 1.5-1.5 3-1.5S7.5 19 9 19s1.5-1.5 3-1.5 1.5 1.5 3 1.5 1.5-1.5 3-1.5 1.5 1.5 3 1.5"/>',
  camera: '<path d="M4.5 8A1.5 1.5 0 0 1 6 6.5h2.5L10 4.5h4l1.5 2H18A1.5 1.5 0 0 1 19.5 8v10a1.5 1.5 0 0 1-1.5 1.5H6A1.5 1.5 0 0 1 4.5 18Z"/><circle cx="12" cy="12.75" r="3.25"/>',
  image: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><circle cx="9" cy="9.5" r="1.5"/><path d="m3.5 17 5-5 4 4 2.5-2.5 5.5 5.5"/>',
  music: '<path d="M9 18V5.5l10-2V16"/><circle cx="6.5" cy="18" r="2.5"/><circle cx="16.5" cy="16" r="2.5"/>',
  film: '<rect x="3.5" y="4" width="17" height="16" rx="2"/><path d="M7.5 4v16M16.5 4v16M3.5 8h4M3.5 12h17M3.5 16h4M16.5 8h4M16.5 16h4"/>',
  gift: '<rect x="4" y="8" width="16" height="4" rx="1"/><path d="M5.5 12v7A1.5 1.5 0 0 0 7 20.5h10a1.5 1.5 0 0 0 1.5-1.5v-7M12 8v12.5M12 8c-1.5-3-5-4-5-2s3 2 5 2 5 0 5-2-3.5-1-5 2Z"/>',
  coffee: '<path d="M5 9.5h11v5a5 5 0 0 1-5 5h-1a5 5 0 0 1-5-5Z"/><path d="M16 11h1.5a2.5 2.5 0 0 1 0 5h-2M8 3.5V6M11.5 3.5V6"/>',
  cart: '<path d="M3.5 4.5h2l2 10.5A1.5 1.5 0 0 0 9 16.25h8.25a1.5 1.5 0 0 0 1.45-1.15L20.5 8h-14"/><circle cx="9.5" cy="19.5" r="1.25"/><circle cx="17" cy="19.5" r="1.25"/>',
  wallet: '<path d="M18.5 7.5v-2A1.5 1.5 0 0 0 17 4H6a2.5 2.5 0 0 0-2.5 2.5v11A2.5 2.5 0 0 0 6 20h12.5a1.5 1.5 0 0 0 1.5-1.5V9a1.5 1.5 0 0 0-1.5-1.5H6A2.5 2.5 0 0 1 3.5 5M16.5 14h.01"/>',
  coin: '<circle cx="12" cy="12" r="8.5"/><path d="M14.5 9A2.5 2 0 0 0 12 7.5c-1.5 0-2.5.8-2.5 2s1 1.7 2.5 2 2.5.8 2.5 2-1 2-2.5 2A2.5 2 0 0 1 9.5 15M12 6v1.5M12 16.5V18"/>',
  key: '<circle cx="8" cy="15" r="3.75"/><path d="m10.75 12.25 8.25-8.25M15.5 7.5l2.5 2.5M17.75 5.25l1.75 1.75"/>',
  lock: '<rect x="5" y="10.5" width="14" height="10" rx="2"/><path d="M8.5 10.5V7.5a3.5 3.5 0 0 1 7 0v3"/>',
  shield: '<path d="M12 3.5 5 6.25v5.5c0 4.25 2.9 7.6 7 8.75 4.1-1.15 7-4.5 7-8.75v-5.5Z"/>',
  trophy: '<path d="M8 4.5h8V10a4 4 0 0 1-8 0Z"/><path d="M8 6H5.5v1.5a3 3 0 0 0 3 3M16 6h2.5v1.5a3 3 0 0 1-3 3M12 14v3M8.5 20.5h7M9.5 20.5c0-2 1-3.5 2.5-3.5s2.5 1.5 2.5 3.5"/>',
  crown: '<path d="m3.5 8 4.25 4L12 5l4.25 7 4.25-4-2 10.5h-13Z"/><path d="M5.5 20.5h13"/>',
  anchor: '<circle cx="12" cy="5.5" r="2"/><path d="M12 7.5v13M8 11h8M4.5 13.5a7.5 7.5 0 0 0 15 0"/>',
  bell: '<path d="M6 16.5V11a6 6 0 0 1 12 0v5.5l1.5 1.5h-15Z"/><path d="M10 20.5a2 2 0 0 0 4 0"/>',
  link: '<path d="M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1"/><path d="M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1"/>',
  bolt: '<path d="M13 3.5 5.5 13.5H12l-1 7 7.5-10H12Z"/>',
  atom: '<circle cx="12" cy="12" r="1.5"/><ellipse cx="12" cy="12" rx="8.5" ry="3.5"/><ellipse cx="12" cy="12" rx="8.5" ry="3.5" transform="rotate(60 12 12)"/><ellipse cx="12" cy="12" rx="8.5" ry="3.5" transform="rotate(-60 12 12)"/>',
  flask: '<path d="M9.5 3.5h5M10.5 3.5V9L5 18.5a1.4 1.4 0 0 0 1.2 2h11.6a1.4 1.4 0 0 0 1.2-2L13.5 9V3.5M7.5 14.5h9"/>',
  cap: '<path d="M2.5 9.5 12 5l9.5 4.5L12 14Z"/><path d="M6.5 11.5V16c0 1.4 2.5 2.5 5.5 2.5s5.5-1.1 5.5-2.5v-4.5M21.5 9.5V15"/>',
  palette: '<path d="M12 3.5a8.5 8.5 0 0 0 0 17c1.1 0 1.75-.75 1.75-1.6 0-.45-.2-.8-.45-1.1-.25-.3-.45-.65-.45-1.1 0-.9.75-1.6 1.65-1.6h1.9a4.1 4.1 0 0 0 4.1-4.1c0-4.2-3.8-7.5-8.5-7.5Z"/><circle cx="7.75" cy="11" r="1"/><circle cx="10" cy="7.5" r="1"/><circle cx="14.5" cy="7.5" r="1"/>',
  brush: '<path d="M18.5 3.5 10 12l2 2 8.5-8.5a1.4 1.4 0 0 0-2-2Z"/><path d="M10 12c-2.5 0-4 1.5-4 3.5 0 1.6-.8 2.7-2.5 3.5 4 1.5 9 .5 8.5-5Z"/>',
  dice: '<rect x="4" y="4" width="16" height="16" rx="3"/><path d="M8.5 8.5h.01M15.5 8.5h.01M12 12h.01M8.5 15.5h.01M15.5 15.5h.01"/>',
  plane: '<path d="M10.5 4.5c0-.8.7-1.5 1.5-1.5s1.5.7 1.5 1.5V9l7 4v2l-7-2v4.5l2.5 2v1.5L12 20l-4 1v-1.5l2.5-2V13l-7 2v-2l7-4Z"/>',
  car: '<path d="M5 16.5V12l1.8-4.6a1.5 1.5 0 0 1 1.4-.9h7.6a1.5 1.5 0 0 1 1.4.9L19 12v4.5Z"/><path d="M5 12h14M5 16.5v2M19 16.5v2M7.5 14.25h.01M16.5 14.25h.01"/>',
  paw: '<circle cx="7" cy="10" r="1.75"/><circle cx="10.25" cy="6.25" r="1.75"/><circle cx="13.75" cy="6.25" r="1.75"/><circle cx="17" cy="10" r="1.75"/><path d="M12 11.5c-2.5 0-5 3.5-5 5.5 0 1.5 1 2.5 2.5 2.5 1 0 1.5-.5 2.5-.5s1.5.5 2.5.5c1.5 0 2.5-1 2.5-2.5 0-2-2.5-5.5-5-5.5Z"/>',
  fish: '<path d="M3.5 12c2.5-4 6-6 9.5-6 4 0 7 3 7.5 6-.5 3-3.5 6-7.5 6-3.5 0-7-2-9.5-6Zm0 0L2 8.5M3.5 12 2 15.5M16.5 10.5h.01"/>',
  apple: '<path d="M12 7.5c-1.5-1-5.5-1.5-7 2-1.5 3.5.5 8.5 3 10 1.2.7 2.5.5 4-.2 1.5.7 2.8.9 4 .2 2.5-1.5 4.5-6.5 3-10-1.5-3.5-5.5-3-7-2Z"/><path d="M12 7.5c0-2 1-3.5 3-4"/>',
  eye: '<path d="M2.5 12S6 5.5 12 5.5 21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z"/><circle cx="12" cy="12" r="3"/>',
  box: '<path d="m12 3.5 8 4v9l-8 4-8-4v-9Z"/><path d="m4 7.5 8 4 8-4M12 11.5v9"/>',
  grid: '<rect x="4" y="4" width="6.5" height="6.5" rx="1.25"/><rect x="13.5" y="4" width="6.5" height="6.5" rx="1.25"/><rect x="4" y="13.5" width="6.5" height="6.5" rx="1.25"/><rect x="13.5" y="13.5" width="6.5" height="6.5" rx="1.25"/>',
  table: '<rect x="3.5" y="4.5" width="17" height="15" rx="2"/><path d="M3.5 9.5h17M3.5 14.5h17M9.5 9.5v10"/>',
  scale: '<path d="M12 4v16.5M8 20.5h8M5 7.5h14"/><path d="m5 7.5-2.5 6a2.5 2.5 0 0 0 5 0Zm14 0-2.5 6a2.5 2.5 0 0 0 5 0Z"/>',
  cross: '<path d="M9.5 3.5h5v6h6v5h-6v6h-5v-6h-6v-5h6Z"/>',
};

const WORDS: Record<string, string> = {
  page: "document file doc",
  note: "sticky memo",
  notebook: "journal diary",
  book: "read library manual guide",
  bookmark: "save reading",
  pen: "write ink fountain sign",
  pencil: "edit draft write",
  clipboard: "brief task todo",
  list: "items bullets",
  checklist: "todo tasks done",
  quote: "citation saying",
  inbox: "incoming tray",
  archive: "storage box old",
  folder: "directory files",
  paperclip: "attachment",
  tag: "label",
  calendar: "date schedule event",
  clock: "time hour",
  hourglass: "wait time deadline",
  alarm: "reminder wake",
  briefcase: "work job business",
  "chart-bar": "graph stats metrics",
  "chart-line": "graph trend growth",
  "chart-pie": "graph share",
  target: "goal aim okr",
  flag: "milestone goal",
  rocket: "launch ship start",
  lightbulb: "idea insight",
  puzzle: "plugin piece",
  layers: "stack levels",
  kanban: "board tasks",
  code: "developer program",
  terminal: "console shell cli",
  bug: "issue defect",
  database: "data storage",
  server: "infrastructure host",
  cloud: "hosting weather",
  gear: "settings config",
  wrench: "tool fix repair",
  user: "person profile",
  users: "team people group",
  chat: "message talk comment",
  mail: "email letter",
  phone: "call contact",
  megaphone: "announce marketing",
  heart: "love like health",
  star: "favorite rating",
  home: "house start",
  building: "office company",
  map: "travel route",
  pin: "location place",
  globe: "world web international",
  compass: "direction explore",
  leaf: "nature plant green",
  tree: "nature forest",
  flower: "garden nature",
  sun: "day weather summer",
  moon: "night sleep",
  mountain: "outdoor peak",
  flame: "fire hot",
  drop: "water liquid",
  snowflake: "winter cold",
  wave: "sea water",
  camera: "photo picture",
  image: "picture photo",
  music: "song audio",
  film: "movie video",
  gift: "present birthday",
  coffee: "cup break",
  cart: "shopping buy",
  wallet: "money budget finance",
  coin: "money dollar finance",
  key: "access password",
  lock: "secure private",
  shield: "security safe",
  trophy: "award win",
  crown: "king premium",
  anchor: "harbor sea",
  bell: "notification alert",
  link: "url chain",
  bolt: "fast energy power",
  atom: "science physics",
  flask: "lab science experiment",
  cap: "education school learn",
  palette: "design art color",
  brush: "paint art",
  dice: "game random",
  plane: "travel flight",
  car: "travel drive",
  paw: "pet animal",
  fish: "animal sea",
  apple: "food fruit",
  eye: "view watch",
  box: "package product",
  grid: "layout apps",
  table: "collection database spreadsheet",
  scale: "law balance legal",
  cross: "health medical",
};

export const GLYPH_NAMES: readonly string[] = Object.keys(GLYPHS);

export const TINTS = ["violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"] as const;
export type Tint = (typeof TINTS)[number];

export function hasGlyph(name: string): boolean {
  return Object.hasOwn(GLYPHS, name);
}

export function glyphWords(name: string): string {
  return `${name.replace(/-/g, " ")} ${WORDS[name] ?? ""}`;
}

export function glyphSvg(name: string, size = 16): string {
  const body = GLYPHS[name] ?? GLYPHS.page ?? "";
  return (
    `<svg class="icon glyph" viewBox="0 0 24 24" width="${size}" height="${size}" fill="none" stroke="currentColor" ` +
    `stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">${body}</svg>`
  );
}

const cache = new Map<string, SVGSVGElement>();

export function glyphNode(name: string): SVGSVGElement {
  const key = hasGlyph(name) ? name : "page";
  let proto = cache.get(key);
  if (!proto) {
    const tpl = document.createElement("template");
    tpl.innerHTML = glyphSvg(key);
    proto = tpl.content.firstElementChild as SVGSVGElement;
    cache.set(key, proto);
  }
  return proto.cloneNode(true) as SVGSVGElement;
}

export type IconRef = { kind: "glyph"; name: string; tint: string } | { kind: "file"; id: string } | { kind: "none" };

export function parseIcon(ref: string): IconRef {
  if (ref.startsWith("f:")) return { kind: "file", id: ref.slice(2) };
  if (ref.startsWith("i:")) {
    const [, name = "", tint = ""] = ref.split(":");
    return { kind: "glyph", name, tint };
  }
  return { kind: "none" };
}

export function iconRef(name: string, tint = ""): string {
  return tint ? `i:${name}:${tint}` : `i:${name}`;
}
