import type { Client } from "./client";
import type { CommentListing, CommentThread, LiveEvent } from "./types";

type Listener = (pageId: string) => void;

const EMPTY: CommentListing = { threads: [], can_comment: false, can_moderate: false };

export class CommentStore {
  private cache = new Map<string, CommentListing>();
  private loading = new Map<string, Promise<CommentListing>>();
  private listeners = new Set<Listener>();
  private watched = new Map<string, number>();

  constructor(
    private readonly api: Client,
    private readonly onError: (e: unknown) => void,
  ) {}

  subscribe(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(pageId: string): void {
    for (const fn of [...this.listeners]) {
      try {
        fn(pageId);
      } catch (e) {
        console.error(e);
      }
    }
  }

  get(pageId: string): CommentListing {
    return this.cache.get(pageId) ?? EMPTY;
  }

  has(pageId: string): boolean {
    return this.cache.has(pageId);
  }

  thread(pageId: string, id: string): CommentThread | undefined {
    return this.cache.get(pageId)?.threads.find((t) => t.id === id);
  }

  seed(pageId: string, listing: CommentListing): void {
    const prev = this.cache.get(pageId);
    if (prev && JSON.stringify(prev) === JSON.stringify(listing)) return;
    this.cache.set(pageId, listing);
    this.emit(pageId);
  }

  load(pageId: string, force = false): Promise<CommentListing> {
    const have = this.cache.get(pageId);
    if (have && !force) return Promise.resolve(have);
    const running = this.loading.get(pageId);
    if (running) return running;
    const job = this.api.comments
      .list(pageId)
      .then((l) => {
        this.seed(pageId, l);
        return l;
      })
      .finally(() => this.loading.delete(pageId));
    this.loading.set(pageId, job);
    return job;
  }

  watch(pageId: string): () => void {
    this.watched.set(pageId, (this.watched.get(pageId) ?? 0) + 1);
    return () => {
      const n = (this.watched.get(pageId) ?? 1) - 1;
      if (n <= 0) this.watched.delete(pageId);
      else this.watched.set(pageId, n);
    };
  }

  apply(e: LiveEvent): void {
    if (e.type !== "comments") return;
    if (this.watched.has(e.page_id)) void this.load(e.page_id, true).catch(this.onError);
    else this.cache.delete(e.page_id);
  }

  resync(): void {
    for (const id of this.cache.keys()) if (!this.watched.has(id)) this.cache.delete(id);
    for (const id of this.watched.keys()) void this.load(id, true).catch(this.onError);
  }

  async run<T>(pageId: string, work: () => Promise<T>): Promise<T> {
    const out = await work();
    await this.load(pageId, true);
    return out;
  }
}
