const ASSETS = "formmd-assets";
const SHELL = "formmd-shell";
const SHELL_URL = "/offline-shell";
const APP_ROUTES = /^\/(p\/[^/]+\/?|library\/?|trash\/?|inbox\/?)?$/;

async function precache(all) {
  const res = await fetch("/offline.json", { cache: "no-store" });
  if (!res.ok) return;
  const manifest = await res.json();
  const cache = await caches.open(ASSETS);
  const have = new Set((await cache.keys()).map((r) => new URL(r.url).pathname));
  for (const path of all ? manifest.assets : manifest.core) {
    if (have.has(path)) continue;
    const r = await fetch(path);
    if (r.ok) await cache.put(path, r);
  }
  if (all) {
    const keep = new Set(manifest.assets);
    for (const req of await cache.keys()) if (!keep.has(new URL(req.url).pathname)) await cache.delete(req);
  }
}

async function refreshShell() {
  const res = await fetch(SHELL_URL, { credentials: "same-origin", cache: "no-store" });
  if (!res.ok) return;
  const text = await res.clone().text();
  if (!text.includes('data-auth="app"') || !text.includes('data-shell="offline"')) return;
  await (await caches.open(SHELL)).put(SHELL_URL, res);
}

async function fromCache(request) {
  const cache = await caches.open(ASSETS);
  const hit = await cache.match(request, { ignoreSearch: true, ignoreVary: true });
  if (hit) return hit;
  const res = await fetch(request);
  if (res.ok && res.type === "basic") await cache.put(request, res.clone());
  return res;
}

async function navigate(request) {
  try {
    return await fetch(request);
  } catch (err) {
    const shell = await (await caches.open(SHELL)).match(SHELL_URL);
    if (!shell) throw err;
    const path = new URL(request.url).pathname;
    const route = path.startsWith("/p/") ? "page" : path.replace(/\//g, "") || "home";
    const html = (await shell.text()).replace('data-shell="offline"', `data-shell="offline" data-route="${route}"`);
    const headers = new Headers(shell.headers);
    headers.delete("Content-Length");
    headers.delete("Content-Encoding");
    return new Response(html, { status: 200, headers });
  }
}

self.addEventListener("install", (event) => {
  event.waitUntil(precache(false).catch(() => undefined));
  self.skipWaiting();
});

self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (url.pathname.startsWith("/assets/")) event.respondWith(fromCache(request));
  else if (request.mode === "navigate" && APP_ROUTES.test(url.pathname)) event.respondWith(navigate(request));
});

self.addEventListener("message", (event) => {
  const type = event.data && event.data.type;
  if (type === "offline-shell") event.waitUntil(refreshShell().catch(() => undefined));
  else if (type === "offline-precache") event.waitUntil(precache(true).catch(() => undefined));
  else if (type === "offline-clear") event.waitUntil(caches.delete(SHELL));
});

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch {
    data = { title: event.data ? event.data.text() : "" };
  }
  const options = { body: data.body || "", data: { url: data.url || "/inbox" }, icon: "/favicon.svg" };
  if (data.tag) options.tag = data.tag;
  event.waitUntil(self.registration.showNotification(data.title || "form.md", options));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const url = new URL((event.notification.data && event.notification.data.url) || "/inbox", self.location.origin).href;
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((list) => {
      for (const client of list) {
        if (client.url.startsWith(self.location.origin)) {
          client.postMessage({ type: "open", url });
          return client.focus();
        }
      }
      return self.clients.openWindow(url);
    }),
  );
});
