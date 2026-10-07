import type { FetchFn } from "./api";

export type HealthPhase = "idle" | "ok" | "degraded" | "down";

export interface Health {
  phase: HealthPhase;
  version: string | null;
  storage: string | null;
  latencyMs: number | null;
  checkedAt: number | null;
}

export const initialHealth: Health = {
  phase: "idle",
  version: null,
  storage: null,
  latencyMs: null,
  checkedAt: null,
};

interface HealthBody {
  status?: string;
  version?: string;
  storage?: string;
}

export interface HealthSink {
  get(): Health;
  set(health: Health): void;
  setChecking(checking: boolean): void;
}

const UP_INTERVAL = 30_000;
const DOWN_INTERVAL = 5_000;
const TIMEOUT = 6_000;

export class HealthMonitor {
  private timer: ReturnType<typeof setTimeout> | undefined;
  private inflight: Promise<void> | null = null;

  constructor(
    private readonly fetchFn: FetchFn,
    private readonly sink: HealthSink,
    private readonly now: () => number,
  ) {}

  start(): void {
    void this.check();
  }

  stop(): void {
    clearTimeout(this.timer);
    this.timer = undefined;
  }

  check(): Promise<void> {
    if (this.inflight) return this.inflight;
    clearTimeout(this.timer);
    this.sink.setChecking(true);
    this.inflight = this.probe()
      .then((health) => {
        this.sink.set(health);
      })
      .finally(() => {
        this.inflight = null;
        this.sink.setChecking(false);
        const next = this.sink.get().phase === "ok" ? UP_INTERVAL : DOWN_INTERVAL;
        this.timer = setTimeout(() => void this.check(), next);
      });
    return this.inflight;
  }

  private async probe(): Promise<Health> {
    const started = this.now();
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), TIMEOUT);
    const previous = this.sink.get();
    try {
      const res = await this.fetchFn("/healthz", { cache: "no-store", signal: controller.signal });
      const body = (await res.json().catch(() => null)) as HealthBody | null;
      const latencyMs = Math.max(0, Math.round(this.now() - started));
      const checkedAt = Date.now();
      if (!body) return { ...previous, phase: "down", latencyMs: null, checkedAt };
      const phase: HealthPhase = res.ok && body.status === "ok" ? "ok" : "degraded";
      return {
        phase,
        version: body.version ?? previous.version,
        storage: body.storage ?? null,
        latencyMs,
        checkedAt,
      };
    } catch {
      return { ...previous, phase: "down", latencyMs: null, checkedAt: Date.now() };
    } finally {
      clearTimeout(timeout);
    }
  }
}
