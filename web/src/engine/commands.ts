import type { Unsubscribe } from "./state";

export interface CommandParam {
  type: "string" | "number" | "integer" | "boolean" | "array" | "object";
  required?: boolean;
  enum?: ReadonlyArray<string | number>;
  min?: number;
  max?: number;
  description?: string;
}

export interface Command {
  id: string;
  label: () => string;
  group: () => string;
  run: (params?: Record<string, unknown>) => unknown;
  enabled?: () => boolean;
  description?: string;
  when?: string;
  params?: Record<string, CommandParam>;
  mutates?: boolean;
  risky?: boolean;
}

export interface CommandSpec {
  id: string;
  label: string;
  group: string;
  description?: string;
  when?: string;
  keys?: string[];
  params?: Record<string, CommandParam>;
  mutates: boolean;
  risky?: boolean;
}

export class Commands {
  onChange: (() => void) | null = null;
  keysOf: ((id: string) => string[]) | null = null;
  private byId = new Map<string, Command>();

  register(command: Command): Unsubscribe {
    if (this.byId.has(command.id)) throw new Error(`duplicate command: ${command.id}`);
    this.byId.set(command.id, command);
    this.onChange?.();
    return () => {
      this.byId.delete(command.id);
      this.onChange?.();
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

  async invoke(id: string, params: Record<string, unknown> = {}): Promise<unknown> {
    const command = this.byId.get(id);
    if (!command) throw new Error(`unknown command: ${id}`);
    if (command.enabled?.() === false) throw new Error(`${id} is not available right now`);
    for (const [name, p] of Object.entries(command.params ?? {})) {
      const v = params[name];
      if (v === undefined) {
        if (p.required) throw new Error(`${id}: ${name} is required`);
        continue;
      }
      const kind = Array.isArray(v) ? "array" : v === null ? "null" : typeof v;
      const ok = p.type === "integer" ? Number.isInteger(v) : p.type === "number" ? typeof v === "number" && Number.isFinite(v) : p.type === kind;
      if (!ok) throw new Error(`${id}: ${name} must be ${p.type}`);
      if (p.enum && !p.enum.includes(v as string | number)) throw new Error(`${id}: ${name} must be one of ${p.enum.join(", ")}`);
      if (typeof v === "number" && ((p.min !== undefined && v < p.min) || (p.max !== undefined && v > p.max))) throw new Error(`${id}: ${name} must be between ${p.min ?? "-inf"} and ${p.max ?? "inf"}`);
    }
    const out = await command.run(params);
    return out === undefined ? { ok: true } : out;
  }

  catalogue(): CommandSpec[] {
    return this.list()
      .map((c) => ({
        id: c.id,
        label: c.label(),
        group: c.group(),
        ...(c.description ? { description: c.description } : {}),
        ...(c.when ? { when: c.when } : {}),
        keys: this.keysOf?.(c.id) ?? [],
        ...(c.params ? { params: c.params } : {}),
        mutates: c.mutates ?? true,
        ...(c.risky ? { risky: true } : {}),
      }))
      .sort((a, b) => a.id.localeCompare(b.id));
  }
}
