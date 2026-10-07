package page

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Favorite struct {
	*Meta
	FavoriteKey string `json:"favorite_key"`
}

func (s *Service) visibleByID(ctx context.Context, p *policy.Principal, workspaces map[string]bool) (map[string]*Meta, error) {
	out := map[string]*Meta{}
	for ws := range workspaces {
		pages, err := s.Tree(ctx, p, ws)
		if hidden(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, m := range pages {
			out[m.ID] = m
		}
	}
	return out, nil
}

func (s *Service) Favorites(ctx context.Context, p *policy.Principal) ([]Favorite, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT f.page_id, f.sort_key, p.workspace_id FROM favorites f JOIN pages p ON p.id = f.page_id
		WHERE f.user_id = ? AND p.trashed_at = 0 ORDER BY f.sort_key`, p.UserID)
	if err != nil {
		return nil, err
	}
	type fav struct{ id, key string }
	var favs []fav
	workspaces := map[string]bool{}
	for rows.Next() {
		var f fav
		var ws string
		if err := rows.Scan(&f.id, &f.key, &ws); err != nil {
			rows.Close()
			return nil, err
		}
		favs = append(favs, f)
		workspaces[ws] = true
	}
	rows.Close()
	visible, err := s.visibleByID(ctx, p, workspaces)
	if err != nil {
		return nil, err
	}
	out := []Favorite{}
	for _, f := range favs {
		if m, ok := visible[f.id]; ok {
			out = append(out, Favorite{Meta: m, FavoriteKey: f.key})
		}
	}
	return out, nil
}

type FavoriteInput struct {
	AfterID string `json:"after_id"`
	Top     bool   `json:"top"`
}

func (s *Service) SetFavorite(ctx context.Context, p *policy.Principal, pageID string, in FavoriteInput) error {
	m, _, err := s.readable(ctx, p, pageID)
	if err != nil {
		return err
	}
	if m.TrashedAt > 0 {
		return errTrashed
	}
	afterID := in.AfterID
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var key string
		if afterID == "" {
			var existing string
			err := tx.QueryRowContext(ctx, "SELECT sort_key FROM favorites WHERE user_id = ? AND page_id = ?", p.UserID, pageID).Scan(&existing)
			if err == nil && !in.Top {
				return nil
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			var edge sql.NullString
			q := "SELECT MAX(sort_key) FROM favorites WHERE user_id = ? AND page_id != ?"
			if in.Top {
				q = "SELECT MIN(sort_key) FROM favorites WHERE user_id = ? AND page_id != ?"
			}
			if err := tx.QueryRowContext(ctx, q, p.UserID, pageID).Scan(&edge); err != nil {
				return err
			}
			if in.Top && edge.String != "" {
				key = order.Before(edge.String)
			} else {
				key = order.After(edge.String)
			}
		} else {
			var a string
			if err := tx.QueryRowContext(ctx, "SELECT sort_key FROM favorites WHERE user_id = ? AND page_id = ?", p.UserID, afterID).Scan(&a); err != nil {
				return httpx.Fail(http.StatusBadRequest, "bad_sibling", "that page is not a favorite")
			}
			var b sql.NullString
			if err := tx.QueryRowContext(ctx, "SELECT MIN(sort_key) FROM favorites WHERE user_id = ? AND sort_key > ? AND page_id != ?", p.UserID, a, pageID).Scan(&b); err != nil {
				return err
			}
			k, err := order.Between(a, b.String)
			if err != nil {
				return err
			}
			key = k
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO favorites (user_id, page_id, sort_key, created_at) VALUES (?, ?, ?, ?)
			ON CONFLICT (user_id, page_id) DO UPDATE SET sort_key = excluded.sort_key`, p.UserID, pageID, key, store.NowMS())
		return err
	})
	if err != nil {
		return err
	}
	s.hub.User(p.UserID, "favorites")
	return nil
}

func (s *Service) RemoveFavorite(ctx context.Context, p *policy.Principal, pageID string) error {
	if _, err := s.st.W.ExecContext(ctx, "DELETE FROM favorites WHERE user_id = ? AND page_id = ?", p.UserID, pageID); err != nil {
		return err
	}
	s.hub.User(p.UserID, "favorites")
	return nil
}

func (s *Service) Visit(ctx context.Context, p *policy.Principal, pageID string) error {
	if !p.IsSession() {
		return nil
	}
	if _, _, err := s.readable(ctx, p, pageID); err != nil {
		return err
	}
	_, err := s.st.W.ExecContext(ctx, `INSERT INTO page_visits (user_id, page_id, visited_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id, page_id) DO UPDATE SET visited_at = excluded.visited_at`, p.UserID, pageID, store.NowMS())
	return err
}

type Recent struct {
	*Meta
	VisitedAt int64 `json:"visited_at"`
}

func (s *Service) Recent(ctx context.Context, p *policy.Principal, limit int) ([]Recent, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT v.page_id, v.visited_at, p.workspace_id FROM page_visits v JOIN pages p ON p.id = v.page_id
		WHERE v.user_id = ? AND p.trashed_at = 0 ORDER BY v.visited_at DESC LIMIT ?`, p.UserID, limit*2)
	if err != nil {
		return nil, err
	}
	type visit struct {
		id string
		at int64
	}
	var visits []visit
	workspaces := map[string]bool{}
	for rows.Next() {
		var v visit
		var ws string
		if err := rows.Scan(&v.id, &v.at, &ws); err != nil {
			rows.Close()
			return nil, err
		}
		visits = append(visits, v)
		workspaces[ws] = true
	}
	rows.Close()
	visible, err := s.visibleByID(ctx, p, workspaces)
	if err != nil {
		return nil, err
	}
	out := []Recent{}
	for _, v := range visits {
		if m, ok := visible[v.id]; ok && len(out) < limit {
			out = append(out, Recent{Meta: m, VisitedAt: v.at})
		}
	}
	return out, nil
}
