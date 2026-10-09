package feed

import (
	"context"
	"net/http"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/comment"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxHTML = 60000

type Item struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Icon      string             `json:"icon"`
	CreatedAt int64              `json:"created_at"`
	CreatedBy string             `json:"created_by"`
	Author    string             `json:"author"`
	UpdatedAt int64              `json:"updated_at"`
	HTML      string             `json:"html"`
	Clipped   bool               `json:"clipped,omitempty"`
	Reactions []comment.Reaction `json:"reactions"`
	Comments  int                `json:"comments"`
	Values    map[string]any     `json:"values"`
}

type Input struct {
	ViewID string `json:"view_id"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
	Search string `json:"search,omitempty"`
}

type Service struct {
	st       *store.Store
	auth     *auth.Service
	cols     *collection.Service
	docs     *doc.Service
	comments *comment.Service
}

func New(st *store.Store, a *auth.Service, cols *collection.Service, docs *doc.Service, comments *comment.Service) *Service {
	return &Service{st: st, auth: a, cols: cols, docs: docs, comments: comments}
}

func (s *Service) Feed(ctx context.Context, p *policy.Principal, collectionID string, in Input) ([]Item, int, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	limit = min(limit, 25)
	res, err := s.cols.Query(ctx, p, collectionID, collection.QueryInput{ViewID: in.ViewID, Offset: in.Offset, Limit: limit, Search: in.Search, NoGroups: true})
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(res.Rows))
	for _, r := range res.Rows {
		ids = append(ids, r.ID)
	}
	reacts := s.comments.PageReactions(ctx, p.UserID, ids)
	counts := s.comments.CommentCounts(ctx, ids)
	names := map[string]string{}
	items := make([]Item, 0, len(res.Rows))
	for _, r := range res.Rows {
		it := Item{ID: r.ID, Title: r.Title, Icon: r.Icon, CreatedAt: r.CreatedAt, CreatedBy: r.CreatedBy, UpdatedAt: r.UpdatedAt, Reactions: reacts[r.ID], Comments: counts[r.ID], Values: r.Values}
		if it.Reactions == nil {
			it.Reactions = []comment.Reaction{}
		}
		if n, ok := names[r.CreatedBy]; ok {
			it.Author = n
		} else if r.CreatedBy != "" {
			_ = s.st.R.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", r.CreatedBy).Scan(&it.Author)
			names[r.CreatedBy] = it.Author
		}
		if html, err := s.docs.Render(ctx, p, r.ID); err == nil {
			if len(html) > maxHTML {
				it.Clipped = true
				if cut := strings.LastIndex(html[:maxHTML], "</p>"); cut > 0 {
					html = html[:cut+len("</p>")]
				} else {
					html = ""
				}
			}
			it.HTML = html
		}
		items = append(items, it)
	}
	return items, res.Total, nil
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "feed" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	r.HandleFunc("POST /api/collections/{id}/feed", s.auth.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Input
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		items, total, err := s.Feed(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
		return nil
	}))
	return nil
}
