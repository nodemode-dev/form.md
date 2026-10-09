package ai

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

var Kinds = map[string]string{"anthropic": "https://api.anthropic.com", "openai": "https://api.openai.com/v1", "gemini": "https://generativelanguage.googleapis.com/v1beta/openai", "compatible": ""}

type Service struct {
	st     *store.Store
	box    *secrets.Box
	set    *settings.Service
	gate   *policy.Gate
	audit  *audit.Log
	log    *slog.Logger
	client *http.Client
	limit  *httpx.Limiter
	stub   func(kind, base, key string) (Provider, bool)
}

func New(st *store.Store, box *secrets.Box, set *settings.Service, gate *policy.Gate, al *audit.Log, log *slog.Logger) *Service {
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
		MaxIdleConnsPerHost:   4,
	}
	return &Service{st: st, box: box, set: set, gate: gate, audit: al, log: log, client: &http.Client{Transport: tr}, limit: httpx.NewLimiter(30, 20)}
}

type ProviderInfo struct {
	ID      string       `json:"id"`
	Kind    string       `json:"kind"`
	Name    string       `json:"name"`
	BaseURL string       `json:"base_url"`
	HasKey  bool         `json:"has_key"`
	Enabled bool         `json:"enabled"`
	Models  []ModelEntry `json:"models"`
}

type ModelEntry struct {
	Model   string `json:"model"`
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Audio   bool   `json:"audio"`
}

type providerRow struct {
	id, kind, name, base, key string
	enabled                   bool
}

func (s *Service) row(ctx context.Context, id string) (*providerRow, error) {
	var r providerRow
	var enabled int
	err := s.st.R.QueryRowContext(ctx, "SELECT id, kind, name, base_url, api_key, enabled FROM ai_providers WHERE id = ?", id).Scan(&r.id, &r.kind, &r.name, &r.base, &r.key, &enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	r.enabled = enabled == 1
	return &r, err
}

func (s *Service) build(r *providerRow) (Provider, error) {
	key := ""
	if r.key != "" {
		k, err := s.box.Unseal(r.key)
		if err != nil {
			return nil, err
		}
		key = k
	}
	base := r.base
	if base == "" {
		base = Kinds[r.kind]
	}
	if s.stub != nil {
		if p, ok := s.stub(r.kind, base, key); ok {
			return p, nil
		}
	}
	switch r.kind {
	case "anthropic":
		return &anthropic{base: base, key: key, client: s.client}, nil
	case "openai":
		return &openai{base: base, key: key, client: s.client, maxField: "max_completion_tokens"}, nil
	case "gemini", "compatible":
		if base == "" {
			return nil, httpx.Fail(http.StatusBadRequest, "no_base_url", "this provider needs a base URL")
		}
		return &openai{base: base, key: key, client: s.client, maxField: "max_tokens"}, nil
	}
	return nil, httpx.Fail(http.StatusBadRequest, "bad_kind", "unknown provider kind")
}

func (s *Service) Providers(ctx context.Context) ([]ProviderInfo, error) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, kind, name, base_url, api_key != '', enabled FROM ai_providers ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	out := []ProviderInfo{}
	for rows.Next() {
		var p ProviderInfo
		var hasKey, enabled int
		if err := rows.Scan(&p.ID, &p.Kind, &p.Name, &p.BaseURL, &hasKey, &enabled); err != nil {
			rows.Close()
			return nil, err
		}
		p.HasKey, p.Enabled, p.Models = hasKey == 1, enabled == 1, []ModelEntry{}
		out = append(out, p)
	}
	rows.Close()
	for i := range out {
		mr, err := s.st.R.QueryContext(ctx, "SELECT model, label, enabled, audio FROM ai_models WHERE provider_id = ? ORDER BY enabled DESC, model", out[i].ID)
		if err != nil {
			return nil, err
		}
		for mr.Next() {
			var m ModelEntry
			var en, au int
			if mr.Scan(&m.Model, &m.Label, &en, &au) == nil {
				m.Enabled, m.Audio = en == 1, au == 1
				out[i].Models = append(out[i].Models, m)
			}
		}
		mr.Close()
	}
	return out, nil
}

