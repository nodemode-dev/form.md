import type { Engine } from "../../engine";
import type { ActionOutcome, Automation, AutomationAction, AutomationRun } from "../../engine/types";

const enc = encodeURIComponent;

export function automationApi(engine: Engine) {
  const { get, post, patch, del } = engine.api.http;
  return {
    list: (colId: string) => get<{ automations: Automation[]; can_design: boolean }>(`/api/collections/${enc(colId)}/automations`),
    create: (colId: string, body: Partial<Automation>) => post<{ automation: Automation }>(`/api/collections/${enc(colId)}/automations`, body),
    update: (id: string, body: Partial<Automation>) => patch<{ automation: Automation }>(`/api/automations/${enc(id)}`, body),
    remove: (id: string) => del(`/api/automations/${enc(id)}`),
    runs: (id: string) => get<{ runs: AutomationRun[] }>(`/api/automations/${enc(id)}/runs`),
    run: (id: string, rowId = "") => post<{ outcome: ActionOutcome }>(`/api/automations/${enc(id)}/run`, rowId ? { row_id: rowId } : {}),
    key: (ws: string) => get<{ key: string }>(`/api/workspaces/${enc(ws)}/automation-key`),
    rotateKey: (ws: string) => post<{ key: string }>(`/api/workspaces/${enc(ws)}/automation-key`),
    button: (body: { page_id: string; row_id?: string; name?: string; actions: AutomationAction[] }) => post<{ outcome: ActionOutcome }>("/api/buttons/run", body),
    property: (rowId: string, propId: string) => post<{ outcome: ActionOutcome }>(`/api/rows/${enc(rowId)}/buttons/${enc(propId)}`),
  };
}
