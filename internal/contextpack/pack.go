package contextpack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/search"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxFocus = 8

type PackInput struct {
	WorkspaceID string
	Task        string
	TaskType    string
	Focus       []string
	Format      string
	Since       string
	Budget      int
	AgentKey    string
}

type Source struct {
	PageID  string   `json:"page_id"`
	Title   string   `json:"title"`
	Heading string   `json:"heading,omitempty"`
	Section string   `json:"section"`
	Score   float64  `json:"score,omitempty"`
	Signals []string `json:"signals,omitempty"`
}

type Part struct {
	Kind    string `json:"kind"`
	Budget  int    `json:"budget_tokens"`
	Used    int    `json:"used_tokens"`
	Dropped int    `json:"dropped,omitempty"`
	Text    string `json:"text"`
}

type Pack struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	Task        string   `json:"task"`
	TaskType    string   `json:"task_type"`
	Format      string   `json:"format"`
	Budget      int      `json:"budget_tokens"`
	Used        int      `json:"used_tokens"`
	Since       int64    `json:"since,omitempty"`
	Notes       []string `json:"notes,omitempty"`
	Sections    []Part   `json:"sections"`
	Sources     []Source `json:"sources"`
	Knowledge   []string `json:"knowledge_ids"`

	Family []FamilyHint `json:"family,omitempty"`
}

type FamilyHint struct {
	Product string   `json:"product"`
	Purpose string   `json:"purpose"`
	Running bool     `json:"running"`
	MCP     string   `json:"mcp,omitempty"`
	Connect string   `json:"connect,omitempty"`
	Score   int      `json:"score"`
	Tools   []string `json:"tools"`
}

func (h FamilyHint) line() string {
	s := "- " + h.Product + ": " + h.Purpose
	if h.Running {
		s += " Running, MCP at " + h.MCP + "."
	} else {
		s += " Not running now."
	}
	if h.Connect != "" {
		s += " Connect: " + h.Connect + "."
	}
	if len(h.Tools) > 0 {
		s += " Matching tools: " + strings.Join(h.Tools, ", ") + "."
	}
	return s
}

type packing struct {
	s        *Service
	p        *policy.Principal
	in       PackInput
	pack     *Pack
	query    map[string]bool
	words    []string
	see      func(string) bool
	focus    []*page.Detail
	focusSet map[string]bool
	full     map[string]bool
	linked   map[string]bool
	used     map[string]bool
	items    []*Item
	known    map[string]bool
	cutoff   int64
}

