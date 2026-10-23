package automation

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/webhook"
)

const agentPrefix = "Automation: "

func (s *Service) runAs(ctx context.Context, a *Automation) (*policy.Principal, error) {
	p, err := s.Auth.PrincipalFor(ctx, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	p.Kind, p.Scope, p.AllWorkspaces = policy.KindSession, "write", false
	p.Workspaces = map[string]bool{a.WorkspaceID: true}
	p.AgentName = agentPrefix + a.Name
	p.CredentialID = "automation:" + a.ID
	return p, nil
}

func (s *Service) OnEvent(ctx context.Context, e webhook.Event) {
	if e.PageID == "" || strings.HasPrefix(e.Actor.Agent, agentPrefix) {
		return
	}
	var kind string
	switch e.Type {
	case "row.create":
		kind = "row_added"
	case "row.update":
		kind = "prop_edited"
	default:
		return
	}
	colID, err := s.Cols.RowCollection(ctx, e.PageID)
	if err != nil || colID == "" {
		return
	}
	var tpl int
	if s.Store.R.QueryRowContext(ctx, "SELECT is_template FROM pages WHERE id = ?", e.PageID).Scan(&tpl) != nil || tpl == 1 {
		return
	}
	if kind == "row_added" && s.Cols.ResponseOf(ctx, e.PageID) != nil {
		return
	}
	list := s.enabledFor(ctx, colID, kind)
	if kind == "row_added" {
		for _, a := range s.enabledFor(ctx, colID, "response_received") {
			if a.Trigger.Manual {
				list = append(list, a)
			}
		}
	}
	if len(list) == 0 {
		return
	}
	var changed []struct {
		Field string `json:"field"`
	}
	_ = json.Unmarshal(e.Changes, &changed)
	for _, a := range list {
		if kind == "prop_edited" && len(a.Trigger.Props) > 0 {
			hit := false
			for _, c := range changed {
				for _, want := range a.Trigger.Props {
					if c.Field == want {
						hit = true
					}
				}
			}
			if !hit {
				continue
			}
		}
		s.debounce(a.ID, e.PageID, e.Actor.ID, e.Actor.Label, kind, nil)
	}
}

func (s *Service) enabledFor(ctx context.Context, colID, kind string) []*Automation {
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+cols+" FROM automations WHERE collection_id = ? AND enabled = 1 AND trigger_kind = ?", colID, kind)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var list []*Automation
	for rows.Next() {
		if a, err := scan(rows); err == nil {
			list = append(list, a)
		}
	}
	return list
}

// OnResponse is the "response received" event. It carries the validated
// response as submitted; the automation's condition judges those answers.
// A "row added" automation on a database that takes form responses is
// carried over, so it keeps running for a response too.
func (s *Service) OnResponse(ctx context.Context, e collection.ResponseEvent) {
	var tpl int
	if s.Store.R.QueryRowContext(ctx, "SELECT is_template FROM pages WHERE id = ?", e.RowID).Scan(&tpl) != nil || tpl == 1 {
		return
	}
	list := append(s.enabledFor(ctx, e.CollectionID, "response_received"), s.enabledFor(ctx, e.CollectionID, "row_added")...)
	for _, a := range list {
		s.debounce(a.ID, e.RowID, e.ActorID, e.ActorLabel, "response_received", e.Response)
	}
}

func (s *Service) debounce(automationID, rowID, actorID, actorName, kind string, rec *collection.Response) {
	key := automationID + ":" + rowID
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.pending[key]; ok {
		t.Stop()
	}
	delay := debounceDelay
	if kind == "row_added" || kind == "response_received" {
		delay = 300 * time.Millisecond
	}
	s.pending[key] = time.AfterFunc(delay, func() {
		s.mu.Lock()
		delete(s.pending, key)
		s.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		a, err := s.get(ctx, automationID)
		if err != nil || !a.Enabled {
			return
		}
		s.fire(ctx, a, rowID, actorID, actorName, kind, rec)
	})
}

func (s *Service) fire(ctx context.Context, a *Automation, rowID, actorID, actorName, cause string, rec *collection.Response) Outcome {
	p, err := s.runAs(ctx, a)
	if err != nil {
		s.record(ctx, a, rowID, "failed", "the automation's owner can no longer act here")
		return Outcome{Failed: []string{"owner unavailable"}}
	}
	if rowID != "" && a.Trigger.Filter != nil {
		if rec == nil && (a.TriggerKind == "response_received" || a.TriggerKind == "row_added") {
			rec = s.Cols.ResponseOf(ctx, rowID)
		}
		var ok bool
		var err error
		if rec != nil {
			ok, err = s.Cols.ResponseMatches(ctx, p, collection.ResponseEvent{CollectionID: a.CollectionID, RowID: rowID, Response: rec}, a.Trigger.Filter)
		} else {
			ok, err = s.Cols.RowMatches(ctx, p, rowID, a.Trigger.Filter)
		}
		if err != nil || !ok {
			return Outcome{}
		}
	}
	env := Env{WorkspaceID: a.WorkspaceID, CollectionID: a.CollectionID, AutomationID: a.ID, Name: a.Name, RowID: rowID, ActorID: actorID, ActorName: actorName, CreatorID: a.CreatedBy, RunAs: p, Base: a.baseURL, Trigger: cause}
	if env.ActorID == "" {
		env.ActorID, env.ActorName = a.CreatedBy, p.Name
	}
	o := s.execute(ctx, env, a.Actions)
	s.record(ctx, a, rowID, o.status(), o.detail())
	return o
}

