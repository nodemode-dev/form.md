import { t } from "../../engine/i18n";
import type { Collection, Property, PropOption, PropType, SchemaOp } from "../../engine/types";
import { h, setText } from "../../ui/dom";
import { iconNode } from "../../ui/icons";
import { type MenuEntry, openMenu } from "../../ui/menu";
import { pageIcon } from "../../ui/pageicon";
import { AUTOFILL_TYPES, autofillChanged, autofillEditor, runAutofill } from "./autofill";
import type { DbCtx } from "./cells";
import { openPanel } from "./panel";
import { DATE_FORMATS, NUMBER_FORMATS, OPTION_COLORS, ROLLUP_FNS, TYPE_INFO, typeEntries, typeLabel } from "./props";

function schemaOp(ctx: DbCtx, ops: SchemaOp[]): Promise<void> {
  return ctx.engine.collections.schema(ctx.colId, ops).catch((e) => {
    ctx.engine.reportError(e);
    throw e;
  });
}

function uniqueName(col: Collection, base: string): string {
  let name = base;
  for (let i = 2; col.schema.some((p) => p.name.toLowerCase() === name.toLowerCase()); i++) name = `${base} ${i}`;
  return name;
}

export function openAddProperty(anchor: HTMLElement, ctx: DbCtx, after = "", onAdded?: (id: string) => void): void {
  const entries: MenuEntry[] = [{ kind: "heading", label: t("Property type") }];
  entries.push(
    ...typeEntries((type) => ({
      label: typeLabel(type),
      icon: TYPE_INFO[type].icon,
      run: () => {
        const col = ctx.col();
        const prop: Partial<Property> = { name: uniqueName(col, typeLabel(type)), type };
        if (type === "relation") prop.collection = ctx.colId;
        const op: SchemaOp = { op: "add", prop };
        if (after) op.after = after;
        void schemaOp(ctx, [op])
          .then(() => {
            const added = ctx.engine.collections.get(ctx.colId)?.schema.find((p) => !col.schema.some((q) => q.id === p.id));
            if (added) {
              onAdded?.(added.id);
              if (["select", "multi_select", "status", "relation", "rollup", "formula", "number"].includes(type)) setTimeout(() => openPropertyEditor(anchor, added, ctx), 0);
              if (type === "button") void import("./buttonprop").then((m) => m.editButtonProperty(ctx, added));
            }
          })
          .catch(() => undefined);
      },
    })),
  );
  openMenu(anchor, entries, { side: "bottom", align: "end", label: t("New property") });
}

