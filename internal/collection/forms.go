package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type FormQuestion struct {
	Prop     string  `json:"prop"`
	Label    string  `json:"label,omitempty"`
	Help     string  `json:"help,omitempty"`
	Required bool    `json:"required,omitempty"`
	When     *Filter `json:"when,omitempty"`
}

type FormConfig struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Questions   []FormQuestion `json:"questions,omitempty"`
	Submit      string         `json:"submit,omitempty"`
	Done        string         `json:"done,omitempty"`
	Anonymous   bool           `json:"anonymous,omitempty"`
	Closed      bool           `json:"closed,omitempty"`
}

type QuestionOut struct {
	FormQuestion
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Options    []Option `json:"options,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Collection string   `json:"collection,omitempty"`
	Format     string   `json:"format,omitempty"`
}

type FormDef struct {
	ViewID       string        `json:"view_id"`
	CollectionID string        `json:"collection_id"`
	WorkspaceID  string        `json:"workspace_id,omitempty"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Icon         string        `json:"icon"`
	Questions    []QuestionOut `json:"questions"`
	Submit       string        `json:"submit"`
	Done         string        `json:"done"`
	Public       bool          `json:"public"`
	Token        string        `json:"token,omitempty"`
	Closed       bool          `json:"closed"`
	Anonymous    bool          `json:"anonymous"`
}

var publicTypes = map[string]bool{TTitle: true, TText: true, TNumber: true, TSelect: true, TMulti: true, TStatus: true, TDate: true,
	TCheckbox: true, TURL: true, TEmail: true, TPhone: true, TPlace: true}

func formQuestions(schema Schema, cfg *FormConfig) []FormQuestion {
	if cfg != nil && len(cfg.Questions) > 0 {
		return cfg.Questions
	}
	var out []FormQuestion
	for _, p := range schema {
		if !computed(p.Type) {
			out = append(out, FormQuestion{Prop: p.ID, Required: p.Type == TTitle})
		}
	}
	return out
}

func (s *Service) formDef(ctx context.Context, v *View, public bool) (*FormDef, error) {
	m, err := s.pages.Load(ctx, v.CollectionID)
	if err != nil {
		return nil, err
	}
	schema, err := s.loadSchema(ctx, s.st.R, v.CollectionID)
	if err != nil {
		return nil, err
	}
	cfg := v.Config.Form
	if cfg == nil {
		cfg = &FormConfig{}
	}
	def := &FormDef{ViewID: v.ID, CollectionID: v.CollectionID, Title: cfg.Title, Description: cfg.Description, Icon: m.Icon,
		Submit: cfg.Submit, Done: cfg.Done, Public: v.FormToken != "", Closed: cfg.Closed, Anonymous: cfg.Anonymous, Questions: []QuestionOut{}}
	if def.Title == "" {
		def.Title = m.Title
	}
	if def.Submit == "" {
		def.Submit = "Submit"
	}
	if def.Done == "" {
		def.Done = "Thank you. Your response was recorded."
	}
	for _, q := range formQuestions(schema, cfg) {
		p := schema.ByID(q.Prop)
		if p == nil || computed(p.Type) {
			continue
		}
		if public && !publicTypes[p.Type] {
			continue
		}
		out := QuestionOut{FormQuestion: q, Name: p.Name, Type: p.Type, Options: p.Options, Limit: p.Limit, Format: p.Format}
		if !public {
			out.Collection = p.Collection
		}
		if out.Label == "" {
			out.Label = p.Name
		}
		if out.Help == "" {
			out.Help = p.Description
		}
		def.Questions = append(def.Questions, out)
	}
	return def, nil
}

func (s *Service) Form(ctx context.Context, p *policy.Principal, viewID string) (*FormDef, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return nil, err
	}
	m, canWrite, err := s.readable(ctx, p, v.CollectionID)
	if err != nil {
		return nil, err
	}
	def, err := s.formDef(ctx, v, false)
	if err != nil {
		return nil, err
	}
	def.WorkspaceID = m.WorkspaceID
	if canWrite {
		def.Token = v.FormToken
	}
	return def, nil
}

func (s *Service) SetFormLink(ctx context.Context, p *policy.Principal, viewID string, public bool) (string, error) {
	v, err := s.writableView(ctx, p, viewID)
	if err != nil {
		return "", err
	}
	if v.Type != "form" {
		return "", httpx.Fail(http.StatusBadRequest, "not_a_form", "only a form view can be shared as a form")
	}
	if !public {
		_, err := s.st.W.ExecContext(ctx, "DELETE FROM form_links WHERE view_id = ?", viewID)
		s.emitViews(ctx, v.CollectionID)
		return "", err
	}
	if v.FormToken != "" {
		return v.FormToken, nil
	}
	token := newToken()
	if _, err := s.st.W.ExecContext(ctx, "INSERT INTO form_links (id, view_id, token, created_by, created_at) VALUES (?, ?, ?, ?, ?)",
		ids.New(), viewID, token, p.UserID, store.NowMS()); err != nil {
		return "", err
	}
	s.emitViews(ctx, v.CollectionID)
	return token, nil
}

