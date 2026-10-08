package files

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/netguard"
	"github.com/nodemode-ai/form.md/internal/policy"
)

type ImportOptions struct {
	Purpose     string
	WorkspaceID string
	PageID      string
	Limit       int64
	ImagesOnly  bool
}

func (s *Service) Import(ctx context.Context, p *policy.Principal, raw string, o ImportOptions) (*File, error) {
	if s.fetch == nil {
		return nil, httpx.Fail(http.StatusServiceUnavailable, "fetch_off", "fetching links is not available")
	}
	u, err := netguard.CheckURL(raw)
	if err != nil {
		code, msg := netguard.Reason(err)
		return nil, httpx.Fail(http.StatusBadRequest, code, msg)
	}
	if _, err := s.authorize(ctx, p, o.Purpose, o.WorkspaceID, o.PageID); err != nil {
		return nil, err
	}
	limit := int64(s.settings.Int("files.max_upload_mb")) << 20
	if o.Limit > 0 && o.Limit < limit {
		limit = o.Limit
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, httpx.Fail(http.StatusBadRequest, "bad_url", "use an http or https link")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; form.md/"+app.Version+")")
	req.Header.Set("Accept", "image/avif,image/webp,image/*,video/*,audio/*,application/pdf,*/*;q=0.5")
	resp, err := s.fetch.Do(req)
	if err != nil {
		code, msg := netguard.Reason(err)
		if code == "private_address" || code == "bad_url" {
			return nil, httpx.Fail(http.StatusBadRequest, code, msg)
		}
		return nil, httpx.Fail(http.StatusBadGateway, code, msg)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, httpx.Fail(http.StatusBadGateway, "fetch_failed", fmt.Sprintf("the link answered %d", resp.StatusCode))
	}
	if resp.ContentLength > limit {
		return nil, uploadError(&http.MaxBytesError{Limit: limit}, limit)
	}
	ctype, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if ctype == "text/html" || ctype == "application/xhtml+xml" {
		return nil, httpx.Fail(http.StatusUnprocessableEntity, "not_media", "the link points to a web page, not a file")
	}
	if o.ImagesOnly && !strings.HasPrefix(ctype, "image/") && ctype != "" && ctype != "application/octet-stream" {
		return nil, httpx.Fail(http.StatusUnprocessableEntity, "not_an_image", "the link does not point to an image")
	}
	f, err := s.store(ctx, p, resp.Body, map[string]string{"purpose": o.Purpose, "workspace_id": o.WorkspaceID, "page_id": o.PageID, "name": remoteName(resp, ctype)}, limit)
	if err != nil {
		return nil, err
	}
	if o.ImagesOnly && !strings.HasPrefix(f.Mime, "image/") {
		return nil, httpx.Fail(http.StatusUnprocessableEntity, "not_an_image", "the link does not point to an image")
	}
	if strings.HasPrefix(f.Mime, "image/") {
		f.W, f.H = s.dims(f)
	}
	return f, nil
}

func remoteName(resp *http.Response, ctype string) string {
	if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); err == nil && params["filename"] != "" {
		return params["filename"]
	}
	name := path.Base(resp.Request.URL.Path)
	if name == "/" || name == "." || name == "" {
		name = resp.Request.URL.Hostname()
	}
	if cleanExt(name) == "" && ctype != "" {
		if exts, _ := mime.ExtensionsByType(ctype); len(exts) > 0 {
			name += exts[len(exts)-1]
		}
	}
	return name
}

func (s *Service) dims(f *File) (int, int) {
	fh, err := os.Open(s.path(f.ID, f.ext))
	if err != nil {
		return 0, 0
	}
	defer fh.Close()
	if f.Mime == "image/webp" {
		return webpDims(fh)
	}
	c, _, err := image.DecodeConfig(fh)
	if err != nil {
		return 0, 0
	}
	return c.Width, c.Height
}

func webpDims(r io.Reader) (int, int) {
	b := make([]byte, 30)
	if _, err := io.ReadFull(r, b); err != nil || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0
	}
	switch string(b[12:16]) {
	case "VP8 ":
		return int(b[26]) | int(b[27]&0x3f)<<8, int(b[28]) | int(b[29]&0x3f)<<8
	case "VP8L":
		bits := uint32(b[21]) | uint32(b[22])<<8 | uint32(b[23])<<16 | uint32(b[24])<<24
		return int(bits&0x3fff) + 1, int(bits>>14&0x3fff) + 1
	case "VP8X":
		return 1 + (int(b[24]) | int(b[25])<<8 | int(b[26])<<16), 1 + (int(b[27]) | int(b[28])<<8 | int(b[29])<<16)
	}
	return 0, 0
}

func (s *Service) importHandler(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
	var in struct {
		URL         string `json:"url"`
		Purpose     string `json:"purpose"`
		WorkspaceID string `json:"workspace_id"`
		PageID      string `json:"page_id"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if in.Purpose == "" {
		in.Purpose = "attachment"
	}
	if in.Purpose != "attachment" && in.Purpose != "cover" && in.Purpose != "icon" {
		return httpx.Fail(http.StatusBadRequest, "bad_purpose", "purpose is attachment, cover or icon")
	}
	f, err := s.Import(r.Context(), p, in.URL, ImportOptions{Purpose: in.Purpose, WorkspaceID: in.WorkspaceID, PageID: in.PageID, ImagesOnly: in.Purpose != "attachment"})
	if err != nil {
		return err
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"file": f})
	return nil
}
