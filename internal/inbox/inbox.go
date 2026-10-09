package inbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/comment"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/i18n"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	pushDelay  = 10 * time.Second
	maxBody    = 280
	listLimit  = 50
	pushPeriod = 4 * time.Second
)

type Notification struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Kind        string `json:"kind"`
	PageID      string `json:"page_id"`
	PageIcon    string `json:"page_icon"`
	ThreadID    string `json:"thread_id"`
	ActorID     string `json:"actor_id"`
	ActorName   string `json:"actor_name"`
	ActorColor  string `json:"actor_color"`
	ActorAvatar string `json:"actor_avatar"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	CreatedAt   int64  `json:"created_at"`
	ReadAt      int64  `json:"read_at"`
	ArchivedAt  int64  `json:"archived_at"`
}

type Service struct {
	st       *store.Store
	pages    *page.Service
	hub      *events.Hub
	mailer   *mail.Mailer
	settings *settings.Service
	client   *http.Client
	log      *slog.Logger
	vapid    *vapid
}

func New(ctx context.Context, st *store.Store, pages *page.Service, hub *events.Hub, mailer *mail.Mailer, set *settings.Service, client *http.Client, log *slog.Logger) (*Service, error) {
	v, err := loadVAPID(ctx, set)
	if err != nil {
		return nil, err
	}
	return &Service{st: st, pages: pages, hub: hub, mailer: mailer, settings: set, client: client, log: log, vapid: v}, nil
}

func clip(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func (s *Service) principal(ctx context.Context, userID string) *policy.Principal {
	var role, name, email string
	var disabled int
	if err := s.st.R.QueryRowContext(ctx, "SELECT role, name, email, disabled FROM users WHERE id = ?", userID).Scan(&role, &name, &email, &disabled); err != nil || disabled == 1 {
		return nil
	}
	return &policy.Principal{UserID: userID, Name: name, Email: email, Role: role, Kind: policy.KindSession, Scope: "write", AllWorkspaces: true}
}

func (s *Service) canSee(ctx context.Context, userID, pageID string) bool {
	p := s.principal(ctx, userID)
	if p == nil {
		return false
	}
	if pageID == "" {
		return true
	}
	_, _, err := s.pages.Level(ctx, p, pageID)
	return err == nil
}

type draft struct {
	kind, pageID, threadID, body string
	actor                        *policy.Principal
}

func (s *Service) deliver(ctx context.Context, d draft, users []string) int {
	if len(users) == 0 {
		return 0
	}
	m, err := s.pages.Load(ctx, d.pageID)
	if err != nil {
		return 0
	}
	actorID, actorName := "", ""
	if d.actor != nil {
		actorID, actorName = d.actor.UserID, d.actor.Name
	}
	seen := map[string]bool{actorID: true}
	now := store.NowMS()
	var reached []string
	for _, u := range users {
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		if !s.canSee(ctx, u, d.pageID) {
			continue
		}
		if _, err := s.st.W.ExecContext(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, page_id, thread_id, actor_id, actor_name, title, body, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, ids.New(), u, m.WorkspaceID, d.kind, d.pageID, d.threadID, actorID, actorName, m.Title, clip(d.body, maxBody), now); err != nil {
			s.log.Warn("inbox: insert", "err", err)
			continue
		}
		reached = append(reached, u)
	}
	for _, u := range reached {
		s.hub.User(u, "inbox")
	}
	return len(reached)
}

func (s *Service) FromComment(ctx context.Context, e comment.Event) {
	if e.Page == nil {
		return
	}
	var mentioned []string
	for _, m := range e.Mentions {
		if m.Kind == "user" {
			mentioned = append(mentioned, m.ID)
		}
	}
	threadID := ""
	if e.Thread != nil {
		threadID = e.Thread.ID
	}
	body := ""
	if e.Comment != nil {
		body = comment.PlainText(e.Comment.Body)
	}
	s.deliver(ctx, draft{kind: "mention", pageID: e.Page.ID, threadID: threadID, body: body, actor: e.Actor}, mentioned)
	skip := map[string]bool{}
	for _, u := range mentioned {
		skip[u] = true
	}
	without := func(list []string) []string {
		out := list[:0:0]
		for _, u := range list {
			if !skip[u] {
				out = append(out, u)
			}
		}
		return out
	}
	switch e.Kind {
	case "thread":
		s.deliver(ctx, draft{kind: "comment", pageID: e.Page.ID, threadID: threadID, body: body, actor: e.Actor}, without([]string{e.Page.CreatedBy}))
	case "reply":
		s.deliver(ctx, draft{kind: "reply", pageID: e.Page.ID, threadID: threadID, body: body, actor: e.Actor}, without(s.participants(ctx, threadID)))
	case "suggestion":
		if e.Suggestion != nil {
			s.deliver(ctx, draft{kind: "suggestion", pageID: e.Page.ID, body: suggestionText(e.Suggestion), actor: e.Actor}, []string{e.Page.CreatedBy})
		}
	case "decision":
		if e.Suggestion != nil {
			s.deliver(ctx, draft{kind: "decision", pageID: e.Page.ID, body: e.Suggestion.Status + ": " + suggestionText(e.Suggestion), actor: e.Actor}, []string{e.Suggestion.AuthorID})
		}
	}
}

func suggestionText(x *comment.Suggestion) string {
	switch {
	case x.Original != "" && x.Text != "":
		return fmt.Sprintf("%q to %q", x.Original, x.Text)
	case x.Text != "":
		return fmt.Sprintf("add %q", x.Text)
	}
	return fmt.Sprintf("delete %q", x.Original)
}

func (s *Service) participants(ctx context.Context, threadID string) []string {
	rows, err := s.st.R.QueryContext(ctx, "SELECT DISTINCT author_id FROM comments WHERE thread_id = ? AND author_id != '' UNION SELECT created_by FROM comment_threads WHERE id = ?", threadID, threadID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var u string
		if rows.Scan(&u) == nil {
			out = append(out, u)
		}
	}
	return out
}

func (s *Service) FromRefs(ctx context.Context, pageID, actor string, mentioned []string, reminders []doc.Reminder) {
	if len(mentioned) > 0 {
		var p *policy.Principal
		if actor != "" {
			p = s.principal(ctx, actor)
		}
		s.deliver(ctx, draft{kind: "mention", pageID: pageID, actor: p}, mentioned)
	}
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		keep := map[string]bool{}
		for _, r := range reminders {
			if r.At <= 0 || r.User == "" {
				continue
			}
			keep[r.User+"\x00"+r.Ref] = true
			if _, err := tx.ExecContext(ctx, `INSERT INTO reminders (page_id, user_id, ref, due_at) VALUES (?, ?, ?, ?)
				ON CONFLICT (page_id, user_id, ref) DO UPDATE SET due_at = excluded.due_at, fired_at = CASE WHEN reminders.due_at = excluded.due_at THEN reminders.fired_at ELSE 0 END`,
				pageID, r.User, r.Ref, r.At); err != nil {
				return err
			}
		}
		rows, err := tx.QueryContext(ctx, "SELECT user_id, ref FROM reminders WHERE page_id = ?", pageID)
		if err != nil {
			return err
		}
		var gone [][2]string
		for rows.Next() {
			var u, ref string
			if rows.Scan(&u, &ref) == nil && !keep[u+"\x00"+ref] {
				gone = append(gone, [2]string{u, ref})
			}
		}
		rows.Close()
		for _, g := range gone {
			if _, err := tx.ExecContext(ctx, "DELETE FROM reminders WHERE page_id = ? AND user_id = ? AND ref = ?", pageID, g[0], g[1]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.log.Warn("inbox: reminders", "page", pageID, "err", err)
	}
}

func (s *Service) FromShare(ctx context.Context, m *page.Meta, kind, subject string) {
	actor := policy.From(ctx)
	users := []string{subject}
	if kind == "group" {
		users = nil
		rows, err := s.st.R.QueryContext(ctx, "SELECT user_id FROM group_members WHERE group_id = ?", subject)
		if err != nil {
			return
		}
		for rows.Next() {
			var u string
			if rows.Scan(&u) == nil {
				users = append(users, u)
			}
		}
		rows.Close()
	}
	s.deliver(ctx, draft{kind: "share", pageID: m.ID, actor: actor}, users)
}

func (s *Service) FromAssign(ctx context.Context, p *policy.Principal, rowID string, users []string) {
	s.deliver(ctx, draft{kind: "assign", pageID: rowID, actor: p}, users)
}

func (s *Service) FromAutomation(ctx context.Context, name, pageID, body string, users []string) int {
	return s.deliver(ctx, draft{kind: "automation", pageID: pageID, body: body, actor: &policy.Principal{Name: name}}, users)
}

func (s *Service) fireReminders(ctx context.Context) {
	now := store.NowMS()
	rows, err := s.st.R.QueryContext(ctx, "SELECT page_id, user_id, ref FROM reminders WHERE fired_at = 0 AND due_at <= ? LIMIT 200", now)
	if err != nil {
		return
	}
	type due struct{ page, user, ref string }
	var list []due
	for rows.Next() {
		var d due
		if rows.Scan(&d.page, &d.user, &d.ref) == nil {
			list = append(list, d)
		}
	}
	rows.Close()
	for _, d := range list {
		if _, err := s.st.W.ExecContext(ctx, "UPDATE reminders SET fired_at = ? WHERE page_id = ? AND user_id = ? AND ref = ?", now, d.page, d.user, d.ref); err != nil {
			continue
		}
		s.deliver(ctx, draft{kind: "reminder", pageID: d.page, body: d.ref}, []string{d.user})
	}
}

func (s *Service) List(ctx context.Context, p *policy.Principal, filter string, before int64) ([]Notification, int, error) {
	cond := "n.archived_at = 0"
	switch filter {
	case "unread":
		cond = "n.archived_at = 0 AND n.read_at = 0"
	case "archived":
		cond = "n.archived_at > 0"
	}
	if before <= 0 {
		before = 1 << 62
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT n.id, n.workspace_id, n.kind, n.page_id, COALESCE(pg.icon, ''), COALESCE(pg.title, n.title), n.thread_id, n.actor_id, COALESCE(u.name, n.actor_name),
			COALESCE(u.color, ''), COALESCE(u.avatar, ''), n.body, n.created_at, n.read_at, n.archived_at
		FROM notifications n LEFT JOIN pages pg ON pg.id = n.page_id LEFT JOIN users u ON u.id = n.actor_id
		WHERE n.user_id = ? AND `+cond+` AND n.created_at < ? ORDER BY n.created_at DESC LIMIT ?`, p.UserID, before, listLimit)
	if err != nil {
		return nil, 0, err
	}
	out := []Notification{}
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.WorkspaceID, &n.Kind, &n.PageID, &n.PageIcon, &n.Title, &n.ThreadID, &n.ActorID, &n.ActorName, &n.ActorColor, &n.ActorAvatar, &n.Body, &n.CreatedAt, &n.ReadAt, &n.ArchivedAt); err != nil {
			rows.Close()
			return nil, 0, err
		}
		out = append(out, n)
	}
	rows.Close()
	visible := out[:0]
	for _, n := range out {
		if n.PageID == "" {
			visible = append(visible, n)
			continue
		}
		if _, _, err := s.pages.Level(ctx, p, n.PageID); err == nil {
			visible = append(visible, n)
		}
	}
	unread, err := s.Unread(ctx, p)
	return visible, unread, err
}

