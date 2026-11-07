import { yXmlFragmentToProseMirrorRootNode } from "y-prosemirror";
import * as Y from "yjs";
import { ROOT, schema } from "../../../../web/src/engine/doc/schema";
import type { CommentListing, CommentThread, Crumb, Favorite, Me, Page, PageDetail, PageVersion, Share, ShareLink, Site, SiteInput, SiteStatus } from "../../../../web/src/engine/types";
import { pushLive } from "../platform";
import { dropDoc, favorites, links, pages, passwords, people, person, serverDoc, shares, sites, state, threads, versions, visits, type Version } from "./db";
import { fresh, pid } from "./ids";
import { WS_ID } from "./seed";

type Body = Record<string, unknown>;

declare global {
  interface Window {
    __unhandled?: string[];
  }
}

const ok = (data: unknown, status = 200): Response => new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json" } });
const none = (): Response => new Response(null, { status: 204 });
const fail = (status: number, code: string, message: string): Response => ok({ error: { code, message } }, status);

let seq = 1;

function b64(u: Uint8Array): string {
  let s = "";
  for (const c of u) s += String.fromCharCode(c);
  return btoa(s);
}

function me(): Me {
  return {
    user: state.me!,
    credential: { kind: "session", scope: "write" },
    instance: { name: "form.md" },
    workspaces: [state.workspace!],
    withheld_workspaces: 0,
    can_create_workspaces: false,
  };
}

function find(id: string): Page | undefined {
  return pages.find((p) => p.id === id);
}

function path(p: Page): Crumb[] {
  const out: Crumb[] = [];
  for (let c: Page | undefined = p; c; c = c.parent_id ? find(c.parent_id) : undefined) out.unshift({ id: c.id, title: c.title, icon: c.icon });
  return out;
}

function detail(p: Page): PageDetail {
  return { ...p, path: path(p), can_write: true, level: "full", can_edit: true, can_comment: true, can_share: true };
}

function emitPage(p: Page): void {
  pushLive({ type: "page", seq: seq++, workspace_id: WS_ID, page: p });
}

function emitComments(pageId: string): void {
  pushLive({ type: "comments", seq: seq++, workspace_id: WS_ID, page_id: pageId });
}

export function docWords(id: string): number {
  const doc = yXmlFragmentToProseMirrorRootNode(serverDoc(id).getXmlFragment(ROOT), schema);
  const t = doc.textBetween(0, doc.content.size, " ", " ").trim();
  return t ? t.split(/\s+/).length : 0;
}

export function docText(id: string): string {
  const doc = yXmlFragmentToProseMirrorRootNode(serverDoc(id).getXmlFragment(ROOT), schema);
  return doc.textBetween(0, doc.content.size, " ", " ").trim();
}

function listing(pageId: string): CommentListing {
  return { threads: threads.get(pageId) ?? [], suggestions: [], can_comment: true, can_moderate: true, can_decide: true };
}

function threadOf(id: string): CommentThread | undefined {
  for (const list of threads.values()) {
    const t = list.find((x) => x.id === id);
    if (t) return t;
  }
  return undefined;
}

function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
}

function snapshot(pageId: string, kind: PageVersion["kind"]): Version {
  const p = find(pageId)!;
  const st = b64(Y.encodeStateAsUpdate(serverDoc(pageId)));
  const v: Version = {
    meta: { id: fresh("version"), page_id: pageId, created_at: Date.now(), kind, title: p.title, editors: [state.me!.id], names: [state.me!.name], words: docWords(pageId), size: st.length },
    state: st,
  };
  const list = versions.get(pageId) ?? [];
  list.unshift(v);
  versions.set(pageId, list);
  return v;
}

export function siteFor(pageId: string): Site | undefined {
  return sites.get(pageId);
}

function siteStatus(p: Page): SiteStatus {
  return { site: sites.get(p.id) ?? null, can_publish: true, allowed: true, can_host: false, suggested_slug: slugify(p.title) || "page" };
}

