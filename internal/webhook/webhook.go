package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	maxAttempts    = 8
	disableAfter   = 100
	keepDeliveries = 30 * 24 * time.Hour
)

var backoff = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute, 2 * time.Hour, 6 * time.Hour, 12 * time.Hour}

type Hook struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	Kind        string   `json:"kind"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Events      []string `json:"events"`
	Format      string   `json:"format"`
	HasAuth     bool     `json:"has_auth"`
	Active      bool     `json:"active"`
	Failures    int      `json:"failures"`
	CreatedBy   string   `json:"created_by"`
	CreatedAt   int64    `json:"created_at"`
	UpdatedAt   int64    `json:"updated_at"`
	LastStatus  string   `json:"last_status"`
	LastAt      int64    `json:"last_at"`

	secret string
	auth   string
}

type Delivery struct {
	ID          string `json:"id"`
	WebhookID   string `json:"webhook_id"`
	Event       string `json:"event"`
	EventID     string `json:"event_id"`
	Status      string `json:"status"`
	Attempts    int    `json:"attempts"`
	NextAt      int64  `json:"next_at"`
	LastStatus  int    `json:"last_status"`
	LastError   string `json:"last_error"`
	CreatedAt   int64  `json:"created_at"`
	DeliveredAt int64  `json:"delivered_at"`
	Payload     string `json:"payload,omitempty"`
}

type Actor struct {
	Kind  string `json:"kind"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Agent string `json:"agent,omitempty"`
}

type Event struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	At          int64           `json:"at"`
	WorkspaceID string          `json:"workspace_id"`
	PageID      string          `json:"page_id,omitempty"`
	Actor       Actor           `json:"actor"`
	Detail      string          `json:"detail,omitempty"`
	Changes     json.RawMessage `json:"changes,omitempty"`
	Data        any             `json:"data,omitempty"`
	Audit       bool            `json:"-"`
}

type Service struct {
	st     *store.Store
	gate   *policy.Gate
	box    *secrets.Box
	audit  *audit.Log
	log    *slog.Logger
	client *http.Client
	wake   chan struct{}
	plain  bool
	subs   []func(ctx context.Context, e Event)
}

func (s *Service) OnEvent(fn func(ctx context.Context, e Event)) {
	s.subs = append(s.subs, fn)
}

func New(st *store.Store, gate *policy.Gate, box *secrets.Box, al *audit.Log, fetcher *http.Client, allowHTTP bool, log *slog.Logger) *Service {
	c := *fetcher
	c.Timeout = 10 * time.Second
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Service{st: st, gate: gate, box: box, audit: al, log: log, client: &c, wake: make(chan struct{}, 1), plain: allowHTTP}
}

func (s *Service) poke() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func matches(patterns []string, event string, isAudit bool, kind string) bool {
	if kind == "siem" {
		return isAudit
	}
	for _, p := range patterns {
		if p == "*" || p == event {
			return true
		}
		if ok, _ := path.Match(p, event); ok {
			return true
		}
	}
	return false
}

