package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxHooks = 25

var eventPattern = regexp.MustCompile(`^(\*|[a-z_]+(\.(\*|[a-z_]+))?)$`)

const hookColumns = `w.id, w.workspace_id, w.kind, w.url, w.description, w.events, w.format, w.auth_header != '', w.active, w.failures, w.created_by, w.created_at, w.updated_at,
	COALESCE((SELECT d.status FROM webhook_deliveries d WHERE d.webhook_id = w.id AND d.status != 'pending' ORDER BY d.created_at DESC LIMIT 1), ''),
	COALESCE((SELECT MAX(d.created_at) FROM webhook_deliveries d WHERE d.webhook_id = w.id), 0), w.secret, w.auth_header`

func scanHook(row interface{ Scan(...any) error }) (*Hook, error) {
	var h Hook
	var events string
	var active int
	if err := row.Scan(&h.ID, &h.WorkspaceID, &h.Kind, &h.URL, &h.Description, &events, &h.Format, &h.HasAuth, &active, &h.Failures, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt, &h.LastStatus, &h.LastAt, &h.secret, &h.auth); err != nil {
		return nil, err
	}
	h.Active = active == 1
	_ = json.Unmarshal([]byte(events), &h.Events)
	if h.Events == nil {
		h.Events = []string{}
	}
	return &h, nil
}

func (s *Service) hook(ctx context.Context, id string) (*Hook, error) {
	h, err := scanHook(s.st.R.QueryRowContext(ctx, `SELECT `+hookColumns+` FROM webhooks w WHERE w.id = ?`, id))
	if err != nil {
		return nil, store.NoRows(err)
	}
	return h, nil
}

