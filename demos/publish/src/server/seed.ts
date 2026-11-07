import { DOMParser as PMDOMParser, type Node as PMNode } from "prosemirror-model";
import { absolutePositionToRelativePosition, initProseMirrorDoc, prosemirrorToYXmlFragment } from "y-prosemirror";
import * as Y from "yjs";
import { BLOCKS_WITH_ID, ROOT, schema } from "../../../../web/src/engine/doc/schema";
import type { CommentThread, Member, Page, PageComment, User, Workspace } from "../../../../web/src/engine/types";
import { favorites, links, pages, people, serverDoc, shares, sites, state, threads, versions, type Person } from "./db";
import { FILES, fileId } from "./files";
import { pid } from "./ids";

const DAY = 86_400_000;
const NOW = Date.now();

export const WS_ID = pid("ws:field-desk");
export const PAGE = {
  article: pid("page:article"),
  sources: pid("page:sources"),
  checklist: pid("page:checklist"),
  guide: pid("page:guide"),
};

const cast: [string, string, string, string][] = [
  ["ines", "Ines Varga", "ines@fielddesk.example", "violet"],
  ["mara", "Mara Quinn", "mara@fielddesk.example", "teal"],
  ["idris", "Idris Okafor", "idris@fielddesk.example", "amber"],
  ["tomas", "Tomas Lind", "tomas@fielddesk.example", "rose"],
];

for (const [key, name, email, color] of cast) people.push({ id: pid(`user:${key}`), name, email, color });

const who = (key: string): Person => people.find((p) => p.id === pid(`user:${key}`)) as Person;

const um = (key: string) => `<span class="mention" data-kind="user" data-ref="${who(key).id}"><span class="mention__label">${who(key).name}</span></span>`;
const pm = (page: string, label: string) => `<span class="mention" data-kind="page" data-ref="${page}"><span class="mention__label">${label}</span></span>`;

const fig = (name: string, alt: string, caption: string) => {
  const f = FILES.get(fileId(name))!;
  return `<figure class="media media--image"><div class="media__frame"><img src="/files/${f.id}" alt="${alt}" width="${f.w}" height="${f.h}"></div><figcaption class="media__caption">${caption}</figcaption></figure>`;
};
const cols = (...cells: string[]) => `<div class="columns">${cells.map((c) => `<div class="column">${c}</div>`).join("")}</div>`;

interface B {
  k: string;
  html: string;
  at?: Record<number, string | null>;
}

