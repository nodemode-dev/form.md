package transfer

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection"
	"github.com/nodemode-ai/form.md/internal/doc"
	"github.com/nodemode-ai/form.md/internal/files"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/page"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/site"
	"github.com/nodemode-ai/form.md/internal/store"
)

const (
	maxUpload   = 512 << 20
	maxRunning  = 2
	jobLifetime = 2 * time.Hour
	jobTimeout  = 45 * time.Minute
)

type Service struct {
	st    *store.Store
	auth  *auth.Service
	gate  *policy.Gate
	pages *page.Service
	docs  *doc.Service
	cols  *collection.Service
	files *files.Service
	site  *site.Service
	log   *slog.Logger
	dir   string

	mu   sync.Mutex
	jobs map[string]*Job
	sem  chan struct{}
}

func New(st *store.Store, a *auth.Service, gate *policy.Gate, pages *page.Service, docs *doc.Service, cols *collection.Service, fs *files.Service, sites *site.Service, log *slog.Logger, dataDir string) *Service {
	dir := filepath.Join(dataDir, "tmp")
	_ = os.MkdirAll(dir, 0o700)
	return &Service{st: st, auth: a, gate: gate, pages: pages, docs: docs, cols: cols, files: fs, site: sites, log: log, dir: dir, jobs: map[string]*Job{}, sem: make(chan struct{}, maxRunning)}
}

type Job struct {
	mu          *sync.Mutex
	ID          string   `json:"id"`
	UserID      string   `json:"-"`
	WorkspaceID string   `json:"workspace_id"`
	ParentID    string   `json:"parent_id"`
	Name        string   `json:"name"`
	Format      string   `json:"format"`
	State       string   `json:"state"`
	Done        int      `json:"done"`
	Total       int      `json:"total"`
	Pages       int      `json:"pages"`
	Rows        int      `json:"rows"`
	Files       int      `json:"files"`
	RootIDs     []string `json:"root_ids"`
	Warnings    []string `json:"warnings"`
	Error       string   `json:"error,omitempty"`
	CreatedAt   int64    `json:"created_at"`
	FinishedAt  int64    `json:"finished_at,omitempty"`
	cancel      context.CancelFunc
}

func (j *Job) setState(s string) {
	j.mu.Lock()
	j.State = s
	j.mu.Unlock()
}

func (j *Job) setTotal(n int) {
	j.mu.Lock()
	j.Total = n
	j.mu.Unlock()
}

func (j *Job) addPage(id string, root bool) {
	j.mu.Lock()
	if root {
		j.RootIDs = append(j.RootIDs, id)
	}
	j.mu.Unlock()
}

func (j *Job) progress(pages, rows int) {
	j.mu.Lock()
	j.Done += pages
	j.Pages += pages - rows
	j.Rows += rows
	j.mu.Unlock()
}

func (j *Job) files() {
	j.mu.Lock()
	j.Files++
	j.mu.Unlock()
}

func (j *Job) snapshot() Job {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := Job{ID: j.ID, WorkspaceID: j.WorkspaceID, ParentID: j.ParentID, Name: j.Name, Format: j.Format, State: j.State, Done: j.Done, Total: j.Total, Pages: j.Pages, Rows: j.Rows, Files: j.Files, Error: j.Error, CreatedAt: j.CreatedAt, FinishedAt: j.FinishedAt}
	out.RootIDs = append([]string{}, j.RootIDs...)
	out.Warnings = append([]string{}, j.Warnings...)
	return out
}

type upload struct {
	name string
	path string
	size int64
}

func (s *Service) prune() {
	for id, j := range s.jobs {
		j.mu.Lock()
		old := j.FinishedAt > 0 && time.Since(time.UnixMilli(j.FinishedAt)) > jobLifetime
		j.mu.Unlock()
		if old {
			delete(s.jobs, id)
		}
	}
}

var formats = map[string]bool{"": true, "auto": true, "notion": true, "evernote": true, "markdown": true, "html": true, "confluence": true, "text": true, "opml": true, "trello": true, "asana": true, "csv": true, "docx": true, "xlsx": true, "pdf": true}

func (s *Service) target(ctx context.Context, p *policy.Principal, workspaceID, parentID string) (string, error) {
	if parentID != "" {
		m, err := s.pages.Writable(ctx, p, parentID)
		if err != nil {
			return "", err
		}
		if m.Kind == "collection" {
			return "", httpx.Fail(http.StatusBadRequest, "bad_parent", "import into a page, not a database")
		}
		return m.WorkspaceID, nil
	}
	a, err := s.gate.Workspace(ctx, p, workspaceID)
	if err != nil {
		return "", err
	}
	if !a.CanWrite() {
		return "", httpx.ErrForbidden
	}
	return workspaceID, nil
}

