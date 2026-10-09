package automation

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/agent"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/inbox"
	"github.com/nodemode-ai/form.md/internal/mail"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/webhook"
)

const (
	maxActions    = 10
	maxPerCol     = 50
	maxEditRows   = 500
	keepRuns      = 50
	debounceDelay = 1500 * time.Millisecond
)

type Trigger struct {
	Props   []string           `json:"props,omitempty"`
	Filter  *collection.Filter `json:"filter,omitempty"`
	Every   string             `json:"every,omitempty"`
	N       int                `json:"n,omitempty"`
	At      string             `json:"at,omitempty"`
	Weekday int                `json:"weekday,omitempty"`
	TZ      string             `json:"tz,omitempty"`
}

type Action struct {
	Kind       string                     `json:"kind"`
	Collection string                     `json:"collection,omitempty"`
	Target     string                     `json:"target,omitempty"`
	Filter     *collection.Filter         `json:"filter,omitempty"`
	Title      string                     `json:"title,omitempty"`
	Values     map[string]json.RawMessage `json:"values,omitempty"`
	People     []string                   `json:"people,omitempty"`
	Subject    string                     `json:"subject,omitempty"`
	Message    string                     `json:"message,omitempty"`
	URL        string                     `json:"url,omitempty"`
	Markdown   string                     `json:"markdown,omitempty"`
	Where      string                     `json:"where,omitempty"`
	Page       string                     `json:"page,omitempty"`
}

type Automation struct {
	ID           string   `json:"id"`
	WorkspaceID  string   `json:"workspace_id"`
	CollectionID string   `json:"collection_id"`
	CreatedBy    string   `json:"created_by"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	TriggerKind  string   `json:"trigger_kind"`
	Trigger      Trigger  `json:"trigger"`
	Actions      []Action `json:"actions"`
	NextRunAt    int64    `json:"next_run_at"`
	LastRunAt    int64    `json:"last_run_at"`
	LastStatus   string   `json:"last_status"`
	Runs         int      `json:"runs"`
	CreatedAt    int64    `json:"created_at"`
	UpdatedAt    int64    `json:"updated_at"`
	baseURL      string
}

type Run struct {
	ID     string `json:"id"`
	At     int64  `json:"at"`
	RowID  string `json:"row_id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type Deps struct {
	Store  *store.Store
	Auth   *auth.Service
	Gate   *policy.Gate
	Pages  *page.Service
	Cols   *collection.Service
	Inbox  *inbox.Service
	Mail   *mail.Mailer
	Hooks  *webhook.Service
	Box    *secrets.Box
	Public func() string
	Log    *slog.Logger
}

type Service struct {
	Deps
	mu      sync.Mutex
	pending map[string]*time.Timer
	wake    chan struct{}
}

func New(d Deps) *Service {
	return &Service{Deps: d, pending: map[string]*time.Timer{}, wake: make(chan struct{}, 1)}
}

var triggerKinds = map[string]bool{"row_added": true, "prop_edited": true, "schedule": true}

var serverKinds = map[string]bool{"edit": true, "add": true, "notify": true, "mail": true, "webhook": true}

var buttonKinds = map[string]bool{"edit": true, "add": true, "notify": true, "mail": true, "webhook": true, "insert": true, "open": true, "confirm": true}

const cols = "id, workspace_id, collection_id, created_by, name, enabled, trigger_kind, trigger_config, actions, base_url, next_run_at, last_run_at, last_status, runs, created_at, updated_at"

