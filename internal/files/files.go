package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

var imageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

var inlineTypes = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "image/svg+xml": true,
	"application/pdf": true, "audio/mpeg": true, "audio/ogg": true, "audio/wav": true, "video/mp4": true, "video/webm": true,
	"text/plain; charset=utf-8": true,
}

type File struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	PageID      string `json:"page_id"`
	OwnerID     string `json:"owner_id"`
	Purpose     string `json:"purpose"`
	Name        string `json:"name"`
	Mime        string `json:"mime"`
	Size        int64  `json:"size"`
	CreatedAt   int64  `json:"created_at"`
	URL         string `json:"url"`
	Ref         string `json:"ref"`
	W           int    `json:"w,omitempty"`
	H           int    `json:"h,omitempty"`
	ext         string
}

type Service struct {
	st       *store.Store
	gate     *policy.Gate
	pages    *page.Service
	settings *settings.Service
	audit    *audit.Log
	log      *slog.Logger
	dir      string
	fetch    *http.Client
}

func (s *Service) UseFetcher(c *http.Client) {
	s.fetch = c
}

func New(st *store.Store, gate *policy.Gate, pages *page.Service, set *settings.Service, al *audit.Log, log *slog.Logger, dataDir string) (*Service, error) {
	dir := filepath.Join(dataDir, "files")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Service{st: st, gate: gate, pages: pages, settings: set, audit: al, log: log, dir: dir}, nil
}

func (s *Service) path(id, ext string) string {
	return filepath.Join(s.dir, id+ext)
}

func cleanName(raw string) string {
	name := filepath.Base(strings.ReplaceAll(raw, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' {
			return -1
		}
		return r
	}, name)
	if name == "." || name == "/" || name == "" {
		name = "file"
	}
	if utf8.RuneCountInString(name) > 200 {
		name = string([]rune(name)[:200])
	}
	return name
}

func cleanExt(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	var b strings.Builder
	for _, r := range ext {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		}
	}
	e := b.String()
	if len(e) < 2 || len(e) > 12 {
		return ""
	}
	return e
}

func sniff(head []byte, ext string) string {
	t := http.DetectContentType(head)
	if ext == ".svg" && (bytes.Contains(head, []byte("<svg")) || strings.HasPrefix(t, "text/xml")) {
		return "image/svg+xml"
	}
	if t == "application/octet-stream" {
		if guess := mime.TypeByExtension(ext); guess != "" {
			return guess
		}
	}
	return t
}

type target struct {
	workspaceID string
	pageID      string
}

func (s *Service) authorize(ctx context.Context, p *policy.Principal, purpose, workspaceID, pageID string) (target, error) {
	switch purpose {
	case "avatar":
		if !p.IsSession() {
			return target{}, httpx.ErrSessionOnly
		}
		return target{}, nil
	case "workspace":
		a, err := s.gate.Workspace(ctx, p, workspaceID)
		if err != nil {
			return target{}, err
		}
		if !a.IsAdmin() {
			return target{}, httpx.ErrForbidden
		}
		return target{workspaceID: workspaceID}, nil
	case "attachment", "cover", "icon":
		if pageID != "" {
			ws, err := s.pages.CanWrite(ctx, p, pageID)
			if err != nil {
				return target{}, err
			}
			return target{workspaceID: ws, pageID: pageID}, nil
		}
		a, err := s.gate.Workspace(ctx, p, workspaceID)
		if err != nil {
			return target{}, err
		}
		if !a.CanWrite() {
			return target{}, httpx.ErrForbidden
		}
		return target{workspaceID: workspaceID}, nil
	}
	return target{}, httpx.Fail(http.StatusBadRequest, "bad_purpose", "purpose is attachment, cover, icon, avatar or workspace")
}

func (s *Service) Upload(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	limit := int64(s.settings.Int("files.max_upload_mb")) << 20
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		return httpx.Fail(http.StatusBadRequest, "bad_upload", "send the file as multipart form data")
	}
	fields := map[string]string{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			return httpx.Fail(http.StatusBadRequest, "bad_upload", "the file part is missing")
		}
		if err != nil {
			return uploadError(err, limit)
		}
		if part.FormName() != "file" {
			b, _ := io.ReadAll(io.LimitReader(part, 200))
			fields[part.FormName()] = strings.TrimSpace(string(b))
			continue
		}
		if fields["name"] == "" {
			fields["name"] = part.FileName()
		}
		f, err := s.store(r.Context(), p, part, fields, limit)
		part.Close()
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"file": f})
		return nil
	}
}

