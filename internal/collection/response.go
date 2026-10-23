package collection

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

// Response is one form submission, stored whole on the row it created.
// The row's answer columns are computed from Answers when the row is read.
// A value kept in the row's own vals is an override and wins over the
// computed value (a JSON null there clears the computed value).
type Response struct {
	FormID      string                     `json:"formId"`
	Version     int64                      `json:"version"`
	Answers     map[string]json.RawMessage `json:"answers"`
	SubmittedAt int64                      `json:"submittedAt"`
}

// ResponseEvent is the "response received" event: it carries the validated
// response as submitted, so listeners judge the answers, not stored columns.
type ResponseEvent struct {
	WorkspaceID  string
	CollectionID string
	RowID        string
	ActorID      string
	ActorLabel   string
	Response     *Response
}

func parseResponse(raw string) *Response {
	if raw == "" {
		return nil
	}
	var r Response
	if json.Unmarshal([]byte(raw), &r) != nil || r.Answers == nil {
		return nil
	}
	return &r
}

// columns are the property values the response computes for its row. The
// title answer names the page and is not a column.
func (r *Response) columns() map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(r.Answers))
	for k, v := range r.Answers {
		if k == TTitle || isNull(v) {
			continue
		}
		out[k] = v
	}
	return out
}

// effectiveVals is what readers see: the computed columns with the row's own
// overrides laid over them. A row with no response keeps its stored values.
func effectiveVals(stored map[string]json.RawMessage, r *Response) map[string]json.RawMessage {
	if r == nil {
		return stored
	}
	out := r.columns()
	for k, v := range stored {
		if isNull(v) {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

func sameJSON(a, b json.RawMessage) bool {
	var x, y bytes.Buffer
	if json.Compact(&x, a) != nil || json.Compact(&y, b) != nil {
		return bytes.Equal(a, b)
	}
	return bytes.Equal(x.Bytes(), y.Bytes())
}

// storedVals turns a full set of effective values back into the overrides a
// response row keeps: only what differs from the computed columns.
func storedVals(eff map[string]json.RawMessage, r *Response) map[string]json.RawMessage {
	if r == nil {
		return eff
	}
	base := r.columns()
	out := map[string]json.RawMessage{}
	for k, v := range eff {
		if b, ok := base[k]; ok && sameJSON(v, b) {
			continue
		}
		out[k] = v
	}
	for k := range base {
		if _, ok := eff[k]; !ok {
			out[k] = json.RawMessage("null")
		}
	}
	return out
}

func (s *Service) OnResponse(fn func(ctx context.Context, e ResponseEvent)) {
	s.responders = append(s.responders, fn)
}

func (s *Service) respond(ctx context.Context, e ResponseEvent) {
	for _, fn := range s.responders {
		go fn(context.WithoutCancel(ctx), e)
	}
}

// ResponseOf returns the response record stored on a row, or nil.
func (s *Service) ResponseOf(ctx context.Context, rowID string) *Response {
	var raw string
	if s.st.R.QueryRowContext(ctx, "SELECT response FROM collection_rows WHERE page_id = ?", rowID).Scan(&raw) != nil {
		return nil
	}
	return parseResponse(raw)
}

// ResponseMatches judges a filter against the answers as submitted.
func (s *Service) ResponseMatches(ctx context.Context, p *policy.Principal, e ResponseEvent, f *Filter) (bool, error) {
	if _, _, err := s.readable(ctx, p, e.CollectionID); err != nil {
		return false, err
	}
	if f == nil || e.Response == nil {
		return true, nil
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(e.CollectionID)
	if sn == nil {
		return false, store.ErrNotFound
	}
	shadow := &Row{ID: e.RowID, Vals: e.Response.columns()}
	if t, ok := e.Response.Answers[TTitle]; ok {
		shadow.Title = str(t)
	}
	return ev.match(sn, shadow, f, 0), nil
}
