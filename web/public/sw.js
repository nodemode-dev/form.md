self.addEventListener("install", () => self.skipWaiting());

self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));

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
