import { t } from "../../engine/i18n";
import type { CollectionView, ComputedValue, Filter, Grouping, Member, Property, SortKey, ViewConfig, ViewProp } from "../../engine/types";
import { avatar } from "../../ui/avatar";
import { h, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { openMenu } from "../../ui/menu";
import { type DbCtx, optionChip } from "./cells";
import { openPanel } from "./panel";
import { openAddProperty, openPropertyEditor } from "./propedit";
import { CONDS, condKind, RELATIVE_DAYS, TYPE_INFO, WITHIN } from "./props";

export function viewOf(ctx: DbCtx, viewId: string): CollectionView | undefined {
  return ctx.col().views.find((v) => v.id === viewId);
}

export function saveConfig(ctx: DbCtx, viewId: string, change: (cfg: ViewConfig) => ViewConfig, debounce = 250): void {
  const v = viewOf(ctx, viewId);
  if (!v) return;
  const next = change(structuredClone(v.config ?? {}));
  if (!ctx.canDesign()) {
    ctx.engine.collections.localView(ctx.colId, viewId, { config: next });
    return;
  }
  void ctx.engine.collections.updateView(ctx.colId, viewId, { config: next }, debounce);
}

function sampleOf(ctx: DbCtx, p: Property): ComputedValue | null {
  for (const r of ctx.session.rows.values()) {
    const v = r.values[p.id] as ComputedValue | null | undefined;
    if (v && typeof v === "object" && "type" in v) return v;
  }
  return null;
}

export function propSelect(ctx: DbCtx, value: string, onChange: (id: string) => void, filter: (p: Property) => boolean = () => true): HTMLButtonElement {
  const b = h("button", { class: "select-btn select-btn--sm", attrs: { type: "button", "aria-haspopup": "menu" } });
  const paint = (id: string) => {
    const p = ctx.col().schema.find((x) => x.id === id);
    b.replaceChildren(iconNode(p ? TYPE_INFO[p.type].icon : "text"), h("span", { text: p?.name ?? t("Choose") }), iconNode("chevron-down"));
  };
  paint(value);
  b.addEventListener("click", () =>
    openMenu(
      b,
      ctx
        .col()
        .schema.filter(filter)
        .map((p) => ({
          label: p.name,
          icon: TYPE_INFO[p.type].icon,
          checked: p.id === value,
          run: () => {
            value = p.id;
            paint(p.id);
            onChange(p.id);
          },
        })),
      { side: "bottom", align: "start", label: t("Property") },
    ),
  );
  return b;
}

function filterValueControl(ctx: DbCtx, p: Property, f: Filter, changed: () => void): HTMLElement {
  const kind = condKind(p, sampleOf(ctx, p));
  const set = (v: unknown) => {
    f.value = v;
    changed();
  };
  const text = (type = "text") => {
    const i = h("input", { class: "input input--sm filter__value", attrs: { type, "aria-label": t("Value"), placeholder: t("Value") } });
    i.value = f.value === undefined || f.value === null ? "" : String(f.value);
    i.addEventListener("input", () => set(type === "number" && i.value !== "" ? Number(i.value) : i.value));
    return i;
  };
  if (kind === "text" || kind === "files" || kind === "valuelist") return text();
  if (kind === "number") return text("number");
  if (kind === "checkbox") {
    const s = h("select", { class: "input input--sm filter__value", attrs: { "aria-label": t("Value") } });
    s.append(h("option", { attrs: { value: "true" }, text: t("Checked") }), h("option", { attrs: { value: "false" }, text: t("Unchecked") }));
    s.value = f.value === true || f.value === "true" ? "true" : "false";
    if (f.value === undefined) f.value = false;
    s.addEventListener("change", () => set(s.value === "true"));
    return s;
  }
  if (kind === "date") {
    const wrap = h("span", { class: "filter__date" });
    const val = (f.value ?? {}) as { date?: string; to?: string; relative?: string; within?: string };
    if (f.cond === "within") {
      const s = h("select", { class: "input input--sm", attrs: { "aria-label": t("Period") } });
      for (const [k, l] of WITHIN) s.append(h("option", { attrs: { value: k }, text: t(l) }));
      s.value = val.within ?? "past_week";
      if (!val.within) f.value = { within: s.value };
      s.addEventListener("change", () => set({ within: s.value }));
      wrap.append(s);
      return wrap;
    }
    const mode = h("select", { class: "input input--sm", attrs: { "aria-label": t("Date kind") } });
    mode.append(h("option", { attrs: { value: "exact" }, text: t("Exact date") }));
    for (const [k, l] of RELATIVE_DAYS) mode.append(h("option", { attrs: { value: k }, text: t(l) }));
    mode.value = val.relative ?? "exact";
    const day = h("input", { class: "input input--sm", attrs: { type: "date", "aria-label": t("Date") } });
    day.value = val.date ?? "";
    const to = h("input", { class: "input input--sm", attrs: { type: "date", "aria-label": t("End date") } });
    to.value = val.to ?? "";
    const emit = () => {
      const out: Record<string, string> = {};
      if (mode.value !== "exact") out.relative = mode.value;
      else if (day.value) out.date = day.value;
      if (f.cond === "between" && to.value) out.to = to.value;
      day.hidden = mode.value !== "exact";
      set(out);
    };
    mode.addEventListener("change", emit);
    day.addEventListener("change", emit);
    to.addEventListener("change", emit);
    day.hidden = mode.value !== "exact";
    wrap.append(mode, day);
    if (f.cond === "between") wrap.append(h("span", { text: t("and") }), to);
    return wrap;
  }
  const chosen = new Set(Array.isArray(f.value) ? (f.value as string[]) : typeof f.value === "string" && f.value ? [f.value] : []);
  const btn = h("button", { class: "select-btn select-btn--sm filter__value", attrs: { type: "button" } });
  const paint = () => {
    btn.replaceChildren();
    if (!chosen.size) btn.append(h("span", { class: "faint", text: t("Select") }));
    for (const id of chosen) {
      if (p.type === "person" || p.type === "created_by" || p.type === "last_edited_by") {
        const m = ctx.engine.collections.person(ctx.workspaceId(), id);
        btn.append(h("span", { class: "chip", text: id === "me" ? t("Me") : (m?.name ?? "") }));
      } else if (p.type === "relation") btn.append(h("span", { class: "chip", text: ctx.session.refs.get(id)?.title ?? t("Page") }));
      else {
        const o = p.options?.find((x) => x.id === id);
        if (o) btn.append(optionChip(o, p.type === "status"));
      }
    }
    btn.append(iconNode("chevron-down"));
  };
  paint();
  btn.addEventListener("click", () => {
    openPanel(
      btn,
      (panel) => {
        const list = h("div", { class: "opt-editor__list" });
        const toggle = (id: string) => {
          if (chosen.has(id)) chosen.delete(id);
          else chosen.add(id);
          set([...chosen]);
          paint();
          fill();
        };
        let members: Member[] = [];
        let rels: { id: string; title: string }[] = [];
        const fill = () => {
          list.replaceChildren();
          const add = (id: string, lead: HTMLElement) => {
            const b = h("div", { class: "opt-editor__item", attrs: { role: "option" } }, lead);
            if (chosen.has(id)) b.append(h("span", { class: "menu__check" }, iconNode("check")));
            b.addEventListener("click", () => toggle(id));
            list.append(b);
          };
          if (p.type === "person" || p.type === "created_by" || p.type === "last_edited_by") {
            add("me", h("span", { class: "menu__label", text: t("Me") }));
            for (const m of members) add(m.user_id, h("span", { class: "person" }, avatar(m.name, m.color, false, m.avatar), h("span", { text: m.name })));
          } else if (p.type === "relation") {
            for (const r of rels) add(r.id, h("span", { class: "menu__label", text: r.title || t("Untitled") }));
          } else for (const o of p.options ?? []) add(o.id, optionChip(o, p.type === "status"));
        };
        fill();
        if (p.type === "person" || p.type === "created_by" || p.type === "last_edited_by") {
          void ctx.engine.collections.people(ctx.workspaceId()).then((m) => {
            members = m;
            fill();
          });
        } else if (p.type === "relation" && p.collection) {
          void ctx.engine.api.collections.search(p.collection, "", 100).then((r) => {
            rels = r.rows;
            fill();
          });
        }
        panel.append(list);
      },
      { label: t("Choose values"), focus: false },
    );
  });
  return btn;
}

function ruleRow(ctx: DbCtx, f: Filter, remove: () => void, changed: () => void): HTMLElement {
  const row = h("div", { class: "filter__rule" });
  const build = () => {
    row.replaceChildren();
    const p = ctx.col().schema.find((x) => x.id === f.prop);
    const propBtn = propSelect(ctx, f.prop ?? "", (id) => {
      f.prop = id;
      const np = ctx.col().schema.find((x) => x.id === id);
      f.cond = np ? (CONDS[condKind(np, sampleOf(ctx, np))][0]?.[0] ?? "contains") : "contains";
      delete f.value;
      changed();
      build();
    }, (x) => x.type !== "button");
    row.append(propBtn);
    if (!p) return;
    const kind = condKind(p, sampleOf(ctx, p));
    const cond = h("select", { class: "input input--sm filter__cond", attrs: { "aria-label": t("Condition") } });
    for (const [k, label] of CONDS[kind]) cond.append(h("option", { attrs: { value: k }, text: t(label) }));
    cond.value = f.cond ?? CONDS[kind][0]?.[0] ?? "";
    if (!f.cond) f.cond = cond.value;
    cond.addEventListener("change", () => {
      f.cond = cond.value;
      if (f.cond === "within" || f.cond === "between") delete f.value;
      changed();
      build();
    });
    row.append(cond);
    const needs = CONDS[kind].find((c) => c[0] === f.cond)?.[2] ?? true;
    if (needs) row.append(filterValueControl(ctx, p, f, changed));
  };
  build();
  const rm = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Remove filter") } }, iconNode("close"));
  rm.addEventListener("click", remove);
  const wrap = h("div", { class: "filter__line" }, row, rm);
  return wrap;
}

function groupBox(ctx: DbCtx, g: Filter, depth: number, changed: () => void, removeSelf?: () => void): HTMLElement {
  const box = h("div", { class: "filter__group", attrs: { "data-depth": String(depth) } });
  const paint = () => {
    box.replaceChildren();
    (g.items ??= []).forEach((item, i) => {
      const lead = h("span", { class: "filter__lead" });
      if (i === 0) setText(lead, t("Where"));
      else if (i === 1) {
        const op = h("select", { class: "input input--sm", attrs: { "aria-label": t("Join") } });
        op.append(h("option", { attrs: { value: "and" }, text: t("And") }), h("option", { attrs: { value: "or" }, text: t("Or") }));
        op.value = g.op ?? "and";
        op.addEventListener("change", () => {
          g.op = op.value as "and" | "or";
          changed();
          paint();
        });
        lead.append(op);
      } else setText(lead, g.op === "or" ? t("Or") : t("And"));
      const remove = () => {
        g.items?.splice(i, 1);
        changed();
        paint();
      };
      const body = item.prop !== undefined || !item.items ? ruleRow(ctx, item, remove, changed) : groupBox(ctx, item, depth + 1, changed, remove);
      box.append(h("div", { class: "filter__item" }, lead, body));
    });
    const addRule = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a filter rule") }));
    addRule.addEventListener("click", () => {
      const first = ctx.col().schema[0];
      g.items = [...(g.items ?? []), { prop: first?.id ?? "title", cond: "contains", value: "" }];
      changed();
      paint();
    });
    const actions = h("div", { class: "filter__actions" }, addRule);
    if (depth < 2) {
      const addGroup = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a filter group") }));
      addGroup.addEventListener("click", () => {
        const first = ctx.col().schema[0];
        g.items = [...(g.items ?? []), { op: "and", items: [{ prop: first?.id ?? "title", cond: "contains", value: "" }] }];
        changed();
        paint();
      });
      actions.append(addGroup);
    }
    if (removeSelf) {
      const rm = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("trash"), h("span", { text: t("Remove group") }));
      rm.addEventListener("click", removeSelf);
      actions.append(rm);
    }
    box.append(actions);
  };
  paint();
  return box;
}