function publish(p: Page, input: SiteInput): Response {
  const existing = sites.get(p.id);
  const slug = (input.slug ?? existing?.slug ?? slugify(p.title)).trim().toLowerCase();
  if (!/^[a-z0-9][a-z0-9-]{1,58}$/.test(slug)) return fail(400, "invalid_slug", "The address uses lowercase letters, numbers and dashes, at least two characters.");
  for (const [pid2, s] of sites) if (pid2 !== p.id && s.slug === slug) return fail(409, "slug_taken", "That address is already used by another page.");
  if (input.password) passwords.set(p.id, input.password);
  else if (input.password === "") passwords.delete(p.id);
  const now = Date.now();
  const site: Site = {
    id: existing?.id ?? fresh("site"),
    workspace_id: WS_ID,
    page_id: p.id,
    page_title: p.title,
    slug,
    hostname: "",
    title: input.title ?? existing?.title ?? "",
    description: input.description ?? existing?.description ?? "",
    image: input.image ?? existing?.image ?? "",
    theme: input.theme ?? existing?.theme ?? "auto",
    font: input.font ?? existing?.font ?? "sans",
    password: passwords.has(p.id),
    indexing: passwords.has(p.id) ? false : (input.indexing ?? existing?.indexing ?? false),
    subpages: input.subpages ?? existing?.subpages ?? true,
    created_by: existing?.created_by ?? state.me!.id,
    created_at: existing?.created_at ?? now,
    updated_at: now,
    path: `/s/${slug}`,
  };
  sites.set(p.id, site);
  window.dispatchEvent(new Event("demo:site"));
  return ok({ site });
}

function searchHits(q: string): unknown {
  const needle = q.trim().toLowerCase();
  const hits = pages
    .filter((p) => !p.trashed_at && needle && (p.title.toLowerCase().includes(needle) || docText(p.id).toLowerCase().includes(needle)))
    .map((p) => {
      const inTitle = p.title.toLowerCase().includes(needle);
      const text = docText(p.id);
      const at = text.toLowerCase().indexOf(needle);
      return {
        id: p.id,
        title: p.title,
        title_hl: p.title,
        icon: p.icon,
        kind: p.kind,
        row: false,
        path: path(p).slice(0, -1),
        match: inTitle ? "title" : "body",
        heading: "",
        snippet: inTitle || at < 0 ? p.snippet : text.slice(Math.max(0, at - 40), at + 120),
        created_at: p.created_at,
        created_by: p.created_by,
        updated_at: p.updated_at,
      };
    });
  return { hits, more: false, took_ms: 1 };
}

function reactionToggle(c: { reactions: { icon: string; count: number; mine: boolean; names: string[] }[] }, icon: string, on: boolean): void {
  let r = c.reactions.find((x) => x.icon === icon);
  if (!r && on) {
    r = { icon, count: 0, mine: false, names: [] };
    c.reactions.push(r);
  }
  if (!r) return;
  if (on && !r.mine) {
    r.mine = true;
    r.count++;
    r.names.push(state.me!.name);
  } else if (!on && r.mine) {
    r.mine = false;
    r.count--;
    r.names = r.names.filter((n) => n !== state.me!.name);
  }
  c.reactions = c.reactions.filter((x) => x.count > 0);
}

