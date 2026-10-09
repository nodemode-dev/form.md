package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/agentrun"
	"github.com/nodemode-ai/form.md/internal/ai"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/mcp"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Deps struct {
	Store *store.Store
	Auth  *auth.Service
	Gate  *policy.Gate
	AI    *ai.Service
	Tools *mcp.Server
	Docs  *doc.Service
	Cols  *collection.Service
	Runs  *agentrun.Service
	Log   *slog.Logger
}

type Service struct {
	Deps
	sem     chan struct{}
	mu      sync.Mutex
	pending map[string]bool
	busy    map[string]int
	quiet   map[string]int64
	wake    chan struct{}
}

func New(d Deps) *Service {
	return &Service{Deps: d, sem: make(chan struct{}, 2), pending: map[string]bool{}, busy: map[string]int{}, quiet: map[string]int64{}, wake: make(chan struct{}, 1)}
}

type Chat struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	AgentID     string `json:"agent_id,omitempty"`
	Title       string `json:"title"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	userID      string
}

type Segment struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Label   string `json:"label,omitempty"`
	OK      bool   `json:"ok,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	Parts     []Segment `json:"parts"`
	RunID     string    `json:"run_id,omitempty"`
	RunStatus string    `json:"run_status,omitempty"`
	RunOps    int       `json:"run_ops,omitempty"`
	CreatedAt int64     `json:"created_at"`
}

type Memory struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at"`
}