const article: B[] = [
  {
    k: "lede",
    html: `<p>For most of history the Moon was a light. Then, over about four hundred years, it became a place: first a landscape, then a destination, then a doorstep. This is that story in the order the evidence arrived, with the pictures that carried it.</p>`,
    at: { 1: `<p>The Moon used to be a light in the sky. It stopped being only that.</p>`, 2: `<p>For most of history the Moon was a light. Then it became a place. This is how.</p>` },
  },
  {
    k: "byline",
    html: `<p>Words by ${um("ines")}, edited by ${um("mara")}, pictures checked against the NASA records by ${um("idris")}. Every image and source is listed on ${pm(PAGE.sources, "Sources and credits")}.</p>`,
    at: { 1: null, 2: null, 3: `<p>Words by ${um("ines")}, edited by ${um("mara")}.</p>` },
  },
  { k: "toc", html: `<nav class="toc"></nav>`, at: { 1: null, 2: null, 3: null } },
  { k: "h-1610", html: `<h2>1610: a landscape appears</h2>`, at: { 1: `<h2>Galileo</h2>` } },
  {
    k: "galileo",
    html: `<p>In the winter of 1609 to 1610 Galileo Galilei pointed a telescope of about twenty-fold magnification at the Moon and drew what he saw. The line between the lit and the dark side was not smooth. It was ragged, with bright spots in the dark and dark patches in the light, the way hills catch sunrise before the valleys do. He published the drawings in Venice in March 1610 as part of <em>Sidereus Nuncius</em>, the Starry Messenger, and argued that the Moon was rough and mountainous, like the Earth.</p>`,
    at: { 1: `<p>Galileo looked at the Moon through a telescope in 1609 and saw mountains. He drew them.</p>`, 2: `<p>In 1609 Galileo pointed a telescope at the Moon and drew what he saw. The edge of the shadow was ragged, like hills at sunrise. He published the drawings in Venice in 1610.</p>` },
  },
  {
    k: "galileo-figs",
    html: cols(
      fig("galileo-moon-sketch-1", "An engraving of the Moon with a ragged line between light and dark", "The Moon in Galileo's engraving. Sidereus Nuncius, 1610, Project Gutenberg."),
      fig("galileo-moon-sketch-2", "A second engraving of the Moon at another phase", "The same Moon at another phase. Sidereus Nuncius, 1610, Project Gutenberg."),
    ),
    at: { 1: null, 2: null },
  },
  {
    k: "pull",
    html: `<blockquote><p>A smooth Moon was an idea. A rough one was a measurement.</p></blockquote>`,
    at: { 1: null, 2: null },
  },
  {
    k: "jupiter",
    html: `<p>The same winter he turned the telescope on Jupiter and found three, then four, small stars that changed places from night to night and never left the planet. They were moons. Whatever was true of the Earth, it was no longer the only centre of motion in the sky.</p>`,
    at: { 1: null },
  },
  { k: "jupiter-fig", html: fig("galileo-jupiter-moons", "Galileo's diagram of four small stars in a line beside Jupiter", "Galileo's record of Jupiter's moons, night by night. Sidereus Nuncius, 1610."), at: { 1: null, 2: null } },
  {
    k: "tip",
    html: `<div class="doc-callout" data-tone="green"><div class="callout__body"><p><strong>Why this matters for a writer.</strong> Galileo did not argue from authority. He printed what he saw, drew it, and let readers look for themselves. A good published page does the same: the evidence sits next to the claim.</p></div></div>`,
    at: { 1: null, 2: null },
  },
  { k: "h-1968", html: `<h2>1968: the first look back</h2>`, at: { 1: null } },
  {
    k: "apollo8",
    html: `<p>On 24 December 1968 the crew of Apollo 8, the first people to orbit the Moon, watched the Earth come up over the grey lunar horizon and photographed it. The frame, catalogued AS08-14-2383, is often called Earthrise. It was a picture of the Moon that turned out to be a picture of home.</p>`,
    at: { 1: null },
  },
  { k: "apollo8-fig", html: fig("apollo-8-earthrise", "Earth rising above the grey lunar horizon against black space", "AS08-14-2383, 24 December 1968. Credit: NASA."), at: { 1: null, 2: null } },
  { k: "h-1969", html: `<h2>1969: ground under a boot</h2>`, at: { 1: `<h2>Apollo 11</h2>` } },
  {
    k: "apollo11",
    html: `<p>Apollo 11 left Florida on a Saturn V on 16 July 1969 and put two astronauts on the Sea of Tranquility four days later. The picture people remember is of a person, but the one scientists kept returning to is of a bootprint: fine grey soil, pressed sharp, a record of how the surface behaves when it is stepped on.</p>`,
    at: { 1: `<p>Apollo 11 landed in July 1969.</p>` },
  },
  {
    k: "apollo11-figs",
    html: cols(
      fig("apollo-11-saturn-v", "A Saturn V rocket lifting off in a cloud of flame", "Saturn V liftoff, 16 July 1969. Credit: NASA, Marshall Space Flight Center."),
      fig("apollo-11-bootprint", "A bootprint pressed into fine grey lunar soil", "AS11-40-5877, 20 July 1969. Credit: NASA."),
    ),
    at: { 1: null, 2: null, 3: null },
  },
  { k: "h-2026", html: `<h2>2026: Earthset</h2>`, at: { 1: null, 2: null, 3: null } },
  {
    k: "artemis",
    html: `<p>Echoing Earthrise, the Artemis II crew photographed an Earthset as Orion passed behind the Moon's far side during a seven-hour flyby on 6 April 2026. Fifty-eight years separate the two pictures; the horizon in them is the same grey.</p>`,
    at: { 1: null, 2: null, 3: null },
  },
  { k: "artemis-fig", html: fig("artemis-ii-earthset", "Earth setting behind the cratered horizon of the Moon's far side", "art002e021278, 6 April 2026. Credit: NASA."), at: { 1: null, 2: null, 3: null } },
  {
    k: "table",
    html: `<table><tbody><tr><th><p>Mission</p></th><th><p>Date</p></th><th><p>What it showed</p></th><th><p>NASA ID</p></th></tr><tr><td><p>Apollo 8</p></td><td><p>1968-12-24</p></td><td><p>Earth over the lunar horizon</p></td><td><p>AS08-14-2383</p></td></tr><tr><td><p>Apollo 11</p></td><td><p>1969-07-16</p></td><td><p>Saturn V launch</p></td><td><p>8909250</p></td></tr><tr><td><p>Apollo 11</p></td><td><p>1969-07-20</p></td><td><p>Bootprint in lunar soil</p></td><td><p>AS11-40-5877</p></td></tr><tr><td><p>Artemis II</p></td><td><p>2026-04-06</p></td><td><p>Earthset from Orion</p></td><td><p>art002e021278</p></td></tr></tbody></table>`,
    at: { 1: null, 2: null, 3: null },
  },
  { k: "h-you", html: `<h2>Looking for yourself</h2>`, at: { 1: null, 2: null, 3: null, 4: null } },
  {
    k: "you",
    html: `<p>None of this needs a spacecraft. A full Moon in a clear sky shows the dark plains and bright highlands Galileo argued about, and a pair of binoculars on a steady wall shows the edge of the shadow he drew. Pick a night two or three days after the new Moon, when the terminator is long and the relief is sharpest.</p>`,
    at: { 1: null, 2: null, 3: null, 4: null },
  },
  {
    k: "you-figs",
    html: cols(
      fig("moon-full-mono", "A close-up of the full Moon in black and white", "A full Moon in monochrome. Photo: Yilin Mo, Pexels."),
      fig("moon-full-warm", "The full Moon, warm in colour, against a black night sky", "A warmer full Moon. Photo: tommy picone, Pexels."),
    ),
    at: { 1: null, 2: null, 3: null, 4: null },
  },
  { k: "sky-fig", html: fig("telescope-milky-way", "A telescope on a tripod under a sky full of stars", "A small telescope under a dark sky. Photo: Lucas Pezeta, Pexels."), at: { 1: null, 2: null, 3: null, 4: null } },
  {
    k: "bookmark",
    html: `<div class="bookmark"><a class="bookmark__link" href="https://www.nasa.gov/mission/apollo-11/" target="_blank" rel="noopener noreferrer nofollow"><span class="bookmark__text"><span class="bookmark__title">Apollo 11 | NASA</span><span class="bookmark__desc">The agency's mission page for the first crewed lunar landing.</span><span class="bookmark__url">https://www.nasa.gov/mission/apollo-11/</span></span></a></div>`,
    at: { 1: null, 2: null, 3: null, 4: null },
  },
  {
    k: "faq",
    html: `<details><summary>Where do the pictures come from?</summary><p>The engravings are from a public domain edition of the Starry Messenger on Project Gutenberg, the spacecraft pictures are NASA's, and the photographs of the Moon and the telescope are free to use from Pexels. Each one is credited on ${pm(PAGE.sources, "Sources and credits")}.</p></details>`,
    at: { 1: null, 2: null, 3: null, 4: null },
  },
  {
    k: "end",
    html: `<p>The Moon did not change between 1610 and 2026. What changed is how close we could stand to the evidence, and how many people could stand there with us.</p>`,
    at: { 1: null, 2: null },
  },
];

