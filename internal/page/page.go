package page

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/events"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/icon"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	MaxTitle       = 500
	MaxDescription = 2000
	MaxTags        = 20
	MaxTag         = 40
	nobody         = "-"
)

var Covers = []string{"dawn", "lagoon", "meadow", "ember", "dusk", "glacier", "orchard", "slate", "aurora", "sand", "tide", "plum"}

type Meta struct {
	ID            string   `json:"id"`
	WorkspaceID   string   `json:"workspace_id"`
	ParentID      string   `json:"parent_id"`
	Kind          string   `json:"kind"`
	SortKey       string   `json:"sort_key"`
	Title         string   `json:"title"`
	Icon          string   `json:"icon"`
	Cover         string   `json:"cover"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Visibility    string   `json:"visibility"`
	OwnerID       string   `json:"owner_id"`
	IsTemplate    bool     `json:"is_template"`
	Snippet       string   `json:"snippet"`
	Thumb         string   `json:"thumb"`
	CreatedAt     int64    `json:"created_at"`
	CreatedBy     string   `json:"created_by"`
	UpdatedAt     int64    `json:"updated_at"`
	UpdatedBy     string   `json:"updated_by"`
	TrashedAt     int64    `json:"trashed_at,omitempty"`
	TrashedBy     string   `json:"trashed_by,omitempty"`
	TrashedByKind string   `json:"trashed_by_kind,omitempty"`
	trashBatch    string
}

const metaColumns = `p.id, p.workspace_id, COALESCE(p.parent_id, ''), p.kind, p.sort_key, p.title, p.icon, p.cover, p.description, p.tags,
	p.visibility, COALESCE(p.owner_id, ''), p.is_template, p.snippet, p.thumb, p.created_at, p.created_by, p.updated_at, p.updated_by,
	p.trashed_at, p.trashed_by, p.trashed_by_kind, p.trash_batch`

func scanMeta(row interface{ Scan(...any) error }) (*Meta, error) {
	var m Meta
	var tags string
	var tpl int
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.ParentID, &m.Kind, &m.SortKey, &m.Title, &m.Icon, &m.Cover, &m.Description, &tags,
		&m.Visibility, &m.OwnerID, &tpl, &m.Snippet, &m.Thumb, &m.CreatedAt, &m.CreatedBy, &m.UpdatedAt, &m.UpdatedBy,
		&m.TrashedAt, &m.TrashedBy, &m.TrashedByKind, &m.trashBatch)
	if err != nil {
		return nil, err
	}
	m.IsTemplate = tpl == 1
	m.Tags = []string{}
	if tags != "" {
		if err := json.Unmarshal([]byte(tags), &m.Tags); err != nil {
			m.Tags = []string{}
		}
	}
	return &m, nil
}

type Service struct {
	st       *store.Store
	gate     *policy.Gate
	audit    *audit.Log
	hub      *events.Hub
	settings *settings.Service
	log      *slog.Logger
	copiers  []func(ctx context.Context, tx *sql.Tx, from, to string) error
}

func New(st *store.Store, gate *policy.Gate, al *audit.Log, hub *events.Hub, set *settings.Service, log *slog.Logger) *Service {
	return &Service{st: st, gate: gate, audit: al, hub: hub, settings: set, log: log}
}

func (s *Service) OnDuplicate(fn func(ctx context.Context, tx *sql.Tx, from, to string) error) {
	s.copiers = append(s.copiers, fn)
}

var errTrashed = httpx.Fail(http.StatusConflict, "page_trashed", "the page is in the trash")

func (s *Service) load(ctx context.Context, q store.Querier, id string) (*Meta, error) {
	m, err := scanMeta(q.QueryRowContext(ctx, "SELECT "+metaColumns+" FROM pages p WHERE p.id = ?", id))
	if err != nil {
		return nil, store.NoRows(err)
	}
	return m, nil
}

func (s *Service) privateOwner(ctx context.Context, q store.Querier, id string) (string, error) {
	rows, err := q.QueryContext(ctx, `WITH RECURSIVE chain(id, parent_id, visibility, owner_id) AS (
			SELECT id, parent_id, visibility, owner_id FROM pages WHERE id = ?
			UNION ALL
			SELECT p.id, p.parent_id, p.visibility, p.owner_id FROM pages p JOIN chain c ON p.id = c.parent_id
		) SELECT COALESCE(owner_id, '') FROM chain WHERE visibility = 'private'`, id)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	owner := ""
	for rows.Next() {
		var o string
		if err := rows.Scan(&o); err != nil {
			return "", err
		}
		if o == "" || (owner != "" && owner != o) {
			owner = nobody
		} else if owner != nobody {
			owner = o
		}
	}
	return owner, rows.Err()
}

type access struct {
	ws    policy.Access
	owner string
}

func (s *Service) check(ctx context.Context, q store.Querier, p *policy.Principal, m *Meta, write bool) (access, error) {
	a, err := s.gate.Workspace(ctx, p, m.WorkspaceID)
	if err != nil {
		return access{}, err
	}
	owner, err := s.privateOwner(ctx, q, m.ID)
	if err != nil {
		return access{}, err
	}
	if owner != "" && owner != p.UserID {
		return access{}, policy.ErrHidden
	}
	if write && !a.CanWrite() {
		return access{}, httpx.ErrForbidden
	}
	return access{ws: a, owner: owner}, nil
}

func (s *Service) readable(ctx context.Context, p *policy.Principal, id string) (*Meta, access, error) {
	m, err := s.load(ctx, s.st.R, id)
	if err != nil {
		return nil, access{}, err
	}
	a, err := s.check(ctx, s.st.R, p, m, false)
	return m, a, err
}

func cleanTitle(raw string) (string, error) {
	t := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(raw, "\r", " "), "\n", " "))
	if utf8.RuneCountInString(t) > MaxTitle {
		return "", httpx.Fail(http.StatusBadRequest, "title_too_long", "a title can be up to 500 characters")
	}
	return t, nil
}

func cleanTags(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range in {
		t := strings.Join(strings.Fields(strings.TrimPrefix(strings.TrimSpace(raw), "#")), "-")
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > MaxTag {
			t = string([]rune(t)[:MaxTag])
		}
		k := strings.ToLower(t)
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, t)
		if len(out) == MaxTags {
			break
		}
	}
	return out
}

func validCover(v string) bool {
	if v == "" {
		return true
	}
	if name, ok := strings.CutPrefix(v, "g:"); ok {
		for _, c := range Covers {
			if c == name {
				return true
			}
		}
		return false
	}
	return icon.Valid(v) && strings.HasPrefix(v, "f:")
}

func (s *Service) fileInWorkspace(ctx context.Context, q store.Querier, ref, workspaceID string) error {
	id, ok := strings.CutPrefix(ref, "f:")
	if !ok {
		return nil
	}
	var n int
	if err := q.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE id = ? AND workspace_id = ?", id, workspaceID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return httpx.Fail(http.StatusBadRequest, "bad_file", "that file is not in this workspace")
	}
	return nil
}

func hidden(err error) bool {
	return errors.Is(err, policy.ErrHidden) || errors.Is(err, store.ErrNotFound)
}

func (s *Service) emit(ctx context.Context, m *Meta) {
	owner, err := s.privateOwner(ctx, s.st.R, m.ID)
	if err != nil {
		s.log.Warn("page: emit", "err", err)
		return
	}
	s.hub.Page(m.WorkspaceID, owner, m)
}