export function openFilterPanel(anchor: HTMLElement, ctx: DbCtx, viewId: string, seedProp?: string): void {
  const v = viewOf(ctx, viewId);
  if (!v) return;
  const root: Filter = structuredClone(v.config.filter ?? { op: "and", items: [] });
  if (!root.items) root.items = [];
  if (root.prop) {
    const single = { ...root };
    root.items = [single];
    delete root.prop;
    delete root.cond;
    delete root.value;
    root.op = "and";
  }
  if (seedProp) {
    const p = ctx.col().schema.find((x) => x.id === seedProp);
    if (p) root.items.push({ prop: p.id, cond: CONDS[condKind(p, sampleOf(ctx, p))][0]?.[0] ?? "contains" });
  }
  const changed = () => saveConfig(ctx, viewId, (cfg) => ({ ...cfg, filter: root.items?.length ? structuredClone(root) : undefined } as ViewConfig), 350);
  if (seedProp) changed();
  openPanel(
    anchor,
    (panel) => {
      panel.classList.add("filter-panel");
      panel.append(h("div", { class: "dbpanel__pad" }, groupBox(ctx, root, 0, changed)));
    },
    { label: t("Filters"), focus: false },
  );
}

export function openSortPanel(anchor: HTMLElement, ctx: DbCtx, viewId: string): void {
  const v = viewOf(ctx, viewId);
  if (!v) return;
  const sorts: SortKey[] = structuredClone(v.config.sorts ?? []);
  const changed = () => saveConfig(ctx, viewId, (cfg) => ({ ...cfg, sorts: sorts.length ? [...sorts] : undefined } as ViewConfig));
  openPanel(
    anchor,
    (panel) => {
      const list = h("div", { class: "dbpanel__pad dbpanel__stack" });
      const paint = () => {
        list.replaceChildren();
        sorts.forEach((s, i) => {
          const dir = h("select", { class: "input input--sm", attrs: { "aria-label": t("Direction") } });
          dir.append(h("option", { attrs: { value: "asc" }, text: t("Ascending") }), h("option", { attrs: { value: "desc" }, text: t("Descending") }));
          dir.value = s.dir;
          dir.addEventListener("change", () => {
            s.dir = dir.value as "asc" | "desc";
            changed();
          });
          const up = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Move up") } }, iconNode("arrow-up"));
          up.disabled = i === 0;
          up.addEventListener("click", () => {
            const prev = sorts[i - 1];
            if (!prev) return;
            sorts[i - 1] = s;
            sorts[i] = prev;
            changed();
            paint();
          });
          const rm = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Remove sort") } }, iconNode("close"));
          rm.addEventListener("click", () => {
            sorts.splice(i, 1);
            changed();
            paint();
          });
          list.append(h("div", { class: "dbpanel__row" }, propSelect(ctx, s.prop, (id) => {
            s.prop = id;
            changed();
          }), dir, up, rm));
        });
        const add = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add a sort") }));
        add.addEventListener("click", () => {
          const used = new Set(sorts.map((s) => s.prop));
          const p = ctx.col().schema.find((x) => !used.has(x.id)) ?? ctx.col().schema[0];
          if (!p) return;
          sorts.push({ prop: p.id, dir: "asc" });
          changed();
          paint();
        });
        list.append(add);
        if (sorts.length) {
          const clear = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("trash"), h("span", { text: t("Delete sorts") }));
          clear.addEventListener("click", () => {
            sorts.length = 0;
            changed();
            paint();
          });
          list.append(clear);
        }
      };
      paint();
      panel.append(list);
    },
    { label: t("Sorts"), focus: false },
  );
}

