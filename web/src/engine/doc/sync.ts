import * as decoding from "lib0/decoding";
import * as encoding from "lib0/encoding";

export const KIND_SYNC = 0;
export const KIND_AWARENESS = 1;
export const KIND_CONTROL = 2;
export const KIND_QUERY_AWARENESS = 3;

export type SyncStatus = "connecting" | "live" | "offline";

export interface Control {
  t: string;
  id?: string;
  gen?: number;
  write?: boolean;
  code?: string;
}

export interface Channel {
  onOpened(info: { gen: number; write: boolean }): void;
  onMessage(kind: number, payload: Uint8Array): void;
  onControl(c: Control): void;
  onDisconnect(): void;
}

export interface SyncDeps {
  url: string;
  open: (url: string) => WebSocket;
  onStatus?: (s: SyncStatus) => void;
  onClosed?: (code: number) => void;
}

const WATCHDOG_MS = 45_000;
const MAX_BACKOFF = 15_000;

export class DocSync {
  private ws: WebSocket | null = null;
  private channels = new Map<string, Channel>();
  private status: SyncStatus = "offline";
  private attempt = 0;
  private retry: ReturnType<typeof setTimeout> | undefined;
  private watchdog: ReturnType<typeof setInterval> | undefined;
  private lastSeen = 0;
  private stopped = false;
  private statusListeners = new Set<(s: SyncStatus) => void>();

  constructor(private readonly deps: SyncDeps) {}

  get state(): SyncStatus {
    return this.status;
  }

  onStatus(fn: (s: SyncStatus) => void): () => void {
    this.statusListeners.add(fn);
    return () => this.statusListeners.delete(fn);
  }

  join(id: string, ch: Channel): void {
    this.channels.set(id, ch);
    if (this.status === "live") this.control({ t: "open", id });
    else this.ensure();
  }

  leave(id: string): void {
    if (!this.channels.delete(id)) return;
    if (this.status === "live") this.control({ t: "close", id });
  }

  send(kind: number, id: string, payload: Uint8Array): boolean {
    if (this.status !== "live" || !this.ws) return false;
    const enc = encoding.createEncoder();
    encoding.writeVarUint(enc, kind);
    encoding.writeVarString(enc, id);
    encoding.writeUint8Array(enc, payload);
    this.ws.send(encoding.toUint8Array(enc));
    return true;
  }

  reconnectNow(): void {
    if (this.stopped || this.status === "live") return;
    clearTimeout(this.retry);
    this.attempt = 0;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    clearTimeout(this.retry);
    clearInterval(this.watchdog);
    this.ws?.close(1000);
    this.ws = null;
  }

  private control(c: Control): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const enc = encoding.createEncoder();
    encoding.writeVarUint(enc, KIND_CONTROL);
    encoding.writeVarString(enc, JSON.stringify(c));
    this.ws.send(encoding.toUint8Array(enc));
  }

  private setStatus(s: SyncStatus): void {
    if (this.status === s) return;
    this.status = s;
    this.deps.onStatus?.(s);
    for (const fn of this.statusListeners) fn(s);
  }

  private ensure(): void {
    if (this.stopped || this.ws || this.retry) return;
    this.connect();
  }

  private connect(): void {
    if (this.stopped) return;
    this.retry = undefined;
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
    }
    this.setStatus("connecting");
    let ws: WebSocket;
    try {
      ws = this.deps.open(this.deps.url);
    } catch {
      this.scheduleRetry();
      return;
    }
    ws.binaryType = "arraybuffer";
    this.ws = ws;
    ws.onopen = () => {
      this.attempt = 0;
      this.lastSeen = Date.now();
      this.setStatus("live");
      for (const id of this.channels.keys()) this.control({ t: "open", id });
      clearInterval(this.watchdog);
      this.watchdog = setInterval(() => {
        if (Date.now() - this.lastSeen > WATCHDOG_MS) ws.close(4000, "silent");
      }, 5000);
    };
    ws.onmessage = (ev) => {
      this.lastSeen = Date.now();
      if (!(ev.data instanceof ArrayBuffer)) return;
      this.dispatch(new Uint8Array(ev.data));
    };
    ws.onclose = (ev) => {
      if (this.ws !== ws) return;
      this.ws = null;
      clearInterval(this.watchdog);
      this.setStatus("offline");
      for (const ch of this.channels.values()) ch.onDisconnect();
      this.deps.onClosed?.(ev.code);
      if (ev.code === 4001) return;
      this.scheduleRetry();
    };
  }

  private scheduleRetry(): void {
    if (this.stopped || this.retry) return;
    const base = Math.min(MAX_BACKOFF, 500 * 2 ** this.attempt);
    this.attempt++;
    this.retry = setTimeout(() => this.connect(), base / 2 + Math.random() * (base / 2));
  }

  private dispatch(data: Uint8Array): void {
    const dec = decoding.createDecoder(data);
    const kind = decoding.readVarUint(dec);
    if (kind === KIND_CONTROL) {
      let c: Control;
      try {
        c = JSON.parse(decoding.readVarString(dec)) as Control;
      } catch {
        return;
      }
      if (c.t === "ping" || c.t === "pong") return;
      const ch = c.id ? this.channels.get(c.id) : undefined;
      if (!ch) return;
      if (c.t === "opened") ch.onOpened({ gen: c.gen ?? 1, write: !!c.write });
      else ch.onControl(c);
      return;
    }
    const id = decoding.readVarString(dec);
    const ch = this.channels.get(id);
    if (!ch) return;
    ch.onMessage(kind, decoding.readTailAsUint8Array(dec));
  }
}