const blockKey = (k: string) => pid(`block:${k}`).slice(0, 12);

function pick(blocks: B[], n: number): string[] {
  const out: string[] = [];
  for (const b of blocks) {
    let html = b.html;
    if (b.at) {
      const keys = Object.keys(b.at).map(Number).filter((x) => x >= n).sort((x, y) => x - y);
      if (keys.length) {
        const v = b.at[keys[0] as number];
        if (v === null || v === undefined) continue;
        html = v;
      }
    }
    out.push(`${b.k}\u0000${html}`);
  }
  return out;
}

function parseBlock(entry: string): PMNode {
  const [k, html] = entry.split("\u0000") as [string, string];
  const dom = new DOMParser().parseFromString(`<body>${html}</body>`, "text/html").body;
  const doc = PMDOMParser.fromSchema(schema).parse(dom, { preserveWhitespace: false });
  const node = doc.firstChild as PMNode;
  if (!BLOCKS_WITH_ID.has(node.type.name)) return node;
  return node.type.create({ ...node.attrs, id: blockKey(k) }, node.content, node.marks);
}

function buildDoc(blocks: B[], n: number): PMNode {
  return schema.nodes.doc!.create(null, pick(blocks, n).map(parseBlock));
}

function fill(ydoc: Y.Doc, doc: PMNode): void {
  ydoc.transact(() => prosemirrorToYXmlFragment(doc, ydoc.getXmlFragment(ROOT)));
}