func (s *Service) Pack(ctx context.Context, p *policy.Principal, in PackInput) (*Pack, error) {
	pol := s.Policy
	in.Task = strings.TrimSpace(in.Task)
	if in.Task == "" && len(in.Focus) == 0 {
		return nil, httpx.Fail(http.StatusBadRequest, "empty", "describe the task, or name a focus page")
	}
	if len(in.Focus) > maxFocus {
		return nil, httpx.Fail(http.StatusBadRequest, "too_many", fmt.Sprintf("name up to %d focus pages", maxFocus))
	}
	if in.Format == "" && in.Since != "" {
		in.Format = "delta"
	}
	if in.Format == "" {
		in.Format = pol.Compression.DefaultFormat
	}
	if !slices.Contains(pol.Compression.Formats, in.Format) {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_format", "format must be one of "+strings.Join(pol.Compression.Formats, ", "))
	}
	if in.WorkspaceID == "" && len(in.Focus) > 0 {
		d, err := s.Pages.Get(ctx, p, in.Focus[0])
		if err != nil {
			return nil, err
		}
		in.WorkspaceID = d.WorkspaceID
	}
	ws, err := s.Workspace(ctx, p, in.WorkspaceID)
	if err != nil {
		return nil, err
	}
	in.WorkspaceID = ws
	task, budget := pol.Budget(in.TaskType, in.Budget)
	x := &packing{s: s, p: p, in: in, focusSet: map[string]bool{}, full: map[string]bool{}, linked: map[string]bool{}, used: map[string]bool{}, known: map[string]bool{}}
	x.pack = &Pack{ID: ids.New(), WorkspaceID: ws, Task: in.Task, TaskType: task, Format: in.Format, Budget: budget, Sections: []Part{}, Sources: []Source{}, Knowledge: []string{}}
	x.words = pol.Words(in.Task)
	x.query = pol.bag(in.Task)
	if x.see, err = s.Pages.Filter(ctx, p, ws); err != nil {
		return nil, err
	}
	for _, id := range in.Focus {
		d, err := s.Pages.Get(ctx, p, id)
		if err != nil {
			return nil, err
		}
		if d.WorkspaceID != ws {
			return nil, httpx.Fail(http.StatusBadRequest, "bad_focus", "focus page "+id+" belongs to another workspace")
		}
		if !x.focusSet[d.ID] {
			x.focus = append(x.focus, d)
			x.focusSet[d.ID] = true
		}
	}
	if err := x.since(ctx); err != nil {
		return nil, err
	}
	if err := x.links(ctx); err != nil {
		return nil, err
	}
	if items, err := s.active(ctx, p, ws); err == nil {
		x.items = s.rank(items, in.Task, "", pol.Knowledge.PackLimit)
		for _, it := range x.items {
			if it.SourcePage != "" {
				x.known[it.SourcePage] = true
			}
		}
	} else {
		return nil, err
	}
	total := pol.Chars(budget)
	spent, carry := 0, 0
	for _, sec := range pol.Sections {
		base := int(sec.Share*float64(total)) + carry
		room := min(base, total-spent)
		if sec.Required {
			room = total - spent
		}
		var text string
		dropped := 0
		switch sec.Kind {
		case "rules":
			text, err = x.rules(ctx)
		case "knowledge":
			text, dropped = x.knowledge(room)
		case "focus":
			text, err = x.focusText(ctx, room)
		case "passages":
			text, dropped, err = x.passages(ctx, room)
		case "links":
			text, dropped, err = x.related(ctx, room)
		}
		if err != nil {
			return nil, err
		}
		spent += len(text)
		carry = base - len(text)
		x.pack.Sections = append(x.pack.Sections, Part{Kind: sec.Kind, Budget: pol.Tokens(max(base, 0)), Used: pol.Tokens(len(text)), Dropped: dropped, Text: text})
	}
	if s.Family != nil && in.Task != "" {
		for _, r := range s.Family.Suggest(ctx, in.Task, pol.Family.PackMinScore, pol.Family.PackMaxTools) {
			h := FamilyHint{Product: r.Product, Purpose: r.Purpose, Running: r.Reachable, MCP: r.MCP, Connect: r.Connect, Score: r.Score, Tools: []string{}}
			for _, t := range r.Tools {
				h.Tools = append(h.Tools, t.Name)
			}
			x.pack.Family = append(x.pack.Family, h)
			spent += len(h.line())
		}
	}
	x.pack.Used = pol.Tokens(spent)
	if err := x.record(ctx); err != nil {
		return nil, err
	}
	return x.pack, nil
}

