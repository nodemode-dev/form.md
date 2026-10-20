import type { Engine, Me, Prefs, ThemePref } from "../engine";
import { formatDateTime, weekStart } from "../engine/format";
import { currentLang, LANGUAGES, t } from "../engine/i18n";
import { ACCENTS, type ContrastPref, DARK_THEMES, type Density, LIGHT_THEMES, type Look, readStartup, type StartupPref, writeStartup } from "../engine/prefs";
import { SLASH_KEY } from "../extensions/blocks/triggers";
import { storageAdapter } from "./app";
import type { Tab } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { actions, button, select, status, toggle } from "./form";
import { iconNode } from "./icons";
import { currentLook, setLook, watchLook } from "./theme";

const DARK_LABEL: Record<string, () => string> = {
  dark: () => t("Dark"),
  studio: () => t("Studio"),
  night: () => t("Night"),
  graphite: () => t("Graphite"),
  midnight: () => t("Midnight"),
  forest: () => t("Forest"),
  ember: () => t("Ember"),
  violet: () => t("Violet"),
  slate: () => t("Slate"),
  mono: () => t("Mono"),
  ocean: () => t("Ocean"),
  rose: () => t("Rose"),
  terminal: () => t("Terminal"),
  contrast: () => t("High contrast"),
};

const LIGHT_LABEL: Record<string, () => string> = {
  light: () => t("Light"),
  paper: () => t("Paper"),
  sand: () => t("Sand"),
  sage: () => t("Sage"),
  mist: () => t("Mist"),
  contrast: () => t("High contrast"),
};

const ACCENT_LABEL: Record<string, () => string> = {
  "#f97316": () => t("Orange"),
  "#ef4444": () => t("Red"),
  "#ec4899": () => t("Pink"),
  "#a855f7": () => t("Purple"),
  "#6366f1": () => t("Indigo"),
  "#3b82f6": () => t("Blue"),
  "#14b8a6": () => t("Teal"),
  "#22c55e": () => t("Green"),
  "#eab308": () => t("Yellow"),
};

const REGIONS = ["US", "GB", "IE", "CA", "AU", "NZ", "IN", "ZA", "SG", "DE", "AT", "CH", "FR", "BE", "NL", "LU", "ES", "PT", "IT", "SE", "NO", "DK", "FI", "PL", "CZ", "BR", "MX", "JP", "KR", "CN"];

interface Radios<V extends string> {
  set(v: V): void;
}

function radios<V extends string>(group: HTMLElement, buttons: HTMLElement[], value: V, onPick: (v: V) => void): Radios<V> {
  const set = (v: V) => {
    for (const b of buttons) {
      const on = b.dataset.v === v;
      setAttr(b, "aria-checked", on ? "true" : "false");
      setAttr(b, "tabindex", on ? "0" : "-1");
    }
    if (!buttons.some((b) => b.dataset.v === v) && buttons[0]) setAttr(buttons[0], "tabindex", "0");
  };
  for (const b of buttons) {
    b.addEventListener("click", () => {
      const v = (b.dataset.v ?? "") as V;
      set(v);
      onPick(v);
    });
  }
  group.addEventListener("keydown", (e) => {
    const step = e.key === "ArrowRight" || e.key === "ArrowDown" ? 1 : e.key === "ArrowLeft" || e.key === "ArrowUp" ? -1 : 0;
    if (!step) return;
    const i = buttons.indexOf(document.activeElement as HTMLElement);
    const next = buttons[(i + step + buttons.length) % buttons.length];
    if (i < 0 || !next) return;
    e.preventDefault();
    next.focus();
    next.click();
  });
  set(value);
  return { set };
}

function seg<V extends string>(label: string, options: readonly (readonly [V, string])[], value: V, onPick: (v: V) => void): { el: HTMLElement } & Radios<V> {
  const el = h("div", { class: "prefs-seg", attrs: { role: "radiogroup", "aria-label": label } });
  const buttons = options.map(([v, text]) => h("button", { class: "prefs-seg__btn", attrs: { type: "button", role: "radio", "data-v": v }, text }));
  el.append(...buttons);
  return { el, ...radios(el, buttons, value, onPick) };
}

