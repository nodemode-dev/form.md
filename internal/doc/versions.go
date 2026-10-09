package doc

import (
	"bytes"
	"compress/flate"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	versionEvery   = 10 * time.Minute
	versionsPerDoc = 500
	maxVersionRaw  = 64 << 20
)

type Version struct {
	ID        string   `json:"id"`
	PageID    string   `json:"page_id"`
	CreatedAt int64    `json:"created_at"`
	Kind      string   `json:"kind"`
	Title     string   `json:"title"`
	Editors   []string `json:"editors"`
	Names     []string `json:"names"`
	Words     int      `json:"words"`
	Size      int      `json:"size"`
}

func deflate(b []byte) []byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestSpeed)
	_, _ = w.Write(b)
	_ = w.Close()
	return buf.Bytes()
}

func inflate(b []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(b))
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, maxVersionRaw+1))
	if err != nil {
		return nil, err
	}
	if len(out) > maxVersionRaw {
		return nil, errors.New("doc: version too large")
	}
	return out, nil
}

func (s *Service) lastVersionAt(ctx context.Context, id string) int64 {
	var at sql.NullInt64
	_ = s.st.R.QueryRowContext(ctx, "SELECT MAX(created_at) FROM page_versions WHERE page_id = ?", id).Scan(&at)
	return at.Int64
}

func (s *Service) saveVersion(ctx context.Context, r *room, kind string, force bool) (string, error) {
	r.mu.Lock()
	now := store.NowMS()
	if !r.changedSinceVersion && !(force && kind != "auto") {
		r.mu.Unlock()
		return "", nil
	}
	if !force && r.lastVersion > 0 && now-r.lastVersion < versionEvery.Milliseconds() {
		r.mu.Unlock()
		return "", nil
	}
	if !r.changedSinceVersion && r.lastVersionID != "" {
		id := r.lastVersionID
		r.mu.Unlock()
		return id, nil
	}
	state := r.doc.EncodeStateAsUpdate()
	sum := r.summary(nil)
	editors := make([]string, 0, len(r.editors))
	for u := range r.editors {
		editors = append(editors, u)
	}
	id := ids.New()
	r.editors = map[string]bool{}
	r.changedSinceVersion = false
	r.lastVersion = now
	r.lastVersionID = id
	r.mu.Unlock()
	var title string
	_ = s.st.R.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", r.id).Scan(&title)
	ed, _ := json.Marshal(editors)
	err := s.st.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO page_versions (id, page_id, created_at, kind, title, editors, words, size, state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, r.id, now, kind, title, string(ed), len(strings.Fields(sum.Text)), len(state), deflate(state)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM page_versions WHERE page_id = ? AND id NOT IN (SELECT id FROM page_versions WHERE page_id = ? ORDER BY created_at DESC LIMIT ?)`, r.id, r.id, versionsPerDoc)
		return err
	})
	if err != nil && strings.Contains(err.Error(), "FOREIGN KEY") {
		return "", nil
	}
	return id, err
}

func (s *Service) Versions(ctx context.Context, p *policy.Principal, pageID string) ([]Version, error) {
	if err := s.pages.CanRead(ctx, p, pageID); err != nil {
		return nil, err
	}
	rows, err := s.st.R.QueryContext(ctx, "SELECT id, page_id, created_at, kind, title, editors, words, size FROM page_versions WHERE page_id = ? ORDER BY created_at DESC LIMIT 200", pageID)
	if err != nil {
		return nil, err
	}
	out := []Version{}
	for rows.Next() {
		var v Version
		var ed string
		if err := rows.Scan(&v.ID, &v.PageID, &v.CreatedAt, &v.Kind, &v.Title, &ed, &v.Words, &v.Size); err != nil {
			rows.Close()
			return nil, err
		}
		_ = json.Unmarshal([]byte(ed), &v.Editors)
		out = append(out, v)
	}
	rows.Close()
	names := map[string]string{}
	for i := range out {
		out[i].Names = []string{}
		for _, u := range out[i].Editors {
			n, ok := names[u]
			if !ok {
				_ = s.st.R.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", u).Scan(&n)
				names[u] = n
			}
			if n != "" {
				out[i].Names = append(out[i].Names, n)
			}
		}
	}
	return out, rows.Err()
}

func (s *Service) VersionState(ctx context.Context, p *policy.Principal, id string) (*Version, []byte, error) {
	var v Version
	var blob []byte
	err := s.st.R.QueryRowContext(ctx, "SELECT id, page_id, created_at, kind, title, words, size, state FROM page_versions WHERE id = ?", id).
		Scan(&v.ID, &v.PageID, &v.CreatedAt, &v.Kind, &v.Title, &v.Words, &v.Size, &blob)
	if err != nil {
		return nil, nil, store.NoRows(err)
	}
	if err := s.pages.CanRead(ctx, p, v.PageID); err != nil {
		return nil, nil, err
	}
	state, err := inflate(blob)
	if err != nil {
		return nil, nil, err
	}
	return &v, state, nil
}

func (s *Service) SaveVersion(ctx context.Context, p *policy.Principal, pageID string, beforeRestore bool) (string, error) {
	ws, err := s.pages.CanWrite(ctx, p, pageID)
	if err != nil {
		return "", err
	}
	r, err := s.acquire(ctx, pageID)
	if err != nil {
		return "", err
	}
	defer s.release(r)
	kind := "manual"
	if beforeRestore {
		kind = "restore"
	}
	id, err := s.saveVersion(ctx, r, kind, true)
	if err != nil {
		return "", err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, nil, p, audit.Entry{Action: "page.save_version", WorkspaceID: ws, PageID: pageID, Detail: id})
	}
	return id, nil
}

func (s *Service) Restored(ctx context.Context, p *policy.Principal, versionID string) error {
	v, _, err := s.VersionState(ctx, p, versionID)
	if err != nil {
		return err
	}
	ws, err := s.pages.CanWrite(ctx, p, v.PageID)
	if err != nil {
		return err
	}
	if s.audit != nil {
		return s.audit.Record(ctx, nil, p, audit.Entry{Action: "page.restore_version", WorkspaceID: ws, PageID: v.PageID, Detail: versionID})
	}
	return nil
}

func (s *Service) PurgeVersions(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	res, err := s.st.W.ExecContext(ctx, "DELETE FROM page_versions WHERE created_at < ?", store.NowMS()-int64(days)*24*3600*1000)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Service) SetAudit(a *audit.Log) { s.audit = a }

func (m module) mountVersions(route func(string, func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error)) {
	s := m.s
	route("GET /api/pages/{id}/versions", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		list, err := s.Versions(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"versions": list})
		return nil
	})
	route("POST /api/pages/{id}/versions", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		id, err := s.SaveVersion(r.Context(), p, r.PathValue("id"), r.URL.Query().Get("kind") == "restore")
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
		return nil
	})
	route("GET /api/versions/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		v, state, err := s.VersionState(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"version": v, "state": base64.StdEncoding.EncodeToString(state)})
		return nil
	})
	route("POST /api/versions/{id}/restored", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.Restored(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}
