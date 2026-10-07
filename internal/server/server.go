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
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/static"
	"github.com/nodemode-ai/form.md/internal/store"
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

	ws.SetNotifier(hub)
	ws.SetImageCheck(func(ctx context.Context, ref, workspaceID string) bool {
		return fsvc.InWorkspace(ctx, ref, workspaceID, "workspace")
	})
	authSvc.SetAvatarCheck(func(ctx context.Context, ref, userID string) bool {
		return fsvc.Owned(ctx, ref, userID, "avatar")
	})
	authSvc.OnRevoke(func(_ context.Context, userID string) { hub.CloseUser(userID) })
	authSvc.SetAccountHooks(ws)
	authSvc.SetMailer(mailer)
	ws.SetCreatePolicy(func(p *policy.Principal) bool { return p.IsAdmin() || set.Bool("workspaces.member_create") })
	authSvc.SetMeExtras(func(ctx context.Context, p *policy.Principal) (map[string]any, error) {
		list, withheld, err := ws.List(ctx, p)
		if err != nil {
			return nil, err
		}
		return map[string]any{"workspaces": list, "withheld_workspaces": withheld, "can_create_workspaces": ws.MayCreate(p)}, nil
	})

	s := &Server{Auth: authSvc, Settings: set, Gate: gate, Audit: al, Work: ws, Mail: mailer, Admin: adm, Hub: hub, Pages: pages, Files: fsvc}
	s.Deps = &app.Deps{
		Config: cfg, Store: st, Log: log, Dist: dist, Notices: notices, InstanceID: instanceID,
		Middleware: []httpx.Middleware{authSvc.Middleware},
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
		static.Module(s.renderDocument),
	}
	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	go s.housekeeping(ctx)
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
	if signedIn {
		me, err := s.Auth.Me(ctx, p)
		if err != nil {
			return static.Document{}, err
		}
		boot["me"] = me
		if u, ok := me["user"].(*auth.User); ok {
			userLang = u.Prefs.Lang
			slots["user-name"], slots["user-initials"], slots["user-color"] = u.Name, Initials(u.Name), u.Color
			if u.Prefs.SystemFont {
				attrs["font"] = "system"
			}
		}
		if list, ok := me["workspaces"].([]*workspace.Workspace); ok {
			cur := currentWorkspace(r, list)
			if cur != nil {
				boot["workspace_id"] = cur.ID
				slots["ws-name"], slots["ws-initial"], slots["ws-color"] = cur.Name, firstInitial(cur.Name), TileColor(cur.Name)
			}
		}
	}
	switch {
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
	}
	return static.Document{Attrs: attrs, Slots: slots, Lang: preferredLang(r, userLang), Boot: boot}, nil
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
