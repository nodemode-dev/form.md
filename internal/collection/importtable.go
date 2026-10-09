package collection

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type TableInput struct {
	WorkspaceID string
	ParentID    string
	Title       string
	Headers     []string
	Rows        [][]string
	Types       []string
}

func (s *Service) ImportTable(ctx context.Context, p *policy.Principal, in TableInput) (string, []string, error) {
	if len(in.Headers) == 0 {
		return "", nil, httpx.Fail(http.StatusBadRequest, "bad_table", "a table needs at least one column")
	}
	if len(in.Headers) > maxCSVCols {
		in.Headers = in.Headers[:maxCSVCols]
	}
	if len(in.Rows) > maxCSVRows {
		in.Rows = in.Rows[:maxCSVRows]
	}
	headers := make([]string, len(in.Headers))
	seen := map[string]int{}
	for i, h := range in.Headers {
		h = strings.TrimSpace(h)
		if h == "" {
			h = "Column " + strconv.Itoa(i+1)
		}
		key := strings.ToLower(h)
		if n := seen[key]; n > 0 {
			seen[key] = n + 1
			h = h + " " + strconv.Itoa(n+1)
		} else {
			seen[key] = 1
		}
		headers[i] = trimText(h, maxName)
	}
	kinds := make([]string, len(headers))
	for i := range headers {
		if i < len(in.Types) && in.Types[i] != "" && (types[in.Types[i]] || in.Types[i] == "skip") {
			kinds[i] = in.Types[i]
			continue
		}
		col := make([]string, 0, min(len(in.Rows), 2000))
		for j, rec := range in.Rows {
			if j >= 2000 {
				break
			}
			if i < len(rec) {
				col = append(col, rec[i])
			}
		}
		kinds[i] = guessType(col, i == 0)
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "Imported table"
	}
	ids := []string{}
	token := s.stash.put(&stashed{user: p.UserID, headers: headers, rows: in.Rows, at: time.Now(), name: title, ids: &ids})
	id, _, err := s.ImportCSV(ctx, p, ImportInput{Token: token, Types: kinds, Title: title, WorkspaceID: in.WorkspaceID, ParentID: in.ParentID})
	if err != nil {
		s.stash.take(token, p.UserID)
		return "", nil, err
	}
	return id, ids, nil
}

func (s *Service) AddBoardView(ctx context.Context, p *policy.Principal, collectionID, name, propName string) error {
	schema, err := s.SchemaOf(ctx, collectionID)
	if err != nil {
		return err
	}
	prop := schema.ByName(propName)
	if prop == nil || (prop.Type != TSelect && prop.Type != TStatus) {
		return nil
	}
	typ := "board"
	cfg := json.RawMessage(mustJSON(map[string]any{"group": map[string]string{"prop": prop.ID}}))
	_, err = s.CreateView(ctx, p, collectionID, ViewInput{Name: &name, Type: &typ, Config: &cfg})
	return err
}
