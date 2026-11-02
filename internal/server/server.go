package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/nodemode-ai/form.md/internal/admin"
	"github.com/nodemode-ai/form.md/internal/agent"
	"github.com/nodemode-ai/form.md/internal/agentrun"
	"github.com/nodemode-ai/form.md/internal/ai"
	"github.com/nodemode-ai/form.md/internal/aifill"
	"github.com/nodemode-ai/form.md/internal/analytics"
	"github.com/nodemode-ai/form.md/internal/answer"
	"github.com/nodemode-ai/form.md/internal/apidoc"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auditapi"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/automation"
	"github.com/nodemode-ai/form.md/internal/claudecode"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/comment"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/contextpack"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/family"
	"github.com/nodemode-ai/form.md/internal/feed"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/graph"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/i18n"
	"github.com/nodemode-ai/form.md/internal/inbox"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/mcp"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/oauth"
	"github.com/nodemode-ai/form.md/internal/ops"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/scim"
	"github.com/nodemode-ai/form.md/internal/search"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/site"
	"github.com/nodemode-ai/form.md/internal/static"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/templates"
	"github.com/nodemode-ai/form.md/internal/transfer"
	"github.com/nodemode-ai/form.md/internal/tunnel"
	"github.com/nodemode-ai/form.md/internal/unfurl"
	"github.com/nodemode-ai/form.md/internal/webhook"
	"github.com/nodemode-ai/form.md/internal/workspace"
)

type Server struct {
	Deps     *app.Deps
	Modules  []app.Module
	Auth     *auth.Service
	Settings *settings.Service
	Gate     *policy.Gate
	Audit    *audit.Log
	Work     *workspace.Service
	Mail     *mail.Mailer
	Admin    *admin.Service
	Hub      *events.Hub
	Pages    *page.Service
	Files    *files.Service
	Docs     *doc.Service
	Cols     *collection.Service
	Comments *comment.Service
	Inbox    *inbox.Service
	Views    *analytics.Service
	Hooks    *webhook.Service
	Runs     *agentrun.Service
	OAuth    *oauth.Service
	AI       *ai.Service
	Crew     *agent.Service
	Flows    *automation.Service
	Tunnel   *tunnel.Service
}

