package tunnel

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
)

var quickURL = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

type Status struct {
	Socket    string `json:"socket"`
	Mode      string `json:"mode"`
	Running   bool   `json:"running"`
	URL       string `json:"url"`
	Error     string `json:"error"`
	Binary    string `json:"binary"`
	Restarts  int    `json:"restarts"`
	StartedAt int64  `json:"started_at"`
}

type Service struct {
	set    *settings.Service
	log    *slog.Logger
	socket string

	mu      sync.Mutex
	status  Status
	cancel  context.CancelFunc
	done    chan struct{}
	running bool
}

func New(set *settings.Service, log *slog.Logger, dataDir string) *Service {
	socket := filepath.Join(dataDir, "tunnel.sock")
	if len(socket) > 100 {
		sum := sha256.Sum256([]byte(dataDir))
		socket = filepath.Join(os.TempDir(), "form-md-"+hex.EncodeToString(sum[:6])+".sock")
	}
	return &Service{set: set, log: log, socket: socket}
}

func (s *Service) Socket() string { return s.socket }

func (s *Service) Listen() (net.Listener, error) {
	os.Remove(s.socket)
	ln, err := net.Listen("unix", s.socket)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(s.socket, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	return ln, nil
}

func binary() string {
	if p := os.Getenv("FORMMD_CLOUDFLARED"); p != "" {
		return p
	}
	p, err := exec.LookPath("cloudflared")
	if err != nil {
		return ""
	}
	return p
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.Socket = s.socket
	st.Mode = s.set.Get("tunnel.mode")
	st.Running = s.running
	if st.Binary == "" {
		st.Binary = binary()
	}
	return st
}

func (s *Service) Autostart(ctx context.Context) {
	if s.set.Bool("tunnel.autostart") && s.set.Get("tunnel.mode") != "off" {
		if err := s.Start(ctx); err != nil {
			s.log.Warn("tunnel autostart", "err", err)
		}
	}
}

func (s *Service) Start(ctx context.Context) error {
	mode := s.set.Get("tunnel.mode")
	if mode == "off" || mode == "" {
		return errors.New("choose a tunnel mode first")
	}
	if mode == "token" && s.set.Get("tunnel.token") == "" {
		return errors.New("save the tunnel token first")
	}
	bin := binary()
	if bin == "" {
		return errors.New("cloudflared is not installed on this server (or set FORMMD_CLOUDFLARED to its path)")
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	run, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.cancel, s.done, s.running = cancel, make(chan struct{}), true
	s.status = Status{Binary: bin, StartedAt: time.Now().UnixMilli()}
	done := s.done
	s.mu.Unlock()
	go s.supervise(run, bin, mode, done)
	return nil
}

func (s *Service) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.running, s.cancel = false, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	s.mu.Lock()
	s.status.URL = ""
	s.mu.Unlock()
}

func (s *Service) supervise(ctx context.Context, bin, mode string, done chan struct{}) {
	defer close(done)
	backoff := time.Second
	for ctx.Err() == nil {
		args := []string{"tunnel", "--no-autoupdate"}
		env := os.Environ()
		if mode == "token" {
			args = append(args, "run")
			env = append(env, "TUNNEL_TOKEN="+s.set.Get("tunnel.token"))
		} else {
			args = append(args, "--url", "unix:"+s.socket)
		}
		cmd := exec.CommandContext(ctx, bin, args...)
		cmd.Env = env
		cmd.WaitDelay = 5 * time.Second
		stderr, err := cmd.StderrPipe()
		if err == nil {
			cmd.Stdout = cmd.Stderr
			err = cmd.Start()
		}
		if err != nil {
			s.fail(err)
			return
		}
		started := time.Now()
		s.watch(stderr)
		err = cmd.Wait()
		if ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		s.status.Restarts++
		s.status.URL = ""
		if err != nil {
			s.status.Error = err.Error()
		}
		s.mu.Unlock()
		s.log.Warn("tunnel exited; restarting", "err", err)
		if time.Since(started) > time.Minute {
			backoff = time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, time.Minute)
	}
}

func (s *Service) watch(r io.Reader) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if u := quickURL.FindString(line); u != "" {
			s.mu.Lock()
			s.status.URL, s.status.Error = u, ""
			s.mu.Unlock()
			s.log.Info("tunnel ready", "url", u)
		} else if strings.Contains(line, "Registered tunnel connection") {
			s.mu.Lock()
			s.status.Error = ""
			s.mu.Unlock()
		}
	}
}

func (s *Service) fail(err error) {
	s.mu.Lock()
	s.status.Error, s.running = err.Error(), false
	s.mu.Unlock()
	s.log.Warn("tunnel", "err", err)
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if local, ok := r.Context().Value(http.LocalAddrContextKey).(*net.UnixAddr); ok && local.Name == s.socket {
			r.TLS = &tls.ConnectionState{}
			ip := net.ParseIP(strings.TrimSpace(r.Header.Get("Cf-Connecting-Ip")))
			if ip == nil {
				ip = net.IPv4(0, 0, 0, 0)
			}
			r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
			if u := s.publicHost(); u != "" {
				r.Host = u
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) publicHost() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.URL == "" || s.set.Get("tunnel.mode") != "quick" {
		return ""
	}
	return strings.TrimPrefix(s.status.URL, "https://")
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "tunnel" }

func (m module) Stop() { m.s.Stop() }

func (m module) Listen() (net.Listener, error) { return m.s.Listen() }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	r.HandleFunc("GET /api/admin/tunnel", m.auth.RequireAdmin(func(w http.ResponseWriter, r *http.Request, _ *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, m.s.Status())
		return nil
	}))
	r.HandleFunc("POST /api/admin/tunnel/start", m.auth.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := m.s.Start(r.Context()); err != nil {
			return httpx.Fail(http.StatusConflict, "tunnel", err.Error())
		}
		m.s.log.Info("tunnel started", "by", p.UserID)
		httpx.JSON(w, http.StatusOK, m.s.Status())
		return nil
	}))
	r.HandleFunc("POST /api/admin/tunnel/stop", m.auth.RequireAdmin(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		m.s.Stop()
		m.s.log.Info("tunnel stopped", "by", p.UserID)
		httpx.JSON(w, http.StatusOK, m.s.Status())
		return nil
	}))
	return nil
}