func (s *Service) Start(r *http.Request, p *policy.Principal) (*Job, error) {
	if !p.CanWrite() {
		return nil, httpx.ErrReadOnly
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_upload", "send the files as multipart form data")
	}
	fields := map[string]string{}
	var ups []upload
	cleanup := func() {
		for _, u := range ups {
			os.Remove(u.path)
		}
	}
	var total int64
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cleanup()
			return nil, uploadErr(err)
		}
		if part.FormName() != "file" {
			b, _ := io.ReadAll(io.LimitReader(part, 400))
			fields[part.FormName()] = strings.TrimSpace(string(b))
			part.Close()
			continue
		}
		u, n, err := s.save(part, maxUpload-total)
		part.Close()
		if err != nil {
			cleanup()
			return nil, err
		}
		total += n
		ups = append(ups, u)
		if len(ups) > 100 {
			cleanup()
			return nil, httpx.Fail(http.StatusBadRequest, "too_many_files", "import up to 100 files at once")
		}
	}
	if len(ups) == 0 {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_upload", "choose at least one file")
	}
	format := strings.ToLower(fields["format"])
	if !formats[format] {
		cleanup()
		return nil, httpx.Fail(http.StatusBadRequest, "bad_format", "unknown import format")
	}
	ws, err := s.target(r.Context(), p, fields["workspace_id"], fields["parent_id"])
	if err != nil {
		cleanup()
		return nil, err
	}
	name := ups[0].name
	if len(ups) > 1 {
		name += " and " + itoa(len(ups)-1) + " more"
	}
	job := &Job{mu: &sync.Mutex{}, ID: ids.New(), UserID: p.UserID, WorkspaceID: ws, ParentID: fields["parent_id"], Name: name, Format: format, State: "queued", CreatedAt: store.NowMS(), RootIDs: []string{}, Warnings: []string{}}
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	job.cancel = cancel
	s.mu.Lock()
	s.prune()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	run := *p
	go s.run(ctx, &run, job, ups)
	return job, nil
}

func uploadErr(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", "imports can be up to 512 MB")
	}
	return httpx.Fail(http.StatusBadRequest, "bad_upload", "the upload could not be read")
}

func (s *Service) save(part *multipart.Part, room int64) (upload, int64, error) {
	name := path.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	if name == "" || name == "." || name == "/" {
		name = "import"
	}
	tmp, err := os.CreateTemp(s.dir, ".import-*")
	if err != nil {
		return upload{}, 0, err
	}
	n, err := io.Copy(tmp, io.LimitReader(part, room+1))
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		return upload{}, 0, uploadErr(err)
	}
	if n > room {
		os.Remove(tmp.Name())
		return upload{}, 0, httpx.Fail(http.StatusRequestEntityTooLarge, "too_large", "imports can be up to 512 MB")
	}
	return upload{name: name, path: tmp.Name(), size: n}, n, nil
}

func (s *Service) run(ctx context.Context, p *policy.Principal, job *Job, ups []upload) {
	v := newVFS(s.dir)
	defer func() {
		v.close()
		for _, u := range ups {
			os.Remove(u.path)
		}
		job.cancel()
		if r := recover(); r != nil {
			s.log.Error("import crashed", "job", job.ID, "panic", r)
			job.mu.Lock()
			job.State, job.Error, job.FinishedAt = "failed", "the import stopped unexpectedly", store.NowMS()
			job.mu.Unlock()
		}
	}()
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		s.finish(job, nil, ctx.Err())
		return
	}
	job.setState("reading")
	for _, u := range ups {
		if err := v.addFile(u.name, u.path, u.size); err != nil {
			s.finish(job, nil, err)
			return
		}
	}
	b := &Bundle{}
	kind := readBundle(v, b, job.Format)
	job.mu.Lock()
	if job.Format == "" || job.Format == "auto" {
		job.Format = kind
	}
	job.mu.Unlock()
	if len(b.Roots) == 0 {
		if len(b.warnings) == 0 {
			b.Warn("Nothing in the upload could be imported")
		}
		s.finish(job, b, errors.New("nothing to import"))
		return
	}
	err := s.materialise(ctx, p, job, b)
	s.finish(job, b, err)
}

func (s *Service) finish(job *Job, b *Bundle, err error) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if b != nil {
		job.Warnings = append(job.Warnings, b.warnings...)
	}
	job.FinishedAt = store.NowMS()
	switch {
	case err == nil:
		job.State = "done"
	case errors.Is(err, context.Canceled):
		job.State, job.Error = "cancelled", "the import was cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		job.State, job.Error = "failed", "the import took too long"
	default:
		job.State, job.Error = "failed", err.Error()
	}
}

