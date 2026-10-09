package comment

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	MaxBody   = 10000
	MaxQuote  = 500
	MaxAnchor = 512
)

var Reactions = []string{"heart", "star", "sparkle", "eye", "rocket", "lightbulb"}

var mentionRE = regexp.MustCompile(`@\[([^\]\n]{1,200})\]\((user|page):([a-z2-7]{26})\)`)

type Reaction struct {
	Icon  string   `json:"icon"`
	Count int      `json:"count"`
	Mine  bool     `json:"mine"`
	Names []string `json:"names"`
}

type Comment struct {
	ID           string     `json:"id"`
	ThreadID     string     `json:"thread_id"`
	Body         string     `json:"body"`
	AuthorID     string     `json:"author_id"`
	AuthorName   string     `json:"author_name"`
	AuthorColor  string     `json:"author_color"`
	AuthorAvatar string     `json:"author_avatar"`
	CreatedAt    int64      `json:"created_at"`
	EditedAt     int64      `json:"edited_at"`
	Reactions    []Reaction `json:"reactions"`
}

type Thread struct {
	ID         string    `json:"id"`
	PageID     string    `json:"page_id"`
	Anchor     string    `json:"anchor"`
	Start      string    `json:"start"`
	End        string    `json:"end"`
	Quote      string    `json:"quote"`
	ResolvedAt int64     `json:"resolved_at"`
	ResolvedBy string    `json:"resolved_by"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  int64     `json:"created_at"`
	UpdatedAt  int64     `json:"updated_at"`
	Comments   []Comment `json:"comments"`
}

type Listing struct {
	Threads     []*Thread    `json:"threads"`
	Suggestions []Suggestion `json:"suggestions"`
	CanComment  bool         `json:"can_comment"`
	CanModerate bool         `json:"can_moderate"`
	CanDecide   bool         `json:"can_decide"`
}

type Mention struct {
	Kind string
	ID   string
}

type Event struct {
	Kind       string
	Page       *page.Meta
	Thread     *Thread
	Comment    *Comment
	Suggestion *Suggestion
	Actor      *policy.Principal
	Mentions   []Mention
}

type Service struct {
	st       *store.Store
	pages    *page.Service
	audit    *audit.Log
	hub      *events.Hub
	log      *slog.Logger
	watchers []func(ctx context.Context, e Event)
}

func New(st *store.Store, pages *page.Service, al *audit.Log, hub *events.Hub, log *slog.Logger) *Service {
	return &Service{st: st, pages: pages, audit: al, hub: hub, log: log}
}

func (s *Service) OnComment(fn func(ctx context.Context, e Event)) {
	s.watchers = append(s.watchers, fn)
}

func Mentions(body string) []Mention {
	seen := map[string]bool{}
	var out []Mention
	for _, m := range mentionRE.FindAllStringSubmatch(body, 50) {
		k := m[2] + ":" + m[3]
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, Mention{Kind: m[2], ID: m[3]})
	}
	return out
}

func PlainText(body string) string {
	return mentionRE.ReplaceAllString(body, "@$1")
}

func cleanBody(raw string) (string, error) {
	b := strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	if b == "" {
		return "", httpx.Fail(http.StatusBadRequest, "empty_comment", "write something first")
	}
	if utf8.RuneCountInString(b) > MaxBody {
		return "", httpx.Fail(http.StatusBadRequest, "comment_too_long", "a comment can be up to 10,000 characters")
	}
	return b, nil
}

func (s *Service) emit(ctx context.Context, m *page.Meta) {
	aud, err := s.pages.Audience(ctx, m)
	if err != nil {
		s.log.Warn("comment: audience", "err", err)
		return
	}
	s.hub.PageNote(m.WorkspaceID, aud, "comments", m.ID)
}

func (s *Service) notify(ctx context.Context, e Event) {
	for _, fn := range s.watchers {
		fn(ctx, e)
	}
}

func (s *Service) commentable(ctx context.Context, p *policy.Principal, pageID string) (*page.Meta, error) {
	if !p.CanWrite() {
		return nil, httpx.ErrForbidden
	}
	if _, err := s.pages.CanComment(ctx, p, pageID); err != nil {
		return nil, err
	}
	return s.pages.Load(ctx, pageID)
}

func (s *Service) List(ctx context.Context, p *policy.Principal, pageID string) (*Listing, error) {
	level, _, err := s.pages.Level(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	threads, err := s.threads(ctx, p, "t.page_id = ?", pageID)
	if err != nil {
		return nil, err
	}
	sugg, err := s.suggestions(ctx, pageID)
	if err != nil {
		return nil, err
	}
	write := p.CanWrite()
	return &Listing{Threads: threads, Suggestions: sugg, CanComment: write && level >= page.LevelComment, CanModerate: write && level >= page.LevelEdit, CanDecide: write && level >= page.LevelContent}, nil
}

func (s *Service) threads(ctx context.Context, p *policy.Principal, where string, args ...any) ([]*Thread, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT t.id, t.page_id, t.anchor, t.anchor_start, t.anchor_end, t.quote, t.resolved_at, t.resolved_by, t.created_by, t.created_at, t.updated_at
		FROM comment_threads t WHERE `+where+` ORDER BY t.created_at`, args...)
	if err != nil {
		return nil, err
	}
	out := []*Thread{}
	by := map[string]*Thread{}
	for rows.Next() {
		t := &Thread{Comments: []Comment{}}
		if err := rows.Scan(&t.ID, &t.PageID, &t.Anchor, &t.Start, &t.End, &t.Quote, &t.ResolvedAt, &t.ResolvedBy, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, t)
		by[t.ID] = t
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(out) == 0 {
		return out, err
	}
	rows, err = s.st.R.QueryContext(ctx, `SELECT c.id, c.thread_id, c.body, c.author_id, COALESCE(u.name, c.author_name), COALESCE(u.color, ''), COALESCE(u.avatar, ''), c.created_at, c.edited_at
		FROM comments c JOIN comment_threads t ON t.id = c.thread_id LEFT JOIN users u ON u.id = c.author_id
		WHERE `+where+` ORDER BY c.created_at`, args...)
	if err != nil {
		return nil, err
	}
	comments := map[string]*Comment{}
	var order []*Comment
	for rows.Next() {
		c := &Comment{Reactions: []Reaction{}}
		if err := rows.Scan(&c.ID, &c.ThreadID, &c.Body, &c.AuthorID, &c.AuthorName, &c.AuthorColor, &c.AuthorAvatar, &c.CreatedAt, &c.EditedAt); err != nil {
			rows.Close()
			return nil, err
		}
		comments[c.ID] = c
		order = append(order, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = s.st.R.QueryContext(ctx, `SELECT r.comment_id, r.icon, r.user_id, COALESCE(u.name, '') FROM comment_reactions r
		JOIN comments c ON c.id = r.comment_id JOIN comment_threads t ON t.id = c.thread_id LEFT JOIN users u ON u.id = r.user_id
		WHERE `+where+` ORDER BY r.created_at`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var cid, icon, user, name string
		if err := rows.Scan(&cid, &icon, &user, &name); err != nil {
			rows.Close()
			return nil, err
		}
		c := comments[cid]
		if c == nil {
			continue
		}
		i := -1
		for k := range c.Reactions {
			if c.Reactions[k].Icon == icon {
				i = k
			}
		}
		if i < 0 {
			c.Reactions = append(c.Reactions, Reaction{Icon: icon, Names: []string{}})
			i = len(c.Reactions) - 1
		}
		r := &c.Reactions[i]
		r.Count++
		r.Mine = r.Mine || (p != nil && user == p.UserID)
		if name != "" && len(r.Names) < 12 {
			r.Names = append(r.Names, name)
		}
	}
	rows.Close()
	for _, c := range order {
		if t := by[c.ThreadID]; t != nil {
			t.Comments = append(t.Comments, *c)
		}
	}
	return out, rows.Err()
}

type CreateInput struct {
	Anchor string `json:"anchor"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Quote  string `json:"quote"`
	Body   string `json:"body"`
}

func (s *Service) Create(ctx context.Context, p *policy.Principal, pageID string, in CreateInput) (*Thread, error) {
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	body, err := cleanBody(in.Body)
	if err != nil {
		return nil, err
	}
	anchor := in.Anchor
	if anchor == "" {
		anchor = "page"
	}
	switch anchor {
	case "page":
		in.Start, in.End, in.Quote = "", "", ""
	case "text", "block":
		if in.Start == "" || in.End == "" || len(in.Start) > MaxAnchor || len(in.End) > MaxAnchor {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_anchor", "a text comment needs a start and an end")
		}
	default:
		return nil, httpx.Fail(http.StatusBadRequest, "bad_anchor", "anchor is page, text or block")
	}
	quote := strings.TrimSpace(in.Quote)
	if utf8.RuneCountInString(quote) > MaxQuote {
		quote = string([]rune(quote)[:MaxQuote])
	}
	now := store.NowMS()
	t := &Thread{ID: ids.New(), PageID: pageID, Anchor: anchor, Start: in.Start, End: in.End, Quote: quote, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	c := &Comment{ID: ids.New(), ThreadID: t.ID, Body: body, AuthorID: p.UserID, AuthorName: p.Name, CreatedAt: now, Reactions: []Reaction{}}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO comment_threads (id, page_id, anchor, anchor_start, anchor_end, quote, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.ID, pageID, anchor, t.Start, t.End, quote, p.UserID, now, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO comments (id, thread_id, page_id, body, author_id, author_name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.ID, t.ID, pageID, body, p.UserID, p.Name, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "comment.create", WorkspaceID: m.WorkspaceID, PageID: pageID, Detail: t.ID})
	})
	if err != nil {
		return nil, err
	}
	t.Comments = []Comment{*c}
	s.emit(ctx, m)
	s.notify(ctx, Event{Kind: "thread", Page: m, Thread: t, Comment: c, Actor: p, Mentions: Mentions(body)})
	return t, nil
}

func (s *Service) thread(ctx context.Context, id string) (*Thread, error) {
	var t Thread
	err := s.st.R.QueryRowContext(ctx, "SELECT id, page_id, anchor, quote, resolved_at, created_by FROM comment_threads WHERE id = ?", id).
		Scan(&t.ID, &t.PageID, &t.Anchor, &t.Quote, &t.ResolvedAt, &t.CreatedBy)
	if err != nil {
		return nil, store.NoRows(err)
	}
	return &t, nil
}

func (s *Service) Reply(ctx context.Context, p *policy.Principal, threadID, raw string) (*Comment, error) {
	t, err := s.thread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	m, err := s.commentable(ctx, p, t.PageID)
	if err != nil {
		return nil, err
	}
	body, err := cleanBody(raw)
	if err != nil {
		return nil, err
	}
	now := store.NowMS()
	c := &Comment{ID: ids.New(), ThreadID: threadID, Body: body, AuthorID: p.UserID, AuthorName: p.Name, CreatedAt: now, Reactions: []Reaction{}}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM comments WHERE thread_id = ?", threadID).Scan(&n); err != nil {
			return err
		}
		if n >= 500 {
			return httpx.Fail(http.StatusConflict, "thread_full", "a discussion can hold up to 500 comments")
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO comments (id, thread_id, page_id, body, author_id, author_name, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.ID, threadID, t.PageID, body, p.UserID, p.Name, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE comment_threads SET updated_at = ?, resolved_at = 0, resolved_by = '' WHERE id = ?", now, threadID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "comment.reply", WorkspaceID: m.WorkspaceID, PageID: t.PageID, Detail: threadID})
	})
	if err != nil {
		return nil, err
	}
	s.emit(ctx, m)
	s.notify(ctx, Event{Kind: "reply", Page: m, Thread: t, Comment: c, Actor: p, Mentions: Mentions(body)})
	return c, nil
}

func (s *Service) comment(ctx context.Context, id string) (*Comment, string, error) {
	var c Comment
	var pageID string
	err := s.st.R.QueryRowContext(ctx, "SELECT id, thread_id, page_id, body, author_id, created_at FROM comments WHERE id = ?", id).
		Scan(&c.ID, &c.ThreadID, &pageID, &c.Body, &c.AuthorID, &c.CreatedAt)
	if err != nil {
		return nil, "", store.NoRows(err)
	}
	return &c, pageID, nil
}

func (s *Service) Edit(ctx context.Context, p *policy.Principal, id, raw string) error {
	c, pageID, err := s.comment(ctx, id)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return err
	}
	if c.AuthorID == "" || c.AuthorID != p.UserID {
		return httpx.Fail(http.StatusForbidden, "not_yours", "only the author can edit a comment")
	}
	body, err := cleanBody(raw)
	if err != nil {
		return err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE comments SET body = ?, edited_at = ? WHERE id = ?", body, store.NowMS(), id); err != nil {
		return err
	}
	s.emit(ctx, m)
	before := map[string]bool{}
	for _, x := range Mentions(c.Body) {
		before[x.Kind+":"+x.ID] = true
	}
	var added []Mention
	for _, x := range Mentions(body) {
		if !before[x.Kind+":"+x.ID] {
			added = append(added, x)
		}
	}
	if len(added) > 0 {
		t, err := s.thread(ctx, c.ThreadID)
		if err == nil {
			c.Body = body
			s.notify(ctx, Event{Kind: "edit", Page: m, Thread: t, Comment: c, Actor: p, Mentions: added})
		}
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, p *policy.Principal, id string) error {
	c, pageID, err := s.comment(ctx, id)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return err
	}
	if c.AuthorID == "" || c.AuthorID != p.UserID {
		if _, err := s.pages.CanEdit(ctx, p, pageID); err != nil {
			return httpx.Fail(http.StatusForbidden, "not_yours", "only the author or an editor of the page can delete a comment")
		}
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM comments WHERE id = ?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM comment_threads WHERE id = ? AND NOT EXISTS (SELECT 1 FROM comments WHERE thread_id = ?)", c.ThreadID, c.ThreadID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "comment.delete", WorkspaceID: m.WorkspaceID, PageID: pageID, Detail: c.ThreadID})
	})
	if err != nil {
		return err
	}
	s.emit(ctx, m)
	return nil
}

func (s *Service) Resolve(ctx context.Context, p *policy.Principal, threadID string, resolved bool) error {
	t, err := s.thread(ctx, threadID)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, t.PageID)
	if err != nil {
		return err
	}
	at, by, action := int64(0), "", "comment.reopen"
	if resolved {
		at, by, action = store.NowMS(), p.UserID, "comment.resolve"
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE comment_threads SET resolved_at = ?, resolved_by = ? WHERE id = ?", at, by, threadID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: action, WorkspaceID: m.WorkspaceID, PageID: t.PageID, Detail: threadID})
	})
	if err != nil {
		return err
	}
	s.emit(ctx, m)
	if resolved {
		s.notify(ctx, Event{Kind: "resolve", Page: m, Thread: t, Actor: p})
	}
	return nil
}

func validReaction(icon string) bool {
	for _, r := range Reactions {
		if r == icon {
			return true
		}
	}
	return false
}

func (s *Service) React(ctx context.Context, p *policy.Principal, commentID, icon string, on bool) error {
	if !validReaction(icon) {
		return httpx.Fail(http.StatusBadRequest, "bad_reaction", "use one of the reaction icons")
	}
	_, pageID, err := s.comment(ctx, commentID)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return err
	}
	if on {
		_, err = s.st.W.ExecContext(ctx, "INSERT INTO comment_reactions (comment_id, user_id, icon, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", commentID, p.UserID, icon, store.NowMS())
	} else {
		_, err = s.st.W.ExecContext(ctx, "DELETE FROM comment_reactions WHERE comment_id = ? AND user_id = ? AND icon = ?", commentID, p.UserID, icon)
	}
	if err != nil {
		return err
	}
	detail := icon + " added"
	if !on {
		detail = icon + " removed"
	}
	_ = s.audit.Record(ctx, nil, p, audit.Entry{Action: "comment.react", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: detail})
	s.emit(ctx, m)
	return nil
}
