import { t } from "../../engine/i18n";
import type { Member, Property, PropOption, RowData } from "../../engine/types";
import { avatar } from "../../ui/avatar";
import { h, setAttr } from "../../ui/dom";
import { errorMessage } from "../../ui/errors";
import { iconNode } from "../../ui/icons";
import { pageIcon } from "../../ui/pageicon";
import { pickFile } from "../../ui/pickers";
import { toast } from "../../ui/toast";
import { type DbCtx, optionChip } from "./cells";
import { datePicker } from "./datepicker";
import { openPanel } from "./panel";
import { asList, computed, type DateValue, OPTION_COLORS } from "./props";

export interface EditOptions {
  onClose?: () => void;
  initial?: string;
}

function setValue(ctx: DbCtx, row: RowData, p: Property, value: unknown, debounce = 0): void {
  ctx.engine.collections.setValue(ctx.colId, row.id, p.id, value, debounce);
}

export function editCell(anchor: HTMLElement, p: Property, row: RowData, ctx: DbCtx, o: EditOptions = {}): void {
  if (!ctx.canWrite() || computed(p.type)) {
    o.onClose?.();
    return;
  }
  switch (p.type) {
    case "checkbox":
      setValue(ctx, row, p, row.values[p.id] !== true);
      o.onClose?.();
      return;
    case "title":
    case "text":
    case "url":
    case "email":
    case "phone":
    case "number":
      return textEditor(anchor, p, row, ctx, o);
    case "select":
    case "status":
    case "multi_select":
      return optionEditor(anchor, p, row, ctx, o);
    case "date":
      return dateEditor(anchor, p, row, ctx, o);
    case "person":
      return personEditor(anchor, p, row, ctx, o);
    case "relation":
      return relationEditor(anchor, p, row, ctx, o);
    case "files":
      return filesEditor(anchor, p, row, ctx, o);
    case "place":
      return placeEditor(anchor, p, row, ctx, o);
  }
  o.onClose?.();
}

function textEditor(anchor: HTMLElement, p: Property, row: RowData, ctx: DbCtx, o: EditOptions): void {
  const current = p.type === "title" ? row.title : row.values[p.id];
  const initial = o.initial !== undefined ? o.initial : current === undefined || current === null ? "" : String(current);
  const multi = p.type === "text";
  const field = multi
    ? h("textarea", { class: "cell-input", attrs: { rows: "1", "aria-label": p.name, spellcheck: "true" } })
    : h("input", { class: "cell-input", attrs: { type: p.type === "number" ? "text" : p.type === "email" ? "email" : "text", inputmode: p.type === "number" ? "decimal" : p.type === "phone" ? "tel" : "text", "aria-label": p.name } });
  field.value = initial;
  let cancelled = false;
  const r = anchor.getBoundingClientRect();
  const handle = openPanel(
    () => (anchor.isConnected ? { x: r.left, y: r.top, width: r.width, height: r.height } : null),
    (panel) => {
      panel.classList.add("cell-editor");
      panel.style.minWidth = `${Math.max(r.width, 220)}px`;
      panel.append(field);
    },
    {
      label: p.name,
      side: "bottom",
      align: "start",
      onClose: () => {
        if (!cancelled) commit();
        o.onClose?.();
      },
    },
  );
  handle.el.style.transform = `translateY(-${Math.round(r.height + 4)}px)`;
  const grow = () => {
    if (!multi) return;
    field.style.height = "auto";
    field.style.height = `${Math.min(field.scrollHeight, 320)}px`;
  };
  grow();
  field.focus();
  if (p.type === "number" && o.initial === undefined) field.select();
  else field.setSelectionRange(field.value.length, field.value.length);
  const commit = () => {
    const v = field.value;
    if (v === initial && o.initial === undefined) return;
    if (p.type === "number") {
      const n = v.trim() === "" ? null : Number(v.replace(/,/g, ""));
      if (n !== null && Number.isNaN(n)) {
        toast({ id: "cell", tone: "warning", message: t("{name} needs a number", { name: p.name }) });
        return;
      }
      setValue(ctx, row, p, n);
      return;
    }
    setValue(ctx, row, p, p.type === "title" ? v.replace(/\n/g, " ") : v);
  };
  field.addEventListener("input", grow);
  field.addEventListener("keydown", (e) => {
    const ev = e as KeyboardEvent;
    if (ev.key === "Enter" && !(multi && ev.shiftKey)) {
      ev.preventDefault();
      handle.close();
    } else if (ev.key === "Escape") {
      ev.preventDefault();
      ev.stopPropagation();
      cancelled = true;
      handle.close();
    } else if (ev.key === "Tab") {
      ev.preventDefault();
      handle.close();
    }
  });
}

