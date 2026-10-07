import type { Client, MoveTarget, PageChanges } from "./client";
import { compareKeys, keyBetween } from "./order";
import type { Favorite, LiveEvent, Page, TagInfo } from "./types";

export type StoreChange =
  | { kind: "upsert"; page: Page; prev: Page | undefined }
  | { kind: "remove"; pages: Page[] }
  | { kind: "loaded"; workspaceId: string }
  | { kind: "favorites" }
  | { kind: "tags"; workspaceId: string }
  | { kind: "trash"; workspaceId: string };

interface Edit {
  values: PageChanges;
  inflight: PageChanges | null;
  timer: ReturnType<typeof setTimeout> | undefined;
}

export interface TagCount {
  tag: string;
  count: number;
}

const DEBOUNCE_MS = 350;

const slot = (workspaceId: string, parentId: string) => `${workspaceId}:${parentId}`;

export class PageStore {
  private pages = new Map<string, Page>();
  private byParent = new Map<string, Set<string>>();
  private sorted = new Map<string, Page[]>();
  private loadedWs = new Set<string>();
  private loading = new Map<string, Promise<void>>();
  private favs: { id: string; key: string }[] = [];
  private colors = new Map<string, Map<string, string>>();
  private edits = new Map<string, Edit>();
  private listeners = new Set<(c: StoreChange) => void>();

  constructor(
    private readonly api: Client,
    private readonly onError: (e: unknown) => void,
  ) {}

