import type { Client } from "./client";
import type {
  Collection,
  CollectionView,
  ComputedValue,
  GroupResult,
  LaneResult,
  LiveEvent,
  Member,
  Property,
  QueryInput,
  QueryResult,
  RowData,
  RowRef,
  SchemaOp,
  ViewConfig,
  ViewType,
} from "./types";

export type SessionChange = { kind: "rows"; ids: string[] } | { kind: "reset" } | { kind: "loading"; on: boolean };

export interface SessionOptions {
  limit?: number;
  groupLimit?: number;
  search?: string;
  range?: QueryInput["range"];
  noGroups?: boolean;
  filter?: QueryInput["filter"];
  ids?: string[];
  watch?: () => void;
}

const PAGE = 100;

export class QuerySession {
  readonly rows = new Map<string, RowData>();
  readonly refs = new Map<string, RowRef>();
  slots: (string | undefined)[] = [];
  total = 0;
  groups: GroupResult[] | undefined;
  lanes: LaneResult[] | undefined;
  calcs: Record<string, ComputedValue | null> | undefined;
  loaded = false;
  epoch = 0;
  private listeners = new Set<(c: SessionChange) => void>();
  private seq = 0;
  private windows = new Map<number, AbortController>();
  private visible: [number, number] = [0, PAGE];
  private pumpTimer: ReturnType<typeof setTimeout> | undefined;
  private pumpSince = 0;
  private refreshTimer: ReturnType<typeof setTimeout> | undefined;
  private dirty = new Set<string>();
  private dirtyAll = false;
  private extraGroups = new Map<string, number>();
  closed = false;

  constructor(
    private readonly store: CollectionStore,
    readonly collectionId: string,
    public viewId: string,
    public options: SessionOptions = {},
  ) {}