func (s *Service) Unread(ctx context.Context, p *policy.Principal) (int, error) {
	var n int
	err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at = 0 AND archived_at = 0", p.UserID).Scan(&n)
	return n, err
}

type MarkInput struct {
	IDs      []string `json:"ids"`
	All      bool     `json:"all"`
	Read     *bool    `json:"read"`
	Archived *bool    `json:"archived"`
	Seen     bool     `json:"seen"`
}

func (s *Service) Mark(ctx context.Context, p *policy.Principal, in MarkInput) error {
	if !in.All && len(in.IDs) == 0 {
		return nil
	}
	if len(in.IDs) > 500 {
		return httpx.Fail(http.StatusBadRequest, "too_many", "up to 500 at a time")
	}
	now := store.NowMS()
	var sets []string
	var args []any
	at := func(on bool) int64 {
		if on {
			return now
		}
		return 0
	}
	if in.Read != nil {
		sets = append(sets, "read_at = ?")
		args = append(args, at(*in.Read))
	}
	if in.Archived != nil {
		sets = append(sets, "archived_at = ?")
		args = append(args, at(*in.Archived))
	}
	if in.Seen {
		sets = append(sets, "seen_at = CASE WHEN seen_at = 0 THEN ? ELSE seen_at END")
		args = append(args, now)
	}
	if len(sets) == 0 {
		return nil
	}
	q := "UPDATE notifications SET " + strings.Join(sets, ", ") + " WHERE user_id = ?"
	args = append(args, p.UserID)
	if !in.All {
		q += " AND id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(in.IDs)), ",") + ")"
		for _, id := range in.IDs {
			args = append(args, id)
		}
	} else if in.Read != nil && *in.Read {
		q += " AND read_at = 0"
	}
	if _, err := s.st.W.ExecContext(ctx, q, args...); err != nil {
		return err
	}
	if in.Read != nil || in.Archived != nil {
		s.hub.User(p.UserID, "inbox")
	}
	return nil
}

