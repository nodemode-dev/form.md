package agentrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Undoer func(ctx context.Context, p *policy.Principal, pageID string, data json.RawMessage) error

type Run struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Client    string `json:"client"`
	AgentID   string `json:"agent_id,omitempty"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Ops       int    `json:"ops"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	UndoneAt  int64  `json:"undone_at,omitempty"`
}

type Op struct {
	Seq       int    `json:"seq"`
	Kind      string `json:"kind"`
	PageID    string `json:"page_id,omitempty"`
	Summary   string `json:"summary"`
	CreatedAt int64  `json:"created_at"`
}

type Service struct {
	st      *store.Store
	log     *slog.Logger
	mu      sync.RWMutex
	undoers map[string]Undoer
	auth    *auth.Service
}

func New(st *store.Store, a *auth.Service, log *slog.Logger) *Service {
	return &Service{st: st, auth: a, log: log, undoers: map[string]Undoer{}}
}

func (s *Service) Handle(kind string, fn Undoer) {
	s.mu.Lock()
	s.undoers[kind] = fn
	s.mu.Unlock()
}

var errRun = httpx.Fail(http.StatusNotFound, "run_not_found", "no open agent run with that id belongs to this credential")

func (s *Service) Open(ctx context.Context, p *policy.Principal, runID, kind, client, agentID, title string) (string, error) {
	now := store.NowMS()
	if runID != "" {
		var user, status string
		err := s.st.R.QueryRowContext(ctx, "SELECT user_id, status FROM agent_runs WHERE id = ?", runID).Scan(&user, &status)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (user != p.UserID || status != "open")) {
			return "", errRun
		}
		if err != nil {
			return "", err
		}
		return runID, nil
	}
	id := "run_" + ids.New()
	if len(title) > 200 {
		title = title[:200]
	}
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO agent_runs (id, user_id, credential, kind, client, agent_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, p.UserID, p.CredentialID, kind, client, agentID, title, now, now)
	return id, err
}

func (s *Service) Record(ctx context.Context, runID, kind, pageID, summary string, undo any) error {
	if runID == "" {
		return nil
	}
	b, err := json.Marshal(undo)
	if err != nil {
		return err
	}
	if len(summary) > 300 {
		summary = summary[:300]
	}
	now := store.NowMS()
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		var seq int
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(seq), 0) + 1 FROM agent_ops WHERE run_id = ?", runID).Scan(&seq); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO agent_ops (run_id, seq, kind, page_id, summary, undo, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", runID, seq, kind, pageID, summary, string(b), now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE agent_runs SET ops = ops + 1, updated_at = ? WHERE id = ?", now, runID)
		return err
	})
}

func (s *Service) Finish(ctx context.Context, runID, status string) {
	if runID == "" {
		return
	}
	_, _ = s.st.W.ExecContext(ctx, "UPDATE agent_runs SET status = ?, updated_at = ? WHERE id = ? AND status = 'open'", status, store.NowMS(), runID)
}

func (s *Service) owned(ctx context.Context, p *policy.Principal, runID string) (*Run, error) {
	var r Run
	var user string
	err := s.st.R.QueryRowContext(ctx, "SELECT id, user_id, kind, client, agent_id, title, status, ops, created_at, updated_at, undone_at FROM agent_runs WHERE id = ?", runID).
		Scan(&r.ID, &user, &r.Kind, &r.Client, &r.AgentID, &r.Title, &r.Status, &r.Ops, &r.CreatedAt, &r.UpdatedAt, &r.UndoneAt)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && user != p.UserID) {
		return nil, httpx.ErrNotFound
	}
	return &r, err
}

func (s *Service) Undo(ctx context.Context, p *policy.Principal, runID string) (int, error) {
	r, err := s.owned(ctx, p, runID)
	if err != nil {
		return 0, err
	}
	if r.Status == "undone" {
		return 0, httpx.Fail(http.StatusConflict, "already_undone", "this run was already undone")
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT seq, kind, page_id, undo FROM agent_ops WHERE run_id = ? ORDER BY seq DESC", runID)
	if err != nil {
		return 0, err
	}
	type op struct {
		seq            int
		kind, page, ud string
	}
	var ops []op
	for rows.Next() {
		var o op
		if err := rows.Scan(&o.seq, &o.kind, &o.page, &o.ud); err != nil {
			rows.Close()
			return 0, err
		}
		ops = append(ops, o)
	}
	rows.Close()
	done := 0
	var failed []string
	for _, o := range ops {
		s.mu.RLock()
		fn := s.undoers[o.kind]
		s.mu.RUnlock()
		if fn == nil {
			failed = append(failed, o.kind)
			continue
		}
		if err := fn(ctx, p, o.page, json.RawMessage(o.ud)); err != nil {
			s.log.Warn("agent run: undo step failed", "run", runID, "seq", o.seq, "kind", o.kind, "err", err)
			failed = append(failed, o.kind)
			continue
		}
		done++
	}
	now := store.NowMS()
	_, _ = s.st.W.ExecContext(ctx, "UPDATE agent_runs SET status = 'undone', undone_at = ?, updated_at = ? WHERE id = ?", now, now, runID)
	if len(failed) > 0 {
		return done, httpx.Fail(http.StatusConflict, "partly_undone", "undid "+strconv.Itoa(done)+" steps; could not undo: "+strings.Join(failed, ", "))
	}
	return done, nil
}

func (s *Service) List(ctx context.Context, p *policy.Principal, agentID string, limit int) ([]Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := "SELECT id, kind, client, agent_id, title, status, ops, created_at, updated_at, undone_at FROM agent_runs WHERE user_id = ?"
	args := []any{p.UserID}
	if agentID != "" {
		q += " AND agent_id = ?"
		args = append(args, agentID)
	}
	q += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.st.R.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.Kind, &r.Client, &r.AgentID, &r.Title, &r.Status, &r.Ops, &r.CreatedAt, &r.UpdatedAt, &r.UndoneAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) Ops(ctx context.Context, p *policy.Principal, runID string) (*Run, []Op, error) {
	r, err := s.owned(ctx, p, runID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT seq, kind, page_id, summary, created_at FROM agent_ops WHERE run_id = ? ORDER BY seq", runID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := []Op{}
	for rows.Next() {
		var o Op
		if err := rows.Scan(&o.Seq, &o.Kind, &o.PageID, &o.Summary, &o.CreatedAt); err != nil {
			return nil, nil, err
		}
		out = append(out, o)
	}
	return r, out, rows.Err()
}

func (s *Service) Purge(ctx context.Context, days int) {
	if days <= 0 {
		days = 90
	}
	cut := store.NowMS() - int64(days)*86400000
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM agent_runs WHERE updated_at < ?", cut)
	_, _ = s.st.W.ExecContext(ctx, "DELETE FROM mcp_idempotency WHERE created_at < ?", store.NowMS()-86400000)
}

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (module) ID() string { return "agentrun" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	r.HandleFunc("GET /api/agent-runs", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		list, err := s.List(r.Context(), p, r.URL.Query().Get("agent"), limit)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"runs": list})
		return nil
	}))
	r.HandleFunc("GET /api/agent-runs/{id}", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		run, ops, err := s.Ops(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"run": run, "ops": ops})
		return nil
	}))
	r.HandleFunc("POST /api/agent-runs/{id}/undo", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		n, err := s.Undo(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"undone": n})
		return nil
	}))
	return nil
}