export function groupable(p: Property): boolean {
  return !["files", "button", "place"].includes(p.type);
}

export function openGroupPanel(anchor: HTMLElement, ctx: DbCtx, viewId: string, which: "group" | "sub_group" = "group"): void {
  const v = viewOf(ctx, viewId);
  if (!v) return;
  let g: Grouping | undefined = structuredClone(v.config[which]);
  const changed = () => saveConfig(ctx, viewId, (cfg) => ({ ...cfg, [which]: g?.prop ? { ...g } : undefined } as ViewConfig), 100);
  openPanel(
    anchor,
    (panel) => {
      const pad = h("div", { class: "dbpanel__pad dbpanel__stack" });
      const paint = () => {
        pad.replaceChildren();
        pad.append(
          h(
            "label",
            { class: "field field--compact" },
            h("span", { class: "field__label", text: which === "group" ? t("Group by") : t("Sub-group by") }),
            propSelect(ctx, g?.prop ?? "", (id) => {
              g = { prop: id };
              changed();
              paint();
            }, groupable),
          ),
        );
        if (!g?.prop) return;
        const p = ctx.col().schema.find((x) => x.id === g?.prop);
        if (p && ["date", "created_time", "last_edited_time", "formula", "rollup"].includes(p.type)) {
          const by = h("select", { class: "input input--sm", attrs: { "aria-label": t("Date by") } });
          for (const [k, l] of [["relative", "Relative"], ["day", "Day"], ["week", "Week"], ["month", "Month"], ["year", "Year"]]) by.append(h("option", { attrs: { value: k ?? "" }, text: t(l ?? "") }));
          by.value = g.by ?? "relative";
          by.addEventListener("change", () => {
            if (g) g.by = by.value;
            changed();
          });
          pad.append(h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Date by") }), by));
        }
        if (p?.type === "status") {
          const by = h("select", { class: "input input--sm", attrs: { "aria-label": t("Status by") } });
          by.append(h("option", { attrs: { value: "" }, text: t("Option") }), h("option", { attrs: { value: "group" }, text: t("Group") }));
          by.value = g.by ?? "";
          by.addEventListener("change", () => {
            if (g && by.value) g.by = by.value;
            else if (g) delete g.by;
            changed();
          });
          pad.append(h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Show") }), by));
        }
        if (p?.type === "number") {
          const step = h("input", { class: "input input--sm", attrs: { type: "number", "aria-label": t("Range size") } });
          step.value = String(g.step ?? 10);
          step.addEventListener("change", () => {
            if (g) g.step = Number(step.value) || 10;
            changed();
          });
          pad.append(h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Range size") }), step));
        }
        const sort = h("select", { class: "input input--sm", attrs: { "aria-label": t("Group order") } });
        sort.append(h("option", { attrs: { value: "asc" }, text: t("Ascending") }), h("option", { attrs: { value: "desc" }, text: t("Descending") }), h("option", { attrs: { value: "manual" }, text: t("Manual") }));
        sort.value = g.sort ?? "asc";
        sort.addEventListener("change", () => {
          if (g) g.sort = sort.value as "asc" | "desc" | "manual";
          changed();
        });
        const hide = h("label", { class: "dpick__toggle" }, h("span", { text: t("Hide empty groups") }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
        const hideIn = hide.querySelector("input") as HTMLInputElement;
        hideIn.checked = !!g.hide_empty;
        hideIn.addEventListener("change", () => {
          if (g) g.hide_empty = hideIn.checked;
          changed();
        });
        const rm = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("trash"), h("span", { text: t("Remove grouping") }));
        rm.addEventListener("click", () => {
          g = undefined;
          changed();
          paint();
        });
        pad.append(h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: t("Sort groups") }), sort), hide, rm);
      };
      paint();
      panel.append(pad);
    },
    { label: which === "group" ? t("Group") : t("Sub-group"), focus: false },
  );
}

export function orderedProps(ctx: DbCtx, cfg: ViewConfig): { p: Property; vp: ViewProp }[] {
  const schema = ctx.col().schema;
  const out: { p: Property; vp: ViewProp }[] = [];
  const seen = new Set<string>();
  for (const vp of cfg.props ?? []) {
    const p = schema.find((x) => x.id === vp.id);
    if (p && !seen.has(p.id)) {
      out.push({ p, vp });
      seen.add(p.id);
    }
  }
  for (const p of schema) if (!seen.has(p.id)) out.push({ p, vp: { id: p.id } });
  const title = out.findIndex((x) => x.p.type === "title");
  if (title > 0) out.unshift(...out.splice(title, 1));
  return out;
}

export function openPropsPanel(anchor: HTMLElement, ctx: DbCtx, viewId: string, defaultHidden: (p: Property) => boolean): void {
  const v = viewOf(ctx, viewId);
  if (!v) return;
  openPanel(
    anchor,
    (panel) => {
      const list = h("div", { class: "dbpanel__pad props-list" });
      const paint = () => {
        const cur = viewOf(ctx, viewId);
        if (!cur) return;
        list.replaceChildren(h("div", { class: "menu__heading", text: t("Properties") }));
        const rows = orderedProps(ctx, cur.config);
        rows.forEach(({ p, vp }, i) => {
          const hidden = vp.hidden ?? defaultHidden(p);
          const eye = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": hidden ? t("Show {name}", { name: p.name }) : t("Hide {name}", { name: p.name }), "aria-pressed": hidden ? "false" : "true" } }, iconNode(hidden ? "eye-off" : "eye"));
          eye.disabled = p.type === "title";
          eye.addEventListener("click", () => {
            saveConfig(ctx, viewId, (cfg) => {
              const ordered = orderedProps(ctx, cfg).map((x) => ({ ...x.vp }));
              const me = ordered.find((x) => x.id === p.id);
              if (me) me.hidden = !hidden;
              return { ...cfg, props: ordered };
            }, 100);
            setTimeout(paint, 0);
          });
          const up = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Move up") } }, iconNode("arrow-up"));
          up.disabled = i <= 1 || p.type === "title";
          up.addEventListener("click", () => {
            saveConfig(ctx, viewId, (cfg) => {
              const ordered = orderedProps(ctx, cfg).map((x) => ({ ...x.vp }));
              const k = ordered.findIndex((x) => x.id === p.id);
              if (k > 1) [ordered[k - 1], ordered[k]] = [ordered[k] as ViewProp, ordered[k - 1] as ViewProp];
              return { ...cfg, props: ordered };
            }, 100);
            setTimeout(paint, 0);
          });
          const edit = h("button", { class: "props-list__name", attrs: { type: "button" } }, iconNode(TYPE_INFO[p.type].icon), h("span", { text: p.name }));
          edit.addEventListener("click", () => openPropertyEditor(edit, p, ctx));
          list.append(h("div", { class: "props-list__row" }, edit, up, eye));
        });
        if (ctx.canDesign()) {
          const add = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("New property") }));
          add.addEventListener("click", () => openAddProperty(add, ctx, "", () => setTimeout(paint, 50)));
          list.append(add);
        }
      };
      paint();
      const off = ctx.engine.collections.subscribe(ctx.colId, () => {
        if (list.isConnected) paint();
        else off();
      });
      panel.append(list);
    },
    { label: t("Properties"), focus: false },
  );
}
