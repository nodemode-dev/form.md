package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"net"
	"net/http"
	"os/user"
	"strings"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

const desktopEmail = "owner@localhost"

func (s *Service) SetDesktopKey(key string) { s.desktopKey = key }

func (s *Service) Desktop() bool { return s.desktopKey != "" }

func desktopName() string {
	if u, err := user.Current(); err == nil {
		for _, n := range []string{u.Name, u.Username} {
			if name, err := cleanName(strings.Split(n, ",")[0]); err == nil {
				return name
			}
		}
	}
	return "Owner"
}

func loopbackPeer(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Service) desktopOwner(ctx context.Context) (string, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	var id string
	err := s.st.R.QueryRowContext(ctx, "SELECT id FROM users WHERE role = 'owner' AND disabled = 0 ORDER BY created_at LIMIT 1").Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	need, err := s.SetupRequired(ctx)
	if err != nil {
		return "", err
	}
	if !need {
		return "", httpx.Fail(http.StatusConflict, "no_owner", "this data folder has accounts but no owner")
	}
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		id, err = s.createUser(ctx, tx, desktopEmail, desktopName(), "", "owner", true)
		return err
	})
	return id, err
}

func (s *Service) handleDesktopEnter(w http.ResponseWriter, r *http.Request) error {
	key := r.URL.Query().Get("key")
	if s.desktopKey == "" || !loopbackPeer(r) || subtle.ConstantTimeCompare([]byte(key), []byte(s.desktopKey)) != 1 {
		s.authFailure(r, "desktop_key")
		return httpx.Fail(http.StatusNotFound, "not_found", "not found")
	}
	ctx := r.Context()
	id, err := s.desktopOwner(ctx)
	if err != nil {
		return err
	}
	if p, err := s.fromSession(ctx, r); err != nil || p == nil || p.UserID != id {
		if err := s.StartSession(ctx, w, r, id); err != nil {
			return err
		}
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return nil
}