function b64(u: Uint8Array): string {
  let s = "";
  for (const c of u) s += String.fromCharCode(c);
  return btoa(s);
}

function words(doc: PMNode): number {
  const t = doc.textBetween(0, doc.content.size, " ", " ").trim();
  return t ? t.split(/\s+/).length : 0;
}

function anchor(pageId: string, quote: string): { start: string; end: string } {
  const frag = serverDoc(pageId).getXmlFragment(ROOT);
  const init = initProseMirrorDoc(frag, schema);
  let from = -1;
  init.doc.descendants((node, pos) => {
    if (from >= 0 || !node.isText || !node.text) return from < 0;
    const i = node.text.indexOf(quote);
    if (i >= 0) from = pos + i;
    return from < 0;
  });
  if (from < 0) throw new Error(`comment quote not found: ${quote}`);
  const enc = (p: number) => b64(Y.encodeRelativePosition(absolutePositionToRelativePosition(p, frag, init.mapping as never)));
  return { start: enc(from), end: enc(from + quote.length) };
}

function comment(thread: string, key: string, text: string, ago: number, extra: Partial<PageComment> = {}): PageComment {
  const a = who(key);
  return { id: pid(`comment:${thread}:${text.slice(0, 16)}`), thread_id: thread, body: text, author_id: a.id, author_name: a.name, author_color: a.color, author_avatar: "", created_at: NOW - ago, edited_at: 0, reactions: [], ...extra };
}

function thread(pageId: string, id: string, quote: string, first: PageComment[], resolved = false): CommentThread {
  const a = quote ? anchor(pageId, quote) : { start: "", end: "" };
  const created = first[0]?.created_at ?? NOW;
  const last = first[first.length - 1]?.created_at ?? NOW;
  return {
    id,
    page_id: pageId,
    anchor: quote ? "text" : "page",
    start: a.start,
    end: a.end,
    quote,
    resolved_at: resolved ? last + 1000 : 0,
    resolved_by: resolved ? who("mara").id : "",
    created_by: first[0]?.author_id ?? "",
    created_at: created,
    updated_at: last,
    comments: first,
  };
}

function page(id: string, title: string, o: Partial<Page>): Page {
  return {
    id,
    workspace_id: WS_ID,
    parent_id: "",
    kind: "doc",
    sort_key: "a0",
    title,
    icon: "",
    cover: "",
    description: "",
    tags: [],
    visibility: "workspace",
    owner_id: who("ines").id,
    is_template: false,
    snippet: "",
    thumb: "",
    created_at: NOW - 9 * DAY,
    created_by: who("ines").id,
    updated_at: NOW - 3600_000,
    updated_by: who("ines").id,
    access: "full",
    ...o,
  };
}

const sourcesBlocks: B[] = [
  { k: "s-lede", html: `<p>Everything on ${pm(PAGE.article, "The Moon, in the order we learned it")} that is not original text, with where it came from and the licence its source page states.</p>` },
  { k: "s-h1", html: `<h2>Engravings and text</h2>` },
  {
    k: "s-l1",
    html: `<ul><li><p>Galileo Galilei, <em>Sidereus Nuncius</em> (Venice, 1610), translated by Edward Stafford Carlos (1880). Project Gutenberg ebook 46036. Public domain in the USA.</p></li></ul>`,
  },
  { k: "s-h2", html: `<h2>NASA</h2>` },
  {
    k: "s-l2",
    html: `<ul><li><p>AS08-14-2383, Apollo 8, 24 December 1968.</p></li><li><p>8909250, Saturn Apollo Program, 16 July 1969, Marshall Space Flight Center.</p></li><li><p>AS11-40-5877, Apollo 11, 20 July 1969.</p></li><li><p>art002e021278, Artemis II, 6 April 2026.</p></li></ul><p>NASA content is generally not subject to copyright in the United States, as the agency's media guidelines state.</p>`,
  },
  { k: "s-h3", html: `<h2>Photographs</h2>` },
  {
    k: "s-l3",
    html: `<ul><li><p>Yilin Mo, Close-Up Shot of a Full Moon, Pexels.</p></li><li><p>tommy picone, Full Moon Against Black Night Sky, Pexels.</p></li><li><p>Lucas Pezeta, Sky Filled With Stars at Night, Pexels (also the cover).</p></li></ul><p>Pexels photographs are free to use under the Pexels License; attribution is given anyway.</p>`,
  },
];

