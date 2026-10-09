import type { Engine } from "../engine";

const key = (engine: Engine) => `formmd-offline:${engine.state.get("me")?.user.id ?? ""}`;

export function offlineIds(engine: Engine): string[] {
  try {
    const v = JSON.parse(localStorage.getItem(key(engine)) ?? "[]") as unknown;
    return Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : [];
  } catch {
    return [];
  }
}

export function setOfflineIds(engine: Engine, ids: string[]): void {
  try {
    localStorage.setItem(key(engine), JSON.stringify(ids));
  } catch {
    return;
  }
}
