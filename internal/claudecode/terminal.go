package claudecode

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/nodemode-ai/form.md/internal/ids"
)

const scrollback = 256 * 1024

type Session struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Cwd     string `json:"cwd"`
	Alive   bool   `json:"alive"`
	Created int64  `json:"created_at"`

	owner string
	argv  []string
	env   []string
	mu    sync.Mutex
	f     *os.File
	cmd   *exec.Cmd
	buf   []byte
	subs  map[chan []byte]struct{}
	gen   int
	cols  uint16
	rows  uint16
}

type Terminals struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func newTerminals() *Terminals { return &Terminals{sessions: map[string]*Session{}} }

func loginShell() []string {
	if runtime.GOOS == "windows" {
		if c := os.Getenv("COMSPEC"); c != "" {
			return []string{c}
		}
		return []string{"cmd.exe"}
	}
	sh := os.Getenv("SHELL")
	if sh == "" {
		if p, err := exec.LookPath("bash"); err == nil {
			sh = p
		} else {
			sh = "/bin/sh"
		}
	}
	return []string{sh, "-l"}
}

func (t *Terminals) Open(owner, title, kind, cwd string, argv, env []string, cols, rows int) (*Session, error) {
	if runtime.GOOS == "windows" {
		return nil, errors.New("the embedded terminal needs macOS or Linux; on Windows use form-md claude in a terminal window")
	}
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("no folder %s", cwd)
	}
	if len(argv) == 0 {
		argv = loginShell()
	}
	s := &Session{ID: ids.New(), Title: title, Kind: kind, Cwd: cwd, Created: time.Now().UnixMilli(), owner: owner, argv: argv, env: env,
		subs: map[chan []byte]struct{}{}, cols: clamp(cols, 10, 500, 100), rows: clamp(rows, 4, 300, 30)}
	if err := s.spawn(); err != nil {
		return nil, err
	}
	t.mu.Lock()
	if s.Title == "" {
		s.Title = fmt.Sprintf("Terminal %d", len(t.sessions)+1)
	}
	t.sessions[s.ID] = s
	t.mu.Unlock()
	return s, nil
}

func clamp(v, lo, hi, def int) uint16 {
	if v <= 0 {
		v = def
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return uint16(v)
}

func (s *Session) spawn() error {
	cmd := exec.Command(s.argv[0], s.argv[1:]...)
	cmd.Dir = s.Cwd
	cmd.Env = s.env
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: s.cols, Rows: s.rows})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.f, s.cmd, s.Alive = f, cmd, true
	gen := s.gen
	s.mu.Unlock()
	go s.pump(f, cmd, gen)
	return nil
}

func (s *Session) pump(f *os.File, cmd *exec.Cmd, gen int) {
	chunk := make([]byte, 32*1024)
	for {
		n, err := f.Read(chunk)
		if n > 0 {
			data := append([]byte(nil), chunk[:n]...)
			s.mu.Lock()
			if s.gen != gen {
				s.mu.Unlock()
				_ = cmd.Wait()
				return
			}
			s.buf = append(s.buf, data...)
			if over := len(s.buf) - scrollback; over > 0 {
				s.buf = append([]byte(nil), s.buf[over:]...)
			}
			for ch := range s.subs {
				select {
				case ch <- data:
				default:
				}
			}
			s.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
	_ = cmd.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen != gen {
		return
	}
	s.Alive = false
	for ch := range s.subs {
		close(ch)
		delete(s.subs, ch)
	}
}

func (s *Session) Attach() (chan []byte, []byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan []byte, 512)
	backlog := append([]byte(nil), s.buf...)
	if s.Alive {
		s.subs[ch] = struct{}{}
	} else {
		close(ch)
	}
	return ch, backlog, s.Alive
}

func (s *Session) Detach(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subs[ch]; ok {
		delete(s.subs, ch)
		close(ch)
	}
}

func (s *Session) Write(p []byte) {
	s.mu.Lock()
	f, alive := s.f, s.Alive
	s.mu.Unlock()
	if alive && f != nil {
		_, _ = f.Write(p)
	}
}

func (s *Session) Resize(cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cols, s.rows = clamp(cols, 10, 500, 100), clamp(rows, 4, 300, 30)
	if s.Alive && s.f != nil {
		_ = pty.Setsize(s.f, &pty.Winsize{Cols: s.cols, Rows: s.rows})
	}
}

func (s *Session) stop() { s.halt(true) }

func (s *Session) halt(dropSubs bool) {
	s.mu.Lock()
	s.gen++
	f, cmd := s.f, s.cmd
	if dropSubs {
		for ch := range s.subs {
			close(ch)
			delete(s.subs, ch)
		}
	}
	s.Alive = false
	s.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if f != nil {
		_ = f.Close()
	}
}

func (s *Session) Snapshot() Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Session{ID: s.ID, Title: s.Title, Kind: s.Kind, Cwd: s.Cwd, Alive: s.Alive, Created: s.Created}
}

func (t *Terminals) Restart(owner, id string) (*Session, error) {
	s := t.Get(owner, id)
	if s == nil {
		return nil, errNoTerminal
	}
	s.halt(false)
	s.mu.Lock()
	s.buf = []byte("\x1bc")
	for ch := range s.subs {
		select {
		case ch <- []byte("\x1bc"):
		default:
		}
	}
	s.mu.Unlock()
	return s, s.spawn()
}

var ansi = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[@-_]`)

func plainText(raw []byte) string {
	text := ansi.ReplaceAllString(string(raw), "")
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "")
}

func (s *Session) Scrollback() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.buf...)
}

func (t *Terminals) RunningClaude(owner string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, s := range t.sessions {
		if s.owner == owner && s.Kind == "claude" {
			s.mu.Lock()
			alive := s.Alive
			s.mu.Unlock()
			if alive {
				return s
			}
		}
	}
	return nil
}

var errNoTerminal = errors.New("no such terminal")

func (t *Terminals) Get(owner, id string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := t.sessions[id]
	if s == nil || s.owner != owner {
		return nil
	}
	return s
}

func (t *Terminals) List(owner string) []Session {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := []Session{}
	for _, s := range t.sessions {
		if s.owner == owner {
			out = append(out, s.Snapshot())
		}
	}
	return out
}

func (t *Terminals) Close(owner, id string) bool {
	t.mu.Lock()
	s := t.sessions[id]
	if s == nil || s.owner != owner {
		t.mu.Unlock()
		return false
	}
	delete(t.sessions, id)
	t.mu.Unlock()
	s.stop()
	return true
}

func (t *Terminals) CloseAll() {
	t.mu.Lock()
	all := t.sessions
	t.sessions = map[string]*Session{}
	t.mu.Unlock()
	for _, s := range all {
		s.stop()
	}
}

func runtimeWindows() bool { return runtime.GOOS == "windows" }

func agentDir(dataDir string) string {
	d := filepath.Join(dataDir, "agent")
	if v := strings.TrimSpace(os.Getenv("FORMMD_AGENT_DIR")); v != "" {
		d = v
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		d = filepath.Join(dataDir, "agent")
		_ = os.MkdirAll(d, 0o700)
	}
	return d
}
