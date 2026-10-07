import { createEngine } from "./engine";
import { activateAll, register } from "./extensions/registry";
import shell from "./extensions/shell";
import { mountUI, platformAdapter, storageAdapter } from "./ui/app";

const engine = createEngine({
  storage: storageAdapter(),
  fetch: window.fetch.bind(window),
  platform: platformAdapter(),
  now: () => performance.now(),
});

register(shell);
activateAll(engine);
mountUI(engine);
engine.start();