function themeGrid(kind: "dark" | "light", label: string, labels: Record<string, () => string>, ids: readonly string[], value: string, onPick: (id: string) => void): { el: HTMLElement } & Radios<string> {
  const el = h("div", { class: "theme-grid", attrs: { role: "radiogroup", "aria-label": label } });
  const buttons = ids.map((id) => {
    const name = labels[id]?.() ?? id;
    return h(
      "button",
      { class: "theme-pick", attrs: { type: "button", role: "radio", "data-v": id, "aria-label": name } },
      h("span", { class: "theme-pick__ring" }, h("span", { class: "theme-swatch", attrs: { [`data-${kind}`]: id } })),
      h("span", { class: "theme-pick__name", attrs: { "aria-hidden": "true" }, text: name }),
    );
  });
  el.append(...buttons);
  return { el, ...radios(el, buttons, value, onPick) };
}

function row(label: string, control: HTMLElement, hint?: string, top = false): HTMLElement {
  const box = h("div", { class: "prefs-row__control" }, control);
  if (hint) box.append(h("span", { class: "prefs-row__hint", text: hint }));
  return h("div", { class: top ? "prefs-row prefs-row--top" : "prefs-row" }, h("span", { class: "prefs-row__label", text: label }), box);
}

function group(title: string, ...children: HTMLElement[]): HTMLElement {
  return h("section", { class: "prefs-group" }, h("h3", { class: "prefs-group__title", text: title }), ...children);
}

function me(engine: Engine): Me {
  const m = engine.state.get("me");
  if (!m) throw new Error("not signed in");
  return m;
}

function desktop(engine: Engine): boolean {
  return (engine.state.get("me")?.instance as { desktop?: boolean } | undefined)?.desktop === true;
}

function appearanceGroup(engine: Engine): { el: HTMLElement; refresh(): void } {
  const look = currentLook();
  const mode = seg<ThemePref>(
    t("Appearance"),
    [
      ["auto", t("Use system setting")],
      ["light", t("Light")],
      ["dark", t("Dark")],
    ],
    engine.state.get("themePref"),
    (v) => engine.state.set("themePref", v),
  );
  const dark = themeGrid("dark", t("Dark theme"), DARK_LABEL, DARK_THEMES, look.dark, (id) => setLook({ dark: id }));
  const light = themeGrid("light", t("Light theme"), LIGHT_LABEL, LIGHT_THEMES, look.light, (id) => setLook({ light: id }));

  const accentRow = h("div", { class: "accent-row", attrs: { role: "radiogroup", "aria-label": t("Accent colour") } });
  const accentButtons = [
    h("button", { class: "accent-pick accent-pick--default", attrs: { type: "button", role: "radio", "data-v": "", "aria-label": t("Default"), "data-tip": t("Default") } }),
    ...ACCENTS.map((hex) => {
      const name = ACCENT_LABEL[hex]?.() ?? hex;
      const b = h("button", { class: "accent-pick", attrs: { type: "button", role: "radio", "data-v": hex, "aria-label": name, "data-tip": name } });
      b.style.setProperty("--swatch", hex);
      return b;
    }),
  ];
  const custom = h("input", { attrs: { type: "color", "aria-label": t("Custom colour") } });
  const customBox = h("label", { class: "accent-custom", attrs: { "data-tip": t("Custom colour") } }, custom);
  accentRow.append(...accentButtons, customBox);
  const accent = radios<string>(accentRow, accentButtons, look.accent, (v) => setLook({ accent: v }));
  custom.addEventListener("input", () => setLook({ accent: custom.value.toLowerCase() }));

  const density = seg<Density>(
    t("Density"),
    [
      ["compact", t("Compact")],
      ["standard", t("Standard")],
      ["comfortable", t("Comfortable")],
    ],
    look.density,
    (v) => setLook({ density: v }),
  );
  const contrast = seg<ContrastPref>(
    t("High contrast"),
    [
      ["auto", t("Use system setting")],
      ["standard", t("Standard")],
      ["high", t("High contrast")],
    ],
    look.contrast,
    (v) => setLook({ contrast: v }),
  );

  const font = toggle(t("Use the system font"), me(engine).user.prefs.system_font, t("Instead of Inter for text and JetBrains Mono for code."));
  const fontStatus = status();
  font.input.addEventListener("change", async () => {
    fontStatus.clear();
    try {
      engine.setMe(await engine.api.me.update({ prefs: { ...me(engine).user.prefs, system_font: font.input.checked } }));
    } catch (err) {
      font.input.checked = !font.input.checked;
      fontStatus.error(err);
    }
  });

  const paint = (l: Look) => {
    dark.set(l.dark);
    light.set(l.light);
    accent.set(l.accent);
    const isCustom = l.accent !== "" && !(ACCENTS as readonly string[]).includes(l.accent);
    setAttr(customBox, "data-on", isCustom ? "" : null);
    if (isCustom) {
      customBox.style.setProperty("--swatch", l.accent);
      if (custom.value !== l.accent) custom.value = l.accent;
    } else {
      customBox.style.removeProperty("--swatch");
    }
    density.set(l.density);
    contrast.set(l.contrast);
  };
  paint(look);
  watchLook(paint);
  engine.state.watch("themePref", (v) => mode.set(v));

  const el = group(
    t("Appearance"),
    h("p", { class: "prefs-group__lead", text: t("Changes apply at once and follow your account to your other devices.") }),
    row(t("Appearance"), mode.el),
    row(t("Dark theme"), dark.el, t("Used whenever the appearance is dark."), true),
    row(t("Light theme"), light.el, t("Used whenever the appearance is light."), true),
    row(t("Accent colour"), accentRow),
    row(t("Density"), density.el),
    row(t("High contrast"), contrast.el, t("Stronger text and borders.")),
    font.el,
    fontStatus.el,
  );
  return {
    el,
    refresh: () => {
      mode.set(engine.state.get("themePref"));
      paint(currentLook());
      font.input.checked = me(engine).user.prefs.system_font;
    },
  };
}