function handle(method: string, url: URL, body: Body): Response {
  const p = url.pathname;
  let m: RegExpExecArray | null;

  if (p === "/healthz") return ok({ status: "ok", version: "demo", storage: "memory" });
  if (p === "/api/me" && method === "GET") return ok(me());
  if (p === "/api/me/notifications") return ok({ email: "mentions", push_key: "", subscriptions: 0, mail_enabled: false });
  if (p === "/api/inbox/count") return ok({ unread: 0 });
  if (p === "/api/inbox") return ok({ items: [], unread: 0 });
  if (p === "/api/inbox/mark") return none();
  if (p === "/api/favorites" && method === "GET") {
    const list = favorites.map((id) => find(id)).filter((x): x is Page => !!x).map((x): Favorite => ({ ...x, favorite_key: `fav-${x.id}` }));
    return ok({ favorites: list });
  }
  if ((m = /^\/api\/favorites\/([^/]+)$/.exec(p))) {
    const id = m[1]!;
    const at = favorites.indexOf(id);
    if (method === "PUT" && at < 0) favorites.push(id);
    if (method === "DELETE" && at >= 0) favorites.splice(at, 1);
    return none();
  }
  if (p === "/api/recent") return ok({ pages: pages.filter((x) => !x.trashed_at).map((x) => ({ ...x, visited_at: visits.get(x.id) ?? x.updated_at })) });
  if (p === "/api/search") return ok(searchHits(url.searchParams.get("q") ?? ""));
  if ((m = /^\/api\/workspaces\/([^/]+)\/(pages|tags|teamspaces|members|groups|trash|invites)$/.exec(p))) {
    switch (m[2]) {
      case "pages":
        return ok({ pages: pages.filter((x) => !x.trashed_at) });
      case "tags":
        return ok({ tags: [] });
      case "teamspaces":
        return ok({ teamspaces: [] });
      case "members":
        return ok({ members: state.members });
      case "groups":
        return ok({ groups: [] });
      case "trash":
        return ok({ items: [] });
      default:
        return ok({ invites: [] });
    }
  }
  if (p === "/api/pages" && method === "POST") {
    const now = Date.now();
    const id = fresh("page");
    const page: Page = {
      id,
      workspace_id: WS_ID,
      parent_id: String(body.parent_id ?? ""),
      kind: "doc",
      sort_key: `b${now}`,
      title: String(body.title ?? ""),
      icon: String(body.icon ?? ""),
      cover: "",
      description: "",
      tags: [],
      visibility: "workspace",
      owner_id: state.me!.id,
      is_template: false,
      snippet: "",
      thumb: "",
      created_at: now,
      created_by: state.me!.id,
      updated_at: now,
      updated_by: state.me!.id,
      access: "full",
    };
    pages.push(page);
    emitPage(page);
    return ok({ page }, 201);
  }
  if ((m = /^\/api\/pages\/([^/]+)$/.exec(p))) {
    const page = find(m[1]!);
    if (!page) return fail(404, "not_found", "That page does not exist.");
    if (method === "GET") return ok({ page: detail(page) });
    if (method === "PATCH") {
      for (const k of ["title", "icon", "cover", "description", "tags", "visibility", "style"] as const) if (k in body) (page as unknown as Body)[k] = body[k];
      page.updated_at = Date.now();
      const s = sites.get(page.id);
      if (s && "title" in body) s.page_title = page.title;
      return ok({ page });
    }
    if (method === "DELETE") {
      page.trashed_at = Date.now();
      return ok({ trashed: [page.id] });
    }
  }
  if ((m = /^\/api\/pages\/([^/]+)\/(visit|move|duplicate|restore|backlinks|analytics)$/.exec(p))) {
    visits.set(m[1]!, Date.now());
    if (m[2] === "backlinks") return ok({ pages: [] });
    return m[2] === "visit" ? none() : ok({ page: find(m[1]!) });
  }

  if ((m = /^\/api\/pages\/([^/]+)\/comments$/.exec(p))) {
    const id = m[1]!;
    if (method === "GET") return ok(listing(id));
    const tid = fresh("thread");
    const c = { id: fresh("comment"), thread_id: tid, body: String(body.body ?? ""), author_id: state.me!.id, author_name: state.me!.name, author_color: state.me!.color, author_avatar: "", created_at: Date.now(), edited_at: 0, reactions: [] };
    const th: CommentThread = {
      id: tid,
      page_id: id,
      anchor: body.anchor as CommentThread["anchor"],
      start: String(body.start ?? ""),
      end: String(body.end ?? ""),
      quote: String(body.quote ?? ""),
      resolved_at: 0,
      resolved_by: "",
      created_by: state.me!.id,
      created_at: c.created_at,
      updated_at: c.created_at,
      comments: [c],
    };
    const list = threads.get(id) ?? [];
    list.push(th);
    threads.set(id, list);
    emitComments(id);
    return ok({ thread: th }, 201);
  }
  if ((m = /^\/api\/threads\/([^/]+)\/comments$/.exec(p))) {
    const th = threadOf(m[1]!);
    if (!th) return fail(404, "not_found", "That thread does not exist.");
    const c = { id: fresh("comment"), thread_id: th.id, body: String(body.body ?? ""), author_id: state.me!.id, author_name: state.me!.name, author_color: state.me!.color, author_avatar: "", created_at: Date.now(), edited_at: 0, reactions: [] };
    th.comments.push(c);
    th.updated_at = c.created_at;
    emitComments(th.page_id);
    return ok({ comment: c }, 201);
  }
  if ((m = /^\/api\/threads\/([^/]+)\/resolved$/.exec(p))) {
    const th = threadOf(m[1]!);
    if (!th) return fail(404, "not_found", "That thread does not exist.");
    const on = body.resolved === true;
    th.resolved_at = on ? Date.now() : 0;
    th.resolved_by = on ? state.me!.id : "";
    emitComments(th.page_id);
    return none();
  }
  if ((m = /^\/api\/comments\/([^/]+)(?:\/reactions\/([^/]+))?$/.exec(p))) {
    const id = m[1]!;
    for (const [pageId, list] of threads) {
      for (const th of list) {
        const at = th.comments.findIndex((c) => c.id === id);
        if (at < 0) continue;
        const c = th.comments[at]!;
        if (m[2]) reactionToggle(c, decodeURIComponent(m[2]), method === "PUT");
        else if (method === "PATCH") {
          c.body = String(body.body ?? c.body);
          c.edited_at = Date.now();
        } else if (method === "DELETE") {
          th.comments.splice(at, 1);
          if (!th.comments.length) list.splice(list.indexOf(th), 1);
        }
        emitComments(pageId);
        return none();
      }
    }
    return fail(404, "not_found", "That comment does not exist.");
  }

  if ((m = /^\/api\/pages\/([^/]+)\/versions$/.exec(p))) {
    const id = m[1]!;
    if (method === "GET") return ok({ versions: (versions.get(id) ?? []).map((v) => v.meta) });
    const v = snapshot(id, url.searchParams.get("kind") === "restore" ? "restore" : "manual");
    return ok({ id: v.meta.id }, 201);
  }
  if ((m = /^\/api\/versions\/([^/]+)(\/restored)?$/.exec(p))) {
    if (m[2]) return none();
    for (const list of versions.values()) {
      const v = list.find((x) => x.meta.id === m![1]);
      if (v) return ok({ version: v.meta, state: v.state });
    }
    return fail(404, "not_found", "That version does not exist.");
  }

  if ((m = /^\/api\/pages\/([^/]+)\/site$/.exec(p))) {
    const page = find(m[1]!);
    if (!page) return fail(404, "not_found", "That page does not exist.");
    if (method === "GET") return ok(siteStatus(page));
    if (method === "PUT") return publish(page, body as SiteInput);
    sites.delete(page.id);
    passwords.delete(page.id);
    window.dispatchEvent(new Event("demo:site"));
    return none();
  }
  if ((m = /^\/api\/workspaces\/([^/]+)\/sites$/.exec(p))) return ok({ sites: [...sites.values()], publishing: true, can_manage: true });

  if ((m = /^\/api\/pages\/([^/]+)\/shares$/.exec(p))) {
    const id = m[1]!;
    if (method === "GET") return ok({ general: "workspace", shares: shares.get(id) ?? [], links: links.get(id) ?? [], invites: [], can_share: true });
    const who = people.find((x) => x.email === String(body.email ?? "").toLowerCase()) ?? (body.subject_id ? person(String(body.subject_id)) : undefined);
    if (!who) return fail(404, "not_found", "No one in this workspace has that email.");
    const list = shares.get(id) ?? [];
    const share: Share = { id: fresh("share"), page_id: id, kind: "user", subject_id: who.id, name: who.name, email: who.email, level: String(body.level ?? "edit") as Share["level"], expires_at: Number(body.expires_at ?? 0) };
    shares.set(id, [...list.filter((s) => s.subject_id !== who.id), share]);
    return ok({ share, emailed: false });
  }
  if ((m = /^\/api\/pages\/([^/]+)\/links$/.exec(p))) {
    const id = m[1]!;
    const link: ShareLink = { id: fresh("link"), page_id: id, level: body.level === "comment" ? "comment" : "view", password: !!body.password, expires_at: Number(body.expires_at ?? 0), created_at: Date.now(), token: pid(`token:${Date.now()}`) };
    links.set(id, [...(links.get(id) ?? []), link]);
    return ok({ link }, 201);
  }
  if ((m = /^\/api\/(shares|links)\/([^/]+)$/.exec(p))) {
    const table = m[1] === "shares" ? shares : links;
    for (const [pageId, list] of table as Map<string, { id: string; level?: string }[]>) {
      const at = list.findIndex((x) => x.id === m![2]);
      if (at < 0) continue;
      if (method === "DELETE") list.splice(at, 1);
      else if (body.level) list[at]!.level = String(body.level);
      table.set(pageId, list as never);
    }
    return none();
  }

  if (p === "/api/commands/result") return none();
  if (method === "GET") return ok({});
  return none();
}

export function demoFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
  const url = new URL(raw, "https://demo.invalid");
  const method = (init?.method ?? "GET").toUpperCase();
  let body: Body = {};
  if (typeof init?.body === "string") {
    try {
      body = JSON.parse(init.body) as Body;
    } catch {
      body = {};
    }
  }
  const res = handle(method, url, body);
  if (res.status === 204 && method === "GET") (window.__unhandled ??= []).push(`${method} ${url.pathname}`);
  return new Promise((resolve) => setTimeout(() => resolve(res), 0));
}

export function reset(): void {
  for (const p of pages) dropDoc(p.id);
}
