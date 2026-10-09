package search

import (
	"context"
	"strings"
	"unicode"

	"github.com/nodemode-ai/form.md/internal/policy"
)

type Passage struct {
	PageID    string
	Title     string
	Heading   string
	Kind      string
	Text      string
	Rank      float64
	UpdatedAt int64
	ParentID  string
}

func anyTerms(words []string) string {
	var terms []string
	seen := map[string]bool{}
	for _, w := range words {
		w = strings.ToLower(strings.ReplaceAll(w, `"`, ""))
		if w == "" || seen[w] || strings.IndexFunc(w, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0 {
			continue
		}
		seen[w] = true
		terms = append(terms, `"`+w+`"`)
		if len(terms) == 24 {
			break
		}
	}
	return strings.Join(terms, " OR ")
}

func (s *Service) Passages(ctx context.Context, p *policy.Principal, workspaceID string, words []string, limit int) ([]Passage, error) {
	expr := anyTerms(words)
	if expr == "" || limit <= 0 {
		return nil, nil
	}
	visible, err := s.pages.Filter(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	templates, err := s.templates(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT f.rank, f.heading, f.body, sp.kind, sp.page_id, p.title, p.updated_at, COALESCE(p.parent_id, '')
		FROM (SELECT rowid, rank, heading, body FROM search_fts WHERE search_fts MATCH ? ORDER BY rank LIMIT ?) f
		JOIN search_parts sp ON sp.id = f.rowid JOIN pages p ON p.id = sp.page_id
		WHERE p.trashed_at = 0 AND p.workspace_id = ?
		ORDER BY f.rank`, `{ws} : "`+workspaceID+`" AND {title heading body} : (`+expr+`)`, limit*4, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Passage
	for rows.Next() {
		var ps Passage
		if err := rows.Scan(&ps.Rank, &ps.Heading, &ps.Text, &ps.Kind, &ps.PageID, &ps.Title, &ps.UpdatedAt, &ps.ParentID); err != nil {
			return nil, err
		}
		if !visible(ps.PageID) || templates[ps.PageID] {
			continue
		}
		out = append(out, ps)
		if len(out) == limit {
			break
		}
	}
	return out, rows.Err()
}
