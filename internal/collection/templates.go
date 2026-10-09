package collection

import (
	"context"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type TemplateRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

func badTemplate(msg string) error { return httpx.Fail(http.StatusBadRequest, "bad_template", msg) }

func (s *Service) templates(ctx context.Context, collectionID string) ([]TemplateRef, string) {
	out := []TemplateRef{}
	rows, err := s.st.R.QueryContext(ctx, `SELECT p.id, p.title, p.icon FROM collection_rows r JOIN pages p ON p.id = r.page_id
		WHERE r.collection_id = ? AND p.is_template = 1 AND p.trashed_at = 0 ORDER BY p.sort_key, p.id`, collectionID)
	if err == nil {
		for rows.Next() {
			var t TemplateRef
			if rows.Scan(&t.ID, &t.Title, &t.Icon) == nil {
				out = append(out, t)
			}
		}
		rows.Close()
	}
	var def string
	_ = s.st.R.QueryRowContext(ctx, "SELECT default_template FROM collections WHERE page_id = ?", collectionID).Scan(&def)
	found := false
	for _, t := range out {
		found = found || t.ID == def
	}
	if !found {
		def = ""
	}
	return out, def
}

func (s *Service) CreateTemplate(ctx context.Context, p *policy.Principal, collectionID, title string) (*TemplateRef, error) {
	m, err := s.writable(ctx, p, collectionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.pages.CanEdit(ctx, p, collectionID); err != nil {
		return nil, err
	}
	if title == "" {
		title = "New template"
	}
	id, err := s.createRow(ctx, p, m, RowInput{Title: title, template: true}, p.UserID)
	if err != nil {
		return nil, err
	}
	s.emit(ctx, collectionID, map[string]any{"what": "views"})
	return &TemplateRef{ID: id, Title: title}, nil
}

func (s *Service) SetDefaultTemplate(ctx context.Context, p *policy.Principal, collectionID, templateID string) error {
	if _, err := s.writable(ctx, p, collectionID); err != nil {
		return err
	}
	if _, err := s.pages.CanEdit(ctx, p, collectionID); err != nil {
		return err
	}
	if templateID != "" {
		list, _ := s.templates(ctx, collectionID)
		ok := false
		for _, t := range list {
			ok = ok || t.ID == templateID
		}
		if !ok {
			return badTemplate("that template is not in this database")
		}
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE collections SET default_template = ? WHERE page_id = ?", templateID, collectionID); err != nil {
		return err
	}
	s.emit(ctx, collectionID, map[string]any{"what": "views"})
	return nil
}

func (s *Service) fromTemplate(ctx context.Context, p *policy.Principal, m *page.Meta, in RowInput) (*RowOut, error) {
	var col string
	var tpl int
	err := s.st.R.QueryRowContext(ctx, `SELECT r.collection_id, p.is_template FROM collection_rows r JOIN pages p ON p.id = r.page_id
		WHERE r.page_id = ? AND p.trashed_at = 0`, in.Template).Scan(&col, &tpl)
	if err != nil || col != m.ID || tpl != 1 {
		return nil, badTemplate("that template is not in this database")
	}
	src, err := s.pages.Load(ctx, in.Template)
	if err != nil {
		return nil, err
	}
	title := in.Title
	if title == "" {
		title = src.Title
	}
	copied, err := s.pages.Duplicate(ctx, p, in.Template, page.DuplicateInput{Title: "Template copy"})
	if err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, nil, p, audit.Entry{Action: "row.create", WorkspaceID: m.WorkspaceID, PageID: copied.ID, Detail: m.ID}); err != nil {
		return nil, err
	}
	patch := RowPatch{Title: &title, Values: in.Values, BeforeID: in.BeforeID, AfterID: in.AfterID, Move: in.BeforeID != "" || in.AfterID != ""}
	return s.UpdateRow(ctx, p, copied.ID, patch)
}