type ProviderInput struct {
	Kind    string  `json:"kind"`
	Name    *string `json:"name"`
	BaseURL *string `json:"base_url"`
	APIKey  *string `json:"api_key"`
	Enabled *bool   `json:"enabled"`
}

func validBase(raw string) bool {
	if raw == "" {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.RawQuery == "" && u.Fragment == ""
}

func (s *Service) CreateProvider(ctx context.Context, p *policy.Principal, in ProviderInput) (string, error) {
	if _, ok := Kinds[in.Kind]; !ok {
		return "", httpx.Fail(http.StatusBadRequest, "bad_kind", "kind is anthropic, openai, gemini or compatible")
	}
	name := in.Kind
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		name = strings.TrimSpace(*in.Name)
	}
	base := ""
	if in.BaseURL != nil {
		base = strings.TrimRight(strings.TrimSpace(*in.BaseURL), "/")
	}
	if !validBase(base) || (in.Kind == "compatible" && base == "") {
		return "", httpx.Fail(http.StatusBadRequest, "bad_base_url", "the base URL must be http or https")
	}
	key := ""
	if in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "" {
		key = s.box.Seal(strings.TrimSpace(*in.APIKey))
	}
	id := ids.New()
	now := store.NowMS()
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO ai_providers (id, kind, name, base_url, api_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)", id, in.Kind, name, base, key, now, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "ai.provider_add", Detail: name})
	})
	return id, err
}

func (s *Service) UpdateProvider(ctx context.Context, p *policy.Principal, id string, in ProviderInput) error {
	r, err := s.row(ctx, id)
	if err != nil {
		return err
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		r.name = strings.TrimSpace(*in.Name)
	}
	if in.BaseURL != nil {
		r.base = strings.TrimRight(strings.TrimSpace(*in.BaseURL), "/")
		if !validBase(r.base) || (r.kind == "compatible" && r.base == "") {
			return httpx.Fail(http.StatusBadRequest, "bad_base_url", "the base URL must be http or https")
		}
	}
	if in.APIKey != nil {
		r.key = ""
		if k := strings.TrimSpace(*in.APIKey); k != "" {
			r.key = s.box.Seal(k)
		}
	}
	if in.Enabled != nil {
		r.enabled = *in.Enabled
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE ai_providers SET name = ?, base_url = ?, api_key = ?, enabled = ?, updated_at = ? WHERE id = ?", r.name, r.base, r.key, store.Bool(r.enabled), store.NowMS(), id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "ai.provider_change", Detail: r.name})
	})
}

func (s *Service) DeleteProvider(ctx context.Context, p *policy.Principal, id string) error {
	r, err := s.row(ctx, id)
	if err != nil {
		return err
	}
	return s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM ai_providers WHERE id = ?", id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "ai.provider_remove", Detail: r.name})
	})
}

func (s *Service) RefreshModels(ctx context.Context, id string) (int, error) {
	r, err := s.row(ctx, id)
	if err != nil {
		return 0, err
	}
	prov, err := s.build(r)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	list, err := prov.Models(ctx)
	if err != nil {
		return 0, httpx.Fail(http.StatusBadGateway, "provider_failed", err.Error())
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		for _, m := range list {
			label := m.Label
			if label == "" {
				label = m.ID
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO ai_models (provider_id, model, label) VALUES (?, ?, ?) ON CONFLICT (provider_id, model) DO UPDATE SET label = excluded.label", id, m.ID, label); err != nil {
				return err
			}
		}
		return nil
	})
	return len(list), err
}

type ModelInput struct {
	Model   string  `json:"model"`
	Label   *string `json:"label"`
	Enabled *bool   `json:"enabled"`
	Audio   *bool   `json:"audio"`
}

