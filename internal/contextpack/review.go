package contextpack

import (
	"context"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Pair struct {
	A          *Item   `json:"a"`
	B          *Item   `json:"b"`
	Similarity float64 `json:"similarity"`
}

type Review struct {
	WorkspaceID string  `json:"workspace_id"`
	Active      int     `json:"active"`
	Capacity    int     `json:"capacity"`
	Stale       []*Item `json:"stale"`
	Weak        []*Item `json:"weak"`
	Unsourced   int     `json:"unsourced"`
	Duplicates  []Pair  `json:"duplicates"`
	Conflicts   []Pair  `json:"conflicts"`
}

func (s *Service) Review(ctx context.Context, p *policy.Principal, workspaceID string) (*Review, error) {
	if err := s.reach(ctx, p, workspaceID, false); err != nil {
		return nil, err
	}
	items, err := s.active(ctx, p, workspaceID)
	if err != nil {
		return nil, err
	}
	h := s.Policy.Hygiene
	batch := h.ReviewBatch
	now := store.NowMS()
	r := &Review{WorkspaceID: workspaceID, Active: len(items), Capacity: s.Policy.Knowledge.MaxItems, Stale: []*Item{}, Weak: []*Item{}, Duplicates: []Pair{}, Conflicts: []Pair{}}
	for _, it := range items {
		if it.SourcePage == "" {
			r.Unsourced++
		}
		if it.Effective < h.PruneBelow {
			r.Weak = append(r.Weak, it)
		} else if float64(now-it.ConfirmedAt)/dayMS > h.StaleAfterDays {
			r.Stale = append(r.Stale, it)
		}
	}
	sort.SliceStable(r.Weak, func(i, j int) bool { return r.Weak[i].Effective < r.Weak[j].Effective })
	sort.SliceStable(r.Stale, func(i, j int) bool { return r.Stale[i].ConfirmedAt < r.Stale[j].ConfirmedAt })
	r.Weak, r.Stale = head(r.Weak, batch), head(r.Stale, batch)
	for _, it := range items {
		it.bag = s.Policy.bag(it.Text)
	}
	index := map[string][]int{}
	for i, it := range items {
		for w := range it.bag {
			index[w] = append(index[w], i)
		}
	}
	for i, a := range items {
		shared := map[int]int{}
		for w := range a.bag {
			for _, j := range index[w] {
				if j > i && items[j].Kind == a.Kind {
					shared[j]++
				}
			}
		}
		for j, n := range shared {
			b := items[j]
			sim := float64(n) / float64(len(a.bag)+len(b.bag)-n)
			switch {
			case sim >= h.DuplicateSimilarity:
				r.Duplicates = append(r.Duplicates, Pair{a, b, sim})
			case sim >= h.ConflictSimilarity && a.Subject != "" && strings.EqualFold(a.Subject, b.Subject):
				r.Conflicts = append(r.Conflicts, Pair{a, b, sim})
			}
		}
	}
	bySim := func(list []Pair) {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Similarity > list[j].Similarity })
	}
	bySim(r.Duplicates)
	bySim(r.Conflicts)
	r.Duplicates, r.Conflicts = head(r.Duplicates, batch), head(r.Conflicts, batch)
	return r, nil
}

func head[T any](list []T, n int) []T {
	if len(list) > n {
		return list[:n]
	}
	return list
}
