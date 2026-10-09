import type { Client } from "./client";
import type { InboxItem, LiveEvent } from "./types";

export type InboxFilter = "all" | "unread" | "archived";

export type InboxChange = { kind: "count" } | { kind: "items" } | { kind: "arrived"; items: InboxItem[] };

export class InboxStore {
  unread = 0;
  items: InboxItem[] = [];
  filter: InboxFilter = "all";
  loaded = false;
  more = false;
  private newest = 0;
  private watching = 0;
  private seq = 0;
  private listeners = new Set<(c: InboxChange) => void>();

  constructor(
    private readonly api: Client,
    private readonly onError: (e: unknown) => void,
    private readonly visible: () => boolean,
    unread: number,
  ) {
    this.unread = unread;
    this.newest = Date.now();
  }

  subscribe(fn: (c: InboxChange) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(c: InboxChange): void {
    for (const fn of [...this.listeners]) {
      try {
        fn(c);
      } catch (e) {
        console.error(e);
      }
    }
  }

  private setUnread(n: number): void {
    if (n === this.unread) return;
    this.unread = n;
    this.emit({ kind: "count" });
  }

  watch(): () => void {
    this.watching++;
    if (!this.loaded) void this.load().catch(this.onError);
    return () => {
      this.watching--;
    };
  }

  async load(): Promise<void> {
    const mine = ++this.seq;
    const r = await this.api.inbox.list(this.filter);
    if (mine !== this.seq) return;
    this.items = r.items;
    this.more = r.items.length >= 50;
    this.loaded = true;
    this.setUnread(r.unread);
    this.emit({ kind: "items" });
  }

  async loadMore(): Promise<void> {
    const last = this.items.at(-1);
    if (!last || !this.more) return;
    const r = await this.api.inbox.list(this.filter, last.created_at);
    this.items = [...this.items, ...r.items.filter((x) => !this.items.some((y) => y.id === x.id))];
    this.more = r.items.length >= 50;
    this.emit({ kind: "items" });
  }

  setFilter(f: InboxFilter): void {
    if (f === this.filter) return;
    this.filter = f;
    this.loaded = false;
    void this.load().catch(this.onError);
  }

  private async arrived(): Promise<void> {
    const r = await this.api.inbox.list("all");
    this.setUnread(r.unread);
    const fresh = r.items.filter((x) => x.created_at > this.newest && !x.read_at);
    if (r.items[0]) this.newest = Math.max(this.newest, r.items[0].created_at);
    if (this.filter === "all") {
      this.items = r.items;
      this.loaded = true;
      this.emit({ kind: "items" });
    } else if (this.watching) await this.load();
    if (!fresh.length) return;
    this.emit({ kind: "arrived", items: fresh });
    if (this.visible()) void this.api.inbox.mark({ ids: fresh.map((x) => x.id), seen: true }).catch(() => undefined);
  }

  apply(e: LiveEvent): void {
    if (e.type === "user" && e.what === "inbox") void this.arrived().catch(this.onError);
  }

  async resync(): Promise<void> {
    if (this.watching) await this.load();
    else this.setUnread((await this.api.inbox.count()).unread);
  }

  async mark(ids: string[], change: { read?: boolean; archived?: boolean }): Promise<void> {
    const before = this.items;
    const now = Date.now();
    this.items = this.items
      .map((x) => (ids.includes(x.id) ? { ...x, ...(change.read !== undefined ? { read_at: change.read ? now : 0 } : {}), ...(change.archived !== undefined ? { archived_at: change.archived ? now : 0 } : {}) } : x))
      .filter((x) => (change.archived === undefined ? true : this.filter === "archived" ? !!x.archived_at : !x.archived_at));
    this.emit({ kind: "items" });
    try {
      await this.api.inbox.mark({ ids, ...change });
      this.setUnread((await this.api.inbox.count()).unread);
    } catch (e) {
      this.items = before;
      this.emit({ kind: "items" });
      throw e;
    }
  }

  async markAllRead(): Promise<void> {
    const now = Date.now();
    this.items = this.items.map((x) => (x.read_at ? x : { ...x, read_at: now }));
    if (this.filter === "unread") this.items = [];
    this.emit({ kind: "items" });
    await this.api.inbox.mark({ all: true, read: true });
    this.setUnread(0);
  }
}