  on(fn: (c: SessionChange) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  emit(c: SessionChange): void {
    for (const fn of this.listeners) fn(c);
  }

  close(): void {
    this.closed = true;
    clearTimeout(this.refreshTimer);
    clearTimeout(this.pumpTimer);
    this.dropWindows();
    this.listeners.clear();
    this.store.release(this);
  }

  private input(offset: number, limit: number): QueryInput {
    const o = this.options;
    const input: QueryInput = { view_id: this.viewId, offset, limit };
    if (o.groupLimit) input.group_limit = o.groupLimit;
    if (o.search) input.search = o.search;
    if (o.range) input.range = o.range;
    if (o.noGroups) input.no_groups = true;
    if (o.filter) input.filter = o.filter;
    if (o.ids) input.ids = o.ids;
    return input;
  }

  private absorb(r: QueryResult): void {
    for (const row of r.rows) this.rows.set(row.id, row);
    for (const [k, v] of Object.entries(r.refs)) this.refs.set(k, v);
  }

  seed(r: QueryResult): void {
    this.absorb(r);
    this.apply(r, 0);
    this.loaded = true;
  }

  private apply(r: QueryResult, offset: number): void {
    this.total = r.total;
    this.groups = r.groups;
    this.lanes = r.lanes;
    this.calcs = r.calcs;
    this.epoch = r.epoch;
    if (!r.groups && !r.lanes) {
      if (this.slots.length !== r.total) this.slots.length = r.total;
      r.order.forEach((id, i) => {
        this.slots[offset + i] = id;
      });
    } else {
      this.slots = [];
    }
  }

  async load(): Promise<void> {
    const mine = ++this.seq;
    this.emit({ kind: "loading", on: true });
    try {
      const limit = this.options.limit ?? PAGE;
      const r = await this.store.api.collections.query(this.collectionId, this.input(0, limit));
      if (mine !== this.seq || this.closed) return;
      this.absorb(r);
      this.slots = [];
      this.apply(r, 0);
      this.dropWindows();
      this.extraGroups.clear();
      this.loaded = true;
      this.emit({ kind: "reset" });
    } finally {
      if (mine === this.seq) this.emit({ kind: "loading", on: false });
    }
  }

  setVisible(start: number, end: number): void {
    const moved = start !== this.visible[0] || end !== this.visible[1];
    this.visible = [start, end];
    for (const [off, ctl] of this.windows) {
      if (off + PAGE > start - PAGE && off < end + PAGE) continue;
      ctl.abort();
      this.windows.delete(off);
    }
    if (!moved && this.pumpTimer === undefined) return this.pump();
    const now = performance.now();
    if (this.pumpTimer === undefined) this.pumpSince = now;
    else if (now - this.pumpSince < 220) clearTimeout(this.pumpTimer);
    else return;
    this.pumpTimer = setTimeout(() => {
      this.pumpTimer = undefined;
      this.pump();
    }, 70);
  }

  private pump(): void {
    if (this.closed) return;
    const [start, end] = this.visible;
    for (let off = Math.floor(start / PAGE) * PAGE; off < end && off < this.total && this.windows.size < 3; off += PAGE) {
      if (this.windows.has(off)) continue;
      for (let i = off; i < Math.min(off + PAGE, this.total); i++) {
        if (this.slots[i] !== undefined) continue;
        this.window(off);
        break;
      }
    }
  }

  private dropWindows(): void {
    for (const ctl of this.windows.values()) ctl.abort();
    this.windows.clear();
  }

  private window(offset: number): void {
    const mine = this.seq;
    const ctl = new AbortController();
    this.windows.set(offset, ctl);
    this.store.api.collections
      .query(this.collectionId, this.input(offset, PAGE), ctl.signal)
      .then(
        (r) => {
          if (this.windows.get(offset) === ctl) this.windows.delete(offset);
          if (mine !== this.seq || this.closed || ctl.signal.aborted || !r) return;
          this.absorb(r);
          this.apply(r, offset);
          this.emit({ kind: "reset" });
          this.pump();
        },
        (e) => {
          if (this.windows.get(offset) === ctl) this.windows.delete(offset);
          if (!ctl.signal.aborted) this.store.onError(e);
        },
      );
  }

  async moreInGroup(groupKey: string, laneKey?: string): Promise<void> {
    const k = `${laneKey ?? ""}|${groupKey}`;
    const have = this.extraGroups.get(k) ?? 0;
    const shown = this.groupRows(groupKey, laneKey).length;
    const input = this.input(Math.max(have, shown), PAGE);
    input.group_key = groupKey;
    if (laneKey !== undefined) input.lane_key = laneKey;
    const r = await this.store.api.collections.query(this.collectionId, input);
    if (this.closed) return;
    this.absorb(r);
    const target = this.findGroup(groupKey, laneKey);
    const fresh = laneKey !== undefined ? r.lanes?.[0]?.groups.find((g) => g.key === groupKey) : r.groups?.find((g) => g.key === groupKey);
    if (target && fresh) {
      const seen = new Set(target.rows);
      for (const id of fresh.rows) if (!seen.has(id)) target.rows.push(id);
      target.count = fresh.count;
    }
    this.extraGroups.set(k, target?.rows.length ?? 0);
    this.emit({ kind: "reset" });
  }

  findGroup(key: string, laneKey?: string): GroupResult | undefined {
    if (laneKey !== undefined) return this.lanes?.find((l) => l.key === laneKey)?.groups.find((g) => g.key === key);
    return this.groups?.find((g) => g.key === key);
  }

  groupRows(key: string, laneKey?: string): string[] {
    return this.findGroup(key, laneKey)?.rows ?? [];
  }

  scheduleRefresh(ids: string[]): void {
    if (this.closed) return;
    if (ids.length) for (const id of ids) this.dirty.add(id);
    else this.dirtyAll = true;
    clearTimeout(this.refreshTimer);
    this.refreshTimer = setTimeout(() => {
      if (this.options.watch) {
        this.dirty.clear();
        this.dirtyAll = false;
        this.options.watch();
        return;
      }
      const all = this.dirtyAll;
      const list = [...this.dirty];
      const mine = this.seq + 1;
      void this.refresh(all ? [] : list)
        .then(() => {
          if (this.seq !== mine) return;
          if (all) this.dirtyAll = false;
          for (const id of list) this.dirty.delete(id);
        })
        .catch((e) => this.store.onError(e));
    }, 120);
  }

  async refresh(changed: string[] = []): Promise<void> {
    const mine = ++this.seq;
    if (this.groups || this.lanes) {
      const r = await this.store.api.collections.query(this.collectionId, this.input(0, this.options.limit ?? PAGE));
      if (mine !== this.seq || this.closed) return;
      const keep = new Map<string, number>();
      const visit = (gs: GroupResult[] | undefined, lane = "") => {
        for (const g of gs ?? []) keep.set(`${lane}|${g.key}`, g.rows.length);
      };
      visit(this.groups);
      for (const l of this.lanes ?? []) visit(l.groups, l.key);
      this.absorb(r);
      this.apply(r, 0);
      this.extraGroups.clear();
      this.emit({ kind: "reset" });
      for (const [k, n] of keep) {
        const [lane, key] = k.split("|") as [string, string];
        const g = this.findGroup(key, lane === "" && !this.lanes ? undefined : lane);
        if (g && n > g.rows.length && g.count > g.rows.length) void this.moreInGroup(key, lane === "" && !this.lanes ? undefined : lane);
      }
      return;
    }
    const [start, end] = this.visible;
    const off = Math.max(0, Math.floor(start / PAGE) * PAGE);
    const lim = Math.min(1000, Math.max(PAGE, Math.ceil((end - off) / PAGE) * PAGE));
    const r = await this.store.api.collections.query(this.collectionId, this.input(off, lim));
    if (mine !== this.seq || this.closed) return;
    const lengthChanged = r.total !== this.total;
    const before = this.slots.slice(off, off + lim).join(",");
    this.absorb(r);
    if (lengthChanged) {
      this.slots = [];
      this.slots.length = r.total;
    } else {
      for (let i = 0; i < this.slots.length; i++) if (i < off || i >= off + lim) this.slots[i] = undefined;
    }
    this.apply(r, off);
    this.dropWindows();
    const after = this.slots.slice(off, off + lim).join(",");
    if (lengthChanged || before !== after || this.calcs) this.emit({ kind: "reset" });
    else this.emit({ kind: "rows", ids: changed.length ? changed.filter((id) => this.rows.has(id)) : r.order });
  }

  patch(row: RowData): void {
    if (!this.rows.has(row.id)) return;
    this.rows.set(row.id, row);
    this.emit({ kind: "rows", ids: [row.id] });
  }

  ordered(): string[] {
    if (this.groups || this.lanes) {
      const out: string[] = [];
      const seen = new Set<string>();
      const take = (gs: GroupResult[]) => {
        for (const g of gs) for (const id of g.rows) if (!seen.has(id)) (seen.add(id), out.push(id));
      };
      if (this.lanes) for (const l of this.lanes) take(l.groups);
      if (this.groups) take(this.groups);
      return out;
    }
    return this.slots.filter((x): x is string => x !== undefined);
  }
}

export class CollectionStore {
  private cols = new Map<string, Collection>();
  private listeners = new Map<string, Set<(c: Collection) => void>>();
  private loading = new Map<string, Promise<Collection>>();
  private sessions = new Set<QuerySession>();
  private members = new Map<string, Promise<Member[]>>();
  private memberCache = new Map<string, Map<string, Member>>();
  private pending = new Map<string, { values: Record<string, unknown>; title?: string; timer?: ReturnType<typeof setTimeout>; inflight: boolean }>();
  private firstPages = new Map<string, { result: QueryResult; at: number }>();

