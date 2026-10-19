import type { AIAdmin, AIProvider, Engine } from "../engine";
import { t } from "../engine/i18n";
import { confirmDialog, type Tab } from "./dialog";
import { h, setAttr, setText } from "./dom";
import { actions, busy, button, field, iconButton, input, section, select, status, toggle } from "./form";
import { iconNode } from "./icons";
import { reconcile } from "./list";
import { toast } from "./toast";

const KIND_LABEL: Record<string, string> = { anthropic: "Anthropic", openai: "OpenAI", gemini: "Google Gemini", compatible: "OpenAI-compatible" };

type Row = AIProvider["models"][number] & { provider: string };

export function aiTab(engine: Engine): Tab {
  return {
    id: "ai",
    label: t("AI"),
    icon: "sparkle",
    render(panel) {
      let data: AIAdmin | null = null;
      const st = status();

      const on = toggle(t("AI features"), true, t("Writing help, AI blocks, autofill, the workspace agent, meeting notes and AI answers. Workspace admins can still turn AI off for their workspace."));
      const def = select<string>([["", t("Choose a model")]], "");
      const audio = select<string>([["", t("None")]], "");
      const limit = input({ type: "number", name: "ai-limit", value: "200" });
      limit.min = "1";
      const save = button(t("Save"), "primary");
      const test = button(t("Try the default model"), "ghost", "sparkle");
      const general = section(t("AI"), t("Connect one or more providers, then choose which of their models people may use. API keys are sealed in the database."));
      general.append(on.el, field(t("Default model"), def, t("Used when someone does not pick a model.")), field(t("Transcription model"), audio, t("Turns meeting audio into text; mark a model as audio below.")), field(t("Requests per person per hour"), limit), st.el, actions(test, save));

      const webMode = select<string>([["off", t("Off")], ["searxng", "SearXNG"], ["brave", "Brave Search API"]], "off");
      const webUrl = input({ name: "ai-web-url", placeholder: "https://search.example.com/search" });
      const webKey = input({ type: "password", name: "ai-web-key", autocomplete: "off" });
      const webSave = button(t("Save"), "primary");
      const webStatus = status();
      let webKeySet = "";
      const web = section(t("Web search for answers"), t("Lets people tick Include the web when they ask a question. Only the question is sent to the search service; the top three results are fetched and given to the model as untrusted sources."));
      web.append(
        field(t("Search service"), webMode),
        field(t("Search address"), webUrl, t("SearXNG: the address of its search page, with JSON output allowed. Brave: leave empty for the public API, or give a compatible address.")),
        field(t("Search API key"), webKey, t("Brave only. Sealed in the database; leave empty to keep the saved key.")),
        webStatus.el,
        actions(webSave),
      );
      const loadWeb = async () => {
        try {
          const r = await engine.api.admin.settings();
          webKeySet = r.secret_set;
          webMode.value = r.settings["ai.web.mode"] || "off";
          webUrl.value = r.settings["ai.web.url"] ?? "";
          webKey.placeholder = r.settings["ai.web.key"] === webKeySet ? t("key set") : "";
        } catch (err) {
          webStatus.error(err);
        }
      };
      webSave.addEventListener("click", async () => {
        webStatus.clear();
        try {
          await busy(webSave, () => engine.api.admin.saveSettings({ "ai.web.mode": webMode.value, "ai.web.url": webUrl.value, "ai.web.key": webKey.value || webKeySet }));
          webKey.value = "";
          webStatus.say(t("Saved."), "success");
          await loadWeb();
        } catch (err) {
          webStatus.error(err);
        }
      });

      const provList = h("div", { class: "ai-providers" });
      const noProv = h("p", { class: "muted", text: t("No provider yet.") });
      const provStatus = status();
      const providers = section(t("Providers"));
      providers.append(provList, noProv, provStatus.el);

      const kind = select<string>(Object.entries(KIND_LABEL) as [string, string][], "anthropic");
      const name = input({ name: "ai-name", placeholder: t("for example Company Anthropic") });
      const base = input({ name: "ai-base", placeholder: "https://api.anthropic.com" });
      const key = input({ type: "password", name: "ai-key", autocomplete: "off" });
      const add = button(t("Add provider"), "primary", "plus", "submit");
      const addStatus = status();
      const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, field(t("Kind"), kind), field(t("Name"), name), field(t("Base URL"), base, t("Leave empty for the provider's own address. For a compatible server such as a local model runner, give its /v1 address.")), field(t("API key"), key), addStatus.el, actions(add));
      kind.addEventListener("change", () => {
        base.placeholder = data?.kinds[kind.value] || "http://127.0.0.1:11434/v1";
      });
      const addSection = section(t("Add a provider"));
      addSection.append(form);

      const usage = h("div", { class: "rows" });
      const noUsage = h("p", { class: "muted", text: t("No AI use in the last 30 days.") });
      const usageSection = section(t("Use in the last 30 days"));
      usageSection.append(usage, noUsage);

      const modelRow = (m: Row) => {
        const box = h("input", { attrs: { type: "checkbox" } });
        const au = h("input", { attrs: { type: "checkbox" } });
        const r = h("div", { class: "ai-model" }, h("label", { class: "check" }, box, h("span")), h("label", { class: "check ai-model__audio" }, au, h("span", { text: t("audio") })));
        const send = async (body: { enabled?: boolean; audio?: boolean }) => {
          try {
            await engine.api.ai.setModel(m.provider, { model: r.dataset.model ?? "", ...body });
            await load();
          } catch (err) {
            provStatus.error(err);
          }
        };
        box.addEventListener("change", () => void send({ enabled: box.checked }));
        au.addEventListener("change", () => void send({ audio: au.checked }));
        return r;
      };
      const modelUpdate = (r: HTMLElement, m: Row) => {
        r.dataset.model = m.model;
        const [box, au] = r.querySelectorAll<HTMLInputElement>("input");
        box!.checked = m.enabled;
        au!.checked = m.audio;
        setText(r.querySelector(".check span")!, m.label === m.model ? m.model : `${m.label} (${m.model})`);
        setAttr(r, "data-enabled", m.enabled ? "true" : null);
      };

      const provRow = (p: AIProvider) => {
        const title = h("span", { class: "row__title" });
        const meta = h("span", { class: "row__meta" });
        const refresh = iconButton("refresh", t("Fetch the model list"));
        const remove = iconButton("trash", t("Remove provider"));
        const enabled = h("input", { attrs: { type: "checkbox", role: "switch" } });
        const filter = input({ name: "ai-filter", placeholder: t("Find a model") });
        const models = h("div", { class: "ai-models" });
        const manual = input({ name: "ai-manual", placeholder: t("Add a model by id") });
        const addModel = button(t("Add"), "ghost", "plus");
        const el = h(
          "div",
          { class: "ai-provider" },
          h("div", { class: "row" }, iconNode("sparkle"), h("div", { class: "row__main" }, title, meta), h("div", { class: "row__actions" }, h("label", { class: "switch", attrs: { "aria-label": t("Provider on") } }, enabled), refresh, remove)),
          filter,
          models,
          h("div", { class: "ai-provider__manual" }, manual, addModel),
        );
        const id = () => el.dataset.key ?? "";
        refresh.addEventListener("click", async () => {
          try {
            const res = await busy(refresh, () => engine.api.ai.refreshModels(id()));
            toast({ id: "ai", message: t("{n} models found.", { n: String(res.models) }) });
            await load();
          } catch (err) {
            provStatus.error(err);
          }
        });
        remove.addEventListener("click", async () => {
          const ok = await confirmDialog({ title: t("Remove {name}?", { name: title.textContent ?? "" }), body: t("Its models stop working at once."), confirm: t("Remove"), variant: "danger" });
          if (!ok) return;
          try {
            await engine.api.ai.removeProvider(id());
            await load();
          } catch (err) {
            provStatus.error(err);
          }
        });
        enabled.addEventListener("change", async () => {
          try {
            await engine.api.ai.updateProvider(id(), { enabled: enabled.checked });
            await load();
          } catch (err) {
            provStatus.error(err);
          }
        });
        filter.addEventListener("input", () => paintModels(el));
        addModel.addEventListener("click", async () => {
          if (!manual.value.trim()) return;
          try {
            await engine.api.ai.setModel(id(), { model: manual.value.trim(), enabled: true });
            manual.value = "";
            await load();
          } catch (err) {
            provStatus.error(err);
          }
        });
        return el;
      };
      const paintModels = (el: HTMLElement) => {
        const p = data?.providers.find((x) => x.id === el.dataset.key);
        if (!p) return;
        const q = (el.querySelector<HTMLInputElement>('input[name="ai-filter"]')?.value ?? "").trim().toLowerCase();
        const rows = p.models.filter((m) => m.enabled || !q || m.model.toLowerCase().includes(q) || m.label.toLowerCase().includes(q)).slice(0, q ? 200 : 40);
        reconcile(el.querySelector<HTMLElement>(".ai-models")!, rows.map((m) => ({ ...m, provider: p.id })), (m) => m.model, modelRow, modelUpdate);
      };
      const provUpdate = (el: HTMLElement, p: AIProvider) => {
        const [title, meta] = el.querySelectorAll(".row__main > span") as unknown as [HTMLElement, HTMLElement];
        setText(title, p.name);
        const count = p.models.filter((m) => m.enabled).length;
        setText(meta, [KIND_LABEL[p.kind] ?? p.kind, p.base_url || t("default address"), p.has_key ? t("key set") : t("no key"), t("{n} of {m} models allowed", { n: String(count), m: String(p.models.length) })].join(" · "));
        el.querySelector<HTMLInputElement>('.row__actions input[type="checkbox"]')!.checked = p.enabled;
        paintModels(el);
      };

      const fillSelect = (sel: HTMLSelectElement, items: [string, string][], value: string) => {
        const want = items.map(([v]) => v).join("\n");
        if (sel.dataset.items !== want) {
          sel.replaceChildren(...items.map(([v, label]) => h("option", { attrs: { value: v }, text: label })));
          sel.dataset.items = want;
        }
        sel.value = value;
      };

      const load = async () => {
        try {
          data = await engine.api.ai.admin();
        } catch (err) {
          st.error(err);
          return;
        }
        on.input.checked = data.enabled;
        limit.value = String(data.limit_per_hour);
        const enabledModels = data.providers.flatMap((p) => p.models.filter((m) => m.enabled).map((m) => [`${p.id}:${m.model}`, `${m.label} · ${p.name}`] as [string, string]));
        const audioModels = data.providers.flatMap((p) => p.models.filter((m) => m.enabled && m.audio).map((m) => [`${p.id}:${m.model}`, `${m.label} · ${p.name}`] as [string, string]));
        fillSelect(def, [["", t("Choose a model")], ...enabledModels], data.default_model);
        fillSelect(audio, [["", t("None")], ...audioModels], data.transcribe_model);
        reconcile(provList, data.providers, (p) => p.id, provRow, provUpdate);
        noProv.hidden = data.providers.length > 0;
        reconcile(
          usage,
          data.usage,
          (u) => u.model,
          () => h("div", { class: "row" }, iconNode("chart"), h("div", { class: "row__main" }, h("span", { class: "row__title" }), h("span", { class: "row__meta" })), h("span")),
          (el, u) => {
            setText(el.querySelector(".row__title")!, u.model);
            setText(el.querySelector(".row__meta")!, t("{calls} requests · {in} tokens in · {out} tokens out", { calls: String(u.calls), in: String(u.input), out: String(u.output) }));
          },
        );
        noUsage.hidden = data.usage.length > 0;
      };

      save.addEventListener("click", async () => {
        st.clear();
        try {
          await busy(save, () => engine.api.ai.saveSettings({ enabled: on.input.checked, default_model: def.value, transcribe_model: audio.value, limit_per_hour: Number(limit.value) || 200 }));
          st.say(t("Saved."), "success");
          await load();
        } catch (err) {
          st.error(err);
        }
      });
      test.addEventListener("click", async () => {
        st.clear();
        try {
          const res = await busy(test, () => engine.api.ai.test(def.value));
          st.say(t("The model answered: {reply}", { reply: res.reply.slice(0, 80) }), "success");
        } catch (err) {
          st.error(err);
        }
      });
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        addStatus.clear();
        try {
          const res = await busy(add, () => engine.api.ai.addProvider({ kind: kind.value, name: name.value, base_url: base.value, api_key: key.value }));
          if (res.warning) addStatus.error(new Error(res.warning));
          else toast({ id: "ai", tone: "success", message: t("Provider added with {n} models; allow the ones people may use.", { n: String(res.models) }) });
          name.value = "";
          base.value = "";
          key.value = "";
          await load();
        } catch (err) {
          addStatus.error(err);
        }
      });

      panel.append(general, web, providers, addSection, usageSection);
      void load();
      void loadWeb();
      return () => {
        void load();
        void loadWeb();
      };
    },
  };
}
