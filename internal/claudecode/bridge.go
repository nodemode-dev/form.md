package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/ids"
)

type CommandSpec struct {
	ID          string                    `json:"id"`
	Label       string                    `json:"label"`
	Group       string                    `json:"group"`
	Description string                    `json:"description,omitempty"`
	When        string                    `json:"when,omitempty"`
	Keys        []string                  `json:"keys,omitempty"`
	Params      map[string]map[string]any `json:"params,omitempty"`
	Mutates     bool                      `json:"mutates"`
	Risky       bool                      `json:"risky,omitempty"`
}

func (c *CommandSpec) UnmarshalJSON(b []byte) error {
	type plain CommandSpec
	v := plain{Mutates: true}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*c = CommandSpec(v)
	return nil
}

type Sender interface {
	Send(userID, kind string, data map[string]any)
	Online(userID string) bool
}

type reply struct {
	Result json.RawMessage
	Err    string
}

type Bridge struct {
	hub     Sender
	mu      sync.Mutex
	catalog map[string][]CommandSpec
	waiting map[string]chan reply
}

func NewBridge(hub Sender) *Bridge {
	return &Bridge{hub: hub, catalog: map[string][]CommandSpec{}, waiting: map[string]chan reply{}}
}

func (b *Bridge) SetCatalogue(userID string, specs []CommandSpec) {
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	b.mu.Lock()
	b.catalog[userID] = specs
	b.mu.Unlock()
}

func (b *Bridge) Catalogue(userID, filter string) []CommandSpec {
	b.mu.Lock()
	defer b.mu.Unlock()
	f := strings.ToLower(strings.TrimSpace(filter))
	out := []CommandSpec{}
	for _, c := range b.catalog[userID] {
		if f == "" || strings.Contains(strings.ToLower(c.ID+" "+c.Label+" "+c.Group+" "+c.Description), f) {
			out = append(out, c)
		}
	}
	return out
}

func (b *Bridge) Describe(userID, id string) (CommandSpec, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, c := range b.catalog[userID] {
		if c.ID == id {
			return c, true
		}
	}
	return CommandSpec{}, false
}

var ErrNoWindow = errors.New("no form.md window is open for this account: open form.md in a browser or the desktop app and try again")

func (b *Bridge) Run(ctx context.Context, userID, id string, params map[string]any) (json.RawMessage, error) {
	if _, ok := b.Describe(userID, id); !ok {
		if len(b.Catalogue(userID, "")) == 0 {
			return nil, ErrNoWindow
		}
		return nil, fmt.Errorf("unknown command %q: list the catalogue first", id)
	}
	if !b.hub.Online(userID) {
		return nil, ErrNoWindow
	}
	call := ids.New()
	ch := make(chan reply, 1)
	b.mu.Lock()
	b.waiting[call] = ch
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.waiting, call)
		b.mu.Unlock()
	}()
	if params == nil {
		params = map[string]any{}
	}
	b.hub.Send(userID, "command.run", map[string]any{"call_id": call, "id": id, "params": params})
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	select {
	case r := <-ch:
		if r.Err != "" {
			return nil, errors.New(r.Err)
		}
		if len(r.Result) == 0 {
			r.Result = json.RawMessage(`{"ok":true}`)
		}
		return r.Result, nil
	case <-ctx.Done():
		return nil, errors.New("the window did not answer within a minute")
	}
}

func (b *Bridge) Answer(call string, result json.RawMessage, errText string) bool {
	b.mu.Lock()
	ch := b.waiting[call]
	b.mu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- reply{Result: result, Err: errText}:
	default:
	}
	return true
}
