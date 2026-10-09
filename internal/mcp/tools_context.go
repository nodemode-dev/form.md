package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/contextpack"
)

const skillName = "contextual-compression"

func (s *Server) registerContext() {
	if s.Context == nil {
		return
	}
	pol := s.Context.Policy
	tasks := make([]string, 0, len(pol.Budgets.Tasks))
	for t := range pol.Budgets.Tasks {
		tasks = append(tasks, t)
	}
	sort.Strings(tasks)
	s.add(&Tool{Name: "context_pack", Title: "Pack context for a task", Brief: true,
		Description: "Call this before starting a task: returns the smallest relevant context for it, ranked and cut to a token budget for the task type. Sections in order: the workspace's rules, recorded knowledge, the focus pages, the best matching passages and related pages, each with its source page id. format digest compresses passages, outline keeps only headings, delta returns only what changed since an earlier pack (pass its id as since). Budgets, weights and limits come from the contextual-compression skill's policy.json.",
		Props: map[string]any{"task": str("The task in the person's words"), "task_type": enum("Sets the token budget; default "+pol.Budgets.DefaultTask, tasks...),
			"workspace_id": str("Workspace to pack from (needed when the credential reaches several and no focus page is named)"), "focus": strs("Page ids the task is about, up to 8; included first and in full when they fit"),
			"format": enum("digest (default), outline or delta", pol.Compression.Formats...), "since": str("For delta: an earlier pack id or an RFC 3339 time"),
			"budget": intp(fmt.Sprintf("Token budget overriding the task type's, between %d and %d", pol.Budgets.Min, pol.Budgets.Max)), "agent_key": str("A stable name for this agent, so a delta without since starts from its own last pack")},
		Run: s.contextPack})
	s.add(&Tool{Name: "knowledge_search", Title: "Search knowledge", Brief: true,
		Description: "Facts, decisions, glossary terms, constraints and preferences recorded in the workspace's knowledge bank, best match and highest confidence first. Confidence decays with time since an item was last confirmed.",
		Props:       map[string]any{"query": str("Words to match; omit for the most confident items"), "workspace_id": str("Workspace to search"), "kind": enum("Only this kind", pol.Knowledge.Kinds...), "limit": intp(fmt.Sprintf("Up to 100, default %d", pol.Knowledge.SearchLimit))},
		Run:         s.knowledgeSearch})
	s.add(&Tool{Name: "knowledge_review", Title: "Review knowledge", Brief: true,
		Description: "Hygiene report for the knowledge bank: stale items not confirmed for a long time, weak items whose confidence decayed, likely duplicates, and items with the same subject that may contradict each other. Resolve them with knowledge_confirm.",
		Props:       map[string]any{"workspace_id": str("Workspace to review")}, Run: s.knowledgeReview})
	s.add(&Tool{Name: "knowledge_record", Title: "Record knowledge", Write: true, Idempotent: true, Brief: true,
		Description: "Record one fact, decision, glossary term, constraint or preference that will matter again, with the page (and block) it comes from. Recording the same thing twice confirms the first item instead. Items with the same subject that may disagree are returned as conflicts.",
		Props: map[string]any{"workspace_id": str("Workspace to record in"), "kind": enum("What kind of knowledge", pol.Knowledge.Kinds...), "subject": str(fmt.Sprintf("Short subject it is about, up to %d characters", pol.Knowledge.MaxSubjectChars)),
			"text": str(fmt.Sprintf("The knowledge in one or two sentences, up to %d characters", pol.Knowledge.MaxTextChars)), "source_page": str("Page id it comes from"), "source_block": str("Block id inside source_page")},
		Required: []string{"kind", "text"}, Run: s.knowledgeRecord})
	s.add(&Tool{Name: "knowledge_confirm", Title: "Confirm or correct knowledge", Write: true, Brief: true,
		Description: "Give a verdict on a knowledge item you relied on: confirm raises its confidence and resets its decay, contradict lowers it (retiring it at the floor), supersede retires it in favour of corrected text, retire removes it.",
		Props:       map[string]any{"id": str("Knowledge item id"), "verdict": enum("The verdict", "confirm", "contradict", "supersede", "retire"), "text": str("Corrected text, for supersede")},
		Required:    []string{"id", "verdict"}, Run: s.knowledgeConfirm})
}