func (s *Service) viewByToken(ctx context.Context, token string) (*View, error) {
	var viewID string
	err := s.st.R.QueryRowContext(ctx, "SELECT view_id FROM form_links WHERE token = ?", token).Scan(&viewID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	v, err := s.view(ctx, viewID)
	if err != nil {
		return nil, err
	}
	m, err := s.pages.Load(ctx, v.CollectionID)
	if err != nil || m.TrashedAt > 0 || v.Type != "form" {
		return nil, httpx.ErrNotFound
	}
	return v, nil
}

func (s *Service) PublicForm(ctx context.Context, token string) (*FormDef, error) {
	v, err := s.viewByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	def, err := s.formDef(ctx, v, true)
	if err != nil {
		return nil, err
	}
	def.CollectionID = ""
	def.ViewID = ""
	return def, nil
}

type Answers struct {
	Answers map[string]json.RawMessage `json:"answers"`
}

var errClosed = httpx.Fail(http.StatusConflict, "form_closed", "this form is not taking responses")

func (s *Service) Submit(ctx context.Context, p *policy.Principal, viewID string, in Answers) (string, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return "", err
	}
	m, _, err := s.readable(ctx, p, v.CollectionID)
	if err != nil {
		return "", err
	}
	author := p.UserID
	if v.Config.Form != nil && v.Config.Form.Anonymous {
		author = ""
	}
	return s.submit(ctx, p, m, v, in, author, false)
}

func (s *Service) PublicSubmit(ctx context.Context, token, client string, in Answers) error {
	if !s.formLimit.Allow(client) {
		return httpx.Fail(http.StatusTooManyRequests, "rate_limited", "too many responses from this address; try again in a minute")
	}
	v, err := s.viewByToken(ctx, token)
	if err != nil {
		return err
	}
	m, err := s.pages.Load(ctx, v.CollectionID)
	if err != nil {
		return err
	}
	_, err = s.submit(ctx, nil, m, v, in, "", true)
	if err == nil {
		_, _ = s.st.W.ExecContext(ctx, "UPDATE form_links SET submissions = submissions + 1 WHERE token = ?", token)
	}
	return err
}

func (s *Service) submit(ctx context.Context, p *policy.Principal, m *page.Meta, v *View, in Answers, author string, public bool) (string, error) {
	if v.Type != "form" {
		return "", httpx.Fail(http.StatusBadRequest, "not_a_form", "this view is not a form")
	}
	if v.Config.Form != nil && v.Config.Form.Closed {
		return "", errClosed
	}
	def, err := s.formDef(ctx, v, public)
	if err != nil {
		return "", err
	}
	schema, err := s.loadSchema(ctx, s.st.R, v.CollectionID)
	if err != nil {
		return "", err
	}
	nc := s.normCtx(ctx, m.WorkspaceID)
	if public {
		nc.members = map[string]bool{}
	}
	shadow := &Row{ID: "form", Vals: map[string]json.RawMessage{}}
	sn := &snap{id: v.CollectionID, schema: schema, rows: map[string]*Row{}}
	sn.compile()
	ev := &evaluator{s: s, ctx: ctx, p: p, loc: time.UTC, now: time.Now(), snaps: map[string]*snap{}, memo: map[cellKey]formula.Value{},
		errs: map[cellKey]string{}, visiting: map[cellKey]bool{}, readable: map[string]bool{}, rowCol: map[string]string{}}
	if p == nil {
		ev.p = &policy.Principal{}
	} else {
		ev.loc = s.location(ctx, p)
	}
	values := map[string]json.RawMessage{}
	for _, q := range def.Questions {
		prop := schema.ByID(q.Prop)
		if prop == nil {
			continue
		}
		if q.When != nil && q.When.Prop != "" {
			ev.memo = map[cellKey]formula.Value{}
			if !ev.matchOne(sn, shadow, q.When) {
				continue
			}
		}
		raw := in.Answers[q.Prop]
		if prop.Type == TTitle {
			t := strings.TrimSpace(str(raw))
			if t == "" && q.Required {
				return "", httpx.Fail(http.StatusBadRequest, "required", q.Label+" is required").With("prop", q.Prop)
			}
			shadow.Title = t
			values[TTitle] = mustJSON(t)
			continue
		}
		check := prop
		if prop.Type == TSelect || prop.Type == TMulti || prop.Type == TStatus {
			c := *prop
			c.Options = append([]Option(nil), prop.Options...)
			check = &c
		}
		norm, err := nc.normalize(check, raw)
		if err != nil {
			return "", err
		}
		if norm == nil {
			if q.Required {
				return "", httpx.Fail(http.StatusBadRequest, "required", q.Label+" is required").With("prop", q.Prop)
			}
			continue
		}
		if nc.changed {
			return "", httpx.Fail(http.StatusBadRequest, "bad_value", q.Label+": choose one of the listed options")
		}
		shadow.Vals[q.Prop] = norm
		values[q.Prop] = norm
	}
	if _, ok := values[TTitle]; !ok {
		values[TTitle] = mustJSON("Response " + time.Now().UTC().Format("2006-01-02 15:04"))
	}
	rec := &Response{FormID: v.ID, Version: v.UpdatedAt, SubmittedAt: store.NowMS()}
	return s.createRow(ctx, p, m, RowInput{Values: values, form: rec}, author)
}
