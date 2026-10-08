import type { Engine } from "../../engine";
import { t } from "../../engine/i18n";
import type { CSVPreview, PropType } from "../../engine/types";
import { simpleDialog } from "../../ui/dialog";
import { h, setText } from "../../ui/dom";
import { errorMessage } from "../../ui/errors";
import { pickFile } from "../../ui/pickers";
import { toast } from "../../ui/toast";
import { CREATABLE, typeLabel } from "./props";

export interface ImportTarget {
  collectionId?: string;
  workspaceId: string;
  parentId?: string;
}

export function openCSVImport(engine: Engine, _anchor: HTMLElement | null, target: ImportTarget): void {
  void pickFile(".csv,.tsv,text/csv,text/tab-separated-values").then(async (file) => {
    if (!file) return;
    let preview: CSVPreview;
    try {
      toast({ id: "csv", message: t("Reading {name}", { name: file.name }), durationMs: 0 });
      preview = await engine.api.collections.previewCSV(file);
      toast({ id: "csv", tone: "success", message: t("{n} rows found", { n: preview.rows }) });
    } catch (e) {
      toast({ id: "csv", tone: "danger", message: errorMessage(e) });
      return;
    }
    simpleDialog(t("Import CSV"), (body, close) => {
      const title = h("input", { class: "input", attrs: { type: "text", "aria-label": t("Database name") } });
      title.value = preview.name;
      if (!target.collectionId) body.append(h("label", { class: "field" }, h("span", { class: "field__label", text: t("Database name") }), title));
      body.append(h("p", { class: "field__hint", text: target.collectionId ? t("{n} rows will be added. Columns are matched to properties by name; new names become new properties.", { n: preview.rows }) : t("{n} rows. Choose a type for each column.", { n: preview.rows }) }));
      const table = h("div", { class: "csvmap" });
      const selects: HTMLSelectElement[] = [];
      preview.columns.forEach((c, i) => {
        const sel = h("select", { class: "input input--sm", attrs: { "aria-label": t("Type of {name}", { name: c.name }) } });
        if (i === 0 || c.type === "title") sel.append(h("option", { attrs: { value: "title" }, text: typeLabel("title") }));
        for (const type of CREATABLE) if (!["rollup", "formula", "relation", "created_time", "created_by", "last_edited_time", "last_edited_by", "unique_id", "place"].includes(type)) sel.append(h("option", { attrs: { value: type }, text: typeLabel(type as PropType) }));
        sel.append(h("option", { attrs: { value: "skip" }, text: t("Skip this column") }));
        sel.value = c.type;
        selects.push(sel);
        table.append(h("div", { class: "csvmap__row" }, h("span", { class: "csvmap__name", text: c.name }), sel, h("span", { class: "csvmap__sample", text: c.samples.join(" · ") })));
      });
      body.append(table);
      const status = h("p", { class: "form-status", attrs: { role: "alert" } });
      const go = h("button", { class: "btn btn--primary", attrs: { type: "button" }, text: t("Import") });
      body.append(status, h("div", { class: "dialog__actions" }, go));
      go.addEventListener("click", () => {
        go.disabled = true;
        setText(status, t("Importing"));
        const req: { token: string; types: string[]; title?: string; workspace_id?: string; parent_id?: string; collection_id?: string } = { token: preview.token, types: selects.map((s) => s.value) };
        if (target.collectionId) req.collection_id = target.collectionId;
        else {
          req.title = title.value.trim();
          req.workspace_id = target.workspaceId;
          if (target.parentId) req.parent_id = target.parentId;
        }
        engine.api.collections
          .importCSV(req)
          .then((r) => {
            toast({ id: "csv", tone: "success", message: t("Imported {n} rows", { n: r.rows }) });
            close();
            if (!target.collectionId) engine.openPage(r.collection_id);
          })
          .catch((e: unknown) => {
            go.disabled = false;
            setText(status, errorMessage(e));
          });
      });
    });
  });
}
