package ops

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type Module struct {
	Auth  *auth.Service
	Flush func(context.Context)
}

func (Module) ID() string { return "ops" }

func (m Module) Mount(r *httpx.Router, d *app.Deps) error {
	r.HandleFunc("GET /api/admin/backup", m.Auth.RequireOwner(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if !p.IsSession() {
			return httpx.Fail(http.StatusForbidden, "session_only", "a backup is downloaded from a signed-in browser")
		}
		dir, err := os.MkdirTemp(d.Config.DataDir, ".download-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		out := filepath.Join(dir, "backup.tgz")
		if _, err := Backup(r.Context(), d.Config.DataDir, d.Config.KeyFile, out, func() bool {
			m.Flush(r.Context())
			return true
		}); err != nil {
			return err
		}
		f, err := os.Open(out)
		if err != nil {
			return err
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			return err
		}
		d.Log.Info("backup downloaded", "user", p.UserID, "bytes", st.Size())
		name := "form-md-backup-" + time.Now().Format("2006-01-02-150405") + ".tgz"
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		w.Header().Set("Content-Length", strconv.FormatInt(st.Size(), 10))
		w.Header().Set("Cache-Control", "no-store")
		_, err = io.Copy(w, f)
		return err
	}))
	return nil
}
