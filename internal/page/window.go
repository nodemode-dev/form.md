package page

import (
	"sort"
	"strconv"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

func pageWindow(pages []*Meta, limitStr, cursor string) (map[string]any, error) {
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	c, err := httpx.DecodeCursor(cursor)
	if err != nil {
		return nil, err
	}
	sorted := append([]*Meta(nil), pages...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt != sorted[j].CreatedAt {
			return sorted[i].CreatedAt < sorted[j].CreatedAt
		}
		return sorted[i].ID < sorted[j].ID
	})
	start := 0
	if cursor != "" {
		start = sort.Search(len(sorted), func(i int) bool {
			m := sorted[i]
			return m.CreatedAt > c.At || (m.CreatedAt == c.At && m.ID > c.ID)
		})
	}
	end := min(start+limit, len(sorted))
	out := map[string]any{"pages": sorted[start:end], "next_cursor": ""}
	if end < len(sorted) {
		last := sorted[end-1]
		out["next_cursor"] = httpx.EncodeCursor(httpx.Cursor{At: last.CreatedAt, ID: last.ID})
	}
	return out, nil
}
