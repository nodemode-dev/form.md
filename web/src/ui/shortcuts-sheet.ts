import type { Engine } from "../engine";

export function mountShortcutsSheet(engine: Engine): void {
  let job: Promise<{ open(): void; close(): void }> | null = null;
  const load = () => (job ??= import("./shortcuts-editor").then((m) => m.mountShortcutsEditor(engine)));
  engine.state.watch("overlay", (v) => {
    if (v === "shortcuts")
      void load().then(
        (s) => engine.state.get("overlay") === "shortcuts" && s.open(),
        (e: unknown) => {
          job = null;
          engine.state.set("overlay", null);
          engine.reportError(e);
        },
      );
    else if (job) void job.then((s) => s.close(), () => undefined);
  });
}
