import type { LiveEvent } from "./types";

export type LiveStatus = "connecting" | "live" | "offline";

export interface LiveDeps {
  url: string;
  open: (url: string) => EventSource;
  onEvent: (e: LiveEvent) => void;
  onStatus: (s: LiveStatus) => void;
  onClosed: () => void;
}

const WATCHDOG_MS = 45000;
const RETRY_MS = [1000, 2000, 5000, 10000, 20000];

export class Live {
  private source: EventSource | null = null;
  private seq = 0;
  private hasSeq = false;
  private watchdog: ReturnType<typeof setTimeout> | undefined;
  private retry: ReturnType<typeof setTimeout> | undefined;
  private attempts = 0;
  private stopped = true;

  constructor(private readonly deps: LiveDeps) {}

  start(after?: number): void {
    if (after !== undefined) {
      this.seq = after;
      this.hasSeq = true;
    }
    this.stopped = false;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    clearTimeout(this.watchdog);
    clearTimeout(this.retry);
    this.source?.close();
    this.source = null;
  }

  reconnectNow(): void {
    if (this.stopped) return;
    clearTimeout(this.retry);
    this.attempts = 0;
    this.connect();
  }

  cursor(): number {
    return this.seq;
  }

  private connect(): void {
    this.source?.close();
    this.deps.onStatus("connecting");
    const url = this.hasSeq ? `${this.deps.url}?after=${this.seq}` : this.deps.url;
    const src = this.deps.open(url);
    this.source = src;
    this.arm();
    src.onmessage = (m: MessageEvent<string>) => {
      if (src !== this.source) return;
      this.arm();
      let e: LiveEvent;
      try {
        e = JSON.parse(m.data) as LiveEvent;
      } catch {
        return;
      }
      if (e.type === "hello") {
        this.attempts = 0;
        this.deps.onStatus("live");
        if (e.reset || !this.hasSeq) this.seq = e.seq;
        this.hasSeq = true;
      } else if (m.lastEventId) {
        const n = Number(m.lastEventId);
        if (Number.isFinite(n) && n > this.seq) this.seq = n;
      }
      this.deps.onEvent(e);
    };
    src.onerror = () => {
      if (src !== this.source) return;
      src.close();
      this.source = null;
      clearTimeout(this.watchdog);
      if (this.stopped) return;
      this.deps.onStatus("offline");
      this.deps.onClosed();
      this.schedule();
    };
  }

  private arm(): void {
    clearTimeout(this.watchdog);
    this.watchdog = setTimeout(() => {
      if (this.stopped) return;
      this.deps.onStatus("offline");
      this.connect();
    }, WATCHDOG_MS);
  }

  private schedule(): void {
    const delay = RETRY_MS[Math.min(this.attempts, RETRY_MS.length - 1)] ?? 20000;
    this.attempts++;
    clearTimeout(this.retry);
    this.retry = setTimeout(() => {
      if (!this.stopped) this.connect();
    }, delay);
  }
}