func (s *Service) record(ctx context.Context, a *Automation, rowID, status, detail string) {
	now := store.NowMS()
	_, _ = s.Store.W.ExecContext(ctx, "INSERT INTO automation_runs (id, automation_id, at, row_id, status, detail) VALUES (?, ?, ?, ?, ?, ?)", ids.New(), a.ID, now, rowID, status, detail)
	_, _ = s.Store.W.ExecContext(ctx, "UPDATE automations SET last_run_at = ?, last_status = ?, runs = runs + 1 WHERE id = ?", now, status, a.ID)
	_, _ = s.Store.W.ExecContext(ctx, "DELETE FROM automation_runs WHERE automation_id = ? AND id NOT IN (SELECT id FROM automation_runs WHERE automation_id = ? ORDER BY at DESC LIMIT ?)", a.ID, a.ID, keepRuns)
}

func (s *Service) RunNow(ctx context.Context, p *policy.Principal, id, rowID string) (*Outcome, error) {
	a, err := s.editable(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if rowID != "" {
		col, err := s.Cols.RowCollection(ctx, rowID)
		if err != nil || col != a.CollectionID {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_row", "that page is not in this database")
		}
	}
	o := s.fire(ctx, a, rowID, p.UserID, p.Name, "manual", nil)
	return &o, nil
}

func (s *Service) due(ctx context.Context) {
	now := store.NowMS()
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+cols+" FROM automations WHERE enabled = 1 AND trigger_kind = 'schedule' AND next_run_at > 0 AND next_run_at <= ? LIMIT 50", now)
	if err != nil {
		return
	}
	var list []*Automation
	for rows.Next() {
		if a, err := scan(rows); err == nil {
			list = append(list, a)
		}
	}
	rows.Close()
	for _, a := range list {
		next := s.nextRun(a)
		res, err := s.Store.W.ExecContext(ctx, "UPDATE automations SET next_run_at = ? WHERE id = ? AND next_run_at = ?", next, a.ID, a.NextRunAt)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n != 1 {
			continue
		}
		go func(a *Automation) {
			c, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
			defer cancel()
			s.fire(c, a, "", "", "", "schedule", nil)
		}(a)
	}
}

func (s *Service) Loop(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		s.due(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.wake:
		}
	}
}

type ButtonInput struct {
	PageID  string   `json:"page_id"`
	RowID   string   `json:"row_id"`
	Name    string   `json:"name"`
	Actions []Action `json:"actions"`
}

func (s *Service) RunButton(ctx context.Context, p *policy.Principal, base string, in ButtonInput) (*Outcome, error) {
	if in.PageID == "" {
		return nil, bad("bad_page", "page_id is required")
	}
	m, _, err := s.Pages.ReadableMeta(ctx, p, in.PageID)
	if err != nil {
		return nil, err
	}
	if in.RowID == "" {
		if col, err := s.Cols.RowCollection(ctx, in.PageID); err == nil && col != "" {
			in.RowID = in.PageID
		}
	}
	var server []Action
	for _, a := range in.Actions {
		if serverKinds[a.Kind] {
			server = append(server, a)
		}
	}
	if len(server) == 0 {
		return &Outcome{}, nil
	}
	if err := s.checkActions(ctx, p, m.WorkspaceID, server, serverKinds); err != nil {
		return nil, err
	}
	name := clip(in.Name, 80)
	if name == "" {
		name = "Button"
	}
	run := *p
	run.AgentName = ""
	env := Env{WorkspaceID: m.WorkspaceID, Name: name, PageID: in.PageID, RowID: in.RowID, ActorID: p.UserID, ActorName: p.Name, CreatorID: p.UserID, RunAs: &run, Base: base, Trigger: "button"}
	if in.RowID != "" {
		col, err := s.Cols.RowCollection(ctx, in.RowID)
		if err != nil || col == "" {
			return nil, bad("bad_row", "that page is not a database row")
		}
		env.CollectionID = col
	}
	o := s.execute(ctx, env, server)
	return &o, nil
}

func (s *Service) RunProperty(ctx context.Context, p *policy.Principal, base, rowID, propID string) (*Outcome, error) {
	col, err := s.Cols.RowCollection(ctx, rowID)
	if err != nil || col == "" {
		return nil, httpx.ErrNotFound
	}
	schema, err := s.Cols.SchemaOf(ctx, col)
	if err != nil {
		return nil, err
	}
	prop := schema.ByID(propID)
	if prop == nil || prop.Type != collection.TButton {
		return nil, httpx.ErrNotFound
	}
	raw, _ := json.Marshal(prop.Actions)
	var list []struct {
		Kind   string          `json:"kind"`
		Config json.RawMessage `json:"config"`
	}
	_ = json.Unmarshal(raw, &list)
	var acts []Action
	for _, x := range list {
		var a Action
		if len(x.Config) > 0 {
			_ = json.Unmarshal(x.Config, &a)
		}
		a.Kind = x.Kind
		acts = append(acts, a)
	}
	return s.RunButton(ctx, p, base, ButtonInput{PageID: rowID, RowID: rowID, Name: prop.Name, Actions: acts})
}
