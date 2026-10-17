package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type Service struct {
	Auth    *auth.Service
	Bridge  *Bridge
	Terms   *Terminals
	Chats   *Chats
	addr    string
	dataDir string

	mu       sync.Mutex
	attached map[*attachment]bool
}

type attachment struct {
	user, cred string
	cancel     context.CancelFunc
}

func New(a *auth.Service, hub Sender, addr, dataDir string) *Service {
	return &Service{Auth: a, Bridge: NewBridge(hub), Terms: newTerminals(), Chats: newChats(), addr: addr, dataDir: dataDir, attached: map[*attachment]bool{}}
}

func (s *Service) detachWhere(match func(*attachment) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for at := range s.attached {
		if match(at) {
			at.cancel()
		}
	}
}

func (s *Service) CloseUser(userID string) {
	s.detachWhere(func(at *attachment) bool { return at.user == userID })
}

func (s *Service) CloseCredential(id string) {
	s.detachWhere(func(at *attachment) bool { return at.cred == id })
}

func (s *Service) Module() app.Module { return module{s} }

func (s *Service) Stop() { s.Terms.CloseAll() }

type module struct{ s *Service }

func (module) ID() string { return "claudecode" }

func (m module) Stop() { m.s.Stop() }

func (s *Service) enabled() bool {
	switch strings.ToLower(os.Getenv("FORMMD_TERMINAL")) {
	case "off", "0", "false":
		return false
	case "on", "1", "true":
		return true
	}
	return Local(s.addr)
}

func (s *Service) gate(p *policy.Principal) error {
	if !p.IsSession() {
		return httpx.ErrSessionOnly
	}
	if !s.enabled() {
		return httpx.Fail(http.StatusForbidden, "terminal_off", "the terminal and Claude Code run only on a form.md listening on this computer; set FORMMD_TERMINAL=on to allow them on a server")
	}
	if !p.IsOwner() {
		return httpx.Fail(http.StatusForbidden, "owner_only", "only the instance owner can open a terminal or Claude Code")
	}
	return nil
}

func (s *Service) env() []string { return childEnv(EngineURL(s.addr)) }

