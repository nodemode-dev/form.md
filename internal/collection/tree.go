package collection

func treeOrder(all []*Row, propID string, open map[string]bool) ([]*Row, map[string]int, map[string]int) {
	in := make(map[string]bool, len(all))
	for _, r := range all {
		in[r.ID] = true
	}
	parent := make(map[string]string, len(all))
	children := map[string][]*Row{}
	var roots []*Row
	for _, r := range all {
		p := ""
		if ids := strList(r.Vals[propID]); len(ids) > 0 && ids[0] != r.ID && in[ids[0]] {
			p = ids[0]
		}
		parent[r.ID] = p
		if p == "" {
			roots = append(roots, r)
		} else {
			children[p] = append(children[p], r)
		}
	}
	out := make([]*Row, 0, len(all))
	depth := make(map[string]int, len(all))
	kids := make(map[string]int, len(children))
	for id, list := range children {
		kids[id] = len(list)
	}
	seen := make(map[string]bool, len(all))
	var walk func(r *Row, d int)
	walk = func(r *Row, d int) {
		if seen[r.ID] || d > 32 {
			return
		}
		seen[r.ID] = true
		out = append(out, r)
		depth[r.ID] = d
		if !open[r.ID] {
			return
		}
		for _, c := range children[r.ID] {
			walk(c, d+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
	for _, r := range all {
		if !seen[r.ID] && parent[r.ID] != "" && !reachable(parent, r.ID) {
			walk(r, 0)
		}
	}
	return out, depth, kids
}

func reachable(parent map[string]string, id string) bool {
	hops := map[string]bool{}
	for cur := parent[id]; cur != ""; cur = parent[cur] {
		if hops[cur] {
			return false
		}
		hops[cur] = true
		if parent[cur] == "" {
			return true
		}
	}
	return true
}
