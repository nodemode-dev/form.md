package scim

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Token struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	CreatedBy  string `json:"created_by"`
	CreatedAt  int64  `json:"created_at"`
	LastUsedAt int64  `json:"last_used_at"`
}

func (s *Service) admin(ctx context.Context, p *policy.Principal, workspaceID string) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() || a.Personal {
		return httpx.Fail(http.StatusForbidden, "admin_only", "only admins of a shared workspace set up provisioning")
	}
	if !p.IsSession() {
		return httpx.ErrSessionOnly
	}
	return nil
}

func (s *Service) Tokens(ctx context.Context, p *policy.Principal, workspaceID string) ([]Token, error) {
	if err := s.admin(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, label, created_by, created_at, last_used_at FROM scim_tokens WHERE workspace_id = ? ORDER BY created_at", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.Label, &t.CreatedBy, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) CreateToken(ctx context.Context, p *policy.Principal, workspaceID, label string) (string, *Token, error) {
	if err := s.admin(ctx, p, workspaceID); err != nil {
		return "", nil, err
	}
	if label = clip(label, 80); label == "" {
		label = "Identity provider"
	}
	raw := secrets.Token("scim_", 32)
	t := &Token{ID: ids.New(), Label: label, CreatedBy: p.UserID, CreatedAt: store.NowMS()}
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO scim_tokens (id, workspace_id, token_hash, label, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			t.ID, workspaceID, secrets.Hash(raw), t.Label, t.CreatedBy, t.CreatedAt); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "scim.token_create", WorkspaceID: workspaceID, Detail: t.Label})
	})
	return raw, t, err
}

func (s *Service) RevokeToken(ctx context.Context, p *policy.Principal, id string) error {
	var ws, label string
	if err := s.st.R.QueryRowContext(ctx, "SELECT workspace_id, label FROM scim_tokens WHERE id = ?", id).Scan(&ws, &label); err != nil {
		return store.NoRows(err)
	}
	if err := s.admin(ctx, p, ws); err != nil {
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM scim_tokens WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "scim.token_revoke", WorkspaceID: ws, Detail: label})
	})
}

type module struct{ s *Service }

func (s *Service) Module() app.Module { return module{s} }

func (module) ID() string { return "scim" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, s.auth.Require(h))
	}
	route("GET /api/workspaces/{id}/scim", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Tokens(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"tokens": list, "base_url": httpx.RequestOrigin(r, s.auth.TrustProxy()) + "/scim/v2"})
		return nil
	})
	route("POST /api/workspaces/{id}/scim", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Label string `json:"label"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		raw, t, err := s.CreateToken(r.Context(), p, r.PathValue("id"), in.Label)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"token": raw, "info": t})
		return nil
	})
	route("DELETE /api/scim-tokens/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.RevokeToken(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	return nil
}