type Prefs struct {
	InstructionsPage string `json:"instructions_page"`
	Memory           bool   `json:"memory"`
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

var errChat = httpx.ErrNotFound

func (s *Service) member(ctx context.Context, p *policy.Principal, ws string) (policy.Access, error) {
	acc, err := s.Gate.Workspace(ctx, p, ws)
	if err != nil {
		return acc, httpx.ErrNotFound
	}
	if !acc.CanRead() {
		return acc, httpx.ErrNotFound
	}
	return acc, nil
}

func (s *Service) prefs(ctx context.Context, p *policy.Principal, ws string) Prefs {
	out := Prefs{Memory: true}
	var mem int
	if s.Store.R.QueryRowContext(ctx, "SELECT instructions_page, memory FROM agent_prefs WHERE workspace_id = ? AND user_id = ?", ws, p.UserID).Scan(&out.InstructionsPage, &mem) == nil {
		out.Memory = mem == 1
	}
	return out
}

func (s *Service) SetPrefs(ctx context.Context, p *policy.Principal, ws string, in Prefs) error {
	if _, err := s.member(ctx, p, ws); err != nil {
		return err
	}
	if in.InstructionsPage != "" {
		if _, err := s.Docs.Markdown(ctx, p, in.InstructionsPage); err != nil {
			return httpx.Fail(http.StatusBadRequest, "bad_page", "that instructions page cannot be read")
		}
	}
	_, err := s.Store.W.ExecContext(ctx, `INSERT INTO agent_prefs (workspace_id, user_id, instructions_page, memory) VALUES (?, ?, ?, ?)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET instructions_page = excluded.instructions_page, memory = excluded.memory`, ws, p.UserID, in.InstructionsPage, store.Bool(in.Memory))
	return err
}

func (s *Service) memories(ctx context.Context, userID, ws string) []Memory {
	rows, err := s.Store.R.QueryContext(ctx, "SELECT id, text, created_at FROM agent_memory WHERE user_id = ? AND workspace_id = ? ORDER BY created_at LIMIT 200", userID, ws)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Memory
	for rows.Next() {
		var m Memory
		if rows.Scan(&m.ID, &m.Text, &m.CreatedAt) == nil {
			out = append(out, m)
		}
	}
	return out
}

func (s *Service) remember(ctx context.Context, userID, ws, text string) (string, error) {
	text = strings.TrimSpace(clip(text, 500))
	if text == "" {
		return "", errors.New("nothing to remember")
	}
	var n int
	_ = s.Store.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_memory WHERE user_id = ? AND workspace_id = ?", userID, ws).Scan(&n)
	if n >= 200 {
		return "", errors.New("memory is full (200 facts); forget something first")
	}
	id := "mem_" + ids.New()
	_, err := s.Store.W.ExecContext(ctx, "INSERT INTO agent_memory (id, workspace_id, user_id, text, created_at) VALUES (?, ?, ?, ?, ?)", id, ws, userID, text, store.NowMS())
	return id, err
}

func (s *Service) Forget(ctx context.Context, p *policy.Principal, id string) error {
	res, err := s.Store.W.ExecContext(ctx, "DELETE FROM agent_memory WHERE id = ? AND user_id = ?", id, p.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

func (s *Service) NewChat(ctx context.Context, p *policy.Principal, ws, agentID, title string) (*Chat, error) {
	if _, err := s.member(ctx, p, ws); err != nil {
		return nil, err
	}
	now := store.NowMS()
	c := &Chat{ID: "chat_" + ids.New(), WorkspaceID: ws, AgentID: agentID, Title: clip(title, 120), CreatedAt: now, UpdatedAt: now, userID: p.UserID}
	_, err := s.Store.W.ExecContext(ctx, "INSERT INTO agent_chats (id, workspace_id, user_id, agent_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)", c.ID, ws, p.UserID, agentID, c.Title, now, now)
	return c, err
}

func (s *Service) chat(ctx context.Context, p *policy.Principal, id string) (*Chat, error) {
	c := &Chat{}
	err := s.Store.R.QueryRowContext(ctx, "SELECT id, workspace_id, user_id, agent_id, title, created_at, updated_at FROM agent_chats WHERE id = ?", id).
		Scan(&c.ID, &c.WorkspaceID, &c.userID, &c.AgentID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errChat
	}
	if err != nil {
		return nil, err
	}
	if c.userID != p.UserID {
		if c.AgentID == "" {
			return nil, errChat
		}
		if _, err := s.agentFor(ctx, p, c.AgentID); err != nil {
			return nil, errChat
		}
	}
	return c, nil
}

func (s *Service) Chats(ctx context.Context, p *policy.Principal, ws string, limit int) ([]Chat, error) {
	if _, err := s.member(ctx, p, ws); err != nil {
		return nil, err
	}
	rows, err := s.Store.R.QueryContext(ctx, "SELECT id, workspace_id, agent_id, title, created_at, updated_at FROM agent_chats WHERE user_id = ? AND workspace_id = ? AND agent_id = '' ORDER BY updated_at DESC LIMIT ?", p.UserID, ws, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Chat{}
	for rows.Next() {
		var c Chat
		if rows.Scan(&c.ID, &c.WorkspaceID, &c.AgentID, &c.Title, &c.CreatedAt, &c.UpdatedAt) == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) DeleteChat(ctx context.Context, p *policy.Principal, id string) error {
	res, err := s.Store.W.ExecContext(ctx, "DELETE FROM agent_chats WHERE id = ? AND user_id = ?", id, p.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return httpx.ErrNotFound
	}
	return nil
}

func (s *Service) Messages(ctx context.Context, chatID string) ([]Message, error) {
	rows, err := s.Store.R.QueryContext(ctx, `SELECT m.id, m.role, m.text, m.parts, m.run_id, m.created_at, COALESCE(r.status, ''), COALESCE(r.ops, 0)
		FROM agent_messages m LEFT JOIN agent_runs r ON r.id = m.run_id WHERE m.chat_id = ? ORDER BY m.seq`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Message{}
	for rows.Next() {
		var m Message
		var parts string
		if rows.Scan(&m.ID, &m.Role, &m.Text, &parts, &m.RunID, &m.CreatedAt, &m.RunStatus, &m.RunOps) == nil {
			_ = json.Unmarshal([]byte(parts), &m.Parts)
			if m.Parts == nil {
				m.Parts = []Segment{}
			}
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Service) addMessage(ctx context.Context, chatID string, m *Message) error {
	parts, _ := json.Marshal(m.Parts)
	m.ID = "msg_" + ids.New()
	m.CreatedAt = store.NowMS()
	return s.Store.Tx(ctx, func(tx *sql.Tx) error {
		var seq int
		_ = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(seq), 0) FROM agent_messages WHERE chat_id = ?", chatID).Scan(&seq)
		if _, err := tx.ExecContext(ctx, "INSERT INTO agent_messages (id, chat_id, seq, role, text, parts, run_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			m.ID, chatID, seq+1, m.Role, m.Text, string(parts), m.RunID, m.CreatedAt); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE agent_chats SET updated_at = ? WHERE id = ?", m.CreatedAt, chatID)
		return err
	})
}

func (s *Service) retitle(ctx context.Context, chatID, title string) {
	_, _ = s.Store.W.ExecContext(ctx, "UPDATE agent_chats SET title = ? WHERE id = ? AND title = ''", clip(strings.TrimSpace(strings.ReplaceAll(title, "\n", " ")), 80), chatID)
}
