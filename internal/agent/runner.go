package agent

import (
	"context"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/webhook"
)

func (s *Service) poke() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) principal(ctx context.Context, a *Agent) (*policy.Principal, error) {
	p, err := s.Auth.PrincipalFor(ctx, a.CreatedBy)
	if err != nil {
		return nil, err
	}
	p.Kind, p.Scope, p.AllWorkspaces, p.Workspaces = policy.KindOAuth, "write", false, map[string]bool{a.WorkspaceID: true}
	p.CredentialID, p.ClientName, p.AgentName = "agent:"+a.ID, a.Name, a.Name
	acc, err := s.Gate.Workspace(ctx, p, a.WorkspaceID)
	if err != nil || !acc.CanWrite() || !policy.AgentAllowed(p, acc.AgentAccess) {
		return nil, httpx.Fail(403, "agent_refused", "the agent's creator cannot write here, or this workspace does not allow agents")
	}
	return p, nil
}

func (s *Service) Start(ctx context.Context, p *policy.Principal, id, cause string) (*Chat, error) {
	a, err := s.agentFor(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if !a.CanEdit && a.CreatedBy != p.UserID {
		acc, _ := s.Gate.Workspace(ctx, p, a.WorkspaceID)
		if !acc.CanWrite() {
			return nil, httpx.ErrForbidden
		}
	}
	runAs, err := s.principal(ctx, a)
	if err != nil {
		return nil, err
	}
	if err := s.AI.Allowed(ctx, runAs, a.WorkspaceID); err != nil {
		return nil, err
	}
	if cause == "" {
		cause = "a manual run by " + p.Name
	}
	return s.launch(a, runAs, cause, "Run now")
}

func (s *Service) launch(a *Agent, p *policy.Principal, cause, prompt string) (*Chat, error) {
	ctx := context.Background()
	chat, err := s.NewChat(ctx, p, a.WorkspaceID, a.ID, "")
	if err != nil {
		return nil, err
	}
	now := store.NowMS()
	_, _ = s.Store.W.ExecContext(ctx, "UPDATE agents SET last_run_at = ?, last_status = 'running' WHERE id = ?", now, a.ID)
	go func() {
		s.sem <- struct{}{}
		defer func() { <-s.sem }()
		rctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		text := prompt
		if text == "" {
			text = "Do your job now."
		}
		s.mu.Lock()
		s.busy[a.ID]++
		s.mu.Unlock()
		msg, err := s.Run(rctx, p, &Turn{Chat: chat, Text: text, Agent: a, Cause: cause}, func(string, any) error { return nil })
		touched := []string{}
		if msg != nil && msg.RunID != "" {
			if rows, qerr := s.Store.R.QueryContext(context.Background(), "SELECT DISTINCT page_id FROM agent_ops WHERE run_id = ?", msg.RunID); qerr == nil {
				for rows.Next() {
					var id string
					if rows.Scan(&id) == nil {
						touched = append(touched, id)
					}
				}
				rows.Close()
			}
		}
		s.mu.Lock()
		s.busy[a.ID]--
		for _, id := range touched {
			s.quiet[a.ID+":"+id] = store.NowMS()
		}
		s.mu.Unlock()
		status := "done"
		if err != nil {
			status = "failed"
			s.Log.Warn("agent: run failed", "agent", a.ID, "err", err)
		}
		_, _ = s.Store.W.ExecContext(context.Background(), "UPDATE agents SET last_status = ? WHERE id = ?", status, a.ID)
	}()
	return chat, nil
}

func (s *Service) due(ctx context.Context) {
	now := store.NowMS()
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+agentCols+" FROM agents WHERE enabled = 1 AND trigger_kind = 'schedule' AND next_run_at > 0 AND next_run_at <= ?", now)
	if err != nil {
		return
	}
	var list []*Agent
	for rows.Next() {
		if a, err := scanAgent(rows); err == nil {
			list = append(list, a)
		}
	}
	rows.Close()
	for _, a := range list {
		next := NextRun(a.Trigger, time.Now())
		res, err := s.Store.W.ExecContext(ctx, "UPDATE agents SET next_run_at = ? WHERE id = ? AND next_run_at = ?", next, a.ID, a.NextRunAt)
		if err != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			continue
		}
		p, err := s.principal(ctx, a)
		if err != nil {
			_, _ = s.Store.W.ExecContext(ctx, "UPDATE agents SET last_status = 'failed' WHERE id = ?", a.ID)
			continue
		}
		if s.AI.Allowed(ctx, p, a.WorkspaceID) != nil {
			_, _ = s.Store.W.ExecContext(ctx, "UPDATE agents SET last_status = 'skipped' WHERE id = ?", a.ID)
			continue
		}
		_, _ = s.launch(a, p, "its schedule ("+time.UnixMilli(a.NextRunAt).UTC().Format("2006-01-02 15:04")+" UTC)", "Scheduled run. Do your job now.")
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
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (s *Service) OnEvent(ctx context.Context, e webhook.Event) {
	var kind, colID, pageID string
	switch e.Type {
	case "row.create":
		kind = "row_added"
	case "row.update":
		kind = "row_changed"
	case "page.content":
		kind = "page_changed"
	default:
		return
	}
	if e.PageID == "" || e.Actor.Kind == "agent" {
		return
	}
	if kind == "page_changed" {
		pageID = e.PageID
		if c, err := s.Cols.RowCollection(ctx, e.PageID); err == nil && c != "" {
			colID = c
		}
	} else {
		c, err := s.Cols.RowCollection(ctx, e.PageID)
		if err != nil || c == "" {
			return
		}
		colID = c
	}
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+agentCols+" FROM agents WHERE enabled = 1 AND workspace_id = ? AND trigger_kind IN ('row_added', 'row_changed', 'page_changed')", e.WorkspaceID)
	if err != nil {
		return
	}
	var list []*Agent
	for rows.Next() {
		if a, err := scanAgent(rows); err == nil {
			list = append(list, a)
		}
	}
	rows.Close()
	for _, a := range list {
		match := false
		switch a.TriggerKind {
		case "row_added":
			match = kind == "row_added" && a.Trigger.Collection == colID
		case "row_changed":
			match = (kind == "row_changed" || (kind == "page_changed" && colID != "")) && a.Trigger.Collection == colID
		case "page_changed":
			match = kind == "page_changed" && a.Trigger.Page == pageID
		}
		if !match || e.Actor.ID == "agent:"+a.ID {
			continue
		}
		s.debounce(a, e.PageID, kind)
	}
}

func (s *Service) debounce(a *Agent, pageID, kind string) {
	key := a.ID + ":" + pageID
	s.mu.Lock()
	if s.pending[key] || (kind != "row_added" && (s.busy[a.ID] > 0 || store.NowMS()-s.quiet[key] < 60000)) {
		s.mu.Unlock()
		return
	}
	s.pending[key] = true
	s.mu.Unlock()
	delay := 3 * time.Second
	if kind != "row_added" {
		delay = 15 * time.Second
	}
	time.AfterFunc(delay, func() {
		s.mu.Lock()
		delete(s.pending, key)
		s.mu.Unlock()
		ctx := context.Background()
		p, err := s.principal(ctx, a)
		if err != nil || s.AI.Allowed(ctx, p, a.WorkspaceID) != nil {
			return
		}
		title := ""
		_ = s.Store.R.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", pageID).Scan(&title)
		what := map[string]string{"row_added": "a new row was added to the database", "row_changed": "a row of the database changed", "page_changed": "the page changed"}[kind]
		prompt := "Triggered because " + what + ": \"" + clip(title, 200) + "\" (page_id " + pageID + "). Do your job for it now."
		_, _ = s.launch(a, p, what, prompt)
	})
}