function editOption(anchor: HTMLElement, p: Property, opt: PropOption, ctx: DbCtx, after: () => void): void {
  openPanel(
    anchor,
    (panel, close) => {
      const name = h("input", { class: "input input--sm", attrs: { type: "text", "aria-label": t("Option name") } });
      name.value = opt.name;
      const colors = h("div", { class: "menu", attrs: { role: "listbox", "aria-label": t("Colour") } });
      for (const c of OPTION_COLORS) {
        const b = h("button", { class: "menu__item", attrs: { type: "button", "data-color": c } }, h("span", { class: "opt-swatch", attrs: { "data-color": c } }), h("span", { class: "menu__label", text: t(c.charAt(0).toUpperCase() + c.slice(1)) }));
        if (opt.color === c) b.append(h("span", { class: "menu__check" }, iconNode("check")));
        b.addEventListener("click", () => {
          save({ color: c });
          close();
        });
        colors.append(b);
      }
      const del = h("button", { class: "menu__item", attrs: { type: "button", "data-danger": "true" } }, iconNode("trash"), h("span", { class: "menu__label", text: t("Delete option") }));
      del.addEventListener("click", () => {
        const options = (p.options ?? []).filter((x) => x.id !== opt.id);
        void ctx.engine.collections.schema(ctx.colId, [{ op: "update", id: p.id, prop: { ...p, options } }]).then(after, ctx.engine.reportError);
        close();
      });
      const save = (change: Partial<PropOption>) => {
        const options = (p.options ?? []).map((x) => (x.id === opt.id ? { ...x, ...change } : x));
        void ctx.engine.collections.schema(ctx.colId, [{ op: "update", id: p.id, prop: { ...p, options } }]).then(after, ctx.engine.reportError);
      };
      name.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          close();
        }
      });
      name.addEventListener("change", () => {
        if (name.value.trim() && name.value.trim() !== opt.name) save({ name: name.value.trim() });
      });
      if (p.type === "status") {
        const group = h("select", { class: "input input--sm", attrs: { "aria-label": t("Group") } });
        for (const [g, label] of [["todo", t("To do")], ["doing", t("In progress")], ["done", t("Complete")]] as const) group.append(h("option", { attrs: { value: g }, text: label }));
        group.value = opt.group ?? "todo";
        group.addEventListener("change", () => save({ group: group.value as "todo" | "doing" | "done" }));
        panel.append(h("div", { class: "dbpanel__pad" }, name, group), h("div", { class: "menu__sep" }), del, h("div", { class: "menu__sep" }), h("div", { class: "menu__heading", text: t("Colour") }), colors);
      } else panel.append(h("div", { class: "dbpanel__pad" }, name), del, h("div", { class: "menu__sep" }), h("div", { class: "menu__heading", text: t("Colour") }), colors);
    },
    { label: t("Edit option"), side: "right", align: "start" },
  );
}