func (x *packing) since(ctx context.Context) error {
	pol := x.s.Policy.Compression.Delta
	now := store.NowMS()
	var at int64
	switch {
	case x.in.Since != "":
		if t, err := time.Parse(time.RFC3339, x.in.Since); err == nil {
			at = t.UnixMilli()
		} else {
			err := x.s.Store.R.QueryRowContext(ctx, "SELECT created_at FROM context_packs WHERE id = ? AND user_id = ? AND workspace_id = ?", x.in.Since, x.p.UserID, x.in.WorkspaceID).Scan(&at)
			if errors.Is(err, sql.ErrNoRows) {
				return httpx.Fail(http.StatusBadRequest, "bad_since", "since must be the id of an earlier pack of yours in this workspace, or an RFC 3339 time")
			}
			if err != nil {
				return err
			}
		}
	case x.in.Format != "delta":
		return nil
	case x.in.AgentKey != "":
		err := x.s.Store.R.QueryRowContext(ctx, "SELECT created_at FROM context_packs WHERE user_id = ? AND agent_key = ? AND workspace_id = ? ORDER BY created_at DESC LIMIT 1", x.p.UserID, x.in.AgentKey, x.in.WorkspaceID).Scan(&at)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if at == 0 {
		at = now - int64(pol.DefaultHours)*3600000
		x.pack.Notes = append(x.pack.Notes, fmt.Sprintf("no earlier pack to compare with; showing changes from the last %d hours", pol.DefaultHours))
	}
	if oldest := now - int64(pol.MaxHours)*3600000; at < oldest {
		at = oldest
		x.pack.Notes = append(x.pack.Notes, fmt.Sprintf("the earlier pack is older than %d hours; only changes since then are shown, and a full pack may serve better", pol.MaxHours))
	}
	x.cutoff, x.pack.Since = at, at
	return nil
}

func (x *packing) links(ctx context.Context) error {
	if len(x.focus) == 0 {
		return nil
	}
	marks := strings.TrimSuffix(strings.Repeat("?, ", len(x.focus)), ", ")
	args := make([]any, 0, 2*len(x.focus))
	for _, d := range x.focus {
		args = append(args, d.ID)
	}
	args = append(args, args...)
	rows, err := x.s.Store.R.QueryContext(ctx, "SELECT target FROM doc_refs WHERE kind = 'page' AND page_id IN ("+marks+") UNION SELECT page_id FROM doc_refs WHERE kind = 'page' AND target IN ("+marks+")", args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if !x.focusSet[id] && x.see(id) {
			x.linked[id] = true
		}
	}
	return rows.Err()
}

func (x *packing) source(src Source) {
	x.pack.Sources = append(x.pack.Sources, src)
	x.used[src.PageID] = true
}

func (x *packing) rules(ctx context.Context) (string, error) {
	var rules string
	if err := x.s.Store.R.QueryRowContext(ctx, "SELECT rules FROM workspaces WHERE id = ?", x.in.WorkspaceID).Scan(&rules); err != nil {
		return "", store.NoRows(err)
	}
	return strings.TrimSpace(rules), nil
}

func (x *packing) knowledge(room int) (string, int) {
	var b strings.Builder
	for i, it := range x.items {
		line := it.Line() + "\n"
		if b.Len()+len(line) > room {
			return strings.TrimSpace(b.String()), len(x.items) - i
		}
		b.WriteString(line)
		x.pack.Knowledge = append(x.pack.Knowledge, it.ID)
	}
	return strings.TrimSpace(b.String()), 0
}

func (x *packing) focusText(ctx context.Context, room int) (string, error) {
	var b strings.Builder
	for i, d := range x.focus {
		share := (room - b.Len()) / (len(x.focus) - i)
		head := fmt.Sprintf("### %s (%s)\n", titleOf(d.Title), d.ID)
		if x.cutoff > 0 && d.UpdatedAt <= x.cutoff {
			b.WriteString(head + "Unchanged since the earlier pack.\n\n")
			x.source(Source{PageID: d.ID, Title: d.Title, Section: "focus", Signals: []string{"focus", "unchanged"}})
			continue
		}
		var body string
		if d.Kind == "collection" {
			body = "A database. " + strings.TrimSpace(d.Description) + " Read its rows with query_database rather than from this pack."
		} else {
			md, err := x.s.Docs.Markdown(ctx, x.p, d.ID)
			if err != nil {
				return "", err
			}
			var mode string
			body, mode = x.s.Policy.compress(md, share-len(head), x.in.Format)
			if mode == "full" {
				x.full[d.ID] = true
			}
		}
		b.WriteString(head + strings.TrimSpace(body) + "\n\n")
		x.source(Source{PageID: d.ID, Title: d.Title, Section: "focus", Signals: []string{"focus"}})
	}
	return strings.TrimSpace(b.String()), nil
}

func (p *Policy) compress(md string, room int, format string) (string, string) {
	md = strings.TrimSpace(md)
	if format != "outline" && len(md) <= room {
		return md, "full"
	}
	if format != "outline" {
		var b strings.Builder
		for _, sec := range splitSections(md) {
			if sec.heading != "" {
				b.WriteString(sec.heading + "\n")
			}
			if d := p.Digest(sec.body); d != "" {
				b.WriteString(d + "\n")
			}
		}
		if out := strings.TrimSpace(b.String()); len(out) <= room {
			return out, "digest"
		}
	}
	out, omitted := p.Outline(md)
	if out == "" {
		out = p.Digest(md)
	}
	if omitted > 0 {
		out += fmt.Sprintf("(%d more headings)\n", omitted)
	}
	out, _ = clip(out, max(room, 0))
	return out, "outline"
}

type mdSection struct{ heading, body string }

func splitSections(md string) []mdSection {
	var out []mdSection
	cur := mdSection{}
	var body []string
	inCode := false
	flush := func() {
		cur.body = strings.Join(body, " ")
		if cur.heading != "" || strings.TrimSpace(cur.body) != "" {
			out = append(out, cur)
		}
		body = body[:0]
	}
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if !inCode && strings.HasPrefix(line, "#") {
			flush()
			cur = mdSection{heading: t}
			continue
		}
		if !inCode && t != "" {
			body = append(body, strings.TrimLeft(t, "-*>0123456789. []x"))
		}
	}
	flush()
	return out
}