type Prefs struct {
	Email         string `json:"email"`
	PushKey       string `json:"push_key"`
	Subscriptions int    `json:"subscriptions"`
	MailEnabled   bool   `json:"mail_enabled"`
}

func (s *Service) Prefs(ctx context.Context, p *policy.Principal) (*Prefs, error) {
	out := &Prefs{Email: "all", PushKey: s.vapid.public, MailEnabled: s.mailer != nil && s.mailer.Enabled()}
	_ = s.st.R.QueryRowContext(ctx, "SELECT email FROM notify_prefs WHERE user_id = ?", p.UserID).Scan(&out.Email)
	err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM push_subscriptions WHERE user_id = ?", p.UserID).Scan(&out.Subscriptions)
	return out, err
}

func (s *Service) SetPrefs(ctx context.Context, p *policy.Principal, email string) error {
	if email != "all" && email != "mentions" && email != "off" {
		return httpx.Fail(http.StatusBadRequest, "bad_prefs", "email is all, mentions or off")
	}
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO notify_prefs (user_id, email, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET email = excluded.email, updated_at = excluded.updated_at`, p.UserID, email, store.NowMS())
	return err
}

type SubscribeInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *Service) Subscribe(ctx context.Context, p *policy.Principal, in SubscribeInput) error {
	if !p.IsSession() {
		return httpx.ErrSessionOnly
	}
	if !strings.HasPrefix(in.Endpoint, "https://") && !strings.HasPrefix(in.Endpoint, "http://") || len(in.Endpoint) > 2000 {
		return httpx.Fail(http.StatusBadRequest, "bad_endpoint", "the push endpoint must be a web address")
	}
	if _, err := encrypt(in.Keys.P256dh, in.Keys.Auth, []byte("{}")); err != nil {
		return httpx.Fail(http.StatusBadRequest, "bad_keys", "the push keys are not valid")
	}
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, created_at) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = excluded.user_id, p256dh = excluded.p256dh, auth = excluded.auth, failures = 0`,
		ids.New(), p.UserID, in.Endpoint, in.Keys.P256dh, in.Keys.Auth, store.NowMS())
	return err
}