func (s *Server) contextPack(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Task        string   `json:"task"`
		TaskType    string   `json:"task_type"`
		WorkspaceID string   `json:"workspace_id"`
		Focus       []string `json:"focus"`
		Format      string   `json:"format"`
		Since       string   `json:"since"`
		Budget      int      `json:"budget"`
		AgentKey    string   `json:"agent_key"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	if in.AgentKey == "" {
		in.AgentKey = x.name
	}
	pk, err := s.Context.Pack(ctx, x.p, contextpack.PackInput{WorkspaceID: in.WorkspaceID, Task: in.Task, TaskType: in.TaskType, Focus: in.Focus, Format: in.Format, Since: in.Since, Budget: in.Budget, AgentKey: in.AgentKey})
	if err != nil {
		return nil, "", err
	}
	return pk, pk.Render(x.frame), nil
}

func (s *Server) knowledgeSearch(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		Query       string `json:"query"`
		WorkspaceID string `json:"workspace_id"`
		Kind        string `json:"kind"`
		Limit       int    `json:"limit"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws, err := s.Context.Workspace(ctx, x.p, in.WorkspaceID)
	if err != nil {
		return nil, "", err
	}
	items, err := s.Context.Find(ctx, x.p, ws, in.Query, in.Kind, in.Limit)
	if err != nil {
		return nil, "", err
	}
	if items == nil {
		items = []*contextpack.Item{}
	}
	summary := fmt.Sprintf("%d knowledge items", len(items))
	if len(items) > 0 {
		summary += ":\n" + x.frame("knowledge", lines(items))
	}
	return map[string]any{"workspace_id": ws, "items": items}, summary, nil
}

func lines(items []*contextpack.Item) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Line() + "\n")
	}
	return strings.TrimSpace(b.String())
}

func pairs(list []contextpack.Pair) string {
	var b strings.Builder
	for _, p := range list {
		fmt.Fprintf(&b, "- similarity %.2f\n  %s\n  %s\n", p.Similarity, p.A.Line(), p.B.Line())
	}
	return strings.TrimSpace(b.String())
}

func (s *Server) knowledgeReview(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws, err := s.Context.Workspace(ctx, x.p, in.WorkspaceID)
	if err != nil {
		return nil, "", err
	}
	r, err := s.Context.Review(ctx, x.p, ws)
	if err != nil {
		return nil, "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d active knowledge items of %d allowed, %d without a source. %d weak, %d stale, %d likely duplicates, %d possible conflicts.",
		r.Active, r.Capacity, r.Unsourced, len(r.Weak), len(r.Stale), len(r.Duplicates), len(r.Conflicts))
	for _, part := range []struct{ title, body string }{
		{"Weak (retire, or confirm if still true)", lines(r.Weak)},
		{"Stale (confirm or retire)", lines(r.Stale)},
		{"Likely duplicates (retire one of each pair)", pairs(r.Duplicates)},
		{"Possible conflicts (contradict or supersede the wrong one)", pairs(r.Conflicts)},
	} {
		if part.body != "" {
			b.WriteString("\n\n" + part.title + ":\n" + x.frame("knowledge", part.body))
		}
	}
	return r, b.String(), nil
}

func (s *Server) knowledgeRecord(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		Kind        string `json:"kind"`
		Subject     string `json:"subject"`
		Text        string `json:"text"`
		SourcePage  string `json:"source_page"`
		SourceBlock string `json:"source_block"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	ws := in.WorkspaceID
	if ws == "" && in.SourcePage != "" {
		d, err := s.Pages.Get(ctx, x.p, in.SourcePage)
		if err != nil {
			return nil, "", err
		}
		ws = d.WorkspaceID
	}
	ws, err := s.Context.Workspace(ctx, x.p, ws)
	if err != nil {
		return nil, "", err
	}
	res, err := s.Context.Record(ctx, x.p, contextpack.RecordInput{WorkspaceID: ws, Kind: in.Kind, Subject: in.Subject, Text: in.Text, SourcePage: in.SourcePage, SourceBlock: in.SourceBlock, Agent: x.name})
	if err != nil {
		return nil, "", err
	}
	undo, what := contextpack.Undo{Delete: res.Item.ID}, "Recorded knowledge: "
	if res.Duplicate {
		undo, what = contextpack.Undo{Restore: res.Prior}, "Confirmed knowledge: "
	}
	if err := x.record(ctx, contextpack.UndoKind, res.Item.SourcePage, what+res.Item.Text, undo); err != nil {
		return nil, "", err
	}
	summary := "Recorded " + res.Item.ID
	if res.Duplicate {
		summary = "Already known as " + res.Item.ID + "; confirmed it instead of recording it twice"
	}
	summary += fmt.Sprintf(" (confidence %.2f)", res.Item.Effective)
	if len(res.Conflicts) > 0 {
		summary += ".\nThese items have the same subject and may disagree; contradict or supersede the wrong one:\n" + x.frame("knowledge", lines(res.Conflicts))
	}
	out := map[string]any{"item": res.Item, "duplicate": res.Duplicate}
	if len(res.Conflicts) > 0 {
		out["conflicts"] = res.Conflicts
	}
	return out, summary, nil
}

func (s *Server) knowledgeConfirm(ctx context.Context, x *exec, raw json.RawMessage) (any, string, error) {
	var in struct {
		ID      string `json:"id"`
		Verdict string `json:"verdict"`
		Text    string `json:"text"`
	}
	if err := decode(raw, &in); err != nil {
		return nil, "", err
	}
	prior, err := s.Context.Get(ctx, x.p, in.ID)
	if err != nil {
		return nil, "", err
	}
	prior = prior.Snapshot()
	it, next, err := s.Context.Confirm(ctx, x.p, in.ID, in.Verdict, in.Text, x.name)
	if err != nil {
		return nil, "", err
	}
	if err := x.record(ctx, contextpack.UndoKind, it.SourcePage, "Knowledge "+in.Verdict+": "+it.Text, contextpack.Undo{Restore: prior}); err != nil {
		return nil, "", err
	}
	if next != nil {
		if err := x.record(ctx, contextpack.UndoKind, next.SourcePage, "Recorded knowledge: "+next.Text, contextpack.Undo{Delete: next.ID}); err != nil {
			return nil, "", err
		}
	}
	out := map[string]any{"item": it}
	var summary string
	switch {
	case next != nil:
		out["replacement"] = next
		summary = fmt.Sprintf("Superseded %s with %s (confidence %.2f)", it.ID, next.ID, next.Effective)
	case it.RetiredAt != 0:
		summary = "Retired " + it.ID
	default:
		summary = fmt.Sprintf("%s now has confidence %.2f after %d confirmations and %d contradictions", it.ID, it.Effective, it.Confirmations, it.Contradictions)
	}
	return out, summary, nil
}

type skillDoc struct {
	name, title, description, file, body string
}

func frontmatter(raw string) (map[string]string, string) {
	meta := map[string]string{}
	if !strings.HasPrefix(raw, "---\n") {
		return meta, raw
	}
	end := strings.Index(raw[4:], "\n---\n")
	if end < 0 {
		return meta, raw
	}
	for _, line := range strings.Split(raw[4:4+end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if ok {
			meta[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"`)
		}
	}
	return meta, strings.TrimLeft(raw[4+end+5:], "\n")
}