const checklistBlocks: B[] = [
  { k: "c-lede", html: `<p>Before the Moon essay goes out. Owner: ${um("ines")}.</p>` },
  {
    k: "c-list",
    html: `<ul class="tasks"><li class="task" data-checked="true"><div class="task__body"><p>${um("mara")} signs off on the Galileo section</p></div></li><li class="task" data-checked="true"><div class="task__body"><p>${um("idris")} checks every caption against the NASA record</p></div></li><li class="task" data-checked="false"><div class="task__body"><p>Publish ${pm(PAGE.sources, "Sources and credits")} with the essay</p></div></li><li class="task" data-checked="false"><div class="task__body"><p>Choose the web address, theme and font</p></div></li><li class="task" data-checked="false"><div class="task__body"><p>Export Markdown for the newsletter and HTML for the archive</p></div></li></ul>`,
  },
];

const guideBlocks: B[] = [
  { k: "g-lede", html: `<p>A short guide to a dark-sky night out: where to go, what to bring and what you will see without any equipment.</p>` },
  { k: "g-h1", html: `<h2>Before you leave</h2>` },
  { k: "g-l1", html: `<ul><li><p>Check the Moon phase: a new Moon gives the darkest sky.</p></li><li><p>Give your eyes twenty minutes to adapt, and use a red light.</p></li><li><p>Bring warm layers and something to sit on.</p></li></ul>` },
  { k: "g-fig", html: fig("cover-milkyway", "A band of the Milky Way over mountains", "The Milky Way over the Kyrgyz mountains. Photo: Daniel Torobekov, Pexels.") },
  { k: "g-h2", html: `<h2>What to look for</h2>` },
  { k: "g-p2", html: `<p>The Milky Way, the Pleiades, the Orion Nebula and the Andromeda Galaxy are all visible to the unaided eye from a dark site.</p>` },
];

function seedDoc(id: string, blocks: B[]): void {
  fill(serverDoc(id), buildDoc(blocks, 99));
}

function snippet(blocks: B[]): string {
  const doc = buildDoc(blocks, 99);
  return doc.textBetween(0, doc.content.size, " ", " ").slice(0, 160);
}