func (s *Service) claudeArgv(ctx context.Context, p *policy.Principal, extra ...string) ([]string, error) {
	info := Detect(s.env(), false)
	if !info.Installed {
		return nil, httpx.Fail(http.StatusBadRequest, "claude_missing", Install)
	}
	token, err := s.Auth.AppToken(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	argv := []string{info.Path, "--mcp-config", MCPConfig(EngineURL(s.addr), token), "--append-system-prompt", Note}
	return append(argv, extra...), nil
}

func (m module) Mount(r *httpx.Router, d *app.Deps) error {
	s := m.s
	a := s.Auth
	r.HandleFunc("GET /api/claude", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		on := s.enabled() && p.IsOwner()
		out := map[string]any{"enabled": on, "windows": runtimeWindows(), "folder": agentDir(s.dataDir), "engine_url": EngineURL(s.addr)}
		if on {
			out["claude"] = Detect(s.env(), r.URL.Query().Get("fresh") == "1")
		}
		httpx.JSON(w, http.StatusOK, out)
		return nil
	}))
	r.HandleFunc("GET /api/terminal", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"terminals": s.Terms.List(p.UserID)})
		return nil
	}))
	r.HandleFunc("POST /api/terminal", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		var in struct {
			Kind    string `json:"kind"`
			Cwd     string `json:"cwd"`
			Cols    int    `json:"cols"`
			Rows    int    `json:"rows"`
			Resume  string `json:"resume"`
			Context string `json:"context"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		var argv []string
		title := ""
		cwd := in.Cwd
		if in.Kind == "claude" {
			var extra []string
			if in.Resume != "" {
				extra = append(extra, "--resume", in.Resume)
			} else if strings.TrimSpace(in.Context) != "" {
				extra = append(extra, strings.TrimSpace(in.Context))
			}
			var err error
			if argv, err = s.claudeArgv(r.Context(), p, extra...); err != nil {
				return err
			}
			title = "Claude Code"
			if cwd == "" {
				cwd = agentDir(s.dataDir)
			}
		} else {
			in.Kind = "shell"
		}
		sess, err := s.Terms.Open(p.UserID, title, in.Kind, cwd, argv, s.env(), in.Cols, in.Rows)
		if err != nil {
			return httpx.Fail(http.StatusBadRequest, "terminal_failed", err.Error())
		}
		httpx.JSON(w, http.StatusCreated, sess.Snapshot())
		return nil
	}))
	r.HandleFunc("POST /api/terminal/{id}/restart", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		sess, err := s.Terms.Restart(p.UserID, r.PathValue("id"))
		if err != nil {
			return httpx.Fail(http.StatusNotFound, "not_found", err.Error())
		}
		httpx.JSON(w, http.StatusOK, sess.Snapshot())
		return nil
	}))
	r.HandleFunc("DELETE /api/terminal/{id}", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"closed": s.Terms.Close(p.UserID, r.PathValue("id"))})
		return nil
	}))
	r.HandleFunc("GET /api/terminal/{id}/attach", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		sess := s.Terms.Get(p.UserID, r.PathValue("id"))
		if sess == nil {
			return httpx.Fail(http.StatusNotFound, "not_found", "no such terminal")
		}
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"127.0.0.1:*", "localhost:*", "[::1]:*", "tauri.localhost"}})
		if err != nil {
			return nil
		}
		s.attach(r.Context(), c, sess, p)
		return nil
	}))

	r.HandleFunc("GET /api/claude/chat", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, s.Chats.of(p.UserID).snapshot())
		return nil
	}))
	r.HandleFunc("POST /api/claude/chat", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		var in struct {
			Text    string `json:"text"`
			Context string `json:"context"`
			Note    string `json:"note"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		shown := strings.TrimSpace(in.Text)
		if shown == "" {
			return httpx.Fail(http.StatusBadRequest, "empty", "type a message first")
		}
		prompt := shown
		if c := strings.TrimSpace(in.Context); c != "" {
			prompt = shown + " (" + c + ")"
		}
		ch := s.Chats.of(p.UserID)
		ch.mu.Lock()
		if ch.state.Busy {
			ch.mu.Unlock()
			return httpx.Fail(http.StatusConflict, "busy", "Claude is still answering; wait or press Stop")
		}
		session := ch.state.Session
		ch.mu.Unlock()
		info := Detect(s.env(), true)
		if !info.Installed {
			return httpx.Fail(http.StatusBadRequest, "claude_missing", Install)
		}
		token, err := s.Auth.AppToken(r.Context(), p.UserID)
		if err != nil {
			return err
		}
		ch.mu.Lock()
		ch.state.Busy = true
		ch.state.Messages = append(ch.state.Messages, Message{Role: "user", Text: shown, Note: strings.TrimSpace(in.Note)})
		ch.mu.Unlock()
		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)
		rc := http.NewResponseController(w)
		var wmu sync.Mutex
		gone := false
		send := func(kind string, data map[string]any) {
			wmu.Lock()
			defer wmu.Unlock()
			if gone {
				return
			}
			raw, _ := json.Marshal(data)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", kind, raw); err != nil {
				gone = true
				return
			}
			_ = rc.Flush()
		}
		send("start", map[string]any{"text": shown, "note": strings.TrimSpace(in.Note)})
		done := make(chan struct{})
		go func() {
			ch.run(chatArgv(info.Path, prompt, session, MCPConfig(EngineURL(s.addr), token)), agentDir(s.dataDir), s.env(), send)
			close(done)
		}()
		select {
		case <-done:
		case <-r.Context().Done():
			wmu.Lock()
			gone = true
			wmu.Unlock()
		}
		return nil
	}))
	r.HandleFunc("POST /api/claude/chat/cancel", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"stopped": s.Chats.of(p.UserID).cancel()})
		return nil
	}))
	r.HandleFunc("POST /api/claude/chat/reset", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		ch := s.Chats.of(p.UserID)
		ch.cancel()
		ch.mu.Lock()
		ch.state = ChatState{Messages: []Message{}}
		ch.mu.Unlock()
		httpx.JSON(w, http.StatusOK, map[string]any{"reset": true})
		return nil
	}))
	r.HandleFunc("GET /api/claude/sessions", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		cur := s.Chats.of(p.UserID).snapshot().Session
		httpx.JSON(w, http.StatusOK, map[string]any{"sessions": conversations(agentDir(s.dataDir), cur)})
		return nil
	}))
	r.HandleFunc("POST /api/claude/adopt", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.gate(p); err != nil {
			return err
		}
		var in struct {
			Session string `json:"session"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		ch := s.Chats.of(p.UserID)
		list := conversations(agentDir(s.dataDir), "")
		pick := ""
		for _, c := range list {
			if c.ID == in.Session || (in.Session == "" && pick == "") {
				pick = c.ID
			}
		}
		if pick == "" {
			return httpx.Fail(http.StatusNotFound, "not_found", "no Claude Code conversation was found in form.md's agent folder yet")
		}
		msgs := transcript(projectDir(agentDir(s.dataDir)) + "/" + pick + ".jsonl")
		ch.mu.Lock()
		if ch.state.Busy {
			ch.mu.Unlock()
			return httpx.Fail(http.StatusConflict, "busy", "Claude is still answering; wait or press Stop")
		}
		ch.state = ChatState{Session: pick, Messages: msgs}
		ch.mu.Unlock()
		httpx.JSON(w, http.StatusOK, ch.snapshot())
		return nil
	}))

	r.HandleFunc("PUT /api/commands/catalogue", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Commands []CommandSpec `json:"commands"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		s.Bridge.SetCatalogue(p.UserID, in.Commands)
		httpx.JSON(w, http.StatusOK, map[string]any{"count": len(in.Commands)})
		return nil
	}))
	r.HandleFunc("GET /api/commands", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"commands": s.Bridge.Catalogue(p.UserID, r.URL.Query().Get("q"))})
		return nil
	}))
	r.HandleFunc("GET /api/commands/{id}", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		c, ok := s.Bridge.Describe(p.UserID, r.PathValue("id"))
		if !ok {
			return httpx.Fail(http.StatusNotFound, "not_found", "no such command in the open window")
		}
		httpx.JSON(w, http.StatusOK, c)
		return nil
	}))
	r.HandleFunc("POST /api/commands/run", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			ID     string         `json:"id"`
			Params map[string]any `json:"params"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		out, err := s.RunCommand(r.Context(), p, in.ID, in.Params)
		if err != nil {
			return httpx.Fail(http.StatusBadRequest, "command_failed", err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
		return nil
	}))
	r.HandleFunc("POST /api/commands/result", a.RequireSession(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			CallID string          `json:"call_id"`
			Result json.RawMessage `json:"result"`
			Error  string          `json:"error"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"delivered": s.Bridge.Answer(in.CallID, in.Result, in.Error)})
		return nil
	}))
	return nil
}

