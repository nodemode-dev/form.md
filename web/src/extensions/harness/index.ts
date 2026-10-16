import type { Engine } from "../../engine";
import { request } from "../../engine/api";
import { t } from "../../engine/i18n";
import type { Unsubscribe } from "../../engine/state";
import type { AgentDock } from "../../ui/agent-dock";
import type { Extension } from "../registry";

const GROUP = () => t("Agents");
let teardown: Unsubscribe[] = [];

function setup(engine: Engine): void {
  const { commands, keymap, state } = engine;
  let job: Promise<AgentDock> | null = null;
  const dock = () => {
    job ??= import("../../ui/agent-dock").then((m) => m.mountAgentDock(engine));
    job.catch(() => (job = null));
    return job;
  };
  const fail = (e: unknown) => engine.reportError(e);

  let timer: ReturnType<typeof setTimeout> | undefined;
  const mirror = () => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (!state.get("me")) return;
      void request(fetch, "PUT", "/api/commands/catalogue", { commands: commands.catalogue() }).catch(() => undefined);
    }, 400);
  };
  commands.onChange = mirror;

  teardown = [
    commands.register({
      id: "dock.toggle",
      label: () => t("Show or hide the terminal"),
      group: GROUP,
      description: "Shows or hides the bottom panel with terminals, Claude Code and the Claude Code chat.",
      mutates: false,
      run: () => void dock().then((d) => d.toggle(), fail),
    }),
    commands.register({
      id: "dock.newTerminal",
      label: () => t("New terminal"),
      group: GROUP,
      description: "Opens a new shell tab in the bottom panel, on the computer form.md runs on.",
      mutates: true,
      run: () => dock().then((d) => d.newTerminal()),
    }),
    commands.register({
      id: "agent.claude",
      label: () => t("Claude Code in a terminal"),
      group: GROUP,
      description: "Opens Claude Code in a terminal tab, already connected to this form.md through its MCP tools.",
      when: "Use when the person wants the full Claude Code terminal experience inside form.md.",
      mutates: true,
      run: () => dock().then((d) => d.claude()),
    }),
    commands.register({
      id: "agent.chat",
      label: () => t("Chat with Claude Code"),
      group: GROUP,
      description: "Opens the Claude Code chat tab: Claude Code answers through form.md's tools, with each tool call shown as it runs.",
      mutates: true,
      run: () => dock().then((d) => d.chat()),
    }),
    keymap.register({ id: "dock.toggle", command: "dock.toggle", chords: ["mod+j"], scope: "global", whileTyping: true }),
    keymap.register({ id: "agent.claude", command: "agent.claude", chords: ["mod+alt+j"], scope: "global", whileTyping: true }),
    keymap.register({ id: "agent.chat", command: "agent.chat", chords: ["mod+alt+k"], scope: "global", whileTyping: true }),
    state.watch("me", (me) => {
      if (me) mirror();
    }),
  ];
}

const harness: Extension = {
  id: "harness",
  setup,
  dispose() {
    for (const fn of teardown) fn();
    teardown = [];
  },
};

export default harness;