function optionEditor(anchor: HTMLElement, p0: Property, row0: RowData, ctx: DbCtx, o: EditOptions): void {
  const multi = p0.type === "multi_select";
  const prop = () => ctx.engine.collections.prop(ctx.colId, p0.id) ?? p0;
  const selected = () => {
    const r = ctx.session.rows.get(row0.id) ?? row0;
    return asList(r.values[p0.id]);
  };
  openPanel(
    anchor,
    (panel, close) => {
      panel.classList.add("opt-editor");
      const chips = h("div", { class: "opt-editor__chips" });
      const search = h("input", { class: "opt-editor__search", attrs: { type: "text", placeholder: multi ? t("Search or create options") : t("Search or create an option"), "aria-label": t("Search options") } });
      if (o.initial) search.value = o.initial;
      const list = h("div", { class: "opt-editor__list", attrs: { role: "listbox" } });
      let sel = 0;
      const items = (): { kind: "opt" | "new"; opt?: PropOption; name: string }[] => {
        const q = search.value.trim().toLowerCase();
        const opts = (prop().options ?? []).filter((x) => !q || x.name.toLowerCase().includes(q));
        const out: { kind: "opt" | "new"; opt?: PropOption; name: string }[] = opts.map((x) => ({ kind: "opt" as const, opt: x, name: x.name }));
        if (q && !(prop().options ?? []).some((x) => x.name.toLowerCase() === q)) out.push({ kind: "new", name: search.value.trim() });
        return out;
      };
      const pick = (it: { kind: "opt" | "new"; opt?: PropOption; name: string }) => {
        const cur = selected();
        let next: unknown;
        if (it.kind === "new") {
          next = multi ? [...cur, { new: it.name }] : { new: it.name };
          search.value = "";
          setValue(ctx, row0, p0, next);
          if (!multi) close();
          setTimeout(paint, 250);
          return;
        }
        const id = it.opt?.id ?? "";
        if (multi) next = cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id];
        else next = cur[0] === id ? null : id;
        setValue(ctx, row0, p0, next);
        search.value = "";
        if (!multi) close();
        else paint();
      };
      const paint = () => {
        const p = prop();
        const cur = selected();
        chips.replaceChildren();
        for (const id of cur) {
          const opt = p.options?.find((x) => x.id === id);
          if (!opt) continue;
          const chip = optionChip(opt, p.type === "status");
          const x = h("button", { class: "opt__remove", attrs: { type: "button", "aria-label": t("Remove {name}", { name: opt.name }) } }, iconNode("close"));
          x.addEventListener("click", () => {
            setValue(ctx, row0, p0, multi ? selected().filter((y) => y !== id) : null);
            setTimeout(paint, 0);
          });
          chip.append(x);
          chips.append(chip);
        }
        chips.append(search);
        list.replaceChildren();
        const its = items();
        sel = Math.min(sel, Math.max(0, its.length - 1));
        let lastGroup = "";
        its.forEach((it, i) => {
          if (p.type === "status" && it.opt && it.opt.group !== lastGroup) {
            lastGroup = it.opt.group ?? "todo";
            list.append(h("div", { class: "menu__heading", text: lastGroup === "todo" ? t("To do") : lastGroup === "doing" ? t("In progress") : t("Complete") }));
          }
          const b = h("div", { class: "opt-editor__item", attrs: { role: "option", "aria-selected": i === sel ? "true" : "false" } });
          if (it.kind === "new") b.append(h("span", { class: "opt-editor__create", text: t("Create") }), optionChip({ id: "", name: it.name, color: "default" }));
          else if (it.opt) {
            b.append(optionChip(it.opt, p.type === "status"));
            if (cur.includes(it.opt.id)) b.append(h("span", { class: "menu__check" }, iconNode("check")));
            const more = h("button", { class: "icon-btn icon-btn--sm opt-editor__more", attrs: { type: "button", "aria-label": t("Edit {name}", { name: it.opt.name }) } }, iconNode("more"));
            const opt = it.opt;
            more.addEventListener("click", (e) => {
              e.stopPropagation();
              editOption(more, prop(), opt, ctx, paint);
            });
            b.append(more);
          }
          b.addEventListener("click", () => pick(it));
          b.addEventListener("pointermove", () => {
            if (sel !== i) {
              sel = i;
              for (const [k, c] of Array.from(list.querySelectorAll(".opt-editor__item")).entries()) setAttr(c, "aria-selected", k === sel ? "true" : "false");
            }
          });
          list.append(b);
        });
        if (!its.length) list.append(h("div", { class: "listbox__empty", text: t("Type to create an option") }));
      };
      search.addEventListener("input", () => {
        sel = 0;
        paint();
        search.focus();
      });
      search.addEventListener("keydown", (e) => {
        const its = items();
        if (e.key === "ArrowDown") sel = Math.min(its.length - 1, sel + 1);
        else if (e.key === "ArrowUp") sel = Math.max(0, sel - 1);
        else if (e.key === "Enter") {
          const it = its[sel];
          if (it) pick(it);
          e.preventDefault();
          return;
        } else if (e.key === "Backspace" && !search.value && multi) {
          const cur = selected();
          if (cur.length) setValue(ctx, row0, p0, cur.slice(0, -1));
          setTimeout(paint, 0);
          return;
        } else return;
        e.preventDefault();
        paint();
        search.focus();
      });
      panel.append(chips, list);
      paint();
      ctx.engine.collections.subscribe(ctx.colId, () => {
        if (panel.isConnected) paint();
      });
    },
    { label: p0.name, onClose: () => o.onClose?.() },
  );
}

