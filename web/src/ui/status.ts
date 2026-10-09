import type { Engine, HealthPhase } from "../engine";
import { t } from "../engine/i18n";
import { delayedIndicator, ref, setAttr, setText } from "./dom";
import { toast } from "./toast";

const LABEL: Record<HealthPhase, string> = {
  idle: "Connecting",
  ok: "Connected",
  degraded: "Degraded",
  down: "Unreachable",
};

const CONNECTION_TOAST = "connection";

export function mountStatus(engine: Engine): void {
  const card = ref("status");
  const label = ref("status-label");
  const server = ref("status-server");
  const version = ref("status-version");
  const storage = ref("status-storage");
  const latency = ref("status-latency");
  const sidebarVersion = ref("version-text");
  const liveDot = ref("live-dot");
  engine.state.watch(
    "live",
    (s) => {
      setAttr(liveDot, "data-state", s);
      setAttr(liveDot.parentElement as HTMLElement, "title", s === "live" ? t("Live updates on") : s === "offline" ? t("Live updates paused, reconnecting") : t("Connecting live updates"));
    },
    { immediate: true },
  );

  setText(server, location.host);

  engine.state.watch("health", (health, previous) => {
    setAttr(card, "data-state", health.phase);
    setText(label, t(LABEL[health.phase]));
    setText(version, health.version ?? "");
    if (health.version) setText(sidebarVersion, health.version);
    const role = engine.state.get("me")?.user.role;
    const update = health.latest && (role === "owner" || role === "admin") ? health.latest : "";
    setAttr(sidebarVersion, "data-update", update ? "true" : null);
    setAttr(sidebarVersion, "title", update ? t("{latest} is available", { latest: update }) : null);
    setText(storage, health.storage === null ? "" : health.storage === "ok" ? "OK" : "Error");
    setText(latency, health.latencyMs === null ? (health.phase === "down" ? "n/a" : "") : `${health.latencyMs} ms`);

    if (health.phase === previous.phase) return;
    if (health.phase === "down") {
      toast({
        id: CONNECTION_TOAST,
        tone: "warning",
        message: t("Cannot reach the server. Pages saved on this device stay open, and edits to their text sync when it is back."),
        durationMs: 0,
        action: { label: t("Retry"), run: () => engine.commands.run("shell.checkConnection") },
      });
    } else if (health.phase === "degraded") {
      toast({ id: CONNECTION_TOAST, tone: "warning", message: t("The server is up, but its storage reports an error."), durationMs: 0 });
    } else if (health.phase === "ok" && (previous.phase === "down" || previous.phase === "degraded")) {
      toast({ id: CONNECTION_TOAST, tone: "success", message: t("Connection restored.") });
    }
  });

  let stopIndicator: (() => void) | null = null;
  engine.state.watch("checking", (checking) => {
    if (checking && !stopIndicator) {
      stopIndicator = delayedIndicator(
        () => setAttr(card, "data-checking", "true"),
        () => setAttr(card, "data-checking", null),
      );
    } else if (!checking && stopIndicator) {
      stopIndicator();
      stopIndicator = null;
    }
  });

  window.addEventListener("online", () => void engine.health.check());
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") void engine.health.check();
  });
}