type candidate struct {
	search.Passage
	score   float64
	signals []string
}

func (x *packing) passages(ctx context.Context, room int) (string, int, error) {
	pol := x.s.Policy
	r := pol.Ranking
	list, err := x.s.Search.Passages(ctx, x.p, x.in.WorkspaceID, x.words, r.MaxCandidates)
	if err != nil {
		return "", 0, err
	}
	best := 0.0
	for _, c := range list {
		best = math.Min(best, c.Rank)
	}
	w := r.Weights
	sum := w.Text + w.Title + w.Link + w.Recency + w.Knowledge + w.Focus
	if sum == 0 {
		sum = 1
	}
	now := store.NowMS()
	var cands []candidate
	dropped := 0
	for _, ps := range list {
		if x.full[ps.PageID] || (x.cutoff > 0 && ps.UpdatedAt <= x.cutoff) {
			continue
		}
		body := ps.Text
		if body == "" {
			body = ps.Title
		}
		rank := 0.0
		if best < 0 {
			rank = ps.Rank / best
		}
		sig := map[string]float64{
			"text":      0.5*rank + 0.5*pol.Overlap(x.query, ps.Heading+" "+body),
			"title":     pol.Overlap(x.query, ps.Title),
			"link":      flag(x.linked[ps.PageID]),
			"recency":   math.Pow(0.5, math.Max(0, float64(now-ps.UpdatedAt)/dayMS)/r.RecencyHalfLifeDays),
			"knowledge": flag(x.known[ps.PageID]),
			"focus":     flag(x.focusSet[ps.PageID] || x.focusSet[ps.ParentID]),
		}
		score := (w.Text*sig["text"] + w.Title*sig["title"] + w.Link*sig["link"] + w.Recency*sig["recency"] + w.Knowledge*sig["knowledge"] + w.Focus*sig["focus"]) / sum
		if score < r.MinScore {
			dropped++
			continue
		}
		var names []string
		for _, k := range []string{"text", "title", "link", "knowledge", "focus", "recency"} {
			if sig[k] >= 0.25 {
				names = append(names, k)
			}
		}
		cands = append(cands, candidate{ps, score, names})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	perPage := map[string]int{}
	var b strings.Builder
	for i, c := range cands {
		if perPage[c.PageID] >= r.PerPage {
			dropped++
			continue
		}
		head := fmt.Sprintf("- %s", titleOf(c.Title))
		if c.Heading != "" {
			head += " / " + c.Heading
		}
		head += fmt.Sprintf(" (%s, score %.2f: %s)", c.PageID, c.score, strings.Join(c.signals, ", "))
		entry := head + "\n"
		if x.in.Format != "outline" && c.Kind != "title" && c.Text != "" {
			entry += "  " + pol.Digest(c.Text) + "\n"
		}
		if b.Len()+len(entry) > room {
			dropped += len(cands) - i
			break
		}
		b.WriteString(entry)
		perPage[c.PageID]++
		x.source(Source{PageID: c.PageID, Title: c.Title, Heading: c.Heading, Section: "passages", Score: math.Round(c.score*1000) / 1000, Signals: c.signals})
	}
	return strings.TrimSpace(b.String()), dropped, nil
}

func flag(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func titleOf(t string) string {
	if strings.TrimSpace(t) == "" {
		return "Untitled"
	}
	return t
}

type related struct {
	id, why string
	at      int64
}

func (x *packing) related(ctx context.Context, room int) (string, int, error) {
	var list []related
	seen := map[string]bool{}
	add := func(id, why string, at int64) {
		if id == "" || seen[id] || x.used[id] || x.focusSet[id] || !x.see(id) {
			return
		}
		seen[id] = true
		list = append(list, related{id, why, at})
	}
	if x.cutoff > 0 {
		rows, err := x.s.Store.R.QueryContext(ctx, "SELECT id, updated_at FROM pages WHERE workspace_id = ? AND trashed_at = 0 AND is_template = 0 AND updated_at > ? ORDER BY updated_at DESC LIMIT 200", x.in.WorkspaceID, x.cutoff)
		if err != nil {
			return "", 0, err
		}
		for rows.Next() {
			var id string
			var at int64
			if err := rows.Scan(&id, &at); err != nil {
				rows.Close()
				return "", 0, err
			}
			add(id, "changed "+time.UnixMilli(at).UTC().Format("2006-01-02 15:04")+" UTC", at)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return "", 0, err
		}
	}
	linked := make([]string, 0, len(x.linked))
	for id := range x.linked {
		linked = append(linked, id)
	}
	sort.Strings(linked)
	for _, id := range linked {
		add(id, "linked with a focus page", 0)
	}
	for _, it := range x.items {
		if slices.Contains(x.pack.Knowledge, it.ID) {
			add(it.SourcePage, "source of knowledge "+it.ID, 0)
		}
	}
	if len(list) == 0 {
		return "", 0, nil
	}
	want := make([]string, 0, len(list))
	for _, r := range list {
		want = append(want, r.id)
	}
	titles := x.s.Pages.Titles(ctx, x.p, want)
	var b strings.Builder
	dropped := 0
	for i, r := range list {
		t, ok := titles[r.id]
		if !ok {
			continue
		}
		line := fmt.Sprintf("- %s (%s): %s\n", t, r.id, r.why)
		if b.Len()+len(line) > room {
			dropped = len(list) - i
			break
		}
		b.WriteString(line)
		x.source(Source{PageID: r.id, Title: t, Section: "links", Signals: []string{r.why}})
	}
	return strings.TrimSpace(b.String()), dropped, nil
}

func (x *packing) record(ctx context.Context) error {
	ids := make([]string, 0, len(x.pack.Sources))
	seen := map[string]bool{}
	for _, s := range x.pack.Sources {
		if !seen[s.PageID] {
			seen[s.PageID] = true
			ids = append(ids, s.PageID)
		}
	}
	raw, _ := json.Marshal(ids)
	task, _ := clip(x.in.Task, 500)
	_, err := x.s.Store.W.ExecContext(ctx, "INSERT INTO context_packs (id, workspace_id, user_id, agent_key, task, task_type, created_at, sources) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		x.pack.ID, x.pack.WorkspaceID, x.p.UserID, x.in.AgentKey, task, x.pack.TaskType, store.NowMS(), string(raw))
	return err
}

var sectionTitles = map[string]string{
	"rules":     "Workspace rules (written by the workspace's admins; follow them)",
	"knowledge": "Knowledge (recorded by people and agents; confirm or contradict what you rely on)",
	"focus":     "Focus pages",
	"passages":  "Relevant passages",
	"links":     "Related pages",
}

func (pk *Pack) Render(frame func(label, text string) string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Context pack %s (task type %s, %s format): %d of %d tokens, %d sources.\n", pk.ID, pk.TaskType, pk.Format, pk.Used, pk.Budget, len(pk.Sources))
	if pk.Since > 0 {
		fmt.Fprintf(&b, "Changes since %s UTC.\n", time.UnixMilli(pk.Since).UTC().Format("2006-01-02 15:04"))
	}
	for _, n := range pk.Notes {
		b.WriteString("Note: " + n + ".\n")
	}
	dropped := 0
	for _, sec := range pk.Sections {
		dropped += sec.Dropped
		if sec.Text == "" {
			continue
		}
		b.WriteString("\n## " + sectionTitles[sec.Kind] + "\n")
		b.WriteString(frame(sec.Kind, sec.Text) + "\n")
	}
	if len(pk.Family) > 0 {
		lines := make([]string, len(pk.Family))
		for i, h := range pk.Family {
			lines[i] = h.line()
		}
		b.WriteString("\n## Form family (another app on this computer fits this task better than form.md; family_route has the detail)\n")
		b.WriteString(frame("family", strings.Join(lines, "\n")) + "\n")
	}
	if dropped > 0 {
		fmt.Fprintf(&b, "\n%d candidates were left out by the budget or the score floor; ask for a larger budget or a narrower task to see them.\n", dropped)
	}
	fmt.Fprintf(&b, "\nPass since=%s with format delta to get only what changed after this pack.", pk.ID)
	return b.String()
}
