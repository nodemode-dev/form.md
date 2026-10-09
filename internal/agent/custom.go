package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Trigger struct {
	Every      string `json:"every,omitempty"`
	N          int    `json:"n,omitempty"`
	At         string `json:"at,omitempty"`
	Weekday    int    `json:"weekday,omitempty"`
	TZ         string `json:"tz,omitempty"`
	OnceAt     int64  `json:"once_at,omitempty"`
	Collection string `json:"collection,omitempty"`
	Page       string `json:"page,omitempty"`
}

type Agent struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	CreatedBy    string  `json:"created_by"`
	Name         string  `json:"name"`
	Icon         string  `json:"icon"`
	Instructions string  `json:"instructions"`
	Model        string  `json:"model"`
	TriggerKind  string  `json:"trigger_kind"`
	Trigger      Trigger `json:"trigger"`
	Enabled      bool    `json:"enabled"`
	NextRunAt    int64   `json:"next_run_at"`
	LastRunAt    int64   `json:"last_run_at"`
	LastStatus   string  `json:"last_status"`
	CanEdit      bool    `json:"can_edit"`
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

type AgentInput struct {
	Name         *string  `json:"name"`
	Icon         *string  `json:"icon"`
	Instructions *string  `json:"instructions"`
	Model        *string  `json:"model"`
	TriggerKind  *string  `json:"trigger_kind"`
	Trigger      *Trigger `json:"trigger"`
	Enabled      *bool    `json:"enabled"`
}

var triggerKinds = map[string]bool{"manual": true, "schedule": true, "row_added": true, "row_changed": true, "page_changed": true}

const agentCols = "id, workspace_id, created_by, name, icon, instructions, model, trigger_kind, trigger_config, enabled, next_run_at, last_run_at, last_status, created_at, updated_at"

func scanAgent(row interface{ Scan(...any) error }) (*Agent, error) {
	a := &Agent{}
	var cfg string
	var on int
	if err := row.Scan(&a.ID, &a.WorkspaceID, &a.CreatedBy, &a.Name, &a.Icon, &a.Instructions, &a.Model, &a.TriggerKind, &cfg, &on, &a.NextRunAt, &a.LastRunAt, &a.LastStatus, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, store.NoRows(err)
	}
	a.Enabled = on == 1
	_ = json.Unmarshal([]byte(cfg), &a.Trigger)
	return a, nil
}

func clock(at string) (int, int, bool) {
	h, m, ok := strings.Cut(at, ":")
	if !ok {
		return 0, 0, false
	}
	hh, e1 := strconv.Atoi(h)
	mm, e2 := strconv.Atoi(m)
	if e1 != nil || e2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, 0, false
	}
	return hh, mm, true
}

func NextRun(t Trigger, from time.Time) int64 {
	loc := time.UTC
	if t.TZ != "" {
		if l, err := time.LoadLocation(t.TZ); err == nil {
			loc = l
		}
	}
	now := from.In(loc)
	switch t.Every {
	case "once":
		if t.OnceAt > from.UnixMilli() {
			return t.OnceAt
		}
		return 0
	case "minutes":
		n := max(t.N, 1)
		return now.Add(time.Duration(n) * time.Minute).UnixMilli()
	case "hours":
		n := max(t.N, 1)
		return now.Add(time.Duration(n) * time.Hour).UnixMilli()
	case "day", "week":
		hh, mm, ok := clock(t.At)
		if !ok {
			hh, mm = 9, 0
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, loc)
		for !next.After(now) || (t.Every == "week" && int(next.Weekday()) != t.Weekday%7) {
			next = next.AddDate(0, 0, 1)
		}
		return next.UnixMilli()
	}
	return 0
}