function dateEditor(anchor: HTMLElement, p: Property, row: RowData, ctx: DbCtx, o: EditOptions): void {
  const v = row.values[p.id] as DateValue | undefined;
  openPanel(
    anchor,
    (panel) => {
      panel.append(datePicker({ value: v ?? null, allowEnd: true, allowTime: true, allowRemind: true, onChange: (next) => setValue(ctx, row, p, next, 300) }));
    },
    { label: p.name, onClose: () => o.onClose?.() },
  );
}

function personEditor(anchor: HTMLElement, p: Property, row0: RowData, ctx: DbCtx, o: EditOptions): void {
  const selected = () => asList((ctx.session.rows.get(row0.id) ?? row0).values[p.id]);
  openPanel(
    anchor,
    (panel, close) => {
      const search = h("input", { class: "opt-editor__search", attrs: { type: "text", placeholder: t("Search for a person"), "aria-label": t("Search people") } });
      const list = h("div", { class: "opt-editor__list", attrs: { role: "listbox" } });
      let members: Member[] = [];
      const paint = () => {
        const q = search.value.trim().toLowerCase();
        const cur = selected();
        list.replaceChildren();
        const shown = members.filter((m) => !m.disabled && (!q || m.name.toLowerCase().includes(q) || m.email.toLowerCase().includes(q)));
        for (const m of shown) {
          const b = h("div", { class: "opt-editor__item", attrs: { role: "option" } }, avatar(m.name, m.color, false, m.avatar), h("span", { class: "menu__label", text: m.name }));
          if (cur.includes(m.user_id)) b.append(h("span", { class: "menu__check" }, iconNode("check")));
          b.addEventListener("click", () => {
            const now = selected();
            setValue(ctx, row0, p, now.includes(m.user_id) ? now.filter((x) => x !== m.user_id) : [...now, m.user_id]);
            setTimeout(paint, 0);
          });
          list.append(b);
        }
        if (!shown.length) list.append(h("div", { class: "listbox__empty", text: members.length ? t("No one matches") : t("Loading people") }));
      };
      search.addEventListener("input", paint);
      search.addEventListener("keydown", (e) => {
        if (e.key === "Enter") close();
      });
      panel.append(h("div", { class: "opt-editor__chips" }, search), list);
      paint();
      void ctx.engine.collections.people(ctx.workspaceId()).then((m) => {
        members = m;
        paint();
      }, ctx.engine.reportError);
    },
    { label: p.name, onClose: () => o.onClose?.() },
  );
}

