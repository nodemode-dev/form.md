import type { Unsubscribe } from "./state";

export interface Command {
  id: string;
  label: () => string;
  group: () => string;
  run: () => void | Promise<void>;
  enabled?: () => boolean;
}

export class Commands {
  private byId = new Map<string, Command>();

  register(command: Command): Unsubscribe {
    if (this.byId.has(command.id)) throw new Error(`duplicate command: ${command.id}`);
    this.byId.set(command.id, command);
    return () => {
      this.byId.delete(command.id);
    };
  }

  get(id: string): Command | undefined {
    return this.byId.get(id);
  }

  list(): Command[] {
    return [...this.byId.values()];
  }

  run(id: string): boolean {
    const command = this.byId.get(id);
    if (!command || command.enabled?.() === false) return false;
    void command.run();
    return true;
  }
}