func (s *Service) List(ctx context.Context, p *policy.Principal, workspaceID string) ([]*Hook, error) {
	if err := s.admin(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT `+hookColumns+` FROM webhooks w WHERE w.workspace_id = ? ORDER BY w.created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Hook{}
	for rows.Next() {
		h, err := scanHook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

type Input struct {
	Kind        string    `json:"kind"`
	URL         *string   `json:"url"`
	Description *string   `json:"description"`
	Events      *[]string `json:"events"`
	Format      *string   `json:"format"`
	Auth        *string   `json:"auth_header"`
	Active      *bool     `json:"active"`
}

func (s *Service) checkURL(raw string) (string, error) {
	u, err := netguard.CheckURL(raw)
	if err != nil || (u.Scheme != "https" && !s.plain) {
		return "", httpx.Fail(http.StatusBadRequest, "bad_url", "use an https address")
	}
	return u.String(), nil
}

func checkEvents(list []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, e := range list {
		e = strings.TrimSpace(e)
		if e == "" || seen[e] {
			continue
		}
		if !eventPattern.MatchString(e) {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_event", "events look like page.create, page.* or *")
		}
		seen[e] = true
		out = append(out, e)
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, p *policy.Principal, workspaceID string, in Input) (*Hook, string, error) {
	if err := s.admin(ctx, p, workspaceID); err != nil {
		return nil, "", err
	}
	var n int
	_ = s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhooks WHERE workspace_id = ?", workspaceID).Scan(&n)
	if n >= maxHooks {
		return nil, "", httpx.Fail(http.StatusConflict, "too_many", "a workspace has at most 25 webhooks and streams")
	}
	if in.Kind == "" {
		in.Kind = "webhook"
	}
	if in.Kind != "webhook" && in.Kind != "siem" {
		return nil, "", httpx.Fail(http.StatusBadRequest, "bad_kind", "kind is webhook or siem")
	}
	if in.URL == nil {
		return nil, "", httpx.Fail(http.StatusBadRequest, "bad_url", "an address is required")
	}
	url, err := s.checkURL(*in.URL)
	if err != nil {
		return nil, "", err
	}
	var events []string
	if in.Events != nil {
		events = *in.Events
	}
	if events, err = checkEvents(events); err != nil {
		return nil, "", err
	}
	format := "json"
	if in.Format != nil && *in.Format == "splunk" {
		format = "splunk"
	}
	desc := ""
	if in.Description != nil {
		desc = clip(*in.Description, 200)
	}
	authHeader := ""
	if in.Auth != nil && strings.TrimSpace(*in.Auth) != "" {
		authHeader = s.box.Seal(strings.TrimSpace(*in.Auth))
	}
	secret := secrets.Token("whsec_", 24)
	id := ids.New()
	now := store.NowMS()
	ev, _ := json.Marshal(events)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO webhooks (id, workspace_id, kind, url, description, events, secret, format, auth_header, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, workspaceID, in.Kind, url, desc, string(ev), s.box.Seal(secret), format, authHeader, p.UserID, now, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "webhook.create", WorkspaceID: workspaceID, Detail: in.Kind + " " + url})
	})
	if err != nil {
		return nil, "", err
	}
	h, err := s.hook(ctx, id)
	return h, secret, err
}

func (s *Service) owned(ctx context.Context, p *policy.Principal, id string) (*Hook, error) {
	h, err := s.hook(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.admin(ctx, p, h.WorkspaceID); err != nil {
		return nil, err
	}
	return h, nil
}

func (s *Service) Update(ctx context.Context, p *policy.Principal, id string, in Input) (*Hook, error) {
	h, err := s.owned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	url, desc, events, format, authHeader, active := h.URL, h.Description, h.Events, h.Format, h.auth, h.Active
	if in.URL != nil {
		if url, err = s.checkURL(*in.URL); err != nil {
			return nil, err
		}
	}
	if in.Description != nil {
		desc = clip(*in.Description, 200)
	}
	if in.Events != nil {
		if events, err = checkEvents(*in.Events); err != nil {
			return nil, err
		}
	}
	if in.Format != nil {
		format = "json"
		if *in.Format == "splunk" {
			format = "splunk"
		}
	}
	if in.Auth != nil {
		authHeader = ""
		if v := strings.TrimSpace(*in.Auth); v != "" {
			authHeader = s.box.Seal(v)
		}
	}
	if in.Active != nil {
		active = *in.Active
	}
	ev, _ := json.Marshal(events)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		failures := "failures"
		if active && !h.Active {
			failures = "0"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE webhooks SET url = ?, description = ?, events = ?, format = ?, auth_header = ?, active = ?, failures = `+failures+`, updated_at = ? WHERE id = ?`,
			url, desc, string(ev), format, authHeader, store.Bool(active), store.NowMS(), id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "webhook.update", WorkspaceID: h.WorkspaceID, Detail: id})
	})
	if err != nil {
		return nil, err
	}
	return s.hook(ctx, id)
}

func (s *Service) Delete(ctx context.Context, p *policy.Principal, id string) error {
	h, err := s.owned(ctx, p, id)
	if err != nil {
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM webhooks WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "webhook.delete", WorkspaceID: h.WorkspaceID, Detail: h.URL})
	})
}

func (s *Service) Rotate(ctx context.Context, p *policy.Principal, id string) (string, error) {
	h, err := s.owned(ctx, p, id)
	if err != nil {
		return "", err
	}
	secret := secrets.Token("whsec_", 24)
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE webhooks SET secret = ?, updated_at = ? WHERE id = ?", s.box.Seal(secret), store.NowMS(), id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "webhook.rotate", WorkspaceID: h.WorkspaceID, Detail: id})
	})
	return secret, err
}

func (s *Service) Test(ctx context.Context, p *policy.Principal, id string) error {
	h, err := s.owned(ctx, p, id)
	if err != nil {
		return err
	}
	e := Event{ID: "evt_" + ids.New(), Type: "ping", At: store.NowMS(), WorkspaceID: h.WorkspaceID, Actor: Actor{Kind: "human", ID: p.UserID, Label: p.Name}, Data: map[string]string{"webhook": h.ID}}
	body, _ := json.Marshal(e)
	now := store.NowMS()
	if _, err := s.st.W.ExecContext(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, next_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ids.New(), h.ID, e.Type, e.ID, string(body), now, now); err != nil {
		return err
	}
	s.poke()
	return nil
}

func (s *Service) Deliveries(ctx context.Context, p *policy.Principal, id string) ([]Delivery, error) {
	if _, err := s.owned(ctx, p, id); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT id, webhook_id, event, event_id, status, attempts, next_at, last_status, last_error, created_at, delivered_at, payload
		FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT 50`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.EventID, &d.Status, &d.Attempts, &d.NextAt, &d.LastStatus, &d.LastError, &d.CreatedAt, &d.DeliveredAt, &d.Payload); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Service) Retry(ctx context.Context, p *policy.Principal, deliveryID string) error {
	var hook string
	if err := s.st.R.QueryRowContext(ctx, "SELECT webhook_id FROM webhook_deliveries WHERE id = ?", deliveryID).Scan(&hook); err != nil {
		return store.NoRows(err)
	}
	if _, err := s.owned(ctx, p, hook); err != nil {
		return err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE webhook_deliveries SET status = 'pending', next_at = ?, attempts = 0 WHERE id = ?", store.NowMS(), deliveryID); err != nil {
		return err
	}
	s.poke()
	return nil
}

func clip(v string, n int) string {
	v = strings.TrimSpace(v)
	if r := []rune(v); len(r) > n {
		return string(r[:n])
	}
	return v
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "webhook" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, a.Require(h))
	}
	route("GET /api/workspaces/{id}/webhooks", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.List(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"webhooks": list})
		return nil
	})
	route("POST /api/workspaces/{id}/webhooks", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		h, secret, err := s.Create(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"webhook": h, "secret": secret})
		return nil
	})
	route("PATCH /api/webhooks/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		h, err := s.Update(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"webhook": h})
		return nil
	})
	route("DELETE /api/webhooks/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Delete(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("POST /api/webhooks/{id}/rotate", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		secret, err := s.Rotate(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"secret": secret})
		return nil
	})
	route("POST /api/webhooks/{id}/test", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Test(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusAccepted)
		return nil
	})
	route("GET /api/webhooks/{id}/deliveries", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Deliveries(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"deliveries": list})
		return nil
	})
	route("POST /api/deliveries/{id}/retry", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Retry(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusAccepted)
		return nil
	})
	return nil
}
