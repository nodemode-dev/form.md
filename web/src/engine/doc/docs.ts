import * as decoding from "lib0/decoding";
import * as encoding from "lib0/encoding";
import { IndexeddbPersistence } from "y-indexeddb";
import * as awarenessProtocol from "y-protocols/awareness";
import * as syncProtocol from "y-protocols/sync";
import * as Y from "yjs";
import { ROOT } from "./schema";
import { type Control, type DocSync, KIND_AWARENESS, KIND_QUERY_AWARENESS, KIND_SYNC } from "./sync";

export type SessionPhase = "loading" | "ready" | "denied";
export type SessionEvent = "ready" | "synced" | "write" | "reset" | "denied";

const DB_PREFIX = "formmd-doc-";
const KEEP = 8;

export class DocSession {
  doc!: Y.Doc;
  awareness!: awarenessProtocol.Awareness;
  phase: SessionPhase = "loading";
  write = false;
  synced = false;
  local = false;
  denied = "";
  gen = 0;
  refs = 0;
  readonly ready: Promise<void>;
  private resolveReady!: () => void;
  private idb: IndexeddbPersistence | null = null;
  private live = false;
  private gate: Uint8Array[] | null = null;
  private listeners = new Set<(what: SessionEvent) => void>();
  private detachers: (() => void)[] = [];
  private hidden = false;

  constructor(
    readonly id: string,
    private readonly store: DocStore,
  ) {
    this.ready = new Promise((r) => (this.resolveReady = r));
    this.attach();
    store.sync.join(id, {
      onOpened: (info) => void this.opened(info.gen, info.write),
      onMessage: (kind, payload) => this.message(kind, payload),
      onControl: (c) => this.control(c),
      onDisconnect: () => this.disconnected(),
    });
  }

  get fragment(): Y.XmlFragment {
    return this.doc.getXmlFragment(ROOT);
  }

  on(fn: (what: SessionEvent) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private emit(what: SessionEvent): void {
    for (const fn of [...this.listeners]) {
      try {
        fn(what);
      } catch (e) {
        console.error(e);
      }
    }
  }

  private markReady(): void {
    if (this.phase !== "loading") return;
    this.phase = "ready";
    this.resolveReady();
    this.emit("ready");
  }

  private attach(): void {
    const doc = new Y.Doc();
    this.doc = doc;
    this.awareness = new awarenessProtocol.Awareness(doc);
    if (this.hidden) this.awareness.setLocalState(null);
    const onUpdate = (update: Uint8Array, origin: unknown) => {
      if (origin === this || origin === this.idb || !this.live || !this.write) return;
      const enc = encoding.createEncoder();
      syncProtocol.writeUpdate(enc, update);
      this.store.sync.send(KIND_SYNC, this.id, encoding.toUint8Array(enc));
    };
    doc.on("update", onUpdate);
    const onAwareness = ({ added, updated, removed }: { added: number[]; updated: number[]; removed: number[] }, origin: unknown) => {
      if (origin === this || !this.live) return;
      const changed = [...added, ...updated, ...removed].filter((id) => id === doc.clientID);
      if (!changed.length) return;
      this.store.sync.send(KIND_AWARENESS, this.id, awarenessProtocol.encodeAwarenessUpdate(this.awareness, changed));
    };
    this.awareness.on("update", onAwareness);
    this.detachers = [() => doc.off("update", onUpdate), () => this.awareness.off("update", onAwareness)];
    const name = this.store.dbName(this.id);
    if (name) {
      const idb = new IndexeddbPersistence(name, doc);
      this.idb = idb;
      void idb.whenSynced.then(() => {
        if (this.idb !== idb) return;
        this.local = true;
        if (this.fragment.length > 0) this.markReady();
      });
    }
  }

  private detach(): void {
    for (const d of this.detachers) d();
    this.detachers = [];
    awarenessProtocol.removeAwarenessStates(this.awareness, [this.doc.clientID], this);
    this.awareness.destroy();
    void this.idb?.destroy();
    this.idb = null;
    this.doc.destroy();
  }

  private async opened(gen: number, write: boolean): Promise<void> {
    this.gate = [];
    this.write = write;
    if (this.idb) {
      try {
        await this.idb.whenSynced;
        const stored = (await this.idb.get("gen")) as number | undefined;
        if (stored !== undefined && stored !== gen) await this.reset();
        await this.idb?.set("gen", gen);
      } catch {
        this.idb = null;
      }
    }
    this.gen = gen;
    this.live = true;
    const enc = encoding.createEncoder();
    syncProtocol.writeSyncStep1(enc, this.doc);
    this.store.sync.send(KIND_SYNC, this.id, encoding.toUint8Array(enc));
    if (this.awareness.getLocalState() !== null) {
      this.store.sync.send(KIND_AWARENESS, this.id, awarenessProtocol.encodeAwarenessUpdate(this.awareness, [this.doc.clientID]));
    }
    this.store.sync.send(KIND_QUERY_AWARENESS, this.id, new Uint8Array(0));
    const queued = this.gate;
    this.gate = null;
    for (const m of queued) this.message(KIND_SYNC, m);
    this.emit("write");
  }

  private async reset(): Promise<void> {
    const old = this.idb;
    this.idb = null;
    this.detach();
    await old?.clearData().catch(() => undefined);
    this.local = false;
    this.synced = false;
    this.attach();
    this.emit("reset");
  }

  private message(kind: number, payload: Uint8Array): void {
    if (kind === KIND_SYNC && this.gate) {
      this.gate.push(payload);
      return;
    }
    if (kind === KIND_SYNC) {
      const dec = decoding.createDecoder(payload);
      const enc = encoding.createEncoder();
      const type = syncProtocol.readSyncMessage(dec, enc, this.doc, this, (e) => console.warn("sync", e));
      if (type === syncProtocol.messageYjsSyncStep1 && !this.write) return;
      if (encoding.length(enc) > 0) this.store.sync.send(KIND_SYNC, this.id, encoding.toUint8Array(enc));
      if (type === syncProtocol.messageYjsSyncStep2 && !this.synced) {
        this.synced = true;
        this.markReady();
        this.emit("synced");
      }
      return;
    }
    if (kind === KIND_AWARENESS) awarenessProtocol.applyAwarenessUpdate(this.awareness, payload, this);
  }

  private control(c: Control): void {
    if (c.t === "denied" || c.t === "closed") {
      this.denied = c.code ?? "not_found";
      this.write = false;
      this.phase = "denied";
      this.resolveReady();
      this.emit("denied");
    } else if (c.t === "refused") {
      this.write = false;
      this.emit("write");
    }
  }

  private disconnected(): void {
    this.live = false;
    this.gate = null;
    const others = [...this.awareness.getStates().keys()].filter((id) => id !== this.doc.clientID);
    if (others.length) awarenessProtocol.removeAwarenessStates(this.awareness, others, this);
  }

  setHidden(hidden: boolean): void {
    if (this.hidden === hidden) return;
    this.hidden = hidden;
    this.awareness.setLocalState(hidden ? null : {});
  }

  destroy(): void {
    this.store.sync.leave(this.id);
    this.listeners.clear();
    this.detach();
  }
}

export interface DocStoreDeps {
  sync: DocSync;
  user: () => string;
  persist: boolean;
}

export class DocStore {
  readonly sync: DocSync;
  private sessions = new Map<string, DocSession>();
  private hidden = false;