func (s *Service) checkInput(ctx context.Context, p *policy.Principal, ws string, a *Agent) error {
	a.Name = strings.TrimSpace(clip(a.Name, 80))
	if a.Name == "" {
		return httpx.Fail(http.StatusBadRequest, "bad_name", "an agent needs a name")
	}
	a.Instructions = clip(a.Instructions, 20000)
	if !triggerKinds[a.TriggerKind] {
		return httpx.Fail(http.StatusBadRequest, "bad_trigger", "the trigger must be manual, schedule, row_added, row_changed or page_changed")
	}
	switch a.TriggerKind {
	case "schedule":
		switch a.Trigger.Every {
		case "once", "minutes", "hours", "day", "week":
		default:
			return httpx.Fail(http.StatusBadRequest, "bad_schedule", "a schedule runs once, every few minutes or hours, daily or weekly")
		}
		if a.Trigger.Every == "minutes" && a.Trigger.N < 1 {
			a.Trigger.N = 15
		}
		if a.Trigger.TZ != "" {
			if _, err := time.LoadLocation(a.Trigger.TZ); err != nil {
				return httpx.Fail(http.StatusBadRequest, "bad_timezone", "unknown time zone")
			}
		}
	case "row_added", "row_changed":
		col, err := s.Cols.Get(ctx, p, a.Trigger.Collection)
		if err != nil || col.WorkspaceID != ws {
			return httpx.Fail(http.StatusBadRequest, "bad_collection", "choose a database in this workspace")
		}
	case "page_changed":
		if _, err := s.Docs.Markdown(ctx, p, a.Trigger.Page); err != nil {
			return httpx.Fail(http.StatusBadRequest, "bad_page", "choose a page you can read")
		}
	}
	if a.TriggerKind != "schedule" {
		a.Trigger.Every, a.Trigger.N, a.Trigger.At, a.Trigger.Weekday, a.Trigger.OnceAt = "", 0, "", 0, 0
	}
	if a.TriggerKind != "row_added" && a.TriggerKind != "row_changed" {
		a.Trigger.Collection = ""
	}
	if a.TriggerKind != "page_changed" {
		a.Trigger.Page = ""
	}
	a.NextRunAt = 0
	if a.TriggerKind == "schedule" && a.Enabled {
		a.NextRunAt = NextRun(a.Trigger, time.Now())
	}
	return nil
}

func (s *Service) CreateAgent(ctx context.Context, p *policy.Principal, ws string, in AgentInput) (*Agent, error) {
	acc, err := s.member(ctx, p, ws)
	if err != nil {
		return nil, err
	}
	if !acc.CanWrite() {
		return nil, httpx.ErrForbidden
	}
	a := &Agent{ID: "agt_" + ids.New(), WorkspaceID: ws, CreatedBy: p.UserID, TriggerKind: "manual", Enabled: true}
	apply(a, in)
	if err := s.checkInput(ctx, p, ws, a); err != nil {
		return nil, err
	}
	cfg, _ := json.Marshal(a.Trigger)
	now := store.NowMS()
	a.CreatedAt, a.UpdatedAt, a.CanEdit = now, now, true
	_, err = s.Store.W.ExecContext(ctx, "INSERT INTO agents ("+agentCols+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, ws, p.UserID, a.Name, a.Icon, a.Instructions, a.Model, a.TriggerKind, string(cfg), store.Bool(a.Enabled), a.NextRunAt, 0, "", now, now)
	if err != nil {
		return nil, err
	}
	s.poke()
	return a, nil
}

func apply(a *Agent, in AgentInput) {
	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Icon != nil {
		a.Icon = clip(*in.Icon, 200)
	}
	if in.Instructions != nil {
		a.Instructions = *in.Instructions
	}
	if in.Model != nil {
		a.Model = clip(*in.Model, 200)
	}
	if in.TriggerKind != nil {
		a.TriggerKind = *in.TriggerKind
	}
	if in.Trigger != nil {
		a.Trigger = *in.Trigger
	}
	if in.Enabled != nil {
		a.Enabled = *in.Enabled
	}
}

