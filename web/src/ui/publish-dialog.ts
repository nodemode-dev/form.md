import type { Engine, Site, SiteInput, SiteStatus } from "../engine";
import { t } from "../engine/i18n";
import { confirmDialog, simpleDialog } from "./dialog";
import { disclosure } from "./disclosure";
import { h, setAttr, setText } from "./dom";
import { errorMessage } from "./errors";
import { actions, busy, button, choices, copyText, field, iconButton, input, status, textarea, toggle } from "./form";
import { iconNode } from "./icons";
import { toast } from "./toast";

export function siteUrl(site: Site): string {
  return site.hostname ? `https://${site.hostname}/` : `${location.origin}${site.path}`;
}

export async function openPublishDialog(engine: Engine, pageId: string): Promise<void> {
  let st: SiteStatus;
  try {
    st = await engine.api.sites.status(pageId);
  } catch (e) {
    toast({ tone: "danger", message: errorMessage(e) });
    return;
  }
  const page = engine.pages.get(pageId) ?? engine.state.get("detail");
  simpleDialog(
    t("Publish to the web"),
    (body, close) => {
      const lead = h("p", { class: "section__lead", text: t("Publishing makes this page, and the pages inside it if you choose, readable by anyone with the address. Nobody needs an account.") });
      const off = h("div", { class: "callout callout--warning" }, iconNode("alert"), h("span", { text: t("Publishing is turned off for this workspace. An admin can turn it on under Settings, Sites.") }));
      const link = h("a", { class: "publish-url__link", attrs: { target: "_blank", rel: "noopener" } });
      const copy = button(t("Copy link"), "primary", "copy");
      const open = iconButton("external", t("Open the site"));
      const live = h("div", { class: "publish-url" }, h("span", { class: "publish-url__dot", attrs: { "aria-hidden": "true" } }), link, copy, open);

      const slug = input({ autocomplete: "off", maxlength: 60, placeholder: st.suggested_slug });
      const slugField = field(t("Web address"), h("div", { class: "publish-slug" }, h("span", { class: "publish-slug__prefix", text: `${location.host}/s/` }), slug));
      const title = input({ maxlength: 120, placeholder: page?.title || t("Untitled") });
      const desc = textarea("", 3);
      desc.maxLength = 300;
      desc.placeholder = t("A sentence or two for search results and link previews.");
      const theme = choices<Site["theme"]>(`pub-theme-${pageId}`, [
        ["auto", t("Match the reader")],
        ["light", t("Light")],
        ["dark", t("Dark")],
      ], "auto");
      const font = choices<Site["font"]>(`pub-font-${pageId}`, [
        ["sans", t("Sans")],
        ["serif", t("Serif")],
        ["mono", t("Mono")],
      ], "sans");
      const subpages = toggle(t("Include the pages inside"), true, t("Sub-pages and databases inside this page are published with it. Private pages never are."));
      const indexing = toggle(t("Let search engines list it"), false, t("Adds the site to the sitemap and lets crawlers index it. Off while a password is set."));
      const locked = toggle(t("Require a password"), false);
      const password = input({ type: "password", autocomplete: "new-password", placeholder: t("New password") });
      const passwordField = field(t("Password"), password);
      const host = input({ autocomplete: "off", placeholder: "docs.example.com" });
      const hostField = field(t("Custom domain"), host, t("Point a DNS record for this name at this server, then save. Only workspace admins change it."));
      const image = h("span", { class: "publish-image__name muted" });
      const pick = button(t("Choose a picture"), "default", "image");
      const clear = button(t("Remove"), "ghost");
      const picker = h("input", { attrs: { type: "file", accept: "image/*", hidden: "" } });
      let imageRef = "";
      const imageField = field(t("Preview picture"), h("div", { class: "publish-image" }, pick, image, clear, picker), t("Shown when the address is shared in chats and social posts."));
      const st2 = status();
      const unpublish = button(t("Unpublish"), "danger");
      const save = button(t("Publish"), "primary", "globe", "submit");
      const more = disclosure(
        t("More options"),
        field(t("Title for search and sharing"), title),
        field(t("Description"), desc),
        imageField,
        field(t("Theme"), theme.el),
        field(t("Font"), font.el),
        subpages.el,
        indexing.el,
        locked.el,
        passwordField,
        hostField,
      );
      const bar = actions(unpublish, save);
      bar.classList.add("publish-bar");
      const form = h("form", { class: "form-grid", attrs: { novalidate: "" } }, slugField, st2.el, more.el, bar);
      body.append(h("div", { class: "publish-top" }, lead, live), off, form);

      let site: Site | null = st.site;
      const sync = () => {
        setAttr(live, "data-off", site ? null : "");
        setAttr(live, "aria-hidden", site ? null : "true");
        setAttr(lead, "data-off", site ? "" : null);
        setAttr(lead, "aria-hidden", site ? "true" : null);
        live.inert = !site;
        off.hidden = st.allowed;
        unpublish.hidden = !site || !st.can_publish;
        save.hidden = !st.can_publish || !st.allowed;
        setText(save.lastElementChild as HTMLElement, site ? t("Save changes") : t("Publish"));
        hostField.hidden = !st.can_host;
        passwordField.hidden = !locked.input.checked;
        password.placeholder = site?.password ? t("Leave empty to keep the current password") : t("New password");
        indexing.input.disabled = locked.input.checked;
        clear.hidden = !imageRef;
        setText(image, imageRef ? t("Picture chosen") : t("None"));
        for (const el of form.querySelectorAll<HTMLInputElement | HTMLTextAreaElement>("input, textarea")) el.disabled = !st.can_publish || !st.allowed || (el === indexing.input && locked.input.checked);
        pick.disabled = !st.can_publish || !st.allowed;
        if (site) {
          const url = siteUrl(site);
          setAttr(link, "href", url);
          setText(link, url.replace(/^https?:\/\//, ""));
        }
      };
      const fill = (x: Site | null) => {
        slug.value = x?.slug ?? st.suggested_slug;
        title.value = x?.title ?? "";
        desc.value = x?.description ?? "";
        theme.set(x?.theme ?? "auto");
        font.set(x?.font ?? "sans");
        subpages.input.checked = x?.subpages ?? true;
        indexing.input.checked = x?.indexing ?? false;
        locked.input.checked = x?.password ?? false;
        password.value = "";
        host.value = x?.hostname ?? "";
        imageRef = x?.image ?? "";
      };
      fill(site);
      sync();
      locked.input.addEventListener("change", sync);

      copy.addEventListener("click", async () => {
        if (site) toast({ id: "copy", message: (await copyText(siteUrl(site))) ? t("Address copied") : t("Could not copy") });
      });
      open.addEventListener("click", () => {
        if (site) window.open(siteUrl(site), "_blank", "noopener");
      });
      pick.addEventListener("click", () => picker.click());
      picker.addEventListener("change", async () => {
        const file = picker.files?.[0];
        picker.value = "";
        if (!file) return;
        try {
          const r = await busy(pick, () => engine.api.files.upload(file, "cover", { page_id: pageId }));
          imageRef = r.file.ref;
          sync();
        } catch (e) {
          st2.error(e);
        }
      });
      clear.addEventListener("click", () => {
        imageRef = "";
        sync();
      });
      form.addEventListener("submit", async (e) => {
        e.preventDefault();
        st2.clear();
        const req: SiteInput = {
          slug: slug.value.trim(),
          title: title.value,
          description: desc.value,
          image: imageRef,
          theme: theme.value(),
          font: font.value(),
          subpages: subpages.input.checked,
          indexing: indexing.input.checked && !locked.input.checked,
        };
        if (!locked.input.checked) {
          if (site?.password) req.password = "";
        } else if (password.value) {
          req.password = password.value;
        } else if (!site?.password) {
          st2.say(t("Type a password, or turn the password off."), "error");
          more.el.open = true;
          password.focus({ preventScroll: false });
          return;
        }
        if (st.can_host && host.value.trim() !== (site?.hostname ?? "")) req.hostname = host.value.trim();
        const first = !site;
        try {
          const r = await busy(save, () => engine.api.sites.publish(pageId, req));
          site = r.site;
          fill(site);
          sync();
          toast({ tone: "success", message: first ? t("Published. Anyone with the address can read it.") : t("Changes saved.") });
        } catch (err) {
          st2.error(err);
        }
      });
      unpublish.addEventListener("click", async () => {
        const ok = await confirmDialog({ title: t("Unpublish this page?"), body: t("The address stops working right away. You can publish again later."), confirm: t("Unpublish"), variant: "danger" });
        if (!ok) return;
        try {
          await busy(unpublish, () => engine.api.sites.unpublish(pageId));
          site = null;
          fill(null);
          sync();
          toast({ message: t("Unpublished.") });
        } catch (err) {
          st2.error(err);
        }
      });
      return site ? copy : save;
    },
    undefined,
    "md",
  );
}
