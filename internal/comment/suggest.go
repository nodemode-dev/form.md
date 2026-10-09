package comment

import (
	"context"
	"database/sql"
	"net/http"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const MaxSuggestion = 5000

type Suggestion struct {
	ID           string `json:"id"`
	PageID       string `json:"page_id"`
	Start        string `json:"start"`
	End          string `json:"end"`
	Original     string `json:"original"`
	Text         string `json:"text"`
	AuthorID     string `json:"author_id"`
	AuthorName   string `json:"author_name"`
	AuthorColor  string `json:"author_color"`
	AuthorAvatar string `json:"author_avatar"`
	Status       string `json:"status"`
	DecidedBy    string `json:"decided_by"`
	DecidedAt    int64  `json:"decided_at"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

type SuggestInput struct {
	Start    string `json:"start"`
	End      string `json:"end"`
	Original string `json:"original"`
	Text     string `json:"text"`
}

func (in *SuggestInput) clean() error {
	if in.Start == "" || in.End == "" || len(in.Start) > MaxAnchor || len(in.End) > MaxAnchor {
		return httpx.Fail(http.StatusBadRequest, "bad_anchor", "a suggestion needs a start and an end")
	}
	if utf8.RuneCountInString(in.Text) > MaxSuggestion || utf8.RuneCountInString(in.Original) > MaxSuggestion {
		return httpx.Fail(http.StatusBadRequest, "suggestion_too_long", "a suggestion can change up to 5,000 characters")
	}
	if in.Text == "" && in.Original == "" {
		return httpx.Fail(http.StatusBadRequest, "empty_suggestion", "the suggestion changes nothing")
	}
	return nil
}

func (s *Service) suggestions(ctx context.Context, pageID string) ([]Suggestion, error) {
	rows, err := s.st.R.QueryContext(ctx, `SELECT x.id, x.page_id, x.anchor_start, x.anchor_end, x.original, x.replacement, x.author_id, COALESCE(u.name, x.author_name),
			COALESCE(u.color, ''), COALESCE(u.avatar, ''), x.status, x.decided_by, x.decided_at, x.created_at, x.updated_at
		FROM suggestions x LEFT JOIN users u ON u.id = x.author_id
		WHERE x.page_id = ? AND (x.status = 'open' OR x.decided_at > ?) ORDER BY x.created_at`, pageID, store.NowMS()-30*24*3600*1000)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Suggestion{}
	for rows.Next() {
		var x Suggestion
		if err := rows.Scan(&x.ID, &x.PageID, &x.Start, &x.End, &x.Original, &x.Text, &x.AuthorID, &x.AuthorName, &x.AuthorColor, &x.AuthorAvatar, &x.Status, &x.DecidedBy, &x.DecidedAt, &x.CreatedAt, &x.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) Suggest(ctx context.Context, p *policy.Principal, pageID string, in SuggestInput) (*Suggestion, error) {
	m, err := s.commentable(ctx, p, pageID)
	if err != nil {
		return nil, err
	}
	if err := in.clean(); err != nil {
		return nil, err
	}
	now := store.NowMS()
	x := &Suggestion{ID: ids.New(), PageID: pageID, Start: in.Start, End: in.End, Original: in.Original, Text: in.Text, AuthorID: p.UserID, AuthorName: p.Name, Status: "open", CreatedAt: now, UpdatedAt: now}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO suggestions (id, page_id, anchor_start, anchor_end, original, replacement, author_id, author_name, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, x.ID, pageID, in.Start, in.End, in.Original, in.Text, p.UserID, p.Name, now, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "suggestion.create", WorkspaceID: m.WorkspaceID, PageID: pageID, Detail: x.ID})
	})
	if err != nil {
		return nil, err
	}
	s.emit(ctx, m)
	s.notify(ctx, Event{Kind: "suggestion", Page: m, Actor: p, Suggestion: x})
	return x, nil
}

func (s *Service) suggestion(ctx context.Context, id string) (*Suggestion, error) {
	var x Suggestion
	err := s.st.R.QueryRowContext(ctx, "SELECT id, page_id, author_id, status, original, replacement FROM suggestions WHERE id = ?", id).Scan(&x.ID, &x.PageID, &x.AuthorID, &x.Status, &x.Original, &x.Text)
	if err != nil {
		return nil, store.NoRows(err)
	}
	return &x, nil
}

var errDecided = httpx.Fail(http.StatusConflict, "suggestion_decided", "the suggestion was already accepted or rejected")

func (s *Service) Revise(ctx context.Context, p *policy.Principal, id string, in SuggestInput) error {
	x, err := s.suggestion(ctx, id)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, x.PageID)
	if err != nil {
		return err
	}
	if x.AuthorID == "" || x.AuthorID != p.UserID {
		return httpx.Fail(http.StatusForbidden, "not_yours", "only the author can change a suggestion")
	}
	if x.Status != "open" {
		return errDecided
	}
	if err := in.clean(); err != nil {
		return err
	}
	if _, err := s.st.W.ExecContext(ctx, "UPDATE suggestions SET anchor_start = ?, anchor_end = ?, original = ?, replacement = ?, updated_at = ? WHERE id = ? AND status = 'open'",
		in.Start, in.End, in.Original, in.Text, store.NowMS(), id); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, nil, p, audit.Entry{Action: "suggestion.revise", WorkspaceID: m.WorkspaceID, PageID: m.ID, Detail: id})
	s.emit(ctx, m)
	return nil
}

func (s *Service) Decide(ctx context.Context, p *policy.Principal, id string, accept bool) error {
	x, err := s.suggestion(ctx, id)
	if err != nil {
		return err
	}
	m, err := s.commentable(ctx, p, x.PageID)
	if err != nil {
		return err
	}
	own := x.AuthorID != "" && x.AuthorID == p.UserID
	if accept || !own {
		if _, err := s.pages.CanWrite(ctx, p, x.PageID); err != nil {
			return httpx.Fail(http.StatusForbidden, "cannot_decide", "only someone who can edit the page accepts or rejects suggestions")
		}
	}
	status, action := "rejected", "suggestion.reject"
	if accept {
		status, action = "accepted", "suggestion.accept"
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE suggestions SET status = ?, decided_by = ?, decided_at = ?, updated_at = ? WHERE id = ? AND status = 'open'", status, p.UserID, store.NowMS(), store.NowMS(), id)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errDecided
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: action, WorkspaceID: m.WorkspaceID, PageID: x.PageID, Detail: id})
	})
	if err != nil {
		return err
	}
	s.emit(ctx, m)
	x.Status = status
	s.notify(ctx, Event{Kind: "decision", Page: m, Actor: p, Suggestion: x})
	return nil
}
