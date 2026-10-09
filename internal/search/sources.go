package search

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type Source struct {
	PageID  string       `json:"page_id"`
	Title   string       `json:"title"`
	Heading string       `json:"heading"`
	Text    string       `json:"text"`
	Path    []page.Crumb `json:"path"`
}

var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`a an and are as at be been but by can could did do does doing for from had has have how i if in into is it its me my of on or our please should so tell than that the their them then there these they this those to us was we were what when where which who whom why will with would you your about any all also am get got know find show give list summarize summary explain`) {
		stopWords[w] = true
	}
}

func Keywords(q string) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range strings.FieldsFunc(strings.ToLower(q), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if seen[f] || stopWords[f] || (utf8.RuneCountInString(f) < 3 && !unicode.IsDigit([]rune(f)[0])) {
			continue
		}
		seen[f] = true
		out = append(out, f)
		if len(out) == 10 {
			break
		}
	}
	return out
}

func (s *Service) Sources(ctx context.Context, p *policy.Principal, workspaceID, question string, limit int) ([]Source, error) {
	visible, err := s.pages.Filter(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	kws := Keywords(question)
	if len(kws) == 0 {
		return []Source{}, nil
	}
	terms := make([]string, len(kws))
	for i, k := range kws {
		terms[i] = `"` + k + `"*`
	}
	templates, err := s.templates(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	match := `{ws} : "` + workspaceID + `" AND {title heading body} : (` + strings.Join(terms, " OR ") + `)`
	rows, err := s.st.R.QueryContext(ctx, `SELECT sp.page_id, sp.heading, p.title, COALESCE(p.parent_id, ''), f.body
		FROM (SELECT rowid, rank, body FROM search_fts WHERE search_fts MATCH ? ORDER BY rank LIMIT 300) f
		JOIN search_parts sp ON sp.id = f.rowid JOIN pages p ON p.id = sp.page_id
		WHERE p.trashed_at = 0 AND p.workspace_id = ? ORDER BY f.rank`, match, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	perPage := map[string]int{}
	parents := map[string]string{}
	for rows.Next() && len(out) < limit {
		var src Source
		var parent string
		if err := rows.Scan(&src.PageID, &src.Heading, &src.Title, &parent, &src.Text); err != nil {
			return nil, err
		}
		if !visible(src.PageID) || templates[src.PageID] || perPage[src.PageID] >= 2 || strings.TrimSpace(src.Text) == "" {
			continue
		}
		perPage[src.PageID]++
		if utf8.RuneCountInString(src.Text) > 1500 {
			src.Text = string([]rune(src.Text)[:1500])
		}
		parents[src.PageID] = parent
		out = append(out, src)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range out {
		out[i].Path = s.path(ctx, parents[out[i].PageID])
	}
	return out, nil
}