func (s *Service) Unsubscribe(ctx context.Context, p *policy.Principal, endpoint string) error {
	_, err := s.st.W.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE user_id = ? AND endpoint = ?", p.UserID, endpoint)
	return err
}

type pushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

func (s *Service) pushTo(ctx context.Context, userID string, payload pushPayload) int {
	rows, err := s.st.R.QueryContext(ctx, "SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?", userID)
	if err != nil {
		return 0
	}
	type target struct{ endpoint, p256dh, auth string }
	var subs []target
	for rows.Next() {
		var x target
		if rows.Scan(&x.endpoint, &x.p256dh, &x.auth) == nil {
			subs = append(subs, x)
		}
	}
	rows.Close()
	body, _ := json.Marshal(payload)
	sent := 0
	sub := subject(s.settings)
	for _, x := range subs {
		switch s.vapid.send(ctx, s.client, sub, x.endpoint, x.p256dh, x.auth, body) {
		case pushOK:
			sent++
			_, _ = s.st.W.ExecContext(ctx, "UPDATE push_subscriptions SET failures = 0 WHERE endpoint = ?", x.endpoint)
		case pushGone:
			_, _ = s.st.W.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE endpoint = ?", x.endpoint)
		case pushFailed:
			_, _ = s.st.W.ExecContext(ctx, "UPDATE push_subscriptions SET failures = failures + 1 WHERE endpoint = ?", x.endpoint)
			_, _ = s.st.W.ExecContext(ctx, "DELETE FROM push_subscriptions WHERE endpoint = ? AND failures >= 5", x.endpoint)
		}
	}
	return sent
}

func (s *Service) TestPush(ctx context.Context, p *policy.Principal) (int, error) {
	n := s.pushTo(ctx, p.UserID, pushPayload{Title: s.settings.Get("instance.name"), Body: "Notifications reach this device.", URL: "/inbox", Tag: "test"})
	if n == 0 {
		return 0, httpx.Fail(http.StatusConflict, "no_push", "no device took the test notification")
	}
	return n, nil
}

func headline(lang string, n Notification) string {
	who := n.ActorName
	if who == "" {
		who = i18n.T(lang, "Someone")
	}
	title := n.Title
	if title == "" {
		title = i18n.T(lang, "Untitled")
	}
	switch n.Kind {
	case "mention":
		return i18n.T(lang, "%s mentioned you in %s", who, title)
	case "comment":
		return i18n.T(lang, "%s commented on %s", who, title)
	case "reply":
		return i18n.T(lang, "%s replied in %s", who, title)
	case "suggestion":
		return i18n.T(lang, "%s suggested an edit to %s", who, title)
	case "decision":
		return i18n.T(lang, "%s decided on your suggestion in %s", who, title)
	case "share":
		return i18n.T(lang, "%s shared %s with you", who, title)
	case "assign":
		return i18n.T(lang, "%s assigned you to %s", who, title)
	case "reminder":
		return i18n.T(lang, "Reminder in %s", title)
	case "automation":
		return i18n.T(lang, "%s ran on %s", who, title)
	}
	return title
}

