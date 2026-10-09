import type { Engine } from "../engine";

export function pushSupported(): boolean {
  return typeof navigator !== "undefined" && "serviceWorker" in navigator && typeof window !== "undefined" && "PushManager" in window && "Notification" in window;
}

function keyBytes(b64: string): Uint8Array {
  const pad = "=".repeat((4 - (b64.length % 4)) % 4);
  const raw = atob((b64 + pad).replace(/-/g, "+").replace(/_/g, "/"));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

async function registration(): Promise<ServiceWorkerRegistration> {
  return (await navigator.serviceWorker.getRegistration("/")) ?? navigator.serviceWorker.register("/sw.js", { scope: "/" });
}

export async function currentSubscription(): Promise<PushSubscription | null> {
  if (!pushSupported()) return null;
  const reg = await navigator.serviceWorker.getRegistration("/");
  return reg ? reg.pushManager.getSubscription() : null;
}

export async function enablePush(engine: Engine, publicKey: string): Promise<"on" | "denied"> {
  const permission = await Notification.requestPermission();
  if (permission !== "granted") return "denied";
  const reg = await registration();
  const sub = (await reg.pushManager.getSubscription()) ?? (await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: keyBytes(publicKey) as BufferSource }));
  const json = sub.toJSON() as { endpoint: string; keys: { p256dh: string; auth: string } };
  await engine.api.inbox.subscribe({ endpoint: json.endpoint, keys: { p256dh: json.keys.p256dh, auth: json.keys.auth } });
  return "on";
}

export async function disablePush(engine: Engine): Promise<void> {
  const sub = await currentSubscription();
  if (!sub) return;
  await engine.api.inbox.unsubscribe(sub.endpoint).catch(() => undefined);
  await sub.unsubscribe();
}

export function listenForPushClicks(engine: Engine): void {
  if (!pushSupported()) return;
  navigator.serviceWorker.addEventListener("message", (e) => {
    const data = e.data as { type?: string; url?: string } | null;
    if (data?.type !== "open" || !data.url) return;
    const u = new URL(data.url);
    const m = /^\/p\/([a-z2-7]{26})/.exec(u.pathname);
    if (m?.[1]) {
      engine.openPage(m[1]);
      const thread = u.searchParams.get("thread");
      if (thread) window.dispatchEvent(new CustomEvent("formmd:thread", { detail: { page: m[1], thread } }));
    } else engine.navigate({ name: "inbox" });
  });
}