func (s *Server) skillDocs() []skillDoc {
	if s.Skill == nil {
		return nil
	}
	var out []skillDoc
	_ = fs.WalkDir(s.Skill, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || (p != "SKILL.md" && path.Base(p) != "SUB-SKILL.md") {
			return nil
		}
		raw, err := fs.ReadFile(s.Skill, p)
		if err != nil {
			return nil
		}
		meta, body := frontmatter(string(raw))
		name := meta["name"]
		if name == "" {
			name = path.Base(path.Dir(p))
		}
		title := strings.ReplaceAll(name, "-", " ")
		title = strings.ToUpper(title[:1]) + title[1:]
		out = append(out, skillDoc{name: name, title: title, description: meta["description"], file: p, body: body})
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].file == "SKILL.md" || (out[j].file != "SKILL.md" && out[i].name < out[j].name)
	})
	return out
}

func (s *Server) listPrompts() []map[string]any {
	out := []map[string]any{}
	for _, d := range s.skillDocs() {
		out = append(out, map[string]any{"name": d.name, "title": d.title, "description": d.description})
	}
	return out
}

func (s *Server) getPrompt(name string) (map[string]any, *rpcError) {
	for _, d := range s.skillDocs() {
		if d.name != name {
			continue
		}
		text := "Follow this method from the " + skillName + " skill. Every number it names is in skill://" + skillName + "/policy.json.\n\n" + d.body
		return map[string]any{"description": d.description, "messages": []map[string]any{{"role": "user", "content": map[string]any{"type": "text", "text": text}}}}, nil
	}
	return nil, &rpcError{Code: codeParams, Message: "Unknown prompt: " + name}
}

func mimeOf(p string) string {
	switch path.Ext(p) {
	case ".json":
		return "application/json"
	case ".md":
		return "text/markdown"
	}
	return "text/plain"
}

func (s *Server) listResources() []map[string]any {
	out := []map[string]any{}
	if s.Skill == nil {
		return out
	}
	_ = fs.WalkDir(s.Skill, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, _ := d.Info()
		r := map[string]any{"uri": "skill://" + skillName + "/" + p, "name": skillName + "/" + p, "mimeType": mimeOf(p)}
		if info != nil {
			r["size"] = info.Size()
		}
		out = append(out, r)
		return nil
	})
	return out
}

func (s *Server) readResource(uri string) (map[string]any, *rpcError) {
	p, ok := strings.CutPrefix(uri, "skill://"+skillName+"/")
	if !ok || s.Skill == nil || !fs.ValidPath(p) {
		return nil, &rpcError{Code: -32002, Message: "Resource not found", Data: map[string]any{"uri": uri}}
	}
	raw, err := fs.ReadFile(s.Skill, p)
	if err != nil {
		return nil, &rpcError{Code: -32002, Message: "Resource not found", Data: map[string]any{"uri": uri}}
	}
	return map[string]any{"contents": []map[string]any{{"uri": uri, "mimeType": mimeOf(p), "text": string(raw)}}}, nil
}
