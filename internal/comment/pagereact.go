package comment

import (
	"context"
	"net/http"
	"strings"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func (s *Service) ReactPage(ctx context.Context, p *policy.Principal, pageID, icon string, on bool) ([]Reaction, error) {
	if !validReaction(icon) {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_reaction", "use one of the reaction icons")
	}
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	if on {
		_, err = s.st.W.ExecContext(ctx, "INSERT INTO page_reactions (page_id, user_id, icon, created_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", pageID, p.UserID, icon, store.NowMS())
	} else {
		_, err = s.st.W.ExecContext(ctx, "DELETE FROM page_reactions WHERE page_id = ? AND user_id = ? AND icon = ?", pageID, p.UserID, icon)
	}
	if err != nil {
		return nil, err
	}
	detail := icon + " added"
	if !on {
		detail = icon + " removed"
	}
	_ = s.audit.Record(ctx, nil, p, audit.Entry{Action: "page.react", WorkspaceID: m.WorkspaceID, PageID: pageID, Detail: detail})
	s.emit(ctx, m)
	return s.PageReactions(ctx, p.UserID, []string{pageID})[pageID], nil
}

func (s *Service) PageReactions(ctx context.Context, userID string, pageIDs []string) map[string][]Reaction {
	out := map[string][]Reaction{}
	if len(pageIDs) == 0 {
		return out
	}
	args := make([]any, len(pageIDs))
	for i, id := range pageIDs {
		args[i] = id
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT r.page_id, r.icon, r.user_id, COALESCE(u.name, '') FROM page_reactions r
		LEFT JOIN users u ON u.id = r.user_id WHERE r.page_id IN (`+placeholders(len(pageIDs))+`) ORDER BY r.created_at`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var pid, icon, user, name string
		if rows.Scan(&pid, &icon, &user, &name) != nil {
			continue
		}
		list := out[pid]
		k := -1
		for i := range list {
			if list[i].Icon == icon {
				k = i
			}
		}
		if k < 0 {
			list = append(list, Reaction{Icon: icon, Names: []string{}})
			k = len(list) - 1
		}
		list[k].Count++
		list[k].Mine = list[k].Mine || user == userID
		if len(list[k].Names) < 10 {
			list[k].Names = append(list[k].Names, name)
		}
		out[pid] = list
	}
	return out
}

func (s *Service) CommentCounts(ctx context.Context, pageIDs []string) map[string]int {
	out := map[string]int{}
	if len(pageIDs) == 0 {
		return out
	}
	args := make([]any, len(pageIDs))
	for i, id := range pageIDs {
		args[i] = id
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT t.page_id, COUNT(c.id) FROM comment_threads t JOIN comments c ON c.thread_id = t.id
		WHERE t.page_id IN (`+placeholders(len(pageIDs))+`) GROUP BY t.page_id`, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var pid string
		var n int
		if rows.Scan(&pid, &n) == nil {
			out[pid] = n
		}
	}
	return out
}
