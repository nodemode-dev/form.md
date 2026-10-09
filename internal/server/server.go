package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/nodemode-ai/form.md/internal/admin"
	"github.com/nodemode-ai/form.md/internal/analytics"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auditapi"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/comment"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/inbox"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/search"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/site"
	"github.com/nodemode-ai/form.md/internal/static"
	"github.com/nodemode-ai/form.md/internal/store"
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
}

func Build(ctx context.Context, cfg config.Config, log *slog.Logger, st *store.Store, dist fs.FS, notices, instanceID string) (*Server, error) {
	box, err := secrets.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	set, err := settings.New(ctx, st, box)
	if err != nil {
		return nil, err
	}
	gate := policy.NewGate(st)
	al := audit.New(st)
	mailer := mail.New(set)
	authSvc := auth.New(st, box, set, gate, log)
	ws := workspace.New(st, gate, al, log)
	adm := admin.New(st, authSvc, ws, set, mailer, al, log)
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
	cols.OnRows(finder.RowsChanged)
	sites := site.New(st, gate, pages, docs, cols, fsvc, authSvc, set, al, log)
	views := analytics.New(st, gate, pages, al, log)
	pages.OnVisit(views.FromVisit)
	sites.OnView(views.Record)
	hooks := webhook.New(st, gate, box, al, fetcher, cfg.FetchPrivate, log)
	docs.OnRefs(func(ctx context.Context, pageID, actor string, _ []string, _ []doc.Reminder) {
		var wsID string
		if st.R.QueryRowContext(ctx, "SELECT workspace_id FROM pages WHERE id = ?", pageID).Scan(&wsID) == nil {
			hooks.Emit(ctx, wsID, "page.content", pageID, &policy.Principal{UserID: actor, Kind: policy.KindSession}, nil)
		}
	})
	comments.OnComment(func(ctx context.Context, e comment.Event) {
		kind := map[string]string{"edit": "comment.edit", "resolve": "comment.resolve", "decision": "suggestion.decide"}[e.Kind]
		if kind == "" || e.Page == nil {
			return
		}
		var data any
		switch {
		case e.Suggestion != nil:
			data = map[string]any{"suggestion": e.Suggestion.ID, "status": e.Suggestion.Status}
		case e.Thread != nil:
			data = map[string]any{"thread": e.Thread.ID, "resolved": e.Thread.ResolvedAt > 0}
		}
		hooks.Emit(ctx, e.Page.WorkspaceID, kind, e.Page.ID, e.Actor, data)
	})

	ws.SetNotifier(hub)
	ws.SetImageCheck(func(ctx context.Context, ref, workspaceID string) bool {
		return fsvc.InWorkspace(ctx, ref, workspaceID, "workspace")
	})
	authSvc.SetAvatarCheck(func(ctx context.Context, ref, userID string) bool {
		return fsvc.Owned(ctx, ref, userID, "avatar")
	})
	authSvc.OnRevoke(func(_ context.Context, userID string) {
		hub.CloseUser(userID)
		docs.CloseUser(userID)
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

	s := &Server{Auth: authSvc, Settings: set, Gate: gate, Audit: al, Work: ws, Mail: mailer, Admin: adm, Hub: hub, Pages: pages, Files: fsvc, Docs: docs, Cols: cols, Comments: comments, Inbox: notes, Views: views, Hooks: hooks}
	s.Deps = &app.Deps{
		Config: cfg, Store: st, Log: log, Dist: dist, Notices: notices, InstanceID: instanceID,
		Middleware: []httpx.Middleware{sites.Host, authSvc.Middleware},
	}
	s.Modules = []app.Module{
		app.Health(),
		app.Notices(),
		authSvc.Module(),
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
		static.Module(s.renderDocument),
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	go s.housekeeping(ctx)
	go s.Inbox.Run(ctx)
	go s.Hooks.Run(ctx)
	return app.Run(ctx, s.Deps, s.Modules)
}

func (s *Server) housekeeping(ctx context.Context) {
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		s.Auth.SweepSessions(ctx)
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
	if userLang == "en" || userLang == "de" {
		return userLang
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		base, _, _ := strings.Cut(strings.ToLower(tag), "-")
		if base == "de" || base == "en" {
			return base
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
	if s.Settings.Get("auth.signup") != "invite" {
		signup = "open"
	}
	return map[string]string{"sso": strings.Join(sso, " "), "signup": signup}
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
	slots := map[string]string{"instance-name": s.Settings.Get("instance.name")}
	htmlSlots := map[string]string{}
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
	case !signedIn && path == "/signup" && attrs["signup"] == "open":
		attrs["auth"] = "signup"
	case !signedIn:
		attrs["auth"] = "signed-out"
	case path == "/login" || path == "/signup":
		return static.Document{Redirect: "/"}, nil
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
		}
	}
	var preload []string
	if attrs["auth"] == "form" || boot["collection"] != nil || boot["row"] != nil || strings.Contains(htmlSlots["doc-body"], `class="dbblock"`) {
		preload = append(preload, "database")
	}
	return static.Document{Attrs: attrs, Slots: slots, HTML: htmlSlots, Lang: preferredLang(r, userLang), Boot: boot, Preload: preload}, nil
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
