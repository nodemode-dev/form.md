package contextpack

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/family"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/search"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Deps struct {
	Store  *store.Store
	Gate   *policy.Gate
	Pages  *page.Service
	Docs   *doc.Service
	Search *search.Service
	Policy *Policy
	Log    *slog.Logger

	Family Family
}

type Family interface {
	Suggest(ctx context.Context, task string, minScore, maxTools int) []family.Route
}

type Service struct{ Deps }

func New(d Deps) *Service { return &Service{d} }

const dayMS = 86400000.0

type Item struct {
	ID             string   `json:"id"`
	WorkspaceID    string   `json:"workspace_id"`
	Kind           string   `json:"kind"`
	Subject        string   `json:"subject,omitempty"`
	Text           string   `json:"text"`
	SourcePage     string   `json:"source_page,omitempty"`
	SourceTitle    string   `json:"source_title,omitempty"`
	SourceBlock    string   `json:"source_block,omitempty"`
	Confidence     float64  `json:"confidence"`
	Effective      float64  `json:"effective_confidence"`
	Confirmations  int      `json:"confirmations"`
	Contradictions int      `json:"contradictions"`
	CreatedAt      int64    `json:"created_at"`
	CreatedBy      string   `json:"created_by,omitempty"`
	Agent          string   `json:"agent,omitempty"`
	ConfirmedAt    int64    `json:"confirmed_at"`
	SupersededBy   string   `json:"superseded_by,omitempty"`
	RetiredAt      int64    `json:"retired_at,omitempty"`
	Match          float64  `json:"match,omitempty"`
	bag            bagWords `json:"-"`
}

type bagWords map[string]bool

const itemColumns = "id, workspace_id, kind, subject, text, source_page, source_block, confidence, confirmations, contradictions, created_at, created_by, agent, confirmed_at, superseded_by, retired_at"

func scanItem(row interface{ Scan(...any) error }) (*Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.WorkspaceID, &it.Kind, &it.Subject, &it.Text, &it.SourcePage, &it.SourceBlock, &it.Confidence, &it.Confirmations, &it.Contradictions,
		&it.CreatedAt, &it.CreatedBy, &it.Agent, &it.ConfirmedAt, &it.SupersededBy, &it.RetiredAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Service) effective(it *Item, now int64) float64 {
	half := s.Policy.Knowledge.HalfLifeDays[it.Kind]
	if half <= 0 {
		half = 90
	}
	days := math.Max(0, float64(now-it.ConfirmedAt)/dayMS)
	return it.Confidence * math.Pow(0.5, days/half)
}

func (s *Service) reach(ctx context.Context, p *policy.Principal, workspaceID string, write bool) error {
	a, err := s.Gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return err
	}
	if !a.CanRead() {
		return policy.ErrHidden
	}
	if write && (!a.CanWrite() || !p.CanWrite()) {
		return httpx.Fail(http.StatusForbidden, "forbidden", "this credential cannot change knowledge in this workspace")
	}
	return nil
}

func (s *Service) Workspace(ctx context.Context, p *policy.Principal, want string) (string, error) {
	if want != "" {
		return want, s.reach(ctx, p, want, false)
	}
	list, _, err := s.Gate.Memberships(ctx, p)
	if err != nil {
		return "", err
	}
	var reach []string
	for _, m := range list {
		if p.Reaches(m.WorkspaceID) && policy.AgentAllowed(p, m.AgentAccess) {
			reach = append(reach, m.WorkspaceID)
		}
	}
	if len(reach) != 1 {
		return "", httpx.Fail(http.StatusBadRequest, "workspace_required", "this credential reaches several workspaces; pass workspace_id (see whoami)")
	}
	return reach[0], nil
}