  constructor(private readonly deps: DocStoreDeps) {
    this.sync = deps.sync;
  }

  dbName(id: string): string {
    const user = this.deps.user();
    return this.deps.persist && user ? `${DB_PREFIX}${user}-${id}` : "";
  }

  peek(id: string): DocSession | undefined {
    return this.sessions.get(id);
  }

  open(id: string): DocSession {
    let s = this.sessions.get(id);
    if (s) this.sessions.delete(id);
    else {
      s = new DocSession(id, this);
      if (this.hidden) s.setHidden(true);
    }
    this.sessions.set(id, s);
    s.refs++;
    this.trim();
    return s;
  }

  release(s: DocSession): void {
    s.refs = Math.max(0, s.refs - 1);
    this.trim();
  }

  prefetch(id: string): DocSession {
    const s = this.open(id);
    this.release(s);
    return s;
  }

  drop(id: string): void {
    const s = this.sessions.get(id);
    if (!s || s.refs > 0) return;
    this.sessions.delete(id);
    s.destroy();
  }

  setHidden(hidden: boolean): void {
    this.hidden = hidden;
    for (const s of this.sessions.values()) s.setHidden(hidden);
  }

  private trim(): void {
    if (this.sessions.size <= KEEP) return;
    for (const [id, s] of this.sessions) {
      if (this.sessions.size <= KEEP) break;
      if (s.refs > 0) continue;
      this.sessions.delete(id);
      s.destroy();
    }
  }

  async clearLocal(): Promise<void> {
    for (const s of this.sessions.values()) s.destroy();
    this.sessions.clear();
    this.sync.stop();
    if (typeof caches !== "undefined") await caches.delete("formmd-shell").catch(() => false);
    if (typeof indexedDB === "undefined" || !indexedDB.databases) return;
    const dbs = await indexedDB.databases().catch(() => []);
    await Promise.all(
      dbs
        .filter((d) => d.name?.startsWith(DB_PREFIX))
        .map(
          (d) =>
            new Promise<void>((resolve) => {
              const req = indexedDB.deleteDatabase(d.name as string);
              req.onsuccess = req.onerror = req.onblocked = () => resolve();
            }),
        ),
    );
  }
}