func (s *Service) pending(ctx context.Context, where string, args ...any) ([]struct {
	user string
	n    Notification
}, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT n.user_id, n.id, n.kind, n.page_id, n.thread_id, n.actor_name, COALESCE(pg.title, n.title), n.body, n.created_at
		FROM notifications n LEFT JOIN pages pg ON pg.id = n.page_id WHERE `+where+` ORDER BY n.created_at LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		user string
		n    Notification
	}
	for rows.Next() {
		var x struct {
			user string
			n    Notification
		}
		if err := rows.Scan(&x.user, &x.n.ID, &x.n.Kind, &x.n.PageID, &x.n.ThreadID, &x.n.ActorName, &x.n.Title, &x.n.Body, &x.n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func link(n Notification) string {
	if n.PageID == "" {
		return "/inbox"
	}
	if n.ThreadID != "" {
		return "/p/" + n.PageID + "?thread=" + n.ThreadID
	}
	return "/p/" + n.PageID
}

func (s *Service) sendPushes(ctx context.Context) {
	now := store.NowMS()
	list, err := s.pending(ctx, "n.pushed_at = 0 AND n.read_at = 0 AND n.seen_at = 0 AND n.archived_at = 0 AND n.created_at <= ? AND n.created_at > ?", now-pushDelay.Milliseconds(), now-24*3600*1000)
	if err != nil {
		return
	}
	for _, x := range list {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE notifications SET pushed_at = ? WHERE id = ?", now, x.n.ID)
		s.pushTo(ctx, x.user, pushPayload{Title: headline(i18n.UserLang(ctx, s.st.R, x.user), x.n), Body: x.n.Body, URL: link(x.n), Tag: x.n.ID})
	}
}

func (s *Service) sendMail(ctx context.Context) {
	if s.mailer == nil || !s.mailer.Enabled() {
		return
	}
	now := store.NowMS()
	delay := int64(s.settings.Int("notify.email_delay_minutes")) * 60 * 1000
	list, err := s.pending(ctx, "n.emailed_at = 0 AND n.read_at = 0 AND n.seen_at = 0 AND n.archived_at = 0 AND n.created_at <= ? AND n.created_at > ?", now-delay, now-7*24*3600*1000)
	if err != nil {
		return
	}
	by := map[string][]Notification{}
	var order []string
	for _, x := range list {
		if _, ok := by[x.user]; !ok {
			order = append(order, x.user)
		}
		by[x.user] = append(by[x.user], x.n)
	}
	base := strings.TrimRight(s.settings.Get("instance.public_url"), "/")
	for _, user := range order {
		items := by[user]
		mark := func() {
			for _, n := range items {
				_, _ = s.st.W.ExecContext(ctx, "UPDATE notifications SET emailed_at = ? WHERE id = ?", now, n.ID)
			}
		}
		if s.hub.Online(user) {
			continue
		}
		pref := "all"
		var email string
		_ = s.st.R.QueryRowContext(ctx, "SELECT email FROM notify_prefs WHERE user_id = ?", user).Scan(&pref)
		if err := s.st.R.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ? AND disabled = 0", user).Scan(&email); err != nil || pref == "off" {
			mark()
			continue
		}
		lang := i18n.UserLang(ctx, s.st.R, user)
		var lines []string
		for _, n := range items {
			if pref == "mentions" && n.Kind != "mention" && n.Kind != "reminder" && n.Kind != "assign" {
				continue
			}
			line := "- " + headline(lang, n)
			if n.Body != "" {
				line += "\n  " + n.Body
			}
			lines = append(lines, line+"\n  "+base+link(n))
		}
		mark()
		if len(lines) == 0 {
			continue
		}
		subject := headline(lang, items[0])
		if len(lines) > 1 {
			subject = i18n.T(lang, "%d new notifications", len(lines))
		}
		body := strings.Join(lines, "\n\n") + "\n\n" + i18n.T(lang, "Open your inbox: %s", base+"/inbox") + "\n"
		if err := s.mailer.Send(ctx, email, subject, body); err != nil {
			s.log.Warn("inbox: email", "err", err)
		}
	}
}

func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(pushPeriod)
	defer t.Stop()
	tick := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s.fireReminders(ctx)
		s.sendPushes(ctx)
		tick++
		if tick%15 == 0 {
			s.sendMail(ctx)
		}
	}
}