func (s *Service) active(ctx context.Context, p *policy.Principal, workspaceID string) ([]*Item, error) {
	rows, err := s.Store.R.QueryContext(ctx, "SELECT "+itemColumns+" FROM knowledge WHERE workspace_id = ? AND retired_at = 0 AND superseded_by = '' ORDER BY created_at", workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.visible(ctx, p, workspaceID, out)
}

func (s *Service) visible(ctx context.Context, p *policy.Principal, workspaceID string, items []*Item) ([]*Item, error) {
	var sources []string
	for _, it := range items {
		if it.SourcePage != "" {
			sources = append(sources, it.SourcePage)
		}
	}
	if len(sources) == 0 {
		return items, nil
	}
	see, err := s.Pages.Filter(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	titles := s.Pages.Titles(ctx, p, sources)
	now := store.NowMS()
	out := items[:0]
	for _, it := range items {
		if it.SourcePage != "" {
			if !see(it.SourcePage) {
				continue
			}
			it.SourceTitle = titles[it.SourcePage]
		}
		it.Effective = s.effective(it, now)
		out = append(out, it)
	}
	return out, nil
}

type RecordInput struct {
	WorkspaceID string
	Kind        string
	Subject     string
	Text        string
	SourcePage  string
	SourceBlock string
	Agent       string
}

type RecordResult struct {
	Item      *Item   `json:"item"`
	Duplicate bool    `json:"duplicate"`
	Conflicts []*Item `json:"conflicts,omitempty"`
	Prior     *Item   `json:"-"`
}

func (s *Service) Record(ctx context.Context, p *policy.Principal, in RecordInput) (*RecordResult, error) {
	k := s.Policy.Knowledge
	in.Text = strings.TrimSpace(in.Text)
	in.Subject = strings.TrimSpace(in.Subject)
	switch {
	case !slices.Contains(k.Kinds, in.Kind):
		return nil, httpx.Fail(http.StatusBadRequest, "bad_kind", "kind must be one of "+strings.Join(k.Kinds, ", "))
	case in.Text == "":
		return nil, httpx.Fail(http.StatusBadRequest, "empty", "text is empty")
	case utf8.RuneCountInString(in.Text) > k.MaxTextChars:
		return nil, httpx.Fail(http.StatusBadRequest, "too_long", fmt.Sprintf("text can be up to %d characters; record one fact per item", k.MaxTextChars))
	case utf8.RuneCountInString(in.Subject) > k.MaxSubjectChars:
		return nil, httpx.Fail(http.StatusBadRequest, "too_long", fmt.Sprintf("subject can be up to %d characters", k.MaxSubjectChars))
	}
	if err := s.reach(ctx, p, in.WorkspaceID, true); err != nil {
		return nil, err
	}
	if in.SourcePage != "" {
		m, err := s.Pages.Get(ctx, p, in.SourcePage)
		if err != nil {
			return nil, err
		}
		if m.WorkspaceID != in.WorkspaceID {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_source", "source_page belongs to another workspace")
		}
	} else {
		in.SourceBlock = ""
	}
	items, err := s.active(ctx, p, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if len(items) >= k.MaxItems {
		return nil, httpx.Fail(http.StatusConflict, "bank_full", fmt.Sprintf("this workspace holds %d knowledge items; run knowledge_review and retire what is stale first", len(items)))
	}
	res := &RecordResult{}
	for _, it := range items {
		if it.Kind != in.Kind {
			continue
		}
		sim := s.Policy.Similarity(in.Text, it.Text)
		if sim >= s.Policy.Hygiene.DuplicateSimilarity {
			res.Prior = it.Snapshot()
			got, err := s.verdict(ctx, it, "confirm")
			if err != nil {
				return nil, err
			}
			res.Item, res.Duplicate = got, true
			return res, nil
		}
		if sim >= s.Policy.Hygiene.ConflictSimilarity && in.Subject != "" && strings.EqualFold(in.Subject, it.Subject) {
			res.Conflicts = append(res.Conflicts, it)
		}
	}
	now := store.NowMS()
	conf := k.Confidence.AgentInitial
	if in.SourcePage != "" {
		conf = k.Confidence.Initial
	}
	it := &Item{ID: ids.New(), WorkspaceID: in.WorkspaceID, Kind: in.Kind, Subject: in.Subject, Text: in.Text, SourcePage: in.SourcePage, SourceBlock: in.SourceBlock,
		Confidence: conf, CreatedAt: now, CreatedBy: p.UserID, Agent: in.Agent, ConfirmedAt: now}
	_, err = s.Store.W.ExecContext(ctx, "INSERT INTO knowledge ("+itemColumns+") VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, '', 0)",
		it.ID, it.WorkspaceID, it.Kind, it.Subject, it.Text, it.SourcePage, it.SourceBlock, it.Confidence, it.CreatedAt, it.CreatedBy, it.Agent, it.ConfirmedAt)
	if err != nil {
		return nil, err
	}
	if it.SourcePage != "" {
		it.SourceTitle = s.Pages.Titles(ctx, p, []string{it.SourcePage})[it.SourcePage]
	}
	it.Effective = conf
	res.Item = it
	return res, nil
}

func (s *Service) Get(ctx context.Context, p *policy.Principal, id string) (*Item, error) {
	it, err := scanItem(s.Store.R.QueryRowContext(ctx, "SELECT "+itemColumns+" FROM knowledge WHERE id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.reach(ctx, p, it.WorkspaceID, false); err != nil {
		return nil, err
	}
	list, err := s.visible(ctx, p, it.WorkspaceID, []*Item{it})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, store.ErrNotFound
	}
	return it, nil
}

func (s *Service) Confirm(ctx context.Context, p *policy.Principal, id, verdict, text, agent string) (*Item, *Item, error) {
	it, err := s.Get(ctx, p, id)
	if err != nil {
		return nil, nil, err
	}
	if err := s.reach(ctx, p, it.WorkspaceID, true); err != nil {
		return nil, nil, err
	}
	if it.RetiredAt != 0 || it.SupersededBy != "" {
		return nil, nil, httpx.Fail(http.StatusConflict, "retired", "this item was already retired or superseded")
	}
	if verdict != "supersede" {
		got, err := s.verdict(ctx, it, verdict)
		return got, nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, nil, httpx.Fail(http.StatusBadRequest, "empty", "supersede needs the corrected text")
	}
	res, err := s.Record(ctx, p, RecordInput{WorkspaceID: it.WorkspaceID, Kind: it.Kind, Subject: it.Subject, Text: text, SourcePage: it.SourcePage, SourceBlock: it.SourceBlock, Agent: agent})
	if err != nil {
		return nil, nil, err
	}
	if res.Item.ID == it.ID {
		return nil, nil, httpx.Fail(http.StatusBadRequest, "unchanged", "the new text says the same as the old; use confirm instead")
	}
	now := store.NowMS()
	if _, err := s.Store.W.ExecContext(ctx, "UPDATE knowledge SET superseded_by = ?, retired_at = ? WHERE id = ?", res.Item.ID, now, it.ID); err != nil {
		return nil, nil, err
	}
	it.SupersededBy, it.RetiredAt = res.Item.ID, now
	return it, res.Item, nil
}

func (s *Service) verdict(ctx context.Context, it *Item, verdict string) (*Item, error) {
	c := s.Policy.Knowledge.Confidence
	now := store.NowMS()
	switch verdict {
	case "confirm":
		it.Confidence = math.Min(c.Max, it.Confidence+c.ConfirmStep)
		it.Confirmations++
		it.ConfirmedAt = now
	case "contradict":
		it.Confidence = math.Max(c.Floor, it.Confidence-c.ContradictStep)
		it.Contradictions++
		if it.Confidence <= c.Floor {
			it.RetiredAt = now
		}
	case "retire":
		it.RetiredAt = now
	default:
		return nil, httpx.Fail(http.StatusBadRequest, "bad_verdict", "verdict must be confirm, contradict, supersede or retire")
	}
	_, err := s.Store.W.ExecContext(ctx, "UPDATE knowledge SET confidence = ?, confirmations = ?, contradictions = ?, confirmed_at = ?, retired_at = ? WHERE id = ?",
		it.Confidence, it.Confirmations, it.Contradictions, it.ConfirmedAt, it.RetiredAt, it.ID)
	if err != nil {
		return nil, err
	}
	it.Effective = s.effective(it, now)
	return it, nil
}

func (s *Service) Find(ctx context.Context, p *policy.Principal, workspaceID, query, kind string, limit int) ([]*Item, error) {
	if err := s.reach(ctx, p, workspaceID, false); err != nil {
		return nil, err
	}
	if kind != "" && !slices.Contains(s.Policy.Knowledge.Kinds, kind) {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_kind", "kind must be one of "+strings.Join(s.Policy.Knowledge.Kinds, ", "))
	}
	if limit <= 0 {
		limit = s.Policy.Knowledge.SearchLimit
	}
	limit = min(limit, 100)
	items, err := s.active(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	return s.rank(items, query, kind, limit), nil
}

func (s *Service) rank(items []*Item, query, kind string, limit int) []*Item {
	q := s.Policy.bag(query)
	floor := s.Policy.Knowledge.Confidence.Floor
	var out []*Item
	for _, it := range items {
		if (kind != "" && it.Kind != kind) || it.Effective < floor {
			continue
		}
		if len(q) > 0 {
			it.Match = s.Policy.Overlap(q, it.Subject+" "+it.Text)
			if it.Match == 0 {
				continue
			}
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Effective, out[j].Effective
		if len(q) > 0 {
			a, b = a*out[i].Match, b*out[j].Match
		}
		return a > b
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (it *Item) Line() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("- [%s] %s", it.ID, it.Kind))
	if it.Subject != "" {
		b.WriteString(" / " + it.Subject)
	}
	b.WriteString(fmt.Sprintf(" (confidence %.2f", it.Effective))
	if it.SourcePage != "" {
		src := it.SourceTitle
		if src == "" {
			src = "Untitled"
		}
		b.WriteString(", source " + src + " " + it.SourcePage)
		if it.SourceBlock != "" {
			b.WriteString("#" + it.SourceBlock)
		}
	} else {
		b.WriteString(", no source")
	}
	b.WriteString("): " + strings.Join(strings.Fields(it.Text), " "))
	return b.String()
}

func (s *Service) Brief(ctx context.Context, p *policy.Principal, workspaceID, task string) (string, int) {
	items, err := s.active(ctx, p, workspaceID)
	if err != nil || len(items) == 0 {
		return "", 0
	}
	list := s.rank(items, task, "", s.Policy.Knowledge.PackLimit)
	if len(list) == 0 {
		list = s.rank(items, "", "", s.Policy.Knowledge.PackLimit)
	}
	room := s.Policy.Chars(s.Policy.Budgets.Tasks["answer"])
	var b strings.Builder
	for _, it := range list {
		line := it.Line() + "\n"
		if b.Len()+len(line) > room {
			break
		}
		b.WriteString(line)
	}
	return b.String(), len(list)
}