export function seed(): void {
  const me = who("ines");
  const user: User = {
    id: me.id,
    email: me.email,
    name: me.name,
    color: me.color,
    avatar: "",
    role: "owner",
    email_verified: true,
    disabled: false,
    totp_enabled: false,
    prefs: { lang: "en", region: "", time_zone: "", hour_cycle: "", week_start: "", notes_mode: false, system_font: false },
    created_at: NOW - 40 * DAY,
  };
  const ws: Workspace = {
    id: WS_ID,
    name: "Field Desk",
    icon: "",
    image: "",
    role: "admin",
    personal: false,
    auto_join: false,
    sidebar_mode: "tree",
    agent_access: "closed",
    has_rules: false,
    proposal_waiting: false,
    member_count: people.length,
    owner_id: me.id,
    rules_updated_at: 0,
    ai_enabled: false,
  };
  state.me = user;
  state.workspace = ws;
  state.members = people.map((p): Member => ({ user_id: p.id, name: p.name, email: p.email, color: p.color, avatar: "", role: p.id === me.id ? "admin" : "member", joined_at: NOW - 30 * DAY, disabled: false }));

  seedDoc(PAGE.article, article);
  seedDoc(PAGE.sources, sourcesBlocks);
  seedDoc(PAGE.checklist, checklistBlocks);
  seedDoc(PAGE.guide, guideBlocks);

  pages.push(
    page(PAGE.article, "The Moon, in the order we learned it", {
      icon: "i:moon",
      cover: `f:${fileId("cover-telescope")}`,
      description: "A field essay on how four centuries of looking turned the Moon from a light into a place, from Galileo's engravings to Artemis II.",
      sort_key: "a0",
      snippet: snippet(article),
      tags: ["essay"],
    }),
    page(PAGE.sources, "Sources and credits", { parent_id: PAGE.article, icon: "i:book", cover: `f:${fileId("cover-earthset")}`, sort_key: "a0", snippet: snippet(sourcesBlocks), updated_at: NOW - 2 * 3600_000 }),
    page(PAGE.checklist, "Publishing checklist", { icon: "i:checklist", cover: "g:slate", sort_key: "a1", visibility: "private", snippet: snippet(checklistBlocks), updated_at: NOW - DAY }),
    page(PAGE.guide, "Dark-sky night guide", { icon: "i:star", cover: `f:${fileId("cover-tripod")}`, sort_key: "a2", snippet: snippet(guideBlocks), updated_at: NOW - 5 * DAY, tags: ["guide"] }),
  );
  favorites.push(PAGE.article);

  const stamps: [number, "auto" | "manual", string[], number][] = [
    [6, "auto", ["ines"], 1],
    [4, "auto", ["ines"], 2],
    [3, "auto", ["ines", "mara"], 3],
    [2, "auto", ["idris", "ines"], 4],
    [1, "manual", ["ines"], 5],
  ];
  const vs = stamps.map(([ago, kind, names, n]) => {
    const doc = buildDoc(article, n);
    const tmp = new Y.Doc();
    fill(tmp, doc);
    const state64 = b64(Y.encodeStateAsUpdate(tmp));
    tmp.destroy();
    return {
      meta: { id: pid(`version:${n}`), page_id: PAGE.article, created_at: NOW - ago * DAY - 3 * 3600_000, kind, title: n < 3 ? "The Moon" : "The Moon, in the order we learned it", editors: names.map((k) => who(k).id), names: names.map((k) => who(k).name), words: words(doc), size: state64.length },
      state: state64,
    };
  });
  versions.set(PAGE.article, vs.reverse());

  const t1 = pid("thread:1");
  const t2 = pid("thread:2");
  const t3 = pid("thread:3");
  const t4 = pid("thread:4");
  const t5 = pid("thread:5");
  threads.set(PAGE.article, [
    thread(
      PAGE.article,
      t1,
      "about twenty-fold magnification",
      [
        comment(t1, "mara", "Do we want to say about twenty-fold? The Starry Messenger gives the figure, so I would cite it on the sources page.", 2 * DAY),
        comment(t1, "ines", "Good catch, it is listed under Engravings and text now.", 2 * DAY - 3600_000),
      ],
      true,
    ),
    thread(PAGE.article, t2, "often called Earthrise", [
      comment(t2, "idris", "Put the NASA ID in the caption so readers can find the original record.", DAY + 5 * 3600_000),
      comment(t2, "ines", "Done, it is in the caption and in the table at the end of the section.", DAY + 4 * 3600_000, { reactions: [{ icon: "heart", count: 2, mine: false, names: ["Idris Okafor", "Mara Quinn"] }] }),
    ]),
    thread(PAGE.article, t3, "2026: Earthset", [comment(t3, "tomas", `Could we show Apollo 8 and Artemis II side by side? The echo is the whole point. @[${me.name}](user:${me.id}) what do you think?`, 20 * 3600_000)]),
    thread(PAGE.article, t4, "Why this matters for a writer.", [comment(t4, "idris", "I like this one. Keep it short.", 8 * 3600_000)]),
    thread(PAGE.article, t5, "Every image and source is listed on", [comment(t5, "mara", `Can this go out Thursday morning? The sources page has to go live with it. @[${who("ines").name}](user:${me.id})`, 4 * 3600_000)]),
  ]);

  const published = pid("site:guide");
  sites.set(PAGE.guide, {
    id: published,
    workspace_id: WS_ID,
    page_id: PAGE.guide,
    page_title: "Dark-sky night guide",
    slug: "dark-sky-guide",
    hostname: "",
    title: "",
    description: "",
    image: "",
    theme: "auto",
    font: "serif",
    password: false,
    indexing: true,
    subpages: true,
    created_by: me.id,
    created_at: NOW - 4 * DAY,
    updated_at: NOW - 4 * DAY,
    path: "/s/dark-sky-guide",
  });

  shares.set(PAGE.article, [
    { id: pid("share:mara"), page_id: PAGE.article, kind: "user", subject_id: who("mara").id, name: who("mara").name, email: who("mara").email, level: "edit", expires_at: 0 },
    { id: pid("share:idris"), page_id: PAGE.article, kind: "user", subject_id: who("idris").id, name: who("idris").name, email: who("idris").email, level: "comment", expires_at: 0 },
  ]);
  links.set(PAGE.article, []);
}