func Build(ctx context.Context, cfg config.Config, log *slog.Logger, st *store.Store, dist, skill fs.FS, notices, instanceID string) (*Server, error) {
	packPolicy, err := contextpack.LoadPolicy(skill)
	if err != nil {
		return nil, err
	}
	box, err := secrets.Open(cfg.DataDir, st.Key)
	if err != nil {
		return nil, err
	}
	set, err := settings.New(ctx, st, box)
	if err != nil {
		return nil, err
	}
	if domain := strings.ToLower(strings.TrimSpace(set.Get("https.domain"))); !cfg.TLS() && set.Bool("https.auto") && domain != "" {
		if cfg.DesktopKey != "" {
			log.Warn("automatic HTTPS is ignored in the desktop app, which listens on this computer only", "domain", domain)
		} else {
			cfg.ACMEDomains = []string{domain}
			if cfg.Addr == "127.0.0.1:8750" {
				cfg.Addr = ":443"
			}
			log.Info("automatic HTTPS from Instance settings", "domain", domain, "addr", cfg.Addr)
		}
	}
	gate := policy.NewGate(st)
	al := audit.New(st)
	mailer := mail.New(set)
	authSvc := auth.New(st, box, set, gate, log)
	authSvc.SetDesktopKey(cfg.DesktopKey)
	ws := workspace.New(st, gate, al, log)
	adm := admin.New(st, authSvc, ws, set, mailer, al, log)
	mailAuth := mail.NewOAuth(set)
	mailAuth.Attach(mailer)
	adm.UseMailOAuth(mailAuth)
	hub := events.New(st, log, app.Version)
	pages := page.New(st, gate, al, hub, set, log)
	fsvc, err := files.New(st, gate, pages, set, al, log, cfg.DataDir)
	if err != nil {
		return nil, err
	}
	fetcher := netguard.NewClient(netguard.Options{AllowPrivate: cfg.FetchPrivate})
	fsvc.UseFetcher(fetcher)
	links := unfurl.New(fetcher, pages, fsvc, log)
	docs := doc.New(st, pages, authSvc, log)
	pages.OnDuplicate(docs.Copy)
	pages.OnDuplicated(docs.Remap)
	docs.SetAudit(al)
	docs.SetRemoteImporter(func(ctx context.Context, p *policy.Principal, pageID, url string) (string, error) {
		f, err := fsvc.Import(ctx, p, url, files.ImportOptions{Purpose: "attachment", PageID: pageID, ImagesOnly: true})
		if err != nil {
			return "", err
		}
		return f.ID, nil
	})
	cols := collection.New(st, pages, al, hub, log)
	comments := comment.New(st, pages, al, hub, log)
	notes, err := inbox.New(ctx, st, pages, hub, mailer, set, fetcher, log)
	if err != nil {
		return nil, err
	}
	comments.OnComment(notes.FromComment)
	docs.OnRefs(notes.FromRefs)
	pages.OnShare(notes.FromShare)
	cols.OnAssign(notes.FromAssign)
	finder := search.New(st, gate, authSvc, pages, log)
	finder.SetRows(cols.RowText, cols.RowIDs)
	finder.SetPDF(transfer.PDFText, fsvc.PDFBytes)
	cols.OnRows(finder.RowsChanged)
	sites := site.New(st, gate, pages, docs, cols, fsvc, authSvc, set, al, log)
	views := analytics.New(st, gate, pages, al, log)
	pages.OnVisit(views.FromVisit)
	sites.OnView(views.Record)
	hooks := webhook.New(st, gate, box, al, fetcher, cfg.FetchPrivate, log)
	prov := scim.New(st, gate, authSvc, ws, al, log)
	runs := agentrun.New(st, authSvc, log)
	grants := oauth.New(st, authSvc, gate, al, fetcher, cfg.FetchPrivate, log)
	brain := ai.New(st, box, set, gate, al, log)
	filler := aifill.New(authSvc, cols, docs, brain, log)
	cols.OnRows(filler.RowsChanged)
	docs.OnRefs(filler.PageChanged)
	pageText := func(ctx context.Context, p *policy.Principal, id string) (string, string, error) {
		d, err := pages.Get(ctx, p, id)
		if err != nil {
			return "", "", err
		}
		md, err := docs.Markdown(ctx, p, id)
		if err != nil {
			return "", "", err
		}
		return d.WorkspaceID, md, nil
	}
	ft := packPolicy.Family
	fam := family.New(authSvc, app.Version, cfg.Family, family.Tuning{VocabularyWeight: ft.VocabularyWeight, ToolWeight: ft.ToolWeight, MaxTools: ft.MaxTools,
		Cache: time.Duration(ft.CacheSeconds) * time.Second, Timeout: time.Duration(ft.TimeoutMS) * time.Millisecond})
	packer := contextpack.New(contextpack.Deps{Store: st, Gate: gate, Pages: pages, Docs: docs, Search: finder, Policy: packPolicy, Log: log, Family: fam})
	runs.Handle(contextpack.UndoKind, packer.Undo)
	harness := claudecode.New(authSvc, hub, cfg.Addr, cfg.DataDir)
	agents := mcp.New(mcp.Deps{Store: st, Auth: authSvc, Gate: gate, Pages: pages, Docs: docs, Cols: cols, Search: finder, Comments: comments, Workspace: ws, Runs: runs, Context: packer, Skill: skill, Window: harness, Version: app.Version, Log: log, Family: fam})
	crew := agent.New(agent.Deps{Store: st, Auth: authSvc, Gate: gate, AI: brain, Tools: agents, Docs: docs, Cols: cols, Runs: runs, Context: packer, Log: log})
	hooks.UsePages(pages)
	hooks.OnEvent(crew.OnEvent)
	flows := automation.New(automation.Deps{Store: st, Auth: authSvc, Gate: gate, Pages: pages, Cols: cols, Inbox: notes, Mail: mailer, Hooks: hooks, Box: box, Public: func() string { return set.Get("instance.public_url") }, Log: log})
	hooks.OnEvent(flows.OnEvent)
	cols.OnResponse(flows.OnResponse)
	docs.OnRefs(func(ctx context.Context, pageID, actor string, _ []string, _ []doc.Reminder) {
		var wsID string
		if st.R.QueryRowContext(ctx, "SELECT workspace_id FROM pages WHERE id = ?", pageID).Scan(&wsID) == nil {
			hooks.Emit(ctx, wsID, "page.content", pageID, &policy.Principal{UserID: actor, Kind: policy.KindSession}, nil)
		}
	})
	comments.OnComment(func(ctx context.Context, e comment.Event) {
		kind := map[string]string{"edit": "comment.edit", "resolve": "comment.resolve"}[e.Kind]
		if kind == "" || e.Page == nil {
			return
		}
		var data any
		if e.Thread != nil {
			data = map[string]any{"thread": e.Thread.ID, "resolved": e.Thread.ResolvedAt > 0}
		}
		hooks.Emit(ctx, e.Page.WorkspaceID, kind, e.Page.ID, e.Actor, data)
	})

	ws.SetNotifier(hub)
	ws.SetImageCheck(func(ctx context.Context, ref, workspaceID string) bool {
		return fsvc.InWorkspace(ctx, ref, workspaceID, "workspace")
	})
	ws.SetFontCheck(func(ctx context.Context, ref, workspaceID string) bool {
		return fsvc.FontInWorkspace(ctx, ref, workspaceID)
	})
	authSvc.SetAvatarCheck(func(ctx context.Context, ref, userID string) bool {
		return fsvc.Owned(ctx, ref, userID, "avatar")
	})
	authSvc.OnRevoke(func(_ context.Context, userID string) {
		hub.CloseUser(userID)
		docs.CloseUser(userID)
		harness.CloseUser(userID)
	})
	authSvc.OnCredentialRevoke(func(id string) {
		hub.CloseCredential(id)
		docs.CloseCredential(id)
		harness.CloseCredential(id)
	})
	authSvc.SetAccountHooks(ws)
	authSvc.SetMailer(mailer)
	pages.SetAuth(authSvc)
	authSvc.SetGuestAccept(pages.AcceptGuest)
	ws.SetCreatePolicy(func(p *policy.Principal) bool { return p.IsAdmin() || set.Bool("workspaces.member_create") })
	authSvc.SetMeExtras(func(ctx context.Context, p *policy.Principal) (map[string]any, error) {
		list, withheld, err := ws.List(ctx, p)
		if err != nil {
			return nil, err
		}
		return map[string]any{"workspaces": list, "withheld_workspaces": withheld, "can_create_workspaces": ws.MayCreate(p)}, nil
	})

	s := &Server{Auth: authSvc, Settings: set, Gate: gate, Audit: al, Work: ws, Mail: mailer, Admin: adm, Hub: hub, Pages: pages, Files: fsvc, Docs: docs, Cols: cols, Comments: comments, Inbox: notes, Views: views, Hooks: hooks, Runs: runs, OAuth: grants, AI: brain, Crew: crew, Flows: flows}
	s.Tunnel = tunnel.New(set, log, cfg.DataDir)
	s.Deps = &app.Deps{
		Config: cfg, Store: st, Log: log, Dist: dist, Notices: notices, InstanceID: instanceID,
		Middleware: []httpx.Middleware{s.Tunnel.Middleware, sites.Host, prov.Middleware, authSvc.Middleware},
	}
	s.Modules = []app.Module{
		app.Health(),
		app.Notices(),
		authSvc.Module(),
		fam.Module(),
		ws.Module(authSvc),
		adm.Module(),
		hub.Module(authSvc),
		pages.Module(authSvc),
		fsvc.Module(authSvc),
		links.Module(authSvc),
		docs.Module(),
		cols.Module(authSvc, authSvc.TrustProxy),
		comments.Module(authSvc),
		notes.Module(authSvc),
		finder.Module(),
		sites.Module(authSvc),
		views.Module(authSvc),
		hooks.Module(authSvc),
		auditapi.Module(al, gate, authSvc, set),
		prov.Module(),
		runs.Module(),
		grants.Module(),
		agents.Module(),
		brain.Module(authSvc, pageText),
		filler.Module(),
		crew.Module(),
		harness.Module(),
		flows.Module(),
		feed.New(st, authSvc, cols, docs, comments).Module(),
		templates.New(st, authSvc, gate, pages, docs, cols).Module(),
		transfer.New(st, authSvc, gate, pages, docs, cols, fsvc, sites, log, cfg.DataDir).Module(),
		apidoc.New(authSvc.BaseURL).Module(),
		graph.New(st, authSvc, pages).Module(),
		answer.New(st, authSvc, finder, brain, set, fetcher).Module(),
		ops.Module{Auth: authSvc, Flush: docs.FlushAll},
		s.Tunnel.Module(authSvc),
		static.Module(s.renderDocument),
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	go s.housekeeping(ctx)
	go s.sessionSweep(ctx)
	go s.Docs.Recover(ctx)
	go app.CheckUpdates(ctx, s.Deps.Config.UpdateURL, func() bool { return s.Settings.Bool("update.check") }, s.Deps.Log)
	go s.Inbox.Run(ctx)
	go s.Hooks.Run(ctx)
	go s.Crew.Loop(ctx)
	go s.Flows.Loop(ctx)
	go s.Tunnel.Autostart(ctx)
	return app.Run(ctx, s.Deps, s.Modules)
}

func (s *Server) sessionSweep(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		s.Auth.SweepSessions(ctx)
		s.OAuth.ExpireAccess(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (s *Server) housekeeping(ctx context.Context) {
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		_, _ = s.Deps.Store.W.ExecContext(ctx, "DELETE FROM invites WHERE accepted_at = 0 AND expires_at < ?", store.NowMS()-int64(30*24*time.Hour/time.Millisecond))
		if n, err := s.Pages.Purge(ctx); err != nil {
			s.Deps.Log.Warn("trash purge failed", "err", err)
		} else if n > 0 {
			s.Deps.Log.Info("trash purged", "pages", n)
		}
		s.Files.Sweep(ctx)
		if _, err := s.Docs.PurgeVersions(ctx, s.Settings.Int("history.days")); err != nil {
			s.Deps.Log.Warn("history purge failed", "err", err)
		}
		s.Views.Purge(ctx)
		s.Runs.Purge(ctx, 90)
		s.OAuth.Purge(ctx)
		if _, err := s.Audit.Purge(ctx, s.Settings.Int("audit.retention_days")); err != nil {
			s.Deps.Log.Warn("audit purge failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func preferredLang(r *http.Request, userLang string) string {
	if name, ok := i18n.Resolve(userLang); ok {
		return name
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		if name, ok := i18n.Resolve(tag); ok {
			return name
		}
	}
	return "en"
}

func (s *Server) authAttrs() map[string]string {
	var sso []string
	for _, p := range s.Auth.SSOProviders() {
		sso = append(sso, p.ID)
	}
	signup := "closed"
	if s.Settings.Get("auth.signup") != "invite" && !s.Auth.SAMLOnly() {
		signup = "open"
	}
	out := map[string]string{"sso": strings.Join(sso, " "), "signup": signup}
	if s.Auth.SAMLOnly() {
		out["saml-only"] = "true"
	}
	return out
}

func (s *Server) renderDocument(r *http.Request) (static.Document, error) {
	ctx := r.Context()
	path := r.URL.Path
	attrs := s.authAttrs()
	need, err := s.Auth.SetupRequired(ctx)
	if err != nil {
		return static.Document{}, err
	}
	if need {
		if path != "/" {
			return static.Document{Redirect: "/"}, nil
		}
		attrs["auth"] = "setup"
		slots := map[string]string{"instance-name": s.Settings.Get("instance.name")}
		return static.Document{Attrs: attrs, Slots: slots, Lang: preferredLang(r, ""), Boot: map[string]any{"setup": true}}, nil
	}
	p := policy.From(ctx)
	signedIn := p.IsSession()
	boot := map[string]any{
		"sso":           s.Auth.SSOProviders(),
		"signup":        attrs["signup"],
		"instance_name": s.Settings.Get("instance.name"),
	}
	userLang := ""
	slots := map[string]string{"instance-name": s.Settings.Get("instance.name"), "saml-name": s.Auth.SAMLName(), "version": app.Version}
	htmlSlots := map[string]string{}
	var fontCSS, fontFiles []string
	if signedIn {
		me, err := s.Auth.Me(ctx, p)
		if err != nil {
			return static.Document{}, err
		}
		boot["me"] = me
		if u, ok := me["user"].(*auth.User); ok {
			userLang = u.Prefs.Lang
			slots["user-name"], slots["user-initials"], slots["user-color"] = u.Name, Initials(u.Name), u.Color
			if u.Avatar != "" {
				slots["user-kind"] = "picture"
			}
			if u.Prefs.SystemFont {
				attrs["font"] = "system"
			}
			if a := u.Prefs.Appearance; a != nil {
				if b, err := json.Marshal(a); err == nil {
					attrs["look"] = string(b)
				}
			}
		}
		if list, ok := me["workspaces"].([]*workspace.Workspace); ok {
			boot["seq"] = s.Hub.Seq()
			cur := currentWorkspace(r, list)
			if id, ok := strings.CutPrefix(path, "/p/"); ok {
				if d, err := s.Pages.Get(ctx, p, strings.TrimSuffix(id, "/")); err == nil {
					boot["page"] = d
					s.bootCollection(ctx, p, d, boot)
					if l, err := s.Comments.List(ctx, p, d.ID); err == nil {
						boot["comments"] = l
					}
					if body, err := s.Docs.Render(ctx, p, d.ID); err == nil {
						htmlSlots["doc-body"] = body
						slots["doc-page"] = d.ID
					} else {
						s.Deps.Log.Warn("doc: first paint", "page", d.ID, "err", err)
					}
					for _, w := range list {
						if w.ID == d.WorkspaceID {
							cur = w
						}
					}
					var st map[string]string
					if cur != nil && cur.ID == d.WorkspaceID && len(cur.PageStyle) > 0 {
						_ = json.Unmarshal(cur.PageStyle, &st)
					}
					var own map[string]string
					if len(d.Style) > 0 && json.Unmarshal(d.Style, &own) == nil {
						if st == nil {
							st = map[string]string{}
						}
						for k, v := range own {
							st[k] = v
						}
					}
					for k, v := range st {
						if v != "" && v != "default" && !(k == "font" && v == "sans") && !(k == "align" && v == "start") {
							attrs["style-"+k] = v
						}
					}
					if id, ok := strings.CutPrefix(st["font"], "f:"); ok && cur != nil && cur.ID == d.WorkspaceID {
						for _, f := range cur.Fonts {
							if f.File == st["font"] {
								fontCSS = append(fontCSS, "/api/workspaces/"+cur.ID+"/fonts.css")
								fontFiles = append(fontFiles, "/files/"+id)
							}
						}
					}
				}
			}
			if cur != nil {
				boot["workspace_id"] = cur.ID
				slots["ws-name"], slots["ws-initial"], slots["ws-color"] = cur.Name, firstInitial(cur.Name), TileColor(cur.Name)
				switch {
				case cur.Image != "" || strings.HasPrefix(cur.Icon, "f:"):
					slots["ws-kind"] = "picture"
				case strings.HasPrefix(cur.Icon, "i:"):
					slots["ws-kind"] = "glyph"
				}
				if tree, err := s.Pages.Tree(ctx, p, cur.ID); err == nil {
					boot["pages"] = tree
					if path == "/" {
						if id := s.landing(ctx, p, cur.ID, tree); id != "" {
							return static.Document{Redirect: "/p/" + id}, nil
						}
					}
				}
				if tags, err := s.Pages.Tags(ctx, p, cur.ID); err == nil {
					boot["tags"] = tags
				}
				if teams, err := s.Pages.Teamspaces(ctx, p, cur.ID); err == nil {
					boot["teamspaces"] = teams
				}
			}
			if favs, err := s.Pages.Favorites(ctx, p); err == nil {
				boot["favorites"] = favs
			}
			if n, err := s.Inbox.Unread(ctx, p); err == nil {
				boot["inbox_unread"] = n
			}
		}
	}
	switch {
	case strings.HasPrefix(path, "/form/"):
		attrs["auth"] = "form"
		if def, err := s.Cols.PublicForm(ctx, strings.TrimSuffix(strings.TrimPrefix(path, "/form/"), "/")); err == nil {
			boot["form"] = def
			slots["instance-name"] = def.Title
		}
	case strings.HasPrefix(path, "/invite/"):
		attrs["auth"] = "invite"
	case path == "/oauth/consent":
		if !signedIn {
			return static.Document{Redirect: "/login?next=" + url.QueryEscape(path+"?"+r.URL.RawQuery)}, nil
		}
		attrs["auth"] = "consent"
		view, err := s.OAuth.ConsentView(ctx, p, r.URL.Query().Get("request"))
		if err != nil {
			attrs["consent"] = "expired"
			break
		}
		slots["oauth-client"], slots["oauth-host"] = view.Client, view.Host
		htmlSlots["oauth-scope"], htmlSlots["oauth-workspaces"] = consentChoices(view)
		boot["oauth"] = map[string]string{"request": view.Request}
	case !signedIn && path == "/signup" && attrs["signup"] == "open":
		attrs["auth"] = "signup"
	case !signedIn:
		attrs["auth"] = "signed-out"
	case path == "/login" || path == "/signup":
		return static.Document{Redirect: "/"}, nil
	case path == "/offline-shell":
		attrs["auth"] = "app"
		attrs["shell"] = "offline"
	default:
		attrs["auth"] = "app"
		attrs["route"] = "home"
		switch {
		case boot["page"] != nil:
			attrs["route"] = "page"
		case path == "/library" || path == "/library/":
			attrs["route"] = "library"
		case path == "/trash" || path == "/trash/":
			attrs["route"] = "trash"
		case path == "/inbox" || path == "/inbox/":
			attrs["route"] = "inbox"
		}
	}
	var preload []string
	if attrs["auth"] == "form" || boot["collection"] != nil || boot["row"] != nil || strings.Contains(htmlSlots["doc-body"], `class="dbblock"`) {
		preload = append(preload, "database")
	}
	return static.Document{Attrs: attrs, Slots: slots, HTML: htmlSlots, Lang: preferredLang(r, userLang), Boot: boot, Preload: preload, Styles: fontCSS, Fonts: fontFiles}, nil
}

func (s *Server) bootCollection(ctx context.Context, p *policy.Principal, d *page.Detail, boot map[string]any) {
	if d.Row {
		if row, err := s.Cols.Row(ctx, p, d.ID); err == nil {
			boot["row"] = row
		}
	}
	if d.Kind != "collection" {
		return
	}
	col, err := s.Cols.Get(ctx, p, d.ID)
	if err != nil {
		s.Deps.Log.Warn("collection: first paint", "page", d.ID, "err", err)
		return
	}
	boot["collection"] = col
	if len(col.Views) > 0 {
		v := col.Views[0]
		if res, err := s.Cols.Query(ctx, p, d.ID, collection.QueryInput{ViewID: v.ID, Limit: 100}); err == nil {
			boot["query"] = map[string]any{"view_id": v.ID, "result": res}
		}
	}
}

func (s *Server) landing(ctx context.Context, p *policy.Principal, workspaceID string, tree []*page.Meta) string {
	if recent, err := s.Pages.Recent(ctx, p, 50); err == nil {
		for _, r := range recent {
			if r.WorkspaceID == workspaceID {
				return r.ID
			}
		}
	}
	for _, m := range tree {
		if m.ParentID == "" {
			return m.ID
		}
	}
	return ""
}

const WorkspaceCookie = "formmd_ws"

func currentWorkspace(r *http.Request, list []*workspace.Workspace) *workspace.Workspace {
	if c, err := r.Cookie(WorkspaceCookie); err == nil {
		for _, w := range list {
			if w.ID == c.Value {
				return w
			}
		}
	}
	for _, w := range list {
		if !w.Personal {
			return w
		}
	}
	if len(list) > 0 {
		return list[0]
	}
	return nil
}

func Initials(name string) string {
	var out []rune
	for _, word := range strings.Fields(name) {
		for _, r := range word {
			out = append(out, unicode.ToUpper(r))
			break
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

func firstInitial(name string) string {
	for _, r := range Initials(name) {
		return string(r)
	}
	return ""
}

var tileColors = []string{"violet", "blue", "cyan", "teal", "green", "lime", "amber", "orange", "rose", "pink"}

func TileColor(name string) string {
	var h uint32
	for _, r := range name {
		h = h*31 + uint32(r)
	}
	return tileColors[h%uint32(len(tileColors))]
}