func uploadError(err error, limit int64) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", fmt.Sprintf("files can be up to %d MB", limit>>20)).With("limit_mb", limit>>20)
	}
	return httpx.Fail(http.StatusBadRequest, "bad_upload", "the upload could not be read")
}

func (s *Service) store(ctx context.Context, p *policy.Principal, part io.Reader, fields map[string]string, limit int64) (*File, error) {
	purpose := fields["purpose"]
	if purpose == "" {
		purpose = "attachment"
	}
	t, err := s.authorize(ctx, p, purpose, fields["workspace_id"], fields["page_id"])
	if err != nil {
		return nil, err
	}
	name := cleanName(fields["name"])
	ext := cleanExt(name)
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return nil, err
	}
	keep := false
	defer func() {
		tmp.Close()
		if !keep {
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(part, limit+1))
	if err != nil {
		return nil, uploadError(err, limit)
	}
	if n > limit {
		return nil, uploadError(&http.MaxBytesError{Limit: limit}, limit)
	}
	if n == 0 {
		return nil, httpx.Fail(http.StatusBadRequest, "empty_file", "the file is empty")
	}
	head := make([]byte, 512)
	m, _ := tmp.ReadAt(head, 0)
	ctype := sniff(head[:m], ext)
	if purpose != "attachment" {
		if e, ok := imageTypes[ctype]; ok {
			ext = e
		} else if ctype == "image/svg+xml" && (purpose == "icon" || purpose == "workspace") {
			ext = ".svg"
		} else {
			return nil, httpx.Fail(http.StatusBadRequest, "not_an_image", "use a PNG, JPEG, GIF or WebP image")
		}
	}
	id := ids.New()
	if err := tmp.Sync(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp.Name(), s.path(id, ext)); err != nil {
		return nil, err
	}
	keep = true
	now := store.NowMS()
	sum := hex.EncodeToString(h.Sum(nil))
	err = s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO files (id, workspace_id, page_id, owner_id, purpose, name, ext, mime, size, sha256, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, nullable(t.workspaceID), nullable(t.pageID), p.UserID, purpose, name, ext, ctype, n, sum, now); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, p, audit.Entry{Action: "file.upload", WorkspaceID: t.workspaceID, PageID: t.pageID, Detail: name})
	})
	if err != nil {
		os.Remove(s.path(id, ext))
		return nil, err
	}
	return &File{ID: id, WorkspaceID: t.workspaceID, PageID: t.pageID, OwnerID: p.UserID, Purpose: purpose, Name: name, Mime: ctype, Size: n, CreatedAt: now, URL: "/files/" + id, Ref: "f:" + id, ext: ext}, nil
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func (s *Service) lookup(ctx context.Context, id string) (*File, error) {
	var f File
	var ws, pg, owner sql.NullString
	err := s.st.R.QueryRowContext(ctx, `SELECT id, workspace_id, page_id, owner_id, purpose, name, ext, mime, size, created_at FROM files WHERE id = ?`, id).
		Scan(&f.ID, &ws, &pg, &owner, &f.Purpose, &f.Name, &f.ext, &f.Mime, &f.Size, &f.CreatedAt)
	if err != nil {
		return nil, store.NoRows(err)
	}
	f.WorkspaceID, f.PageID, f.OwnerID = ws.String, pg.String, owner.String
	f.URL, f.Ref = "/files/"+f.ID, "f:"+f.ID
	return &f, nil
}

func (s *Service) canRead(ctx context.Context, p *policy.Principal, f *File) error {
	if f.Purpose == "avatar" {
		return nil
	}
	if f.PageID != "" {
		return s.pages.CanRead(ctx, p, f.PageID)
	}
	if f.WorkspaceID == "" {
		return policy.ErrHidden
	}
	_, err := s.gate.Workspace(ctx, p, f.WorkspaceID)
	return err
}