func scan(r interface{ Scan(...any) error }) (*Automation, error) {
	var a Automation
	var en int
	var trig, acts string
	if err := r.Scan(&a.ID, &a.WorkspaceID, &a.CollectionID, &a.CreatedBy, &a.Name, &en, &a.TriggerKind, &trig, &acts, &a.baseURL, &a.NextRunAt, &a.LastRunAt, &a.LastStatus, &a.Runs, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	a.Enabled = en == 1
	_ = json.Unmarshal([]byte(trig), &a.Trigger)
	_ = json.Unmarshal([]byte(acts), &a.Actions)
	if a.Actions == nil {
		a.Actions = []Action{}
	}
	return &a, nil
}

func bad(code, msg string) error { return httpx.Fail(http.StatusBadRequest, code, msg) }

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

func (s *Service) design(ctx context.Context, p *policy.Principal, collectionID string) (*collection.Out, error) {
	out, err := s.Cols.Get(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	if !out.CanDesign {
		return nil, httpx.Fail(http.StatusForbidden, "forbidden", "only people who can edit this database can change its automations")
	}
	return out, nil
}

func (s *Service) checkActions(ctx context.Context, p *policy.Principal, workspaceID string, list []Action, kinds map[string]bool) error {
	if len(list) == 0 {
		return bad("bad_actions", "add at least one action")
	}
	if len(list) > maxActions {
		return bad("bad_actions", "up to 10 actions")
	}
	for i := range list {
		a := &list[i]
		if !kinds[a.Kind] {
			return bad("bad_action", "unknown action "+a.Kind)
		}
		a.Title = clip(a.Title, 500)
		a.Subject = clip(strings.TrimSpace(a.Subject), 200)
		a.Message = clip(a.Message, 4000)
		a.Markdown = clip(a.Markdown, 20000)
		if len(a.Values) > 50 || len(a.People) > 50 {
			return bad("bad_action", "an action can set up to 50 values or people")
		}
		switch a.Kind {
		case "add":
			if a.Collection == "" {
				return bad("bad_action", "pick a database to add pages to")
			}
		case "edit":
			if a.Target == "" {
				a.Target = "trigger"
			}
			if a.Target != "trigger" && a.Target != "all" {
				return bad("bad_action", "edit the triggering page or all pages that match")
			}
			if a.Target == "all" && a.Collection == "" {
				return bad("bad_action", "pick a database whose pages to edit")
			}
			if len(a.Values) == 0 {
				return bad("bad_action", "pick at least one property to set")
			}
		case "notify", "mail":
			if len(a.People) == 0 {
				return bad("bad_action", "pick who to tell")
			}
			if a.Kind == "mail" && a.Subject == "" {
				return bad("bad_action", "a mail needs a subject")
			}
		case "webhook":
			u, err := s.Hooks.CheckURL(strings.TrimSpace(a.URL))
			if err != nil {
				return err
			}
			a.URL = u
		case "open":
			if a.Page == "" && a.URL == "" {
				return bad("bad_action", "pick a page or address to open")
			}
		}
		if a.Collection != "" {
			out, err := s.Cols.Get(ctx, p, a.Collection)
			if err != nil {
				return bad("bad_action", "that database cannot be reached")
			}
			if out.WorkspaceID != workspaceID {
				return bad("bad_action", "actions stay inside this workspace")
			}
		}
	}
	return nil
}

func (s *Service) checkInput(ctx context.Context, p *policy.Principal, workspaceID string, a *Automation) error {
	a.Name = strings.TrimSpace(clip(a.Name, 120))
	if a.Name == "" {
		a.Name = "Automation"
	}
	if !triggerKinds[a.TriggerKind] {
		return bad("bad_trigger", "the trigger is row_added, prop_edited or schedule")
	}
	if len(a.Trigger.Props) > 50 {
		return bad("bad_trigger", "up to 50 properties")
	}
	if a.TriggerKind == "schedule" {
		switch a.Trigger.Every {
		case "hours", "day", "week":
		default:
			return bad("bad_trigger", "a schedule runs every few hours, every day or every week")
		}
		if a.Trigger.TZ != "" {
			if _, err := time.LoadLocation(a.Trigger.TZ); err != nil {
				return bad("bad_trigger", "unknown time zone")
			}
		}
	}
	return s.checkActions(ctx, p, workspaceID, a.Actions, serverKinds)
}

func (s *Service) nextRun(a *Automation) int64 {
	if !a.Enabled || a.TriggerKind != "schedule" {
		return 0
	}
	return agent.NextRun(agent.Trigger{Every: a.Trigger.Every, N: a.Trigger.N, At: a.Trigger.At, Weekday: a.Trigger.Weekday, TZ: a.Trigger.TZ}, time.Now())
}

func (s *Service) List(ctx context.Context, p *policy.Principal, collectionID string) ([]*Automation, bool, error) {
	out, err := s.Cols.Get(ctx, p, collectionID)
	if err != nil {
		return nil, false, err
	}
	if !out.CanWrite {
		return nil, false, httpx.ErrForbidden
	}
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+cols+" FROM automations WHERE collection_id = ? ORDER BY created_at", collectionID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	list := []*Automation{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		list = append(list, a)
	}
	return list, out.CanDesign, rows.Err()
}

func (s *Service) get(ctx context.Context, id string) (*Automation, error) {
	a, err := scan(s.Store.R.QueryRowContext(ctx, "SELECT "+cols+" FROM automations WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return a, err
}

func (s *Service) editable(ctx context.Context, p *policy.Principal, id string) (*Automation, error) {
	a, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.design(ctx, p, a.CollectionID); err != nil {
		if errors.Is(err, httpx.ErrNotFound) || errors.Is(err, store.ErrNotFound) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

type Input struct {
	Automation
	Enabled *bool `json:"enabled"`
}

func (s *Service) Create(ctx context.Context, p *policy.Principal, collectionID, base string, req Input) (*Automation, error) {
	in := req.Automation
	in.Enabled = req.Enabled == nil || *req.Enabled
	out, err := s.design(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	var n int
	_ = s.Store.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM automations WHERE collection_id = ?", collectionID).Scan(&n)
	if n >= maxPerCol {
		return nil, bad("too_many", "a database can have up to 50 automations")
	}
	a := in
	if err := s.checkInput(ctx, p, out.WorkspaceID, &a); err != nil {
		return nil, err
	}
	now := store.NowMS()
	a.ID, a.WorkspaceID, a.CollectionID, a.CreatedBy, a.CreatedAt, a.UpdatedAt = ids.New(), out.WorkspaceID, collectionID, p.UserID, now, now
	a.NextRunAt = s.nextRun(&a)
	trig, _ := json.Marshal(a.Trigger)
	acts, _ := json.Marshal(a.Actions)
	if _, err := s.Store.W.ExecContext(ctx, "INSERT INTO automations ("+cols+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '', 0, ?, ?)",
		a.ID, a.WorkspaceID, a.CollectionID, a.CreatedBy, a.Name, boolInt(a.Enabled), a.TriggerKind, string(trig), string(acts), base, a.NextRunAt, now, now); err != nil {
		return nil, err
	}
	s.poke()
	return s.get(ctx, a.ID)
}

func (s *Service) Update(ctx context.Context, p *policy.Principal, id, base string, req Input) (*Automation, error) {
	cur, err := s.editable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	a := req.Automation
	a.Enabled = cur.Enabled
	if req.Enabled != nil {
		a.Enabled = *req.Enabled
	}
	if err := s.checkInput(ctx, p, cur.WorkspaceID, &a); err != nil {
		return nil, err
	}
	a.NextRunAt = s.nextRun(&a)
	trig, _ := json.Marshal(a.Trigger)
	acts, _ := json.Marshal(a.Actions)
	if _, err := s.Store.W.ExecContext(ctx, "UPDATE automations SET name = ?, enabled = ?, trigger_kind = ?, trigger_config = ?, actions = ?, base_url = ?, next_run_at = ?, created_by = ?, updated_at = ? WHERE id = ?",
		a.Name, boolInt(a.Enabled), a.TriggerKind, string(trig), string(acts), base, a.NextRunAt, p.UserID, store.NowMS(), id); err != nil {
		return nil, err
	}
	s.poke()
	return s.get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, p *policy.Principal, id string) error {
	if _, err := s.editable(ctx, p, id); err != nil {
		return err
	}
	_, err := s.Store.W.ExecContext(ctx, "DELETE FROM automations WHERE id = ?", id)
	return err
}

func (s *Service) Runs(ctx context.Context, p *policy.Principal, id string) ([]Run, error) {
	a, err := s.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.List(ctx, p, a.CollectionID); err != nil {
		return nil, err
	}
	rows, err := s.Store.R.QueryContext(ctx, "SELECT id, at, row_id, status, detail FROM automation_runs WHERE automation_id = ? ORDER BY at DESC LIMIT ?", id, keepRuns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.At, &r.RowID, &r.Status, &r.Detail); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) key(ctx context.Context, workspaceID string) (string, error) {
	var sealed string
	err := s.Store.R.QueryRowContext(ctx, "SELECT secret FROM automation_keys WHERE workspace_id = ?", workspaceID).Scan(&sealed)
	if err == nil {
		return s.Box.Unseal(sealed)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	return s.rotate(ctx, workspaceID, false)
}

func (s *Service) rotate(ctx context.Context, workspaceID string, replace bool) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	secret := "fmsk_" + hex.EncodeToString(b)
	q := "INSERT INTO automation_keys (workspace_id, secret, created_at) VALUES (?, ?, ?) ON CONFLICT (workspace_id) DO NOTHING"
	if replace {
		q = "INSERT INTO automation_keys (workspace_id, secret, created_at) VALUES (?, ?, ?) ON CONFLICT (workspace_id) DO UPDATE SET secret = excluded.secret, created_at = excluded.created_at"
	}
	if _, err := s.Store.W.ExecContext(ctx, q, workspaceID, s.Box.Seal(secret), store.NowMS()); err != nil {
		return "", err
	}
	if !replace {
		var sealed string
		if err := s.Store.R.QueryRowContext(ctx, "SELECT secret FROM automation_keys WHERE workspace_id = ?", workspaceID).Scan(&sealed); err != nil {
			return "", err
		}
		return s.Box.Unseal(sealed)
	}
	return secret, nil
}

func (s *Service) Key(ctx context.Context, p *policy.Principal, workspaceID string, replace bool) (string, error) {
	acc, err := s.Gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return "", err
	}
	if !acc.IsAdmin() {
		return "", httpx.Fail(http.StatusForbidden, "forbidden", "only workspace admins can see the signing key")
	}
	if replace {
		return s.rotate(ctx, workspaceID, true)
	}
	return s.key(ctx, workspaceID)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Service) poke() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
