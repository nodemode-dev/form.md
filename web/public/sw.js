const ASSETS = "formmd-assets";
const SHELL = "formmd-shell";
const SHELL_URL = "/offline-shell";
const API = "formmd-api";
const API_READS = /^\/api\/(collections\/[^/]+|rows\/[^/]+|pages\/[^/]+)$/;
const QUERY_READS = /^\/api\/collections\/[^/]+\/(query|chart)$/;
const FILES = "formmd-files";

async function queryKey(request) {
  const body = await request.clone().text();
  let q;
  try {
    q = JSON.parse(body || "{}");
  } catch {
    return null;
  }
  if (q.search || q.offset > 0 || (Array.isArray(q.ids) && q.ids.length)) return null;
  const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(body)));
  const hex = [...digest].map((b) => b.toString(16).padStart(2, "0")).join("");
  const path = new URL(request.url).pathname;
  return { exact: new Request(`${path}?q=${hex}`), view: typeof q.view_id === "string" && q.view_id ? new Request(`${path}?view=${encodeURIComponent(q.view_id)}`) : null };
}

async function queryRead(request) {
  const key = await queryKey(request);
  try {
    const res = await fetch(request);
    if (res.ok && key) {
      const cache = await caches.open(API);
      await cache.put(key.exact, res.clone());
      if (key.view) await cache.put(key.view, res.clone());
    }
    return res;
  } catch (err) {
    const cache = await caches.open(API);
    const hit = key ? ((await cache.match(key.exact, { ignoreVary: true })) ?? (key.view ? await cache.match(key.view, { ignoreVary: true }) : undefined)) : undefined;
    if (hit) return hit;
    throw err;
  }
}
const FILE_READS = /^\/files\/[a-z0-9]+$/;

async function fileRead(request) {
  const cache = await caches.open(FILES);
  try {
    const res = await fetch(request);
    if (res.status === 200 && !request.headers.has("range")) await cache.put(request, res.clone());
    return res;
  } catch (err) {
    const hit = await cache.match(request, { ignoreVary: true, ignoreSearch: true });
    if (hit) return hit;
    throw err;
  }
}
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

async function apiRead(request) {
  const cache = await caches.open(API);
  try {
    const res = await fetch(request);
    if (res.ok) await cache.put(request, res.clone());
    return res;
  } catch (err) {
    const kept = new URL(request.url);
    kept.searchParams.set("rows", "1000");
    const hit =
      (kept.pathname.startsWith("/api/collections/") && !new URL(request.url).searchParams.has("rows") ? await cache.match(kept.href, { ignoreVary: true }) : undefined) ??
      (await cache.match(request, { ignoreVary: true })) ??
      (await cache.match(request, { ignoreVary: true, ignoreSearch: true }));
    if (hit) return hit;
    throw err;
  }
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


self.addEventListener("fetch", (event) => {
  const request = event.request;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (request.method === "POST" && QUERY_READS.test(url.pathname)) return event.respondWith(queryRead(request));
  if (request.method !== "GET") return;
  if (url.pathname.startsWith("/assets/") || url.pathname.startsWith("/maps/")) event.respondWith(fromCache(request));
  else if (request.mode === "navigate" && APP_ROUTES.test(url.pathname)) event.respondWith(navigate(request));
  else if (API_READS.test(url.pathname)) event.respondWith(apiRead(request));
  else if (FILE_READS.test(url.pathname)) event.respondWith(fileRead(request));
});

self.addEventListener("message", (event) => {
  const type = event.data && event.data.type;
  if (type === "offline-shell") event.waitUntil(refreshShell().catch(() => undefined));
  else if (type === "offline-precache") event.waitUntil(precache(true).catch(() => undefined));
  else if (type === "offline-api" && Array.isArray(event.data.urls))
    event.waitUntil(
      Promise.all(
        event.data.urls
          .filter((u) => typeof u === "string" && API_READS.test(new URL(u, self.location.origin).pathname))
          .map((u) => apiRead(new Request(new URL(u, self.location.origin).href, { credentials: "same-origin" })).catch(() => undefined)),
      ),
    );
  else if (type === "offline-files" && Array.isArray(event.data.urls))
    event.waitUntil(
      caches.open(FILES).then((cache) =>
        Promise.all(
          event.data.urls
            .filter((u) => typeof u === "string" && FILE_READS.test(new URL(u, self.location.origin).pathname))
            .map(async (u) => {
              const req = new Request(new URL(u, self.location.origin).href, { credentials: "same-origin" });
              if (await cache.match(req, { ignoreVary: true, ignoreSearch: true })) return;
              await fileRead(req).catch(() => undefined);
            }),
        ),
      ),
    );
  else if (type === "offline-clear") event.waitUntil(Promise.all([caches.delete(SHELL), caches.delete(API), caches.delete(FILES)]));
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