function relationEditor(anchor: HTMLElement, p: Property, row0: RowData, ctx: DbCtx, o: EditOptions): void {
  const target = p.collection ?? ctx.colId;
  const selected = () => asList((ctx.session.rows.get(row0.id) ?? row0).values[p.id]);
  openPanel(
    anchor,
    (panel, close) => {
      const search = h("input", { class: "opt-editor__search", attrs: { type: "text", placeholder: t("Link or create a page"), "aria-label": t("Search pages") } });
      const linked = h("div", { class: "opt-editor__list" });
      const list = h("div", { class: "opt-editor__list", attrs: { role: "listbox" } });
      let seq = 0;
      const titles = new Map<string, { title: string; icon: string }>();
      for (const id of selected()) {
        const ref = ctx.session.refs.get(id);
        if (ref) titles.set(id, { title: ref.title, icon: ref.icon });
      }
      const toggle = (id: string) => {
        const now = selected();
        let next: string[];
        if (now.includes(id)) next = now.filter((x) => x !== id);
        else next = p.limit === 1 ? [id] : [...now, id];
        setValue(ctx, row0, p, next);
        if (p.limit === 1 && !now.includes(id)) close();
        else setTimeout(paint, 0);
      };
      const item = (id: string, title: string, icon: string, on: boolean) => {
        const b = h("div", { class: "opt-editor__item", attrs: { role: "option" } }, pageIcon(icon, "page"), h("span", { class: "menu__label", text: title || t("Untitled") }));
        if (on) b.append(h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Remove") } }, iconNode("close")));
        b.addEventListener("click", () => toggle(id));
        return b;
      };
      const paint = () => {
        const cur = selected();
        linked.replaceChildren();
        if (cur.length) linked.append(h("div", { class: "menu__heading", text: t("Linked") }));
        for (const id of cur) {
          const x = titles.get(id) ?? { title: ctx.session.refs.get(id)?.title ?? "", icon: "" };
          linked.append(item(id, x.title, x.icon, true));
        }
      };
      const load = async () => {
        const mine = ++seq;
        try {
          const r = await ctx.engine.api.collections.search(target, search.value, 30);
          if (mine !== seq) return;
          list.replaceChildren(h("div", { class: "menu__heading", text: t("Pages") }));
          const cur = selected();
          for (const x of r.rows) {
            titles.set(x.id, { title: x.title, icon: x.icon });
            if (!cur.includes(x.id)) list.append(item(x.id, x.title, x.icon, false));
          }
          if (search.value.trim()) {
            const create = h("div", { class: "opt-editor__item", attrs: { role: "option" } }, iconNode("plus"), h("span", { class: "menu__label", text: t("New page “{name}”", { name: search.value.trim() }) }));
            create.addEventListener("click", () => {
              const name = search.value.trim();
              void ctx.engine.api.collections
                .createRow(target, { title: name })
                .then((res) => {
                  titles.set(res.row.id, { title: name, icon: "" });
                  toggle(res.row.id);
                  search.value = "";
                  void load();
                })
                .catch(ctx.engine.reportError);
            });
            list.append(create);
          }
        } catch (e) {
          ctx.engine.reportError(e);
        }
      };
      let timer: ReturnType<typeof setTimeout> | undefined;
      search.addEventListener("input", () => {
        clearTimeout(timer);
        timer = setTimeout(() => void load(), 150);
      });
      panel.classList.add("rel-editor");
      panel.append(h("div", { class: "opt-editor__chips" }, search), linked, list);
      paint();
      void load();
    },
    { label: p.name, onClose: () => o.onClose?.() },
  );
}

function filesEditor(anchor: HTMLElement, p: Property, row0: RowData, ctx: DbCtx, o: EditOptions): void {
  type F = { id?: string; url?: string; name: string; mime?: string };
  const current = (): F[] => {
    const v = (ctx.session.rows.get(row0.id) ?? row0).values[p.id];
    return Array.isArray(v) ? (v as F[]) : [];
  };
  openPanel(
    anchor,
    (panel) => {
      const list = h("div", { class: "opt-editor__list" });
      const link = h("input", { class: "input input--sm", attrs: { type: "url", placeholder: t("Paste a link"), "aria-label": t("File link") } });
      const upload = h("button", { class: "btn btn--sm", attrs: { type: "button" } }, iconNode("upload"), h("span", { text: t("Upload") }));
      const paint = () => {
        list.replaceChildren();
        current().forEach((f, i) => {
          const rm = h("button", { class: "icon-btn icon-btn--sm", attrs: { type: "button", "aria-label": t("Remove {name}", { name: f.name }) } }, iconNode("close"));
          rm.addEventListener("click", () => {
            setValue(ctx, row0, p, current().filter((_, k) => k !== i));
            setTimeout(paint, 0);
          });
          list.append(h("div", { class: "opt-editor__item" }, iconNode("attachment"), h("span", { class: "menu__label", text: f.name }), rm));
        });
        if (!current().length) list.append(h("div", { class: "listbox__empty", text: t("No files yet") }));
      };
      upload.addEventListener("click", () => {
        void pickFile("*/*").then(async (file) => {
          if (!file) return;
          const id = `upload-${row0.id}`;
          toast({ id, message: t("Uploading {name}", { name: file.name }), durationMs: 0 });
          try {
            const r = await ctx.engine.api.files.upload(file, "attachment", { workspace_id: ctx.workspaceId(), page_id: row0.id });
            toast({ id, tone: "success", message: t("Uploaded {name}", { name: file.name }) });
            setValue(ctx, row0, p, [...current(), { id: r.file.id, name: r.file.name, mime: r.file.mime }]);
            setTimeout(paint, 0);
          } catch (e) {
            toast({ id, tone: "danger", message: errorMessage(e) });
          }
        });
      });
      link.addEventListener("keydown", (e) => {
        if (e.key !== "Enter") return;
        e.preventDefault();
        const u = link.value.trim();
        if (!/^https?:\/\//i.test(u)) return;
        setValue(ctx, row0, p, [...current(), { url: u, name: u }]);
        link.value = "";
        setTimeout(paint, 0);
      });
      panel.append(list, h("div", { class: "dbpanel__pad dbpanel__row" }, upload, link));
      paint();
    },
    { label: p.name, onClose: () => o.onClose?.(), focus: false },
  );
}

function placeEditor(anchor: HTMLElement, p: Property, row: RowData, ctx: DbCtx, o: EditOptions): void {
  const v = row.values[p.id] as { lat: number; lng: number; name?: string; address?: string } | undefined;
  openPanel(
    anchor,
    (panel, close) => {
      const name = h("input", { class: "input input--sm", attrs: { type: "text", placeholder: t("Name"), "aria-label": t("Place name") } });
      const address = h("input", { class: "input input--sm", attrs: { type: "text", placeholder: t("Address"), "aria-label": t("Address") } });
      const lat = h("input", { class: "input input--sm", attrs: { type: "text", inputmode: "decimal", placeholder: t("Latitude"), "aria-label": t("Latitude") } });
      const lng = h("input", { class: "input input--sm", attrs: { type: "text", inputmode: "decimal", placeholder: t("Longitude"), "aria-label": t("Longitude") } });
      name.value = v?.name ?? "";
      address.value = v?.address ?? "";
      lat.value = v ? String(v.lat) : "";
      lng.value = v ? String(v.lng) : "";
      const save = h("button", { class: "btn btn--primary btn--sm", attrs: { type: "button" }, text: t("Save") });
      const clear = h("button", { class: "btn btn--sm", attrs: { type: "button" }, text: t("Clear") });
      const hint = h("p", { class: "field__hint", text: t("Coordinates in decimal degrees, for example 52.52, 13.405.") });
      save.addEventListener("click", () => {
        const la = Number(lat.value.replace(",", "."));
        const lo = Number(lng.value.replace(",", "."));
        if (!Number.isFinite(la) || !Number.isFinite(lo) || lat.value.trim() === "" || lng.value.trim() === "") {
          toast({ id: "place", tone: "warning", message: t("Enter a latitude and a longitude") });
          return;
        }
        setValue(ctx, row, p, { lat: la, lng: lo, name: name.value.trim(), address: address.value.trim() });
        close();
      });
      clear.addEventListener("click", () => {
        setValue(ctx, row, p, null);
        close();
      });
      panel.append(h("div", { class: "dbpanel__pad dbpanel__stack" }, name, address, h("div", { class: "dbpanel__row" }, lat, lng), hint, h("div", { class: "dbpanel__row" }, clear, save)));
    },
    { label: p.name, onClose: () => o.onClose?.() },
  );
}