func (s *Service) SetModel(ctx context.Context, p *policy.Principal, providerID string, in ModelInput) error {
	if _, err := s.row(ctx, providerID); err != nil {
		return err
	}
	model := strings.TrimSpace(in.Model)
	if model == "" || len(model) > 200 {
		return httpx.Fail(http.StatusBadRequest, "bad_model", "name the model")
	}
	label := model
	if in.Label != nil && strings.TrimSpace(*in.Label) != "" {
		label = strings.TrimSpace(*in.Label)
	}
	if _, err := s.st.W.ExecContext(ctx, "INSERT INTO ai_models (provider_id, model, label) VALUES (?, ?, ?) ON CONFLICT (provider_id, model) DO NOTHING", providerID, model, label); err != nil {
		return err
	}
	if in.Label != nil {
		if _, err := s.st.W.ExecContext(ctx, "UPDATE ai_models SET label = ? WHERE provider_id = ? AND model = ?", label, providerID, model); err != nil {
			return err
		}
	}
	if in.Enabled != nil {
		if _, err := s.st.W.ExecContext(ctx, "UPDATE ai_models SET enabled = ? WHERE provider_id = ? AND model = ?", store.Bool(*in.Enabled), providerID, model); err != nil {
			return err
		}
	}
	if in.Audio != nil {
		if _, err := s.st.W.ExecContext(ctx, "UPDATE ai_models SET audio = ? WHERE provider_id = ? AND model = ?", store.Bool(*in.Audio), providerID, model); err != nil {
			return err
		}
	}
	return s.audit.Record(ctx, nil, p, audit.Entry{Action: "ai.model_change", Detail: model})
}

type Choice struct {
	Ref      string `json:"ref"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
	Audio    bool   `json:"audio,omitempty"`
}

func (s *Service) Choices(ctx context.Context) ([]Choice, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT p.id, p.name, m.model, m.label, m.audio FROM ai_models m JOIN ai_providers p ON p.id = m.provider_id
		WHERE m.enabled = 1 AND p.enabled = 1 ORDER BY p.created_at, m.label`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Choice{}
	for rows.Next() {
		var pid, pname, model, label string
		var audio int
		if err := rows.Scan(&pid, &pname, &model, &label, &audio); err != nil {
			return nil, err
		}
		out = append(out, Choice{Ref: pid + ":" + model, Label: label, Provider: pname, Audio: audio == 1})
	}
	return out, rows.Err()
}

var ErrOff = httpx.Fail(http.StatusConflict, "ai_off", "AI is turned off or no model is set up; an instance admin adds a provider under Instance settings, AI")

func (s *Service) WorkspaceOn(ctx context.Context, workspaceID string) bool {
	if workspaceID == "" {
		return true
	}
	on := 1
	_ = s.st.R.QueryRowContext(ctx, "SELECT ai_enabled FROM workspaces WHERE id = ?", workspaceID).Scan(&on)
	return on == 1
}

func (s *Service) resolve(ctx context.Context, ref string, audio bool) (Provider, string, string, error) {
	if !s.set.Bool("ai.enabled") {
		return nil, "", "", ErrOff
	}
	choices, err := s.Choices(ctx)
	if err != nil {
		return nil, "", "", err
	}
	pick := func(want string) *Choice {
		for i := range choices {
			if choices[i].Ref == want && (!audio || choices[i].Audio) {
				return &choices[i]
			}
		}
		return nil
	}
	c := pick(ref)
	if c == nil && ref != "" {
		return nil, "", "", httpx.Fail(http.StatusBadRequest, "model_unavailable", "that model is not on the instance allow-list")
	}
	if c == nil && audio {
		c = pick(s.set.Get("ai.transcribe_model"))
	}
	if c == nil && !audio {
		c = pick(s.set.Get("ai.default_model"))
	}
	if c == nil {
		for i := range choices {
			if choices[i].Audio == audio || (!audio && !choices[i].Audio) {
				c = &choices[i]
				break
			}
		}
	}
	if c == nil {
		return nil, "", "", ErrOff
	}
	pid, model, _ := strings.Cut(c.Ref, ":")
	r, err := s.row(ctx, pid)
	if err != nil {
		return nil, "", "", err
	}
	prov, err := s.build(r)
	return prov, pid, model, err
}