  subscribe(fn: (c: StoreChange) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(c: StoreChange): void {
    for (const fn of [...this.listeners]) {
      try {
        fn(c);
      } catch (e) {
        console.error(e);
      }
    }
  }

  get(id: string): Page | undefined {
    return this.pages.get(id);
  }

  isLoaded(workspaceId: string): boolean {
    return this.loadedWs.has(workspaceId);
  }

  loadedWorkspaces(): string[] {
    return [...this.loadedWs];
  }

  children(workspaceId: string, parentId: string): Page[] {
    const k = slot(workspaceId, parentId);
    let list = this.sorted.get(k);
    if (!list) {
      list = [...(this.byParent.get(k) ?? [])]
        .map((id) => this.pages.get(id))
        .filter((p): p is Page => p !== undefined)
        .sort((a, b) => compareKeys(a.sort_key, b.sort_key) || compareKeys(a.id, b.id));
      this.sorted.set(k, list);
    }
    return list;
  }

  hasChildren(page: Page): boolean {
    return (this.byParent.get(slot(page.workspace_id, page.id))?.size ?? 0) > 0;
  }

  all(workspaceId: string): Page[] {
    const out: Page[] = [];
    for (const p of this.pages.values()) if (p.workspace_id === workspaceId) out.push(p);
    return out;
  }

  ancestors(id: string): Page[] {
    const out: Page[] = [];
    const seen = new Set<string>();
    let p = this.pages.get(id);
    while (p?.parent_id && !seen.has(p.parent_id)) {
      seen.add(p.parent_id);
      const parent = this.pages.get(p.parent_id);
      if (!parent) break;
      out.unshift(parent);
      p = parent;
    }
    return out;
  }

  subtree(id: string): Page[] {
    const root = this.pages.get(id);
    if (!root) return [];
    const out: Page[] = [root];
    for (let i = 0; i < out.length; i++) {
      const cur = out[i];
      if (cur) out.push(...this.children(cur.workspace_id, cur.id));
    }
    return out;
  }

  private index(p: Page): void {
    const k = slot(p.workspace_id, p.parent_id);
    let set = this.byParent.get(k);
    if (!set) {
      set = new Set();
      this.byParent.set(k, set);
    }
    set.add(p.id);
    this.sorted.delete(k);
  }

  private unindex(p: Page): void {
    const k = slot(p.workspace_id, p.parent_id);
    this.byParent.get(k)?.delete(p.id);
    this.sorted.delete(k);
  }

  private withEdits(p: Page): Page {
    const e = this.edits.get(p.id);
    if (!e) return p;
    return { ...p, ...e.inflight, ...e.values };
  }

  private put(next: Page): void {
    const prev = this.pages.get(next.id);
    if (prev) {
      if (prev.parent_id !== next.parent_id || prev.workspace_id !== next.workspace_id || prev.sort_key !== next.sort_key) {
        this.unindex(prev);
        this.index(next);
      }
    } else {
      this.index(next);
    }
    this.pages.set(next.id, next);
    this.emit({ kind: "upsert", page: next, prev });
  }

  upsert(server: Page): void {
    this.put(this.withEdits(server));
  }

  private drop(ids: Iterable<string>): void {
    const gone: Page[] = [];
    for (const id of ids) {
      const p = this.pages.get(id);
      if (!p) continue;
      this.unindex(p);
      this.pages.delete(id);
      this.byParent.delete(slot(p.workspace_id, p.id));
      gone.push(p);
    }
    if (gone.length) this.emit({ kind: "remove", pages: gone });
  }

  seed(workspaceId: string, pages: Page[]): void {
    for (const p of this.all(workspaceId)) {
      this.unindex(p);
      this.pages.delete(p.id);
    }
    for (const p of pages) {
      const live = this.withEdits(p);
      this.pages.set(p.id, live);
      this.index(live);
    }
    this.loadedWs.add(workspaceId);
    this.emit({ kind: "loaded", workspaceId });
  }

  load(workspaceId: string, force = false): Promise<void> {
    if (!force && this.loadedWs.has(workspaceId)) return Promise.resolve();
    const running = this.loading.get(workspaceId);
    if (running) return running;
    const job = this.api.workspaces
      .pages(workspaceId)
      .then((r) => this.seed(workspaceId, r.pages))
      .finally(() => this.loading.delete(workspaceId));
    this.loading.set(workspaceId, job);
    return job;
  }

  forget(workspaceId: string): void {
    this.loadedWs.delete(workspaceId);
    this.drop(this.all(workspaceId).map((p) => p.id));
  }

  async reloadAll(): Promise<void> {
    await Promise.all([...this.loadedWs].map((ws) => this.load(ws, true).catch(this.onError)));
    await this.loadFavorites().catch(this.onError);
  }

  apply(e: LiveEvent): void {
    switch (e.type) {
      case "page":
        if (e.page.trashed_at) this.drop([e.page.id]);
        else if (this.loadedWs.has(e.workspace_id) || this.pages.has(e.page.id) || this.isFavorite(e.page.id)) this.upsert(e.page);
        return;
      case "page_removed":
        this.drop(e.ids);
        return;
      case "workspace":
        if (e.what === "trash") this.emit({ kind: "trash", workspaceId: e.workspace_id });
        else if (e.what === "tags") void this.loadTags(e.workspace_id).catch(this.onError);
        return;
      case "user":
        if (e.what === "favorites") void this.loadFavorites().catch(this.onError);
        return;
      default:
        return;
    }
  }

  favorites(): Page[] {
    const out: Page[] = [];
    for (const f of this.favs) {
      const p = this.pages.get(f.id);
      if (p) out.push(p);
    }
    return out;
  }

  isFavorite(id: string): boolean {
    return this.favs.some((f) => f.id === id);
  }

  seedFavorites(list: Favorite[]): void {
    const next = list.map((f) => ({ id: f.id, key: f.favorite_key })).sort((a, b) => compareKeys(a.key, b.key));
    for (const f of list) {
      const { favorite_key: _, ...page } = f;
      if (!this.pages.has(f.id) || !this.loadedWs.has(f.workspace_id)) this.upsert(page);
    }
    const same = next.length === this.favs.length && next.every((f, i) => f.id === this.favs[i]?.id && f.key === this.favs[i]?.key);
    this.favs = next;
    if (!same) this.emit({ kind: "favorites" });
  }

  async loadFavorites(): Promise<void> {
    const r = await this.api.favorites.list();
    this.seedFavorites(r.favorites);
  }

  async toggleFavorite(id: string): Promise<boolean> {
    const was = this.isFavorite(id);
    const before = this.favs;
    if (was) this.favs = this.favs.filter((f) => f.id !== id);
    else this.favs = [...this.favs, { id, key: keyBetween(this.favs.at(-1)?.key ?? "", "") }];
    this.emit({ kind: "favorites" });
    try {
      if (was) await this.api.favorites.remove(id);
      else await this.api.favorites.add(id);
    } catch (e) {
      this.favs = before;
      this.emit({ kind: "favorites" });
      throw e;
    }
    return !was;
  }

  async moveFavorite(id: string, afterId: string): Promise<void> {
    const rest = this.favs.filter((f) => f.id !== id);
    const i = afterId ? rest.findIndex((f) => f.id === afterId) : -1;
    const prev = i >= 0 ? (rest[i]?.key ?? "") : "";
    const next = rest[i + 1]?.key ?? "";
    const before = this.favs;
    rest.splice(i + 1, 0, { id, key: keyBetween(prev, next) });
    this.favs = rest;
    this.emit({ kind: "favorites" });
    try {
      await this.api.favorites.add(id, afterId ? { after_id: afterId } : { top: true });
    } catch (e) {
      this.favs = before;
      this.emit({ kind: "favorites" });
      throw e;
    }
  }

  tagColor(workspaceId: string, tag: string): string {
    return this.colors.get(workspaceId)?.get(tag.toLowerCase()) ?? "";
  }

  seedTags(workspaceId: string, tags: TagInfo[]): void {
    const m = new Map<string, string>();
    for (const t of tags) if (t.color) m.set(t.tag.toLowerCase(), t.color);
    this.colors.set(workspaceId, m);
    this.emit({ kind: "tags", workspaceId });
  }

  async loadTags(workspaceId: string): Promise<void> {
    const r = await this.api.workspaces.tags(workspaceId);
    this.seedTags(workspaceId, r.tags);
  }

  async setTagColor(workspaceId: string, tag: string, color: string): Promise<void> {
    let m = this.colors.get(workspaceId);
    if (!m) {
      m = new Map();
      this.colors.set(workspaceId, m);
    }
    const before = m.get(tag.toLowerCase());
    if (color) m.set(tag.toLowerCase(), color);
    else m.delete(tag.toLowerCase());
    this.emit({ kind: "tags", workspaceId });
    try {
      await this.api.workspaces.setTagColor(workspaceId, tag, color);
    } catch (e) {
      if (before) m.set(tag.toLowerCase(), before);
      else m.delete(tag.toLowerCase());
      this.emit({ kind: "tags", workspaceId });
      throw e;
    }
  }

  tagCounts(workspaceId: string): TagCount[] {
    const byKey = new Map<string, Map<string, number>>();
    for (const p of this.pages.values()) {
      if (p.workspace_id !== workspaceId) continue;
      for (const t of p.tags) {
        const k = t.toLowerCase();
        const variants = byKey.get(k) ?? new Map<string, number>();
        variants.set(t, (variants.get(t) ?? 0) + 1);
        byKey.set(k, variants);
      }
    }
    const out: TagCount[] = [];
    for (const variants of byKey.values()) {
      let best = "";
      let bestN = 0;
      let total = 0;
      for (const [name, n] of variants) {
        total += n;
        if (n > bestN) {
          best = name;
          bestN = n;
        }
      }
      out.push({ tag: best, count: total });
    }
    return out.sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
  }

  async create(workspaceId: string, parentId = "", afterId = ""): Promise<Page> {
    const body: { workspace_id: string; parent_id?: string; after_id?: string } = { workspace_id: workspaceId };
    if (parentId) body.parent_id = parentId;
    if (afterId) body.after_id = afterId;
    const r = await this.api.pages.create(body);
    this.upsert(r.page);
    return r.page;
  }

  edit(id: string, changes: PageChanges, debounce = true): void {
    const p = this.pages.get(id);
    if (!p) return;
    let e = this.edits.get(id);
    if (!e) {
      e = { values: {}, inflight: null, timer: undefined };
      this.edits.set(id, e);
    }
    Object.assign(e.values, changes);
    this.put({ ...p, ...changes } as Page);
    clearTimeout(e.timer);
    if (debounce) e.timer = setTimeout(() => void this.flush(id), DEBOUNCE_MS);
    else void this.flush(id);
  }

  async flush(id: string): Promise<void> {
    const e = this.edits.get(id);
    if (!e || e.inflight || Object.keys(e.values).length === 0) return;
    clearTimeout(e.timer);
    e.timer = undefined;
    const body = e.values;
    e.inflight = body;
    e.values = {};
    try {
      const r = await this.api.pages.update(id, body);
      e.inflight = null;
      if (Object.keys(e.values).length === 0 && !e.timer) this.edits.delete(id);
      this.upsert(r.page);
      if (Object.keys(e.values).length > 0 && !e.timer) void this.flush(id);
    } catch (err) {
      this.edits.delete(id);
      this.onError(err);
      try {
        const fresh = await this.api.pages.get(id);
        this.upsert(fresh.page);
      } catch {
        return;
      }
    }
  }

  pendingEdits(): boolean {
    return this.edits.size > 0;
  }

  flushAll(): void {
    for (const id of this.edits.keys()) void this.flush(id);
  }

  private keyFor(workspaceId: string, parentId: string, moving: string, beforeId: string, afterId: string): string {
    const sibs = this.children(workspaceId, parentId).filter((p) => p.id !== moving);
    let i = sibs.length;
    if (beforeId) {
      const at = sibs.findIndex((p) => p.id === beforeId);
      if (at >= 0) i = at;
    } else if (afterId) {
      const at = sibs.findIndex((p) => p.id === afterId);
      if (at >= 0) i = at + 1;
    }
    return keyBetween(sibs[i - 1]?.sort_key ?? "", sibs[i]?.sort_key ?? "");
  }

  wouldCycle(id: string, parentId: string): boolean {
    let cur = parentId;
    const seen = new Set<string>();
    while (cur && !seen.has(cur)) {
      if (cur === id) return true;
      seen.add(cur);
      cur = this.pages.get(cur)?.parent_id ?? "";
    }
    return false;
  }

  async move(id: string, target: MoveTarget): Promise<void> {
    const p = this.pages.get(id);
    if (!p) return;
    const ws = target.workspace_id || p.workspace_id;
    const parent = target.parent_id ?? p.parent_id;
    if (ws === p.workspace_id) {
      if (this.wouldCycle(id, parent)) return;
      const key = this.keyFor(ws, parent, id, target.before_id ?? "", target.after_id ?? "");
      this.put({ ...p, parent_id: parent, sort_key: key });
    }
    const body: MoveTarget = { workspace_id: ws, parent_id: parent };
    if (target.before_id) body.before_id = target.before_id;
    if (target.after_id) body.after_id = target.after_id;
    try {
      const r = await this.api.pages.move(id, body);
      if (r.page.workspace_id !== p.workspace_id && !this.loadedWs.has(r.page.workspace_id)) this.drop(this.subtree(id).map((x) => x.id));
      else this.upsert(r.page);
    } catch (e) {
      this.upsert(p);
      throw e;
    }
  }

  async duplicate(id: string, title = ""): Promise<Page> {
    const r = await this.api.pages.duplicate(id, title);
    this.upsert(r.page);
    return r.page;
  }

  async trash(id: string): Promise<Page[]> {
    const gone = this.subtree(id);
    this.drop(gone.map((p) => p.id));
    try {
      await this.api.pages.trash(id);
    } catch (e) {
      for (const p of gone) this.upsert(p);
      throw e;
    }
    return gone;
  }

  async restore(id: string): Promise<Page> {
    const r = await this.api.pages.restore(id);
    if (this.loadedWs.has(r.page.workspace_id)) this.upsert(r.page);
    return r.page;
  }
}
