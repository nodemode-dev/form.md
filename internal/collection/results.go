package collection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

const maxResults = 24

type resultEntry struct {
	key  string
	rows []*Row
}

type resultCache struct {
	mu      sync.Mutex
	entries []resultEntry
}

func newResultCache() *resultCache { return &resultCache{} }

func (c *resultCache) lookup(key string) ([]*Row, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.entries {
		if e.key == key {
			if i > 0 {
				copy(c.entries[1:i+1], c.entries[:i])
				c.entries[0] = e
			}
			return e.rows, true
		}
	}
	return nil, false
}

func (c *resultCache) store(key string, rows []*Row) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append([]resultEntry{{key: key, rows: rows}}, c.entries...)
	if len(c.entries) > maxResults {
		c.entries = c.entries[:maxResults]
	}
}

func (c *resultCache) ordered(ev *evaluator, sn *snap, cfg ViewConfig, in QueryInput) []*Row {
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode(cfg.Filter)
	_ = enc.Encode(cfg.Sorts)
	_ = enc.Encode(in.Range)
	h.Write([]byte(strings.ToLower(strings.TrimSpace(in.Search))))
	usesNow := false
	for _, p := range sn.schema {
		if p.Type == TFormula && strings.Contains(strings.ToLower(p.Expr), "now") {
			usesNow = true
		}
	}
	stamp := ""
	if usesNow || cfg.Filter != nil {
		stamp = strconv.FormatInt(ev.now.Unix()/60, 10)
	}
	key := sn.id + "|" + ev.p.UserID + "|" + strconv.FormatInt(ev.s.epoch.Load(), 10) + "|" + stamp + "|" + hex.EncodeToString(h.Sum(nil))
	if rows, ok := c.lookup(key); ok {
		return rows
	}
	base := sn.ordered()
	out := make([]*Row, 0, len(base))
	q := strings.ToLower(strings.TrimSpace(in.Search))
	for _, r := range base {
		if q != "" && !ev.search(sn, r, q) {
			continue
		}
		if in.Range != nil && !ev.overlaps(sn, r, in.Range) {
			continue
		}
		if !ev.match(sn, r, cfg.Filter, 0) {
			continue
		}
		out = append(out, r)
	}
	ev.sortRows(sn, out, cfg.Sorts)
	c.store(key, out)
	return out
}