function languageGroup(engine: Engine): { el: HTMLElement; refresh(): void } {
  const prefs = () => me(engine).user.prefs;
  const detectedZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const regionNames = new Intl.DisplayNames([currentLang()], { type: "region" });
  const zones: string[] = (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.("timeZone") ?? [detectedZone];

  const lang = select([["", t("Automatic (browser)")], ...LANGUAGES] as ReadonlyArray<readonly [string, string]>, prefs().lang);
  setAttr(lang, "aria-label", t("Language"));
  const region = select(
    [["", t("Automatic (from language)")] as const, ...REGIONS.map((r) => [r, regionNames.of(r) ?? r] as const).sort((a, b) => a[1].localeCompare(b[1]))],
    prefs().region,
  );
  setAttr(region, "aria-label", t("Region"));
  const zone = select([["", t("Automatic ({zone})", { zone: detectedZone })] as const, ...zones.map((z) => [z, z.replace(/_/g, " ")] as const)], prefs().time_zone);
  setAttr(zone, "aria-label", t("Time zone"));
  const week = toggle(t("Start week on Monday"), prefs().week_start === "monday", t("Calendars and date pickers begin the week on Monday."));

  const preview = h("p", { class: "preview-line" });
  const paintPreview = () => {
    const day = new Intl.DateTimeFormat(currentLang(), { weekday: "long" }).format(new Date(2024, 0, 7 + weekStart()));
    setText(preview, t("Now: {time}. Weeks start on {day}.", { time: formatDateTime(Date.now()), day }));
  };
  paintPreview();

  const st = status();
  const reload = button(t("Reload now"), "default", "refresh");
  reload.addEventListener("click", () => location.reload());
  const reloadNote = h("div", { class: "callout" }, iconNode("info"), h("div", { class: "form-grid" }, h("p", { text: t("The new language applies everywhere after a reload.") }), actions(reload)));
  reloadNote.hidden = true;

  let cycleValue: Prefs["hour_cycle"] = prefs().hour_cycle;
  const save = async () => {
    st.clear();
    const before = prefs();
    const next: Prefs = {
      ...before,
      lang: lang.value,
      region: region.value,
      time_zone: zone.value,
      hour_cycle: cycleValue,
      week_start: week.input.checked ? "monday" : before.week_start === "monday" ? "" : before.week_start,
    };
    try {
      engine.setMe(await engine.api.me.update({ prefs: next }));
      paintPreview();
      if (next.lang !== before.lang) reloadNote.hidden = false;
    } catch (err) {
      st.error(err);
    }
  };
  const cycle = seg<Prefs["hour_cycle"]>(
    t("Clock"),
    [
      ["", t("Automatic")],
      ["h12", t("12-hour")],
      ["h23", t("24-hour")],
    ],
    cycleValue,
    (v) => {
      cycleValue = v;
      void save();
    },
  );
  for (const el of [lang, region, zone]) el.addEventListener("change", () => void save());
  week.input.addEventListener("change", () => void save());

  const el = group(
    t("Language and time"),
    row(t("Language"), lang),
    reloadNote,
    row(t("Region"), region),
    row(t("Time zone"), zone),
    row(t("Clock"), cycle.el),
    week.el,
    preview,
    st.el,
  );
  return {
    el,
    refresh: () => {
      const p = prefs();
      lang.value = p.lang;
      region.value = p.region;
      zone.value = p.time_zone;
      cycleValue = p.hour_cycle;
      cycle.set(p.hour_cycle);
      week.input.checked = p.week_start === "monday";
      paintPreview();
    },
  };
}

function writingGroup(): HTMLElement {
  let slashOn = false;
  try {
    slashOn = localStorage.getItem(SLASH_KEY) === "1";
  } catch {
    slashOn = false;
  }
  const slash = toggle(t("Open the block menu only at the start of a line"), slashOn, t("Typing / in the middle of a sentence then stays a slash."));
  slash.input.addEventListener("change", () => {
    try {
      if (slash.input.checked) localStorage.setItem(SLASH_KEY, "1");
      else localStorage.removeItem(SLASH_KEY);
    } catch {
      slash.input.checked = !slash.input.checked;
    }
  });
  return group(t("Writing"), slash.el);
}

function privacyGroup(engine: Engine): HTMLElement {
  const st = status();
  const record = toggle(t("Show my name in page analytics"), true, t("When off, your visits still count toward view totals but your name is not listed."));
  record.el.dataset.loading = "";
  void engine.api.analytics
    .prefs()
    .then((r) => {
      record.input.checked = r.record;
    })
    .catch(() => undefined)
    .finally(() => delete record.el.dataset.loading);
  record.input.addEventListener("change", async () => {
    st.clear();
    try {
      await engine.api.analytics.setPrefs(record.input.checked);
    } catch (e) {
      record.input.checked = !record.input.checked;
      st.error(e);
    }
  });
  return group(t("Privacy"), record.el, st.el);
}

function desktopGroup(engine: Engine): { el: HTMLElement; refresh(): void } {
  const store = storageAdapter();
  const startup = seg<StartupPref>(
    t("On startup"),
    [
      ["continue", t("Continue where you left off")],
      ["home", t("Home")],
    ],
    readStartup(store),
    (v) => writeStartup(store, v),
  );
  const el = group(t("Desktop app"), row(t("On startup"), startup.el, t("What the window shows when the app opens.")));
  el.hidden = !desktop(engine);
  return {
    el,
    refresh: () => {
      el.hidden = !desktop(engine);
      startup.set(readStartup(store));
    },
  };
}

export function preferencesTab(engine: Engine): Tab {
  return {
    id: "preferences",
    label: t("Preferences"),
    icon: "sliders",
    render(panel) {
      const appearance = appearanceGroup(engine);
      const language = languageGroup(engine);
      const app = desktopGroup(engine);
      panel.append(h("div", { class: "prefs" }, appearance.el, language.el, writingGroup(), privacyGroup(engine), app.el));
      return () => {
        appearance.refresh();
        language.refresh();
        app.refresh();
      };
    },
  };
}
