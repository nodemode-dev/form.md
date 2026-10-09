package collection

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/order"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxViewConfig = 64 << 10

type ViewInput struct {
	Name    *string          `json:"name"`
	Type    *string          `json:"type"`
	Config  *json.RawMessage `json:"config"`
	AfterID string           `json:"after_id"`
}

func badView(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_view", msg) }

func filterDepth(f *Filter, depth int) bool {
	if f == nil {
		return true
	}
	if depth > 3 {
		return false
	}
	if len(f.Items) > 100 {
		return false
	}
	for i := range f.Items {
		if !filterDepth(&f.Items[i], depth+1) {
			return false
		}
	}
	return true
}

func parseConfig(raw json.RawMessage) (ViewConfig, string, error) {
	var cfg ViewConfig
	if len(raw) > maxViewConfig {
		return cfg, "", badView("the view settings are too large")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return cfg, "", badView("the view settings are not valid: " + err.Error())
	}
	if !filterDepth(cfg.Filter, 0) {
		return cfg, "", badView("filters nest up to three groups deep")
	}
	if len(cfg.Sorts) > 20 {
		return cfg, "", badView("up to 20 sorts")
	}
	for _, s := range cfg.Sorts {
		if s.Dir != "asc" && s.Dir != "desc" {
			return cfg, "", badView("a sort is asc or desc")
		}
	}
	for _, fn := range cfg.Calcs {
		if !calcFns[fn] {
			return cfg, "", badView("unknown calculation " + fn)
		}
	}
	b, _ := json.Marshal(cfg)
	return cfg, string(b), nil
}

func cleanViewName(raw, typ string) (string, error) {
	n := strings.Join(strings.Fields(raw), " ")
	if n == "" {
		n = defaultViewName(typ)
	}
	if utf8.RuneCountInString(n) > 100 {
		return "", badView("a view name can be up to 100 characters")
	}
	return n, nil
}

func (s *Service) emitViews(ctx context.Context, collectionID string) {
	s.bump()
	s.emit(ctx, collectionID, map[string]any{"what": "views"})
}

func (s *Service) CreateView(ctx context.Context, p *policy.Principal, collectionID string, in ViewInput) (*View, error) {
	m, err := s.structural(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	typ := "table"
	if in.Type != nil {
		typ = *in.Type
	}
	if !viewTypes[typ] {
		return nil, badView("unknown view type " + typ)
	}
	name := ""
	if in.Name != nil {
		name = *in.Name
	}
	if name, err = cleanViewName(name, typ); err != nil {
		return nil, err
	}
	cfgJSON := "{}"
	if in.Config != nil {
		if _, cfgJSON, err = parseConfig(*in.Config); err != nil {
			return nil, err
		}
	}
	id := ids.New()
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM collection_views WHERE collection_id = ?", collectionID).Scan(&n); err != nil {
			return err
		}
		if n >= 100 {
			return badView("a database can have up to 100 views")
		}
		key, err := s.viewKey(ctx, tx, collectionID, in.AfterID)
		if err != nil {
			return err
		}
		now := store.NowMS()
		if _, err := tx.ExecContext(ctx, `INSERT INTO collection_views (id, collection_id, sort_key, name, type, config, created_at, created_by, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, collectionID, key, name, typ, cfgJSON, now, p.UserID, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "view.create", WorkspaceID: m.WorkspaceID, PageID: collectionID, Detail: name})
	})
	if err != nil {
		return nil, err
	}
	s.emitViews(ctx, collectionID)
	return s.view(ctx, id)
}

func (s *Service) viewKey(ctx context.Context, q store.Querier, collectionID, afterID string) (string, error) {
	if afterID != "" {
		var a string
		if err := q.QueryRowContext(ctx, "SELECT sort_key FROM collection_views WHERE id = ? AND collection_id = ?", afterID, collectionID).Scan(&a); err != nil {
			return "", badView("that view was not found")
		}
		var b sql.NullString
		err := q.QueryRowContext(ctx, "SELECT sort_key FROM collection_views WHERE collection_id = ? AND sort_key > ? ORDER BY sort_key LIMIT 1", collectionID, a).Scan(&b)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return order.Between(a, b.String)
	}
	var last sql.NullString
	if err := q.QueryRowContext(ctx, "SELECT MAX(sort_key) FROM collection_views WHERE collection_id = ?", collectionID).Scan(&last); err != nil {
		return "", err
	}
	return order.After(last.String), nil
}

func (s *Service) writableView(ctx context.Context, p *policy.Principal, viewID string) (*View, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return nil, err
	}
	if _, err := s.structural(ctx, p, v.CollectionID); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) UpdateView(ctx context.Context, p *policy.Principal, viewID string, in ViewInput) (*View, error) {
	v, err := s.writableView(ctx, p, viewID)
	if err != nil {
		return nil, err
	}
	sets, args := []string{"updated_at = ?"}, []any{store.NowMS()}
	typ := v.Type
	if in.Type != nil {
		if !viewTypes[*in.Type] {
			return nil, badView("unknown view type " + *in.Type)
		}
		typ = *in.Type
		sets, args = append(sets, "type = ?"), append(args, typ)
	}
	if in.Name != nil {
		name, err := cleanViewName(*in.Name, typ)
		if err != nil {
			return nil, err
		}
		sets, args = append(sets, "name = ?"), append(args, name)
	}
	if in.Config != nil {
		_, cfg, err := parseConfig(*in.Config)
		if err != nil {
			return nil, err
		}
		sets, args = append(sets, "config = ?"), append(args, cfg)
	}
	args = append(args, viewID)
	if _, err := s.st.W.ExecContext(ctx, "UPDATE collection_views SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		return nil, err
	}
	s.emitViews(ctx, v.CollectionID)
	s.viewAudit(ctx, p, v, "view.update")
	return s.view(ctx, viewID)
}

func (s *Service) viewAudit(ctx context.Context, p *policy.Principal, v *View, action string) {
	var ws string
	_ = s.st.R.QueryRowContext(ctx, "SELECT workspace_id FROM pages WHERE id = ?", v.CollectionID).Scan(&ws)
	_ = s.audit.Record(ctx, nil, p, audit.Entry{Action: action, WorkspaceID: ws, PageID: v.CollectionID, Detail: v.Name})
}

func (s *Service) DeleteView(ctx context.Context, p *policy.Principal, viewID string) error {
	v, err := s.writableView(ctx, p, viewID)
	if err != nil {
		return err
	}
	var n int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM collection_views WHERE collection_id = ?", v.CollectionID).Scan(&n); err != nil {
		return err
	}
	if n <= 1 {
		return httpx.Fail(http.StatusConflict, "last_view", "a database keeps at least one view")
	}
	if _, err := s.st.W.ExecContext(ctx, "DELETE FROM collection_views WHERE id = ?", viewID); err != nil {
		return err
	}
	s.emitViews(ctx, v.CollectionID)
	s.viewAudit(ctx, p, v, "view.delete")
	return nil
}

func (s *Service) MoveView(ctx context.Context, p *policy.Principal, viewID, afterID string) error {
	v, err := s.writableView(ctx, p, viewID)
	if err != nil {
		return err
	}
	var key string
	if afterID == "" {
		var first sql.NullString
		if err := s.st.R.QueryRowContext(ctx, "SELECT MIN(sort_key) FROM collection_views WHERE collection_id = ? AND id != ?", v.CollectionID, viewID).Scan(&first); err != nil {
			return err
		}
		if key, err = order.Between("", first.String); err != nil {
			return err
		}
	} else if key, err = s.viewKey(ctx, s.st.R, v.CollectionID, afterID); err != nil {
		return err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE collection_views SET sort_key = ? WHERE id = ?", key, viewID); err != nil {
		return err
	}
	s.emitViews(ctx, v.CollectionID)
	s.viewAudit(ctx, p, v, "view.move")
	return nil
}

func (s *Service) DuplicateView(ctx context.Context, p *policy.Principal, viewID string) (*View, error) {
	v, err := s.writableView(ctx, p, viewID)
	if err != nil {
		return nil, err
	}
	name := v.Name + " copy"
	typ := v.Type
	raw, _ := json.Marshal(v.Config)
	cfg := json.RawMessage(raw)
	return s.CreateView(ctx, p, v.CollectionID, ViewInput{Name: &name, Type: &typ, Config: &cfg, AfterID: viewID})
}

var tokenEnc = base32.StdEncoding.WithPadding(base32.NoPadding)

func newToken() string {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return strings.ToLower(tokenEnc.EncodeToString(b[:]))
}