func (s *Service) loadAgent(ctx context.Context, id string) (*Agent, error) {
	return scanAgent(s.Store.R.QueryRowContext(ctx, "SELECT "+agentCols+" FROM agents WHERE id = ?", id))
}

func (s *Service) agentFor(ctx context.Context, p *policy.Principal, id string) (*Agent, error) {
	a, err := s.loadAgent(ctx, id)
	if err != nil {
		return nil, httpx.ErrNotFound
	}
	acc, err := s.member(ctx, p, a.WorkspaceID)
	if err != nil {
		return nil, err
	}
	a.CanEdit = a.CreatedBy == p.UserID || acc.IsAdmin()
	return a, nil
}

func (s *Service) UpdateAgent(ctx context.Context, p *policy.Principal, id string, in AgentInput) (*Agent, error) {
	a, err := s.agentFor(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !a.CanEdit {
		return nil, httpx.ErrForbidden
	}
	apply(a, in)
	if err := s.checkInput(ctx, p, a.WorkspaceID, a); err != nil {
		return nil, err
	}
	cfg, _ := json.Marshal(a.Trigger)
	a.UpdatedAt = store.NowMS()
	_, err = s.Store.W.ExecContext(ctx, "UPDATE agents SET name = ?, icon = ?, instructions = ?, model = ?, trigger_kind = ?, trigger_config = ?, enabled = ?, next_run_at = ?, updated_at = ? WHERE id = ?",
		a.Name, a.Icon, a.Instructions, a.Model, a.TriggerKind, string(cfg), store.Bool(a.Enabled), a.NextRunAt, a.UpdatedAt, a.ID)
	s.poke()
	return a, err
}

func (s *Service) DeleteAgent(ctx context.Context, p *policy.Principal, id string) error {
	a, err := s.agentFor(ctx, p, id)
	if err != nil {
		return err
	}
	if !a.CanEdit {
		return httpx.ErrForbidden
	}
	_, err = s.Store.W.ExecContext(ctx, "DELETE FROM agents WHERE id = ?", id)
	return err
}

func (s *Service) Agents(ctx context.Context, p *policy.Principal, ws string) ([]*Agent, error) {
	acc, err := s.member(ctx, p, ws)
	if err != nil {
		return nil, err
	}
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+agentCols+" FROM agents WHERE workspace_id = ? ORDER BY name COLLATE NOCASE", ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Agent{}
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			continue
		}
		a.CanEdit = a.CreatedBy == p.UserID || acc.IsAdmin()
		out = append(out, a)
	}
	return out, nil
}

type RunOut struct {
	ChatID    string `json:"chat_id"`
	Title     string `json:"title"`
	RunID     string `json:"run_id,omitempty"`
	Status    string `json:"status"`
	Ops       int    `json:"ops"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Service) AgentRuns(ctx context.Context, p *policy.Principal, id string) ([]RunOut, error) {
	if _, err := s.agentFor(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.Store.R.QueryContext(ctx, `SELECT c.id, c.title, c.created_at,
		COALESCE((SELECT m.run_id FROM agent_messages m WHERE m.chat_id = c.id AND m.role = 'assistant' ORDER BY m.seq DESC LIMIT 1), '')
		FROM agent_chats c WHERE c.agent_id = ? ORDER BY c.created_at DESC LIMIT 30`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RunOut{}
	for rows.Next() {
		var r RunOut
		if rows.Scan(&r.ChatID, &r.Title, &r.CreatedAt, &r.RunID) == nil {
			out = append(out, r)
		}
	}
	rows.Close()
	for i := range out {
		out[i].Status = "done"
		if out[i].RunID != "" {
			var st string
			var ops int
			if err := s.Store.R.QueryRowContext(ctx, "SELECT status, ops FROM agent_runs WHERE id = ?", out[i].RunID).Scan(&st, &ops); err == nil {
				out[i].Status, out[i].Ops = st, ops
			} else if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
	}
	return out, nil
}
