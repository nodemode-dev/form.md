import type { Engine } from "../engine";
import type { Command } from "../engine/commands";
import { t } from "../engine/i18n";
import { byId, h, setAttr, setText } from "./dom";
import { iconNode } from "./icons";
import { KEYS_STORAGE } from "./keyboard";
import { openLayer } from "./layers";
import { reconcile } from "./list";

type Item = { kind: "head"; key: string; group: string } | { kind: "cmd"; key: string; group: string; command: Command };

export function mountShortcutsEditor(engine: Engine): { open(): void; close(): void } {
  const { commands, keymap, state } = engine;
  let closeLayer: (() => void) | null = null;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  let recording: { id: string; row: HTMLElement } | null = null;
  let clashes = new Map<string, { chord: string; with: string[] }[]>();

  const closeBtn = h("button", { class: "icon-btn", attrs: { type: "button", "aria-label": t("Close"), "data-tip": t("Close"), "data-command": "layers.closeTop" } }, iconNode("close"));
  const reset = h("button", { class: "btn btn--ghost btn--sm", attrs: { type: "button" }, text: t("Reset all") });
  const head = h("header", { class: "dialog__head keys__head" }, h("h2", { class: "dialog__title", attrs: { id: "shortcuts-title" }, text: t("Keyboard shortcuts") }), reset, closeBtn);
  const search = h("input", { class: "keys__search-input", attrs: { type: "search", placeholder: t("Search commands and keys"), "aria-label": t("Search shortcuts"), spellcheck: "false", autocomplete: "off" } });
  const bar = h("label", { class: "keys__search" }, iconNode("search"), search);
  const hint = h("p", { class: "keys__hint", text: t("Click a key combination to change it.") });
  const table = h("div", { class: "keys__table", attrs: { role: "table", "aria-label": t("Keyboard shortcuts") } });
  const empty = h("p", { class: "keys__empty", text: t("No command matches that search.") });
  empty.hidden = true;
  const body = h("div", { class: "keys__body" }, table, empty);
  const dialog = h("div", { class: "dialog keys", attrs: { role: "dialog", "aria-modal": "true", "aria-labelledby": "shortcuts-title", tabindex: "-1" } }, head, h("div", { class: "keys__bar" }, bar, hint), body);
  const scrim = h("div", { class: "scrim", attrs: { "aria-hidden": "true" } });
  scrim.hidden = true;
  dialog.hidden = true;
  byId("overlay-root").append(scrim, dialog);

  const save = () => {
    const custom = keymap.customs();
    try {
      if (Object.keys(custom).length) localStorage.setItem(KEYS_STORAGE, JSON.stringify(custom));
      else localStorage.removeItem(KEYS_STORAGE);
    } catch {
      return;
    }
  };

  const label = (id: string) => commands.get(id)?.label() ?? id;

  const fillRow = (row: HTMLElement, it: Item) => {
    row.dataset.group = it.group;
    if (it.kind === "head") {
      setText(row, it.group);
      return;
    }
    const id = it.command.id;
    const chords = keymap.chordsFor(id);
    const isRec = recording?.id === id;
    const cell = row.querySelector(".keys-row__chords") as HTMLElement;
    const sig = isRec ? "rec" : chords.join(" ");
    if (cell.dataset.sig !== sig) {
      cell.dataset.sig = sig;
      if (isRec) cell.replaceChildren(h("span", { class: "keys-row__rec", text: t("Press keys, Backspace to clear, Escape to cancel") }));
      else if (!chords.length) cell.replaceChildren(h("span", { class: "keys-row__none", text: t("Not set") }));
      else {
        const parts: Node[] = [];
        chords.forEach((c, i) => {
          if (i > 0) parts.push(document.createTextNode(t("or")));
          parts.push(h("span", { class: "keys-row__chord" }, ...keymap.format(c).map((k) => h("kbd", { text: k }))));
        });
        cell.replaceChildren(...parts);
      }
    }
    row.classList.toggle("is-recording", isRec);
    row.classList.toggle("is-custom", keymap.customs()[id] !== undefined);
    setText(row.querySelector(".keys-row__label") as HTMLElement, it.command.label());
    const clash = clashes.get(id) ?? [];
    const note = row.querySelector(".keys-row__note") as HTMLElement;
    const text = clash.length ? clash.map((c) => t("{keys} is also used by {names}", { keys: keymap.text(c.chord), names: c.with.map(label).join(", ") })).join(" · ") : "";
    setText(note, text);
    if (note.hidden !== !text) note.hidden = !text;
    setAttr(cell, "aria-label", t("Change the shortcut for {name}", { name: it.command.label() }));
    row.dataset.search = `${it.command.label()} ${id} ${it.group} ${chords.map((c) => keymap.text(c)).join(" ")}`.toLowerCase();
  };

  const items = (): Item[] => {
    const groups = new Map<string, Command[]>();
    for (const c of commands.list()) {
      if (Object.values(c.params ?? {}).some((p) => p.required)) continue;
      const g = c.group();
      const list = groups.get(g) ?? [];
      list.push(c);
      groups.set(g, list);
    }
    const out: Item[] = [];
    for (const [group, list] of groups) {
      out.push({ kind: "head", key: `g:${group}`, group });
      for (const command of list.sort((a, b) => a.label().localeCompare(b.label()))) out.push({ kind: "cmd", key: `c:${command.id}`, group, command });
    }
    return out;
  };

  let current: Item[] = [];
  const paint = () => {
    clashes = keymap.conflicts();
    reconcile(
      table,
      current,
      (it) => it.key,
      (it) =>
        it.kind === "head"
          ? h("div", { class: "keys__group", attrs: { role: "row" } })
          : h(
              "div",
              { class: "keys-row", attrs: { role: "row" } },
              h("span", { class: "keys-row__main", attrs: { role: "cell" } }, h("span", { class: "keys-row__label" }), h("span", { class: "keys-row__note" })),
              h("button", { class: "keys-row__chords", attrs: { type: "button", role: "cell" } }),
            ),
      fillRow,
    );
    filter();
  };

  const filter = () => {
    const q = search.value.trim().toLowerCase();
    const words = q.split(/\s+/).filter(Boolean);
    const visible = new Set<string>();
    for (const n of Array.from(table.children) as HTMLElement[]) {
      if (!n.classList.contains("keys-row")) continue;
      const hay = n.dataset.search ?? "";
      const ok = words.every((w) => hay.includes(w));
      if (n.hidden !== !ok) n.hidden = !ok;
      if (ok) visible.add(n.dataset.group ?? "");
    }
    for (const n of Array.from(table.children) as HTMLElement[]) {
      if (!n.classList.contains("keys__group")) continue;
      const off = !visible.has(n.dataset.group ?? "");
      if (n.hidden !== off) n.hidden = off;
    }
    if (empty.hidden !== visible.size > 0) empty.hidden = visible.size > 0;
  };

  const repaint = () => {
    clashes = keymap.conflicts();
    for (const n of Array.from(table.children) as HTMLElement[]) {
      const it = current.find((x) => x.key === n.dataset.key);
      if (it) fillRow(n, it);
    }
    filter();
  };

  const stop = () => {
    if (!recording) return;
    recording = null;
    dialog.removeAttribute("data-own-keys");
    repaint();
  };

  const record = (row: HTMLElement) => {
    const id = row.dataset.key?.slice(2);
    if (!id || !commands.get(id)) return;
    stop();
    recording = { id, row };
    dialog.setAttribute("data-own-keys", "");
    repaint();
    (row.querySelector(".keys-row__chords") as HTMLElement).focus({ preventScroll: true });
  };

  table.addEventListener("click", (e) => {
    const cell = (e.target as Element).closest(".keys-row__chords");
    const row = cell?.closest<HTMLElement>(".keys-row");
    if (!row) return;
    if (recording?.row === row) stop();
    else record(row);
  });

  dialog.addEventListener("keydown", (e) => {
    const r = recording;
    if (!r) return;
    const plain = !e.ctrlKey && !e.metaKey && !e.altKey && !e.shiftKey;
    if (plain && e.key === "Tab") {
      stop();
      return;
    }
    e.preventDefault();
    e.stopPropagation();
    if (plain && e.key === "Escape") {
      stop();
      return;
    }
    if (plain && (e.key === "Backspace" || e.key === "Delete")) {
      keymap.setCustom(r.id, keymap.defaultChords(r.id).length ? [] : null);
      save();
      stop();
      return;
    }
    const chord = keymap.fromEvent(e);
    if (!chord) return;
    const defaults = keymap.defaultChords(r.id);
    keymap.setCustom(r.id, defaults.length === 1 && defaults[0] === chord ? null : [chord]);
    save();
    stop();
  });

  dialog.addEventListener("focusout", (e) => {
    if (recording && !(e.relatedTarget instanceof Node && recording.row.contains(e.relatedTarget))) stop();
  });

  search.addEventListener("input", filter);

  reset.addEventListener("click", () => {
    stop();
    keymap.loadCustoms(null);
    save();
    repaint();
  });

  const open = () => {
    clearTimeout(hideTimer);
    if (dialog.hidden) {
      search.value = "";
      current = items();
      paint();
    }
    scrim.hidden = false;
    dialog.hidden = false;
    requestAnimationFrame(() => {
      scrim.classList.add("is-visible");
      dialog.classList.add("is-visible");
    });
    if (!closeLayer) {
      closeLayer = openLayer({
        el: dialog,
        modal: true,
        dismissOnOutside: true,
        initialFocus: search,
        onClose: () => {
          closeLayer = null;
          stop();
          if (state.get("overlay") === "shortcuts") state.set("overlay", null);
          hide();
        },
      });
    }
  };

  const hide = () => {
    if (dialog.hidden) return;
    scrim.classList.remove("is-visible");
    dialog.classList.remove("is-visible");
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => {
      scrim.hidden = true;
      dialog.hidden = true;
    }, 200);
  };

  const close = () => {
    stop();
    if (closeLayer) {
      const c = closeLayer;
      closeLayer = null;
      c();
    }
    hide();
  };

  return { open, close };
}