  constructor(
    readonly api: Client,
    readonly onError: (e: unknown) => void,
  ) {}

  get(id: string): Collection | undefined {
    return this.cols.get(id);
  }

  seed(c: Collection): void {
    this.set(c);
  }

  seedQuery(q: { view_id: string; result: QueryResult }): void {
    this.firstPages.set(q.view_id, { result: q.result, at: Date.now() });
  }

  takeBootQuery(viewId: string): QueryResult | null {
    const q = this.firstPages.get(viewId);
    this.firstPages.delete(viewId);
    if (!q || Date.now() - q.at > 15000) return null;
    return q.result;
  }

  private set(c: Collection): void {
    this.cols.set(c.id, c);
    for (const fn of this.listeners.get(c.id) ?? []) fn(c);
  }

  subscribe(id: string, fn: (c: Collection) => void): () => void {
    let set = this.listeners.get(id);
    if (!set) {
      set = new Set();
      this.listeners.set(id, set);
    }
    set.add(fn);
    return () => set?.delete(fn);
  }

  load(id: string, force = false, view?: string): Promise<Collection> {
    if (!force) {
      const have = this.cols.get(id);
      if (have) return Promise.resolve(have);
      const busy = this.loading.get(id);
      if (busy) return busy;
    }
    const job = this.api.collections
      .get(id, force ? undefined : view)
      .then((r) => {
        if (r.first) this.seedQuery(r.first);
        const c = this.withEdits(r.collection);
        this.set(c);
        return c;
      })
      .finally(() => this.loading.delete(id));
    this.loading.set(id, job);
    return job;
  }

  prop(colId: string, propId: string): Property | undefined {
    return this.cols.get(colId)?.schema.find((p) => p.id === propId);
  }