func (s *Service) allowed(ctx context.Context, p *policy.Principal, workspaceID string) error {
	if workspaceID != "" {
		if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
			return err
		}
		if !s.WorkspaceOn(ctx, workspaceID) {
			return httpx.Fail(http.StatusConflict, "ai_off_workspace", "AI is turned off in this workspace")
		}
	}
	if p != nil && p.UserID != "" {
		limit := s.set.Int("ai.user_limit_per_hour")
		var n int
		_ = s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM ai_usage WHERE user_id = ? AND at > ?", p.UserID, store.NowMS()-3600000).Scan(&n)
		if n >= limit {
			return httpx.Fail(http.StatusTooManyRequests, "ai_limit", "the hourly AI limit for your account is reached")
		}
		if !s.limit.Allow(p.UserID) {
			return httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many AI requests at once")
		}
	}
	return nil
}

func (s *Service) Run(ctx context.Context, p *policy.Principal, workspaceID, feature, ref string, req Request, onText func(string) error) (*Result, error) {
	if err := s.allowed(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	prov, pid, model, err := s.resolve(ctx, ref, false)
	if err != nil {
		return nil, err
	}
	req.Model = model
	if req.MaxTokens <= 0 {
		req.MaxTokens = 2048
	}
	res, err := prov.Stream(ctx, req, onText)
	user := ""
	if p != nil {
		user = p.UserID
	}
	if res != nil {
		_, _ = s.st.W.ExecContext(context.WithoutCancel(ctx), "INSERT INTO ai_usage (user_id, workspace_id, provider_id, model, feature, input_tokens, output_tokens, at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			user, workspaceID, pid, model, feature, res.Usage.Input, res.Usage.Output, store.NowMS())
	}
	if err != nil {
		var pe *ProviderError
		if errors.As(err, &pe) {
			return res, httpx.Fail(http.StatusBadGateway, "provider_failed", pe.Error())
		}
		return res, err
	}
	return res, nil
}

func (s *Service) Transcribe(ctx context.Context, p *policy.Principal, workspaceID, name, mime string, audio interface{ Read([]byte) (int, error) }) (string, error) {
	if err := s.allowed(ctx, p, workspaceID); err != nil {
		return "", err
	}
	prov, pid, model, err := s.resolve(ctx, "", true)
	if err != nil {
		return "", httpx.Fail(http.StatusConflict, "no_transcription", "no transcription model is set up; an instance admin marks one as audio under Instance settings, AI")
	}
	tr, ok := prov.(Transcriber)
	if !ok {
		return "", httpx.Fail(http.StatusConflict, "no_transcription", "the chosen provider cannot transcribe audio")
	}
	text, err := tr.Transcribe(ctx, model, name, mime, audio)
	_, _ = s.st.W.ExecContext(context.WithoutCancel(ctx), "INSERT INTO ai_usage (user_id, workspace_id, provider_id, model, feature, at) VALUES (?, ?, ?, ?, 'transcribe', ?)", p.UserID, workspaceID, pid, model, store.NowMS())
	if err != nil {
		return "", httpx.Fail(http.StatusBadGateway, "provider_failed", err.Error())
	}
	return text, nil
}

type UsageRow struct {
	Model  string `json:"model"`
	Calls  int    `json:"calls"`
	Input  int    `json:"input"`
	Output int    `json:"output"`
}

func (s *Service) Usage(ctx context.Context, days int) ([]UsageRow, error) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT model, COUNT(*), SUM(input_tokens), SUM(output_tokens) FROM ai_usage WHERE at > ? GROUP BY model ORDER BY COUNT(*) DESC", store.NowMS()-int64(days)*86400000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UsageRow{}
	for rows.Next() {
		var u UsageRow
		if rows.Scan(&u.Model, &u.Calls, &u.Input, &u.Output) == nil {
			out = append(out, u)
		}
	}
	return out, rows.Err()
}