func (s *Service) enqueue(ctx context.Context, e Event) {
	for _, fn := range s.subs {
		go fn(context.WithoutCancel(ctx), e)
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, kind, events FROM webhooks WHERE workspace_id = ? AND active = 1", e.WorkspaceID)
	if err != nil {
		return
	}
	type target struct{ id, kind, events string }
	var hooks []target
	for rows.Next() {
		var t target
		if rows.Scan(&t.id, &t.kind, &t.events) == nil {
			hooks = append(hooks, t)
		}
	}
	rows.Close()
	if len(hooks) == 0 {
		return
	}
	body, err := json.Marshal(e)
	if err != nil {
		return
	}
	now := store.NowMS()
	queued := false
	for _, h := range hooks {
		var pats []string
		_ = json.Unmarshal([]byte(h.events), &pats)
		if !matches(pats, e.Type, e.Audit, h.kind) {
			continue
		}
		if _, err := s.st.W.ExecContext(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, next_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			ids.New(), h.id, e.Type, e.ID, string(body), now, now); err != nil {
			s.log.Warn("webhook: enqueue", "err", err)
			continue
		}
		queued = true
	}
	if queued {
		s.poke()
	}
}

func (s *Service) Emit(ctx context.Context, workspaceID, kind, pageID string, actor *policy.Principal, data any) {
	if workspaceID == "" {
		return
	}
	e := Event{ID: "evt_" + ids.New(), Type: kind, At: store.NowMS(), WorkspaceID: workspaceID, PageID: pageID, Actor: Actor{Kind: "system"}, Data: data}
	if actor != nil {
		e.Actor = Actor{Kind: "human", ID: actor.UserID, Label: actor.Name}
		if actor.IsAgent() {
			e.Actor.Kind, e.Actor.Agent = "agent", actor.AgentName
		}
	}
	s.enqueue(ctx, e)
}

func (s *Service) pumpAudit(ctx context.Context) {
	var last int64
	if err := s.st.R.QueryRowContext(ctx, "SELECT last_id FROM event_cursors WHERE name = 'audit'").Scan(&last); err != nil {
		_ = s.st.R.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM audit_log").Scan(&last)
		_, _ = s.st.W.ExecContext(ctx, "INSERT INTO event_cursors (name, last_id) VALUES ('audit', ?) ON CONFLICT (name) DO NOTHING", last)
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT id, at, actor_kind, actor_id, actor_label, agent_name, action, workspace_id, page_id, detail, changes FROM audit_log WHERE id > ? ORDER BY id LIMIT 500`, last)
	if err != nil {
		return
	}
	var batch []Event
	for rows.Next() {
		var id int64
		var e Event
		var changes string
		if rows.Scan(&id, &e.At, &e.Actor.Kind, &e.Actor.ID, &e.Actor.Label, &e.Actor.Agent, &e.Type, &e.WorkspaceID, &e.PageID, &e.Detail, &changes) != nil {
			continue
		}
		e.ID = "aud_" + strconv.FormatInt(id, 10)
		e.Audit = true
		if changes != "" {
			e.Changes = json.RawMessage(changes)
		}
		batch = append(batch, e)
		last = id
	}
	rows.Close()
	for _, e := range batch {
		s.enqueue(ctx, e)
	}
	if len(batch) > 0 {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE event_cursors SET last_id = ? WHERE name = 'audit'", last)
	}
}

func sign(secret string, ts int64, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strconv.FormatInt(ts, 10) + "."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Service) deliverDue(ctx context.Context) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT d.id, d.webhook_id, d.event, d.payload, d.attempts, w.url, w.secret, w.format, w.auth_header
		FROM webhook_deliveries d JOIN webhooks w ON w.id = d.webhook_id
		WHERE d.status = 'pending' AND d.next_at <= ? AND w.active = 1 ORDER BY d.next_at LIMIT 50`, store.NowMS())
	if err != nil {
		return
	}
	type job struct {
		id, hook, event, payload, url, secret, format, auth string
		attempts                                            int
	}
	var jobs []job
	for rows.Next() {
		var j job
		if rows.Scan(&j.id, &j.hook, &j.event, &j.payload, &j.attempts, &j.url, &j.secret, &j.format, &j.auth) == nil {
			jobs = append(jobs, j)
		}
	}
	rows.Close()
	for _, j := range jobs {
		secret, _ := s.box.Unseal(j.secret)
		auth := ""
		if j.auth != "" {
			auth, _ = s.box.Unseal(j.auth)
		}
		body := []byte(j.payload)
		if j.format == "splunk" {
			var e Event
			_ = json.Unmarshal(body, &e)
			body, _ = json.Marshal(map[string]any{"time": float64(e.At) / 1000, "host": "form.md", "source": "form.md", "sourcetype": "_json", "event": json.RawMessage(j.payload)})
		}
		code, derr := s.post(ctx, j.url, j.event, j.id, secret, auth, body)
		now := store.NowMS()
		if derr == nil && code >= 200 && code < 300 {
			_, _ = s.st.W.ExecContext(ctx, "UPDATE webhook_deliveries SET status = 'delivered', attempts = attempts + 1, last_status = ?, last_error = '', delivered_at = ? WHERE id = ?", code, now, j.id)
			_, _ = s.st.W.ExecContext(ctx, "UPDATE webhooks SET failures = 0 WHERE id = ?", j.hook)
			continue
		}
		msg := ""
		if derr != nil {
			_, msg = netguard.Reason(derr)
			if msg == "" {
				msg = derr.Error()
			}
		} else {
			msg = "HTTP " + strconv.Itoa(code)
		}
		attempts := j.attempts + 1
		if attempts >= maxAttempts {
			_, _ = s.st.W.ExecContext(ctx, "UPDATE webhook_deliveries SET status = 'failed', attempts = ?, last_status = ?, last_error = ? WHERE id = ?", attempts, code, msg, j.id)
			s.failed(ctx, j.hook)
			continue
		}
		next := now + backoff[min(attempts-1, len(backoff)-1)].Milliseconds()
		_, _ = s.st.W.ExecContext(ctx, "UPDATE webhook_deliveries SET attempts = ?, next_at = ?, last_status = ?, last_error = ? WHERE id = ?", attempts, next, code, msg, j.id)
	}
}

func (s *Service) failed(ctx context.Context, hook string) {
	var n int
	var ws string
	if s.st.W.QueryRowContext(ctx, "UPDATE webhooks SET failures = failures + 1 WHERE id = ? RETURNING failures, workspace_id", hook).Scan(&n, &ws) != nil {
		return
	}
	if n >= disableAfter {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE webhooks SET active = 0, updated_at = ? WHERE id = ?", store.NowMS(), hook)
		_ = s.audit.Record(ctx, nil, nil, audit.Entry{Action: "webhook.disabled", WorkspaceID: ws, Detail: hook, ActorKind: "system"})
	}
}

func (s *Service) post(ctx context.Context, url, event, delivery, secret, auth string, body []byte) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	ts := time.Now().Unix()
	h := req.Header
	h.Set("Content-Type", "application/json")
	h.Set("User-Agent", "form.md-webhooks/1")
	h.Set("X-FormMD-Event", event)
	h.Set("X-FormMD-Delivery", delivery)
	h.Set("X-FormMD-Signature", "t="+strconv.FormatInt(ts, 10)+",v1="+sign(secret, ts, body))
	if auth != "" {
		h.Set("Authorization", auth)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	res.Body.Close()
	return res.StatusCode, nil
}

func (s *Service) Run(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	sweep := time.Now()
	for {
		s.pumpAudit(ctx)
		s.deliverDue(ctx)
		if time.Since(sweep) > time.Hour {
			sweep = time.Now()
			_, _ = s.st.W.ExecContext(ctx, "DELETE FROM webhook_deliveries WHERE status != 'pending' AND created_at < ?", store.NowMS()-keepDeliveries.Milliseconds())
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-s.wake:
		}
	}
}

func (s *Service) admin(ctx context.Context, p *policy.Principal, workspaceID string) error {
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.IsAdmin() {
		return httpx.Fail(http.StatusForbidden, "admin_only", "only workspace admins manage webhooks")
	}
	return nil
}
