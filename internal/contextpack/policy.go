package contextpack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"slices"
)

type Section struct {
	Kind     string  `json:"kind"`
	Share    float64 `json:"share"`
	Required bool    `json:"required"`
}

type Policy struct {
	Version int `json:"version"`
	Token   struct {
		CharsPerToken float64 `json:"chars_per_token"`
	} `json:"tokens"`
	Budgets struct {
		DefaultTask string         `json:"default_task"`
		Min         int            `json:"min"`
		Max         int            `json:"max"`
		Tasks       map[string]int `json:"tasks"`
	} `json:"budgets"`
	Sections []Section `json:"sections"`
	Ranking  struct {
		Weights struct {
			Text      float64 `json:"text"`
			Title     float64 `json:"title"`
			Link      float64 `json:"link"`
			Recency   float64 `json:"recency"`
			Knowledge float64 `json:"knowledge"`
			Focus     float64 `json:"focus"`
		} `json:"weights"`
		RecencyHalfLifeDays float64  `json:"recency_half_life_days"`
		MaxCandidates       int      `json:"max_candidates"`
		MinScore            float64  `json:"min_score"`
		PerPage             int      `json:"per_page"`
		Stopwords           []string `json:"stopwords"`
	} `json:"ranking"`
	Knowledge struct {
		Kinds           []string `json:"kinds"`
		MaxTextChars    int      `json:"max_text_chars"`
		MaxSubjectChars int      `json:"max_subject_chars"`
		MaxItems        int      `json:"max_items_per_workspace"`
		Confidence      struct {
			Initial        float64 `json:"initial"`
			AgentInitial   float64 `json:"agent_initial"`
			ConfirmStep    float64 `json:"confirm_step"`
			ContradictStep float64 `json:"contradict_step"`
			Max            float64 `json:"max"`
			Floor          float64 `json:"floor"`
		} `json:"confidence"`
		HalfLifeDays map[string]float64 `json:"half_life_days"`
		PackLimit    int                `json:"pack_limit"`
		SearchLimit  int                `json:"search_limit"`
	} `json:"knowledge"`
	Compression struct {
		Formats       []string `json:"formats"`
		DefaultFormat string   `json:"default_format"`
		Digest        struct {
			Sentences int `json:"sentences_per_passage"`
			MaxChars  int `json:"max_passage_chars"`
		} `json:"digest"`
		Outline struct {
			MaxDepth    int `json:"max_depth"`
			MaxHeadings int `json:"max_headings_per_page"`
		} `json:"outline"`
		Delta struct {
			DefaultHours int `json:"default_hours"`
			MaxHours     int `json:"max_hours"`
		} `json:"delta"`
	} `json:"compression"`
	Handoff struct {
		Fields       []string `json:"fields"`
		MaxOpenItems int      `json:"max_open_items"`
		MaxChars     int      `json:"max_chars"`
	} `json:"handoff"`
	Hygiene struct {
		DuplicateSimilarity float64 `json:"duplicate_similarity"`
		ConflictSimilarity  float64 `json:"conflict_similarity"`
		StaleAfterDays      float64 `json:"stale_after_days"`
		PruneBelow          float64 `json:"prune_below"`
		ReviewBatch         int     `json:"review_batch"`
	} `json:"hygiene"`
	Family struct {
		VocabularyWeight int `json:"vocabulary_weight"`
		ToolWeight       int `json:"tool_weight"`
		MaxTools         int `json:"max_tools"`
		CacheSeconds     int `json:"cache_seconds"`
		TimeoutMS        int `json:"timeout_ms"`
		PackMinScore     int `json:"pack_min_score"`
		PackMaxTools     int `json:"pack_max_tools"`
	} `json:"family"`
	stop map[string]bool
}

func LoadPolicy(skill fs.FS) (*Policy, error) {
	raw, err := fs.ReadFile(skill, "policy.json")
	if err != nil {
		return nil, fmt.Errorf("contextual-compression policy: %w", err)
	}
	return ParsePolicy(raw)
}

func ParsePolicy(raw []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("contextual-compression policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("contextual-compression policy: %w", err)
	}
	p.stop = map[string]bool{}
	for _, w := range p.Ranking.Stopwords {
		p.stop[w] = true
	}
	return &p, nil
}