export function openPropertyEditor(anchor: HTMLElement, p0: Property, ctx: DbCtx): void {
  openPanel(
    anchor,
    (panel, close) => {
      panel.classList.add("prop-editor");
      let p: Property = { ...p0, options: p0.options ? p0.options.map((o) => ({ ...o })) : undefined } as Property;
      let twoWay = !!p0.pair;
      let pairName = "";
      const body = h("div", { class: "prop-editor__body" });
      const name = h("input", { class: "input input--sm", attrs: { type: "text", "aria-label": t("Property name") } });
      name.value = p.name;
      const typeBtn = h("button", { class: "select-btn", attrs: { type: "button", "aria-haspopup": "menu" } });
      const paintType = () => typeBtn.replaceChildren(iconNode(TYPE_INFO[p.type].icon), h("span", { text: typeLabel(p.type) }), iconNode("chevron-down"));
      paintType();
      const desc = h("textarea", { class: "input input--sm", attrs: { rows: "2", placeholder: t("Description"), "aria-label": t("Description") } });
      desc.value = p.description ?? "";
      const save = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "button" }, text: t("Done") });
      const status = h("p", { class: "form-status", attrs: { role: "alert" } });
      const head = h("div", { class: "dbpanel__pad dbpanel__stack" }, name);
      if (p.type !== "title") head.append(typeBtn);
      panel.append(head, body, h("div", { class: "dbpanel__pad dbpanel__stack" }, desc, status, h("div", { class: "dbpanel__row dbpanel__row--end" }, save)));

      typeBtn.addEventListener("click", () => {
        openMenu(
          typeBtn,
          typeEntries((type) => ({
            label: typeLabel(type),
            icon: TYPE_INFO[type].icon,
            checked: type === p.type,
            run: () => {
              p = { ...p, type };
              if (type === "relation" && !p.collection) p.collection = ctx.colId;
              paintType();
              build();
            },
          })),
          { side: "bottom", align: "start", label: t("Property type") },
        );
      });

      const field = (label: string, control: HTMLElement) => h("label", { class: "field field--compact" }, h("span", { class: "field__label", text: label }), control);
      const selectOf = (opts: [string, string][], value: string, onChange: (v: string) => void) => {
        const s = h("select", { class: "input input--sm" });
        for (const [v, l] of opts) s.append(h("option", { attrs: { value: v }, text: t(l) }));
        s.value = value;
        s.addEventListener("change", () => onChange(s.value));
        return s;
      };

      const optionsEditor = () => {
        const wrap = h("div", { class: "opts" });
        const paint = () => {
          wrap.replaceChildren(h("div", { class: "menu__heading", text: t("Options") }));
          (p.options ?? []).forEach((o: PropOption, i: number) => {
            const nameIn = h("input", { class: "input input--sm opts__name", attrs: { type: "text", "aria-label": t("Option name") } });
            nameIn.value = o.name;
            nameIn.addEventListener("change", () => {
              o.name = nameIn.value;
            });
            const color = h("select", { class: "input input--sm opts__color", attrs: { "aria-label": t("Colour") } });
            for (const c of OPTION_COLORS) color.append(h("option", { attrs: { value: c }, text: t(c.charAt(0).toUpperCase() + c.slice(1)) }));
            color.value = o.color;
            color.addEventListener("change", () => {
              o.color = color.value;
              swatch.dataset.color = o.color;
            });
            const swatch = h("span", { class: "opt-swatch", attrs: { "data-color": o.color } });
            const up = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Move up") } }, iconNode("arrow-up"));
            const del = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Delete option") } }, iconNode("close"));
            up.disabled = i === 0;
            up.addEventListener("click", () => {
              const list = p.options ?? [];
              const prev = list[i - 1];
              if (!prev) return;
              list[i - 1] = o;
              list[i] = prev;
              paint();
            });
            del.addEventListener("click", () => {
              p.options = (p.options ?? []).filter((x) => x !== o);
              paint();
            });
            const row = h("div", { class: "opts__row" }, swatch, nameIn, color);
            if (p.type === "status") {
              const g = selectOf([["todo", "To do"], ["doing", "In progress"], ["done", "Complete"]], o.group ?? "todo", (v) => {
                o.group = v as "todo" | "doing" | "done";
              });
              g.classList.add("opts__group");
              row.append(g);
            }
            row.append(up, del);
            wrap.append(row);
          });
          const add = h("button", { class: "link-btn", attrs: { type: "button" } }, iconNode("plus"), h("span", { text: t("Add an option") }));
          add.addEventListener("click", () => {
            const list = p.options ?? (p.options = []);
            list.push({ id: "", name: t("Option {n}", { n: list.length + 1 }), color: OPTION_COLORS[(list.length + 1) % OPTION_COLORS.length] ?? "default", ...(p.type === "status" ? { group: "todo" as const } : {}) });
            paint();
            (wrap.querySelectorAll<HTMLInputElement>(".opts__name")[list.length - 1])?.select();
          });
          wrap.append(add);
        };
        paint();
        return wrap;
      };

      const collections = () => {
        const all = ctx.engine.pages.isLoaded(ctx.workspaceId()) ? ctx.engine.pages.all(ctx.workspaceId()) : [];
        return all.filter((x) => x.kind === "collection");
      };

      const formulaEditor = () => {
        const area = h("textarea", { class: "input input--mono formula-input", attrs: { rows: "4", spellcheck: "false", "aria-label": t("Formula"), placeholder: 'prop("Price") * prop("Quantity")' } });
        area.value = p.expr ?? "";
        const msg = h("p", { class: "field__hint formula-msg" });
        const propsList = h("div", { class: "formula-props" });
        for (const x of ctx.col().schema) {
          if (x.id === p.id) continue;
          const b = h("button", { class: "chip chip--btn", attrs: { type: "button" } }, iconNode(TYPE_INFO[x.type].icon), h("span", { text: x.name }));
          b.addEventListener("click", () => {
            const ins = `prop("${x.name.replace(/"/g, '\\"')}")`;
            const s = area.selectionStart;
            area.value = area.value.slice(0, s) + ins + area.value.slice(area.selectionEnd);
            area.focus();
            area.setSelectionRange(s + ins.length, s + ins.length);
            check();
          });
          propsList.append(b);
        }
        let seq = 0;
        let timer: ReturnType<typeof setTimeout> | undefined;
        const check = () => {
          p.expr = area.value;
          clearTimeout(timer);
          timer = setTimeout(async () => {
            const mine = ++seq;
            if (!area.value.trim()) {
              setText(msg, "");
              return;
            }
            try {
              const r = await ctx.engine.api.collections.formula(ctx.colId, area.value);
              if (mine !== seq) return;
              msg.dataset.tone = r.ok ? "ok" : "error";
              setText(msg, r.ok ? t("The formula is valid.") : `${r.error ?? ""} (${t("at character {n}", { n: r.pos + 1 })})`);
            } catch {
              return;
            }
          }, 250);
        };
        area.addEventListener("input", check);
        check();
        const help = h("p", { class: "field__hint", text: t("Use prop(\"Name\") for a property; functions include if, ifs, let, map, filter, sum, dateAdd, dateBetween, formatDate and more.") });
        return h("div", { class: "dbpanel__stack" }, area, msg, propsList, help);
      };

      const build = () => {
        body.replaceChildren();
        const pad = h("div", { class: "dbpanel__pad dbpanel__stack" });
        switch (p.type) {
          case "number":
          case "formula":
          case "rollup":
            if (p.type === "formula") pad.append(formulaEditor());
            if (p.type === "rollup") {
              const rels = ctx.col().schema.filter((x) => x.type === "relation");
              const relSel = selectOf([["", "Choose a relation"], ...rels.map((x) => [x.id, x.name] as [string, string])], p.relation ?? "", (v) => {
                p.relation = v;
                p.target = "";
                build();
              });
              pad.append(field(t("Relation"), relSel));
              const rel = rels.find((x) => x.id === p.relation);
              if (rel?.collection) {
                const targetSel = h("select", { class: "input input--sm" });
                targetSel.append(h("option", { attrs: { value: "" }, text: t("Loading") }));
                void ctx.engine.collections.load(rel.collection).then((tc) => {
                  targetSel.replaceChildren();
                  for (const x of tc.schema) targetSel.append(h("option", { attrs: { value: x.id }, text: x.name }));
                  targetSel.value = p.target || "title";
                  p.target = targetSel.value;
                }, ctx.engine.reportError);
                targetSel.addEventListener("change", () => {
                  p.target = targetSel.value;
                });
                pad.append(field(t("Property"), targetSel));
              }
              pad.append(field(t("Calculate"), selectOf(ROLLUP_FNS, p.fn ?? "show_original", (v) => (p.fn = v))));
            }
            pad.append(field(t("Number format"), selectOf(NUMBER_FORMATS, p.format ?? "", (v) => (p.format = v))));
            {
              const prec = selectOf([["0", "Default"], ["1", "1"], ["2", "2"], ["3", "3"], ["4", "4"]], String(p.precision ?? 0), (v) => (p.precision = Number(v)));
              pad.append(field(t("Decimal places"), prec));
            }
            pad.append(
              field(
                t("Show as"),
                selectOf([["", "Number"], ["bar", "Bar"], ["ring", "Ring"]], p.display ?? "", (v) => {
                  p.display = v as "" | "bar" | "ring";
                  build();
                }),
              ),
            );
            if (p.display) {
              const max = h("input", { class: "input input--sm", attrs: { type: "number", "aria-label": t("Divide by") } });
              max.value = String(p.max ?? 100);
              max.addEventListener("change", () => (p.max = Number(max.value) || 100));
              pad.append(field(t("Divide by"), max), field(t("Colour"), selectOf(OPTION_COLORS.map((c) => [c, c.charAt(0).toUpperCase() + c.slice(1)]), p.color ?? "default", (v) => (p.color = v))));
            }
            break;
          case "select":
          case "multi_select":
          case "status":
            pad.append(optionsEditor());
            break;
          case "date":
          case "created_time":
          case "last_edited_time":
            pad.append(
              field(t("Date format"), selectOf(DATE_FORMATS, p.format ?? "", (v) => (p.format = v))),
              field(t("Time format"), selectOf([["", "Default"], ["12", "12 hour"], ["24", "24 hour"]], p.time_format ?? "", (v) => (p.time_format = v as "" | "12" | "24"))),
            );
            break;
          case "relation": {
            const cols = collections();
            const sel = h("select", { class: "input input--sm" });
            for (const c of cols) sel.append(h("option", { attrs: { value: c.id }, text: c.id === ctx.colId ? t("This database") : c.title || t("Untitled") }));
            if (!cols.some((c) => c.id === ctx.colId)) sel.prepend(h("option", { attrs: { value: ctx.colId }, text: t("This database") }));
            sel.value = p.collection || ctx.colId;
            sel.addEventListener("change", () => {
              p.collection = sel.value;
              build();
            });
            const limit = selectOf([["0", "No limit"], ["1", "1 page"]], String(p.limit ?? 0), (v) => (p.limit = Number(v)));
            const tw = h("label", { class: "dpick__toggle" }, h("span", { text: t("Show on the related database") }), h("input", { attrs: { type: "checkbox", role: "switch" } }));
            const twIn = tw.querySelector("input") as HTMLInputElement;
            twIn.checked = twoWay;
            const pn = h("input", { class: "input input--sm", attrs: { type: "text", placeholder: t("Name on the related database"), "aria-label": t("Related property name") } });
            pn.hidden = !twoWay || !!p0.pair;
            twIn.addEventListener("change", () => {
              twoWay = twIn.checked;
              pn.hidden = !twoWay || !!p0.pair;
            });
            pn.addEventListener("change", () => (pairName = pn.value.trim()));
            pad.append(field(t("Related to"), sel), field(t("Limit"), limit), tw, pn);
            const icon = cols.find((c) => c.id === sel.value);
            if (icon) pad.prepend(h("div", { class: "prop-editor__target" }, pageIcon(icon.icon, "database"), h("span", { text: icon.title || t("Untitled") })));
            break;
          }
          case "button": {
            const n = p.actions?.length ?? 0;
            const edit = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("bolt"), h("span", { text: n ? t("Edit {n} actions", { n }) : t("Choose what it does") }));
            edit.addEventListener("click", () => {
              close();
              void import("./buttonprop").then((m) => m.editButtonProperty(ctx, { ...p, name: name.value.trim() || p.name }));
            });
            pad.append(edit);
            break;
          }
          case "unique_id": {
            const pre = h("input", { class: "input input--sm", attrs: { type: "text", maxlength: "10", "aria-label": t("Prefix") } });
            pre.value = p.prefix ?? "";
            pre.addEventListener("input", () => (p.prefix = pre.value));
            pad.append(field(t("Prefix"), pre));
            break;
          }
        }
        if (AUTOFILL_TYPES.has(p.type)) pad.append(autofillEditor(p));
        if (pad.childElementCount) body.append(pad);
      };
      build();

      save.addEventListener("click", () => {
        p.name = name.value.trim() || p0.name;
        p.description = desc.value.trim();
        if (p.options) p.options = p.options.filter((o) => o.name.trim());
        const op: SchemaOp = { op: "update", id: p0.id, prop: p };
        if (p.type === "relation") {
          op.two_way = twoWay;
          if (pairName) op.pair_name = pairName;
        }
        save.disabled = true;
        if (!AUTOFILL_TYPES.has(p.type)) p.autofill = null;
        ctx.engine.collections
          .schema(ctx.colId, [op])
          .then(() => {
            close();
            if (autofillChanged(p0, p)) runAutofill(ctx, p);
          })
          .catch((e: unknown) => {
            save.disabled = false;
            setText(status, (e as Error).message);
          });
      });
      name.addEventListener("keydown", (e) => {
        if (e.key === "Enter") save.click();
      });
    },
    { label: t("Edit property"), side: "bottom", align: "start" },
  );
}

export function propertyMenu(anchor: HTMLElement, p: Property, ctx: DbCtx, extra: MenuEntry[]): void {
  const entries: MenuEntry[] = [
    { kind: "heading", label: p.name },
    { label: t("Edit property"), icon: "settings", run: () => openPropertyEditor(anchor, p, ctx) },
    ...extra,
  ];
  if (p.autofill && AUTOFILL_TYPES.has(p.type) && ctx.col().can_write) entries.push({ label: t("Fill all rows with AI"), icon: "ai-chip", run: () => runAutofill(ctx, p) });
  if (p.type !== "title") {
    entries.push(
      { kind: "separator" },
      {
        label: t("Duplicate property"),
        icon: "copy",
        run: () => {
          const { id: _id, pair: _pair, ...rest } = p;
          void schemaOp(ctx, [{ op: "add", after: p.id, prop: { ...rest, name: uniqueName(ctx.col(), `${p.name} copy`) } }]).catch(() => undefined);
        },
      },
      { label: t("Delete property"), icon: "trash", danger: true, run: () => void schemaOp(ctx, [{ op: "delete", id: p.id }]).catch(() => undefined) },
    );
  }
  openMenu(anchor, entries, { side: "bottom", align: "start", label: p.name });
}

export type { PropType };
