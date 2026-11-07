import type { Engine } from "../../engine";
import { errorMessage } from "../../ui/errors";
import { toast } from "../../ui/toast";
import type { Extension } from "../registry";

let stop: (() => void) | null = null;
let live = false;

const formext: Extension = {
  id: "formext",
  setup(engine: Engine) {
    live = true;
    void import("./index").then(
      (m) => {
        if (!live) return;
        m.default.setup(engine);
        stop = () => m.default.dispose?.();
      },
      (e: unknown) => toast({ id: "error", tone: "danger", message: errorMessage(e) }),
    );
  },
  dispose() {
    live = false;
    stop?.();
    stop = null;
  },
};

export default formext;