func (p *Policy) Validate() error {
	var errs []error
	bad := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	if p.Version != 1 {
		bad("version must be 1")
	}
	if p.Token.CharsPerToken <= 0 {
		bad("tokens.chars_per_token must be above 0")
	}
	if p.Budgets.Min <= 0 || p.Budgets.Max < p.Budgets.Min {
		bad("budgets.min must be above 0 and at most budgets.max")
	}
	for _, t := range []string{"answer", "edit", "build", "review", "plan", "handoff"} {
		if v, ok := p.Budgets.Tasks[t]; !ok || v < p.Budgets.Min || v > p.Budgets.Max {
			bad("budgets.tasks.%s must exist and lie within budgets.min and budgets.max", t)
		}
	}
	if _, ok := p.Budgets.Tasks[p.Budgets.DefaultTask]; !ok {
		bad("budgets.default_task must name a task")
	}
	sum := 0.0
	seen := map[string]bool{}
	for _, s := range p.Sections {
		if !slices.Contains([]string{"rules", "knowledge", "focus", "passages", "links"}, s.Kind) || seen[s.Kind] {
			bad("sections: %q is unknown or repeated", s.Kind)
		}
		seen[s.Kind] = true
		if s.Share < 0 || s.Share > 1 {
			bad("sections: the share of %s must lie between 0 and 1", s.Kind)
		}
		sum += s.Share
	}
	if math.Abs(sum-1) > 0.01 {
		bad("sections: the shares must add up to 1, not %.2f", sum)
	}
	w := p.Ranking.Weights
	if w.Text < 0 || w.Title < 0 || w.Link < 0 || w.Recency < 0 || w.Knowledge < 0 || w.Focus < 0 {
		bad("ranking.weights must not be negative")
	}
	if p.Ranking.RecencyHalfLifeDays <= 0 || p.Ranking.MaxCandidates <= 0 || p.Ranking.PerPage <= 0 {
		bad("ranking: half-life, max_candidates and per_page must be above 0")
	}
	k := p.Knowledge
	want := []string{"fact", "decision", "glossary", "constraint", "preference"}
	if !slices.Equal(k.Kinds, want) {
		bad("knowledge.kinds must be %v; the database checks this list", want)
	}
	for _, kind := range want {
		if k.HalfLifeDays[kind] <= 0 {
			bad("knowledge.half_life_days.%s must be above 0", kind)
		}
	}
	c := k.Confidence
	if c.Floor < 0 || c.Initial <= c.Floor || c.AgentInitial <= c.Floor || c.Initial > c.Max || c.AgentInitial > c.Max || c.Max > 1 || c.ConfirmStep <= 0 || c.ContradictStep <= 0 {
		bad("knowledge.confidence: need 0 <= floor < initial and agent_initial <= max <= 1, positive steps")
	}
	if k.MaxTextChars <= 0 || k.MaxSubjectChars <= 0 || k.MaxItems <= 0 || k.PackLimit <= 0 || k.SearchLimit <= 0 {
		bad("knowledge: limits must be above 0")
	}
	cp := p.Compression
	if !slices.Contains(cp.Formats, cp.DefaultFormat) || !slices.Equal(cp.Formats, []string{"digest", "outline", "delta"}) {
		bad("compression: formats must be digest, outline and delta, and default_format one of them")
	}
	if cp.Digest.Sentences <= 0 || cp.Digest.MaxChars <= 0 || cp.Outline.MaxDepth <= 0 || cp.Outline.MaxHeadings <= 0 || cp.Delta.DefaultHours <= 0 || cp.Delta.MaxHours < cp.Delta.DefaultHours {
		bad("compression: limits must be above 0 and delta.max_hours at least default_hours")
	}
	if len(p.Handoff.Fields) == 0 || p.Handoff.MaxOpenItems <= 0 || p.Handoff.MaxChars <= 0 {
		bad("handoff: fields and limits are required")
	}
	h := p.Hygiene
	if h.ConflictSimilarity <= 0 || h.DuplicateSimilarity <= h.ConflictSimilarity || h.DuplicateSimilarity > 1 {
		bad("hygiene: need 0 < conflict_similarity < duplicate_similarity <= 1")
	}
	if h.StaleAfterDays <= 0 || h.ReviewBatch <= 0 || h.PruneBelow < 0 {
		bad("hygiene: stale_after_days and review_batch must be above 0")
	}
	f := p.Family
	if f.VocabularyWeight <= 0 || f.ToolWeight < 0 || f.MaxTools <= 0 || f.CacheSeconds < 0 || f.TimeoutMS < 100 || f.TimeoutMS > 10000 || f.PackMinScore <= 0 || f.PackMaxTools < 0 || f.PackMaxTools > f.MaxTools {
		bad("family: need vocabulary_weight, max_tools and pack_min_score above 0, timeout_ms from 100 to 10000, and pack_max_tools at most max_tools")
	}
	return errors.Join(errs...)
}

func (p *Policy) Budget(task string, explicit int) (string, int) {
	if _, ok := p.Budgets.Tasks[task]; !ok {
		task = p.Budgets.DefaultTask
	}
	b := p.Budgets.Tasks[task]
	if explicit > 0 {
		b = explicit
	}
	return task, min(max(b, p.Budgets.Min), p.Budgets.Max)
}

func (p *Policy) Tokens(chars int) int {
	return int(math.Ceil(float64(chars) / p.Token.CharsPerToken))
}

func (p *Policy) Chars(tokens int) int {
	return int(float64(tokens) * p.Token.CharsPerToken)
}
