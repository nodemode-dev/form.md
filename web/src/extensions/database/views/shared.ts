import type { CollectionView, Filter, ViewConfig } from "../../../engine/types";
import type { DbCtx } from "../cells";

export function configKey(cfg: ViewConfig, extra: unknown = null): string {
  return JSON.stringify([cfg.filter ?? null, cfg.sorts ?? null, cfg.group ?? null, cfg.sub_group ?? null, cfg.calcs ?? null, extra]);
}

export function prefillFromFilter(ctx: DbCtx, view: CollectionView, extra: Record<string, unknown> = {}): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  const f: Filter | undefined = view.config.filter;
  const rules: Filter[] = !f ? [] : f.prop ? [f] : f.op === "or" ? [] : (f.items ?? []).filter((x) => x.prop);
  for (const r of rules) {
    const p = ctx.col().schema.find((x) => x.id === r.prop);
    if (!p) continue;
    const v = r.value;
    switch (p.type) {
      case "select":
      case "status":
        if (r.cond === "is" && Array.isArray(v) && v.length === 1) out[p.id] = v[0];
        else if (r.cond === "is" && typeof v === "string" && v) out[p.id] = v;
        break;
      case "multi_select":
        if (r.cond === "contains" && Array.isArray(v) && v.length) out[p.id] = v;
        break;
      case "person":
        if (r.cond === "contains" && Array.isArray(v) && v.length) out[p.id] = v.map((x) => (x === "me" ? (ctx.engine.state.get("me")?.user.id ?? "") : x)).filter(Boolean);
        break;
      case "checkbox":
        if (r.cond === "is" && (v === true || v === "true")) out[p.id] = true;
        break;
      case "text":
      case "url":
      case "email":
      case "phone":
        if (r.cond === "is" && typeof v === "string" && v) out[p.id] = v;
        break;
      case "number":
        if ((r.cond === "eq" || r.cond === "is") && typeof v === "number") out[p.id] = v;
        break;
    }
  }
  return { ...out, ...extra };
}