  session(colId: string, viewId: string, options: SessionOptions = {}): QuerySession {
    const s = new QuerySession(this, colId, viewId, options);
    this.sessions.add(s);
    return s;
  }

  release(s: QuerySession): void {
    this.sessions.delete(s);
  }

  people(workspaceId: string): Promise<Member[]> {
    let job = this.members.get(workspaceId);
    if (!job) {
      job = this.api.workspaces
        .members(workspaceId)
        .then((r) => {
          this.memberCache.set(workspaceId, new Map(r.members.map((m) => [m.user_id, m])));
          return r.members;
        })
        .catch((e) => {
          this.members.delete(workspaceId);
          throw e;
        });
      this.members.set(workspaceId, job);
    }
    return job;
  }

  person(workspaceId: string, id: string): Member | undefined {
    return this.memberCache.get(workspaceId)?.get(id);
  }

  async schema(colId: string, ops: SchemaOp[]): Promise<void> {
    const r = await this.api.collections.schema(colId, ops);
    const c = this.cols.get(colId);
    if (c) this.set({ ...c, schema: r.schema });
    void this.load(colId, true).catch(this.onError);
  }

  localView(colId: string, viewId: string, change: Partial<CollectionView>): void {
    const c = this.cols.get(colId);
    if (!c) return;
    this.set({ ...c, views: c.views.map((v) => (v.id === viewId ? { ...v, ...change } : v)) });
  }

  private viewTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private viewEdits = new Map<string, { change: Partial<CollectionView>; seq: number }>();

  updateView(colId: string, viewId: string, change: { name?: string; type?: ViewType; config?: ViewConfig }, debounce = 0): Promise<void> {
    this.localView(colId, viewId, change);
    const prior = this.viewEdits.get(viewId);
    const seq = (prior?.seq ?? 0) + 1;
    this.viewEdits.set(viewId, { change: { ...prior?.change, ...change }, seq });
    clearTimeout(this.viewTimers.get(viewId));
    const settle = () => {
      if (this.viewEdits.get(viewId)?.seq === seq) this.viewEdits.delete(viewId);
    };
    return new Promise((resolve) => {
      this.viewTimers.set(
        viewId,
        setTimeout(() => {
          const v = this.cols.get(colId)?.views.find((x) => x.id === viewId);
          const body: { name?: string; type?: ViewType; config?: ViewConfig } = {};
          if (change.name !== undefined) body.name = v?.name ?? change.name;
          if (change.type !== undefined) body.type = v?.type ?? change.type;
          if (change.config !== undefined) body.config = v?.config ?? change.config;
          this.api.collections
            .updateView(viewId, body)
            .then(() => {
              settle();
              resolve();
            })
            .catch((e) => {
              settle();
              this.onError(e);
              void this.load(colId, true).catch(() => undefined);
              resolve();
            });
        }, debounce),
      );
    });
  }

  private withEdits(c: Collection): Collection {
    if (!this.viewEdits.size || !c.views.some((v) => this.viewEdits.has(v.id))) return c;
    return { ...c, views: c.views.map((v) => ({ ...v, ...this.viewEdits.get(v.id)?.change })) };
  }

  async createView(colId: string, type: ViewType, name = "", config: ViewConfig = {}, afterId = ""): Promise<CollectionView> {
    const body: { name?: string; type: ViewType; config: ViewConfig; after_id?: string } = { type, config };
    if (name) body.name = name;
    if (afterId) body.after_id = afterId;
    const r = await this.api.collections.createView(colId, body);
    const c = this.cols.get(colId);
    if (c && !c.views.some((v) => v.id === r.view.id)) {
      const views = [...c.views, r.view].sort((a, b) => (a.sort_key < b.sort_key ? -1 : a.sort_key > b.sort_key ? 1 : 0));
      this.set({ ...c, views });
    }
    return r.view;
  }

  async deleteView(colId: string, viewId: string): Promise<void> {
    const c = this.cols.get(colId);
    if (c) this.set({ ...c, views: c.views.filter((v) => v.id !== viewId) });
    try {
      await this.api.collections.deleteView(viewId);
    } catch (e) {
      void this.load(colId, true).catch(() => undefined);
      throw e;
    }
  }