func (s *Service) RunCommand(ctx context.Context, p *policy.Principal, id string, params map[string]any) (json.RawMessage, error) {
	spec, ok := s.Bridge.Describe(p.UserID, id)
	if ok && spec.Mutates && !p.CanWrite() {
		return nil, fmt.Errorf("%s changes the workspace and this credential is read-only", id)
	}
	return s.Bridge.Run(ctx, p.UserID, id, params)
}

func (s *Service) CatalogueFor(p *policy.Principal, filter string) any {
	return s.Bridge.Catalogue(p.UserID, filter)
}

func (s *Service) DescribeFor(p *policy.Principal, id string) (any, bool) {
	return s.Bridge.Describe(p.UserID, id)
}

func (s *Service) attach(ctx context.Context, c *websocket.Conn, sess *Session, p *policy.Principal) {
	defer c.CloseNow()
	ch, backlog, alive := sess.Attach()
	defer sess.Detach(ch)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	at := &attachment{user: p.UserID, cred: p.CredentialID, cancel: cancel}
	s.mu.Lock()
	s.attached[at] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.attached, at)
		s.mu.Unlock()
	}()
	if len(backlog) > 0 {
		if c.Write(ctx, websocket.MessageBinary, backlog) != nil {
			return
		}
	}
	exit := []byte(`{"type":"exit"}`)
	if !alive {
		_ = c.Write(ctx, websocket.MessageText, exit)
	}
	go func() {
		defer cancel()
		for data := range ch {
			if c.Write(ctx, websocket.MessageBinary, data) != nil {
				return
			}
		}
		_ = c.Write(ctx, websocket.MessageText, exit)
	}()
	for {
		_, raw, err := c.Read(ctx)
		if err != nil {
			return
		}
		var msg struct {
			Type string `json:"type"`
			Data string `json:"data"`
			Cols int    `json:"cols"`
			Rows int    `json:"rows"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		switch msg.Type {
		case "input":
			sess.Write([]byte(msg.Data))
		case "resize":
			sess.Resize(msg.Cols, msg.Rows)
		}
	}
}