func readBundle(v *vfs, b *Bundle, format string) string {
	files := v.sorted()
	notion, docs, conf := false, 0, false
	for _, f := range files {
		base := path.Base(f.path)
		if notionID.MatchString(stripExt(base)) || notionID.MatchString(strings.TrimSuffix(stripExt(base), "_all")) {
			notion = true
		}
		switch extOf(f.path) {
		case ".md", ".markdown", ".mdown", ".html", ".htm", ".txt", ".csv":
			docs++
		}
	}
	zipped := false
	for _, f := range files {
		if strings.Contains(f.path, "/") {
			zipped = true
			break
		}
	}
	if docs > 0 && (format == "confluence" || ((format == "" || format == "auto" || format == "html") && looksLikeConfluence(v))) {
		conf = true
	}
	kind := format
	treeDone := false
	switch {
	case format == "notion" || ((format == "" || format == "auto") && notion && docs > 0):
		importTree(v, b, true)
		kind, treeDone = "notion", true
	case conf:
		importConfluence(v, b)
		kind, treeDone = "confluence", true
	case docs > 1 || (docs > 0 && zipped):
		importTree(v, b, false)
		treeDone = true
		if kind == "" || kind == "auto" {
			kind = "markdown"
		}
	}
	for _, f := range files {
		e := extOf(f.path)
		if treeDone && (docExt(e) || e == ".csv") {
			continue
		}
		name := path.Base(f.path)
		switch e {
		case ".enex":
			rc, err := f.open()
			if err != nil {
				continue
			}
			importENEX(name, rc, b)
			rc.Close()
			setKind(&kind, "evernote")
			continue
		}
		known := map[string]bool{".opml": true, ".json": true, ".csv": true, ".tsv": true, ".docx": true, ".xlsx": true, ".pdf": true, ".md": true, ".markdown": true, ".html": true, ".htm": true, ".txt": true, ".xml": true}
		if !known[e] {
			if !treeDone || !isImageName(f.path) {
				b.Warn("Skipped " + f.path + ": this kind of file is not imported")
			}
			continue
		}
		data, err := v.read(f)
		if err != nil {
			b.Warn("Could not read " + f.path)
			continue
		}
		switch e {
		case ".opml":
			importOPML(name, data, b)
			setKind(&kind, "opml")
		case ".json":
			if looksLikeTrello(data) {
				importTrello(name, data, b)
				setKind(&kind, "trello")
			} else {
				b.Warn("Skipped " + name + ": only Trello board exports are read from JSON")
			}
		case ".csv", ".tsv":
			importCSVFile(name, data, b)
			setKind(&kind, "csv")
		case ".docx":
			importDOCX(name, data, b)
			setKind(&kind, "docx")
		case ".xlsx":
			importXLSX(name, data, b)
			setKind(&kind, "xlsx")
		case ".pdf":
			importPDF(name, data, b)
			setKind(&kind, "pdf")
		case ".md", ".markdown":
			importMarkdownFile(name, data, b)
			setKind(&kind, "markdown")
		case ".html", ".htm":
			importHTMLFile(name, data, b)
			setKind(&kind, "html")
		case ".txt":
			importText(name, data, b)
			setKind(&kind, "text")
		case ".xml":
			if looksLikeENEX(data[:min(len(data), 4096)]) {
				importENEX(name, bytes.NewReader(data), b)
				setKind(&kind, "evernote")
			} else {
				b.Warn("Skipped " + name + ": unknown XML")
			}
		}
	}
	return kind
}

func setKind(kind *string, k string) {
	if *kind == "" || *kind == "auto" {
		*kind = k
	}
}

func (s *Service) Job(p *policy.Principal, id string) (*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok || j.UserID != p.UserID {
		return nil, httpx.ErrNotFound
	}
	return j, nil
}

func (s *Service) List(p *policy.Principal, workspaceID string) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Job{}
	for _, j := range s.jobs {
		if j.UserID == p.UserID && (workspaceID == "" || j.WorkspaceID == workspaceID) {
			out = append(out, j.snapshot())
		}
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt > out[k].CreatedAt })
	return out
}

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "transfer" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	route := func(pattern string, h func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error) {
		r.HandleFunc(pattern, s.auth.Require(h))
	}
	route("POST /api/imports", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		r.Body = http.MaxBytesReader(w, r.Body, maxUpload+(4<<20))
		j, err := s.Start(r, p)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusAccepted, map[string]any{"job": j.snapshot()})
		return nil
	})
	route("GET /api/imports", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		httpx.JSON(w, http.StatusOK, map[string]any{"jobs": s.List(p, r.URL.Query().Get("workspace_id"))})
		return nil
	})
	route("GET /api/imports/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		j, err := s.Job(p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"job": j.snapshot()})
		return nil
	})
	route("DELETE /api/imports/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		j, err := s.Job(p, r.PathValue("id"))
		if err != nil {
			return err
		}
		j.cancel()
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	mountExports(r, s, route)
	return nil
}