  async moveView(colId: string, viewId: string, afterId: string): Promise<void> {
    const c = this.cols.get(colId);
    if (c) {
      const moving = c.views.find((v) => v.id === viewId);
      const rest = c.views.filter((v) => v.id !== viewId);
      if (moving) {
        const at = afterId ? rest.findIndex((v) => v.id === afterId) + 1 : 0;
        rest.splice(at, 0, moving);
        this.set({ ...c, views: rest });
      }
    }
    await this.api.collections.moveView(viewId, afterId);
  }

  private patchSessions(row: RowData): void {
    for (const s of this.sessions) if (s.collectionId !== undefined) s.patch(row);
  }

  private local(rowId: string): RowData | undefined {
    for (const s of this.sessions) {
      const r = s.rows.get(rowId);
      if (r) return r;
    }
    return undefined;
  }

  setValue(colId: string, rowId: string, propId: string, value: unknown, debounce = 0): void {
    const now = this.local(rowId);
    if (now) {
      const next: RowData = propId === "title" ? { ...now, title: String(value ?? "") } : { ...now, values: { ...now.values } };
      if (propId !== "title") {
        if (value === null || value === undefined || value === "" || (Array.isArray(value) && value.length === 0)) delete next.values[propId];
        else next.values[propId] = value;
      }
      this.patchSessions(next);
    }
    let p = this.pending.get(rowId);
    if (!p) {
      p = { values: {}, inflight: false };
      this.pending.set(rowId, p);
    }
    if (propId === "title") p.title = String(value ?? "");
    else p.values[propId] = value ?? null;
    clearTimeout(p.timer);
    p.timer = setTimeout(() => void this.flushRow(colId, rowId), debounce);
  }

  pendingFor(rowId: string): boolean {
    return this.pending.has(rowId);
  }

  async flushRow(colId: string, rowId: string): Promise<void> {
    const p = this.pending.get(rowId);
    if (!p || p.inflight) return;
    clearTimeout(p.timer);
    const body: { title?: string; values?: Record<string, unknown> } = {};
    if (p.title !== undefined) body.title = p.title;
    if (Object.keys(p.values).length) body.values = p.values;
    p.values = {};
    delete p.title;
    p.inflight = true;
    try {
      const r = await this.api.collections.updateRow(rowId, body);
      p.inflight = false;
      if (!Object.keys(p.values).length && p.title === undefined) this.pending.delete(rowId);
      else void this.flushRow(colId, rowId);
      if (!this.pending.has(rowId)) this.patchSessions(r.row);
    } catch (e) {
      this.pending.delete(rowId);
      this.onError(e);
      for (const s of this.sessions) if (s.collectionId === colId) s.scheduleRefresh([rowId]);
    }
  }

  async moveRow(_colId: string, rowId: string, target: { before_id?: string; after_id?: string; values?: Record<string, unknown> }): Promise<RowData> {
    const body: { before_id?: string; after_id?: string; values?: Record<string, unknown>; move: boolean } = { move: true };
    if (target.before_id) body.before_id = target.before_id;
    if (target.after_id) body.after_id = target.after_id;
    if (target.values) body.values = target.values;
    const r = await this.api.collections.updateRow(rowId, body);
    this.patchSessions(r.row);
    return r.row;
  }

  async createRow(colId: string, body: { title?: string; values?: Record<string, unknown>; before_id?: string; after_id?: string; top?: boolean } = {}): Promise<RowData> {
    const r = await this.api.collections.createRow(colId, body);
    for (const s of this.sessions) if (s.collectionId === colId) s.scheduleRefresh([r.row.id]);
    return r.row;
  }

  async deleteRows(colId: string, ids: string[]): Promise<void> {
    await this.api.collections.deleteRows(colId, ids);
    for (const s of this.sessions) if (s.collectionId === colId) s.scheduleRefresh([]);
  }

  apply(e: LiveEvent): void {
    if (e.type !== "collection") return;
    if (e.what === "schema" || e.what === "views") {
      if (this.cols.has(e.collection_id)) void this.load(e.collection_id, true).catch(() => undefined);
      if (e.what === "schema") for (const s of this.sessions) if (s.collectionId === e.collection_id) s.scheduleRefresh([]);
      return;
    }
    const ids = (e.ids ?? []).filter((x) => x !== "*");
    for (const s of this.sessions) {
      if (s.collectionId !== e.collection_id) continue;
      if (ids.length && ids.every((id) => this.pending.has(id))) continue;
      s.scheduleRefresh(ids);
    }
  }

  resync(): void {
    for (const id of this.cols.keys()) void this.load(id, true).catch(() => undefined);
    for (const s of this.sessions) s.scheduleRefresh([]);
  }
}
