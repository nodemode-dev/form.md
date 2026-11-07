import * as decoding from "lib0/decoding";
import * as encoding from "lib0/encoding";
import * as syncProtocol from "y-protocols/sync";
import type { Platform } from "../../../web/src/engine";
import type { LiveEvent } from "../../../web/src/engine/types";
import { KIND_CONTROL, KIND_QUERY_AWARENESS, KIND_SYNC } from "../../../web/src/engine/doc/sync";
import { NARROW_QUERY } from "../../../web/src/ui/sidebar";
import { serverDoc } from "./server/db";

const NAV_TOP_KEY = "formmd.demo.navTop";
const navOf = (s: unknown): number => {
  const n = (s as { nav?: unknown } | null)?.nav;
  return typeof n === "number" ? n : 0;
};

export const nav = { at: navOf(history.state), top: navOf(history.state) };

export function navSpan(): { back: boolean; forward: boolean } {
  return { back: nav.at > 0, forward: nav.at < nav.top };
}

function hashPath(): { path: string; search: string } {
  const raw = decodeURIComponent(location.hash.slice(1)) || "/";
  const q = raw.indexOf("?");
  return q < 0 ? { path: raw, search: "" } : { path: raw.slice(0, q), search: raw.slice(q) };
}

const sources = new Set<FakeEventSource>();

export class FakeEventSource {
  onmessage: ((m: MessageEvent<string>) => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;

  constructor(readonly url: string) {
    sources.add(this);
    queueMicrotask(() => this.send({ type: "hello", version: "demo", seq: 0, reset: false }));
  }

  send(e: LiveEvent): void {
    if (this.closed) return;
    this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(e) }));
  }

  close(): void {
    this.closed = true;
    sources.delete(this);
  }
}

export function pushLive(e: LiveEvent): void {
  for (const s of sources) s.send(e);
}

export class FakeWebSocket {
  static readonly OPEN = 1;
  binaryType = "arraybuffer";
  readyState = 0;
  onopen: (() => void) | null = null;
  onmessage: ((m: { data: ArrayBuffer }) => void) | null = null;
  onclose: ((e: { code: number }) => void) | null = null;

  constructor(readonly url: string) {
    setTimeout(() => {
      this.readyState = 1;
      this.onopen?.();
    }, 0);
  }

  private reply(build: (enc: encoding.Encoder) => void): void {
    const enc = encoding.createEncoder();
    build(enc);
    const bytes = encoding.toUint8Array(enc);
    const buf = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
    queueMicrotask(() => this.onmessage?.({ data: buf }));
  }

  private control(c: Record<string, unknown>): void {
    this.reply((enc) => {
      encoding.writeVarUint(enc, KIND_CONTROL);
      encoding.writeVarString(enc, JSON.stringify(c));
    });
  }

  private sync(id: string, payload: Uint8Array): void {
    this.reply((enc) => {
      encoding.writeVarUint(enc, KIND_SYNC);
      encoding.writeVarString(enc, id);
      encoding.writeUint8Array(enc, payload);
    });
  }

  send(data: Uint8Array): void {
    const dec = decoding.createDecoder(data);
    const kind = decoding.readVarUint(dec);
    if (kind === KIND_CONTROL) {
      const c = JSON.parse(decoding.readVarString(dec)) as { t: string; id?: string };
      if (c.t === "open" && c.id) {
        this.control({ t: "opened", id: c.id, gen: 1, write: true });
        const enc = encoding.createEncoder();
        syncProtocol.writeSyncStep1(enc, serverDoc(c.id));
        this.sync(c.id, encoding.toUint8Array(enc));
      }
      return;
    }
    const id = decoding.readVarString(dec);
    if (kind === KIND_QUERY_AWARENESS) return;
    if (kind !== KIND_SYNC) return;
    const body = decoding.createDecoder(decoding.readTailAsUint8Array(dec));
    const out = encoding.createEncoder();
    syncProtocol.readSyncMessage(body, out, serverDoc(id), "client");
    if (encoding.length(out) > 0) this.sync(id, encoding.toUint8Array(out));
  }

  close(): void {
    this.readyState = 3;
  }
}

export function demoPlatform(): Platform {
  const narrow = matchMedia(NARROW_QUERY);
  const light = matchMedia("(prefers-color-scheme: light)");
  return {
    mac: /Mac|iPhone|iPad|iPod/.test(navigator.userAgent),
    isNarrow: () => narrow.matches,
    prefersLight: () => light.matches,
    navigate: (path) => {
      location.hash = path;
    },
    location: () => ({ ...hashPath(), state: history.state as unknown }),
    pushHistory: (state, url) => {
      nav.top = ++nav.at;
      try {
        sessionStorage.setItem(NAV_TOP_KEY, String(nav.top));
      } catch {
        nav.top = nav.at;
      }
      history.pushState({ ...(state as object), nav: nav.at }, "", `#${url}`);
    },
    replaceHistory: (state, url) => history.replaceState({ ...(state as object), nav: nav.at }, "", `#${url}`),
    eventSource: (url) => new FakeEventSource(url) as unknown as EventSource,
    webSocket: (path) => new FakeWebSocket(path) as unknown as WebSocket,
    persistDocs: false,
  };
}