func (s *Service) Serve(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	f, err := s.lookup(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}
	if err := s.canRead(r.Context(), p, f); err != nil {
		return err
	}
	fh, err := os.Open(s.path(f.ID, f.ext))
	if errors.Is(err, os.ErrNotExist) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	defer fh.Close()
	hdr := w.Header()
	hdr.Set("Content-Type", f.Mime)
	hdr.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; sandbox")
	hdr.Set("Cache-Control", "private, max-age=31536000, immutable")
	hdr.Set("ETag", `"`+f.ID+`"`)
	disposition := "attachment"
	if inlineTypes[f.Mime] {
		disposition = "inline"
	}
	hdr.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": f.Name}))
	http.ServeContent(w, r, "", time.UnixMilli(f.CreatedAt), fh)
	return nil
}

func (s *Service) List(ctx context.Context, p *policy.Principal, workspaceID string) ([]File, error) {
	if _, err := s.gate.Workspace(ctx, p, workspaceID); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT id, COALESCE(page_id, ''), COALESCE(owner_id, ''), purpose, name, mime, size, created_at
		FROM files WHERE workspace_id = ? ORDER BY created_at DESC LIMIT 1000`, workspaceID)
	if err != nil {
		return nil, err
	}
	var all []File
	for rows.Next() {
		f := File{WorkspaceID: workspaceID}
		if err := rows.Scan(&f.ID, &f.PageID, &f.OwnerID, &f.Purpose, &f.Name, &f.Mime, &f.Size, &f.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		f.URL, f.Ref = "/files/"+f.ID, "f:"+f.ID
		all = append(all, f)
	}
	rows.Close()
	out := []File{}
	for _, f := range all {
		if f.PageID == "" || s.pages.CanRead(ctx, p, f.PageID) == nil {
			out = append(out, f)
		}
	}
	return out, nil
}

func (s *Service) Owned(ctx context.Context, ref, ownerID, purpose string) bool {
	id, ok := strings.CutPrefix(ref, "f:")
	if !ok {
		return false
	}
	var n int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE id = ? AND owner_id = ? AND purpose = ?", id, ownerID, purpose).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func (s *Service) InWorkspace(ctx context.Context, ref, workspaceID, purpose string) bool {
	id, ok := strings.CutPrefix(ref, "f:")
	if !ok {
		return false
	}
	var n int
	if err := s.st.R.QueryRowContext(ctx, "SELECT COUNT(*) FROM files WHERE id = ? AND workspace_id = ? AND purpose = ?", id, workspaceID, purpose).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func (s *Service) Sweep(ctx context.Context) {
	day := store.NowMS() - int64(24*time.Hour/time.Millisecond)
	_, err := s.st.W.ExecContext(ctx, `DELETE FROM files WHERE created_at < ? AND (
			(purpose IN ('attachment', 'cover', 'icon') AND page_id IS NULL AND workspace_id IS NOT NULL
				AND NOT EXISTS (SELECT 1 FROM pages WHERE cover = 'f:' || files.id OR icon = 'f:' || files.id))
			OR (purpose = 'avatar' AND NOT EXISTS (SELECT 1 FROM users WHERE avatar = 'f:' || files.id))
			OR (purpose = 'workspace' AND NOT EXISTS (SELECT 1 FROM workspaces WHERE image = 'f:' || files.id OR icon = 'f:' || files.id)))`, day)
	if err != nil {
		s.log.Warn("files: sweep rows", "err", err)
		return
	}
	known := map[string]bool{}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id || ext FROM files")
	if err != nil {
		s.log.Warn("files: sweep list", "err", err)
		return
	}
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			known[name] = true
		}
	}
	rows.Close()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-time.Hour)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || known[e.Name()] || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, e.Name())); err != nil {
			s.log.Warn("files: sweep remove", "err", err)
		}
	}
}

type module struct {
	s    *Service
	auth *auth.Service
}

func (s *Service) Module(a *auth.Service) app.Module { return module{s, a} }

func (module) ID() string { return "files" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	r.HandleFunc("POST /api/files", a.Require(s.Upload))
	r.HandleFunc("POST /api/files/import", a.Require(s.importHandler))
	r.HandleFunc("GET /files/{id}", a.Require(s.Serve))
	r.HandleFunc("GET /api/workspaces/{id}/files", a.Require(func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.List(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"files": list})
		return nil
	}))
	return nil
}
