package collection

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/collection/formula"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

type module struct {
	s          *Service
	auth       *auth.Service
	trustProxy func() bool
}

func (s *Service) Module(a *auth.Service, trustProxy func() bool) app.Module {
	return module{s: s, auth: a, trustProxy: trustProxy}
}

func (module) ID() string { return "collection" }

type handler = func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error

func ok(w http.ResponseWriter, v any) error {
	httpx.JSON(w, http.StatusOK, v)
	return nil
}

func okz(w http.ResponseWriter, r *http.Request, v any) error {
	httpx.JSONZ(w, r, http.StatusOK, v)
	return nil
}

type RowDetail struct {
	Collection *Out           `json:"collection"`
	Row        *RowOut        `json:"row"`
	Refs       map[string]Ref `json:"refs"`
}

func (s *Service) Row(ctx context.Context, p *policy.Principal, rowID string) (*RowDetail, error) {
	col, err := s.RowCollection(ctx, rowID)
	if err != nil {
		return nil, err
	}
	if err := s.pages.CanRead(ctx, p, rowID); err != nil {
		return nil, err
	}
	out, err := s.Get(ctx, p, col)
	if err != nil {
		return nil, err
	}
	m, err := s.pages.Load(ctx, rowID)
	if err != nil {
		return nil, err
	}
	if m.TrashedAt > 0 {
		out.CanWrite = false
	}
	res, err := s.Query(ctx, p, col, QueryInput{IDs: []string{rowID}})
	if err != nil {
		return nil, err
	}
	if len(res.Rows) == 0 {
		return &RowDetail{Collection: out, Row: nil, Refs: map[string]Ref{}}, nil
	}
	return &RowDetail{Collection: out, Row: &res.Rows[0], Refs: res.Refs}, nil
}

type FormulaCheck struct {
	OK    bool     `json:"ok"`
	Error string   `json:"error,omitempty"`
	Pos   int      `json:"pos"`
	Props []string `json:"props,omitempty"`
}

func (s *Service) CheckFormula(ctx context.Context, p *policy.Principal, collectionID, expr string) (*FormulaCheck, error) {
	if _, _, err := s.readable(ctx, p, collectionID); err != nil {
		return nil, err
	}
	schema, err := s.loadSchema(ctx, s.st.R, collectionID)
	if err != nil {
		return nil, err
	}
	prog, err := formula.Compile(expr, schema)
	if err != nil {
		var fe *formula.Error
		if errors.As(err, &fe) {
			return &FormulaCheck{Error: fe.Msg, Pos: fe.Pos}, nil
		}
		return &FormulaCheck{Error: err.Error()}, nil
	}
	return &FormulaCheck{OK: true, Props: prog.Props()}, nil
}

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s, a := m.s, m.auth
	route := func(pattern string, h handler) { r.HandleFunc(pattern, a.Require(h)) }

	route("GET /api/collections/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		out, err := s.Get(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		body := map[string]any{"collection": out}
		if want, asked := r.URL.Query()["view"]; asked && len(out.Views) > 0 {
			v := out.Views[0]
			for _, x := range out.Views {
				if len(want) > 0 && x.ID == want[0] {
					v = x
				}
			}
			if v.Type == "table" || v.Type == "list" {
				if res, err := s.Query(r.Context(), p, out.ID, QueryInput{ViewID: v.ID, Limit: 100}); err == nil {
					body["first"] = map[string]any{"view_id": v.ID, "result": res}
				}
			}
		}
		return okz(w, r, body)
	})
	route("POST /api/collections/{id}/schema", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Ops []SchemaOp `json:"ops"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		schema, err := s.UpdateSchema(r.Context(), p, r.PathValue("id"), in.Ops)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"schema": schema, "epoch": s.epoch.Load()})
	})
	route("POST /api/collections/{id}/formula", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Expr string `json:"expr"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		out, err := s.CheckFormula(r.Context(), p, r.PathValue("id"), in.Expr)
		if err != nil {
			return err
		}
		return ok(w, out)
	})
	route("POST /api/collections/{id}/query", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in QueryInput
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		res, err := s.Query(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return okz(w, r, res)
	})
	route("GET /api/collections/{id}/search", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		hits, err := s.SearchRows(r.Context(), p, r.PathValue("id"), r.URL.Query().Get("q"), limit)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"rows": hits})
	})
	route("POST /api/collections/{id}/rows", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in RowInput
		if r.ContentLength != 0 {
			if err := httpx.Decode(r, &in); err != nil {
				return err
			}
		}
		row, err := s.CreateRow(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"row": row, "epoch": s.epoch.Load()})
		return nil
	})
	route("POST /api/collections/{id}/rows/delete", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			IDs []string `json:"ids"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		n, err := s.DeleteRows(r.Context(), p, r.PathValue("id"), in.IDs)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"trashed": n})
	})
	route("GET /api/rows/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		d, err := s.Row(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return okz(w, r, d)
	})
	route("PATCH /api/rows/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in RowPatch
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		row, err := s.UpdateRow(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"row": row, "epoch": s.epoch.Load()})
	})
	route("POST /api/collections/{id}/views", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ViewInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		v, err := s.CreateView(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"view": v})
		return nil
	})
	route("PATCH /api/views/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ViewInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		v, err := s.UpdateView(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"view": v})
	})
	route("DELETE /api/views/{id}", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		if err := s.DeleteView(r.Context(), p, r.PathValue("id")); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("POST /api/views/{id}/move", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			AfterID string `json:"after_id"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.MoveView(r.Context(), p, r.PathValue("id"), in.AfterID); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	route("POST /api/views/{id}/duplicate", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		v, err := s.DuplicateView(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"view": v})
		return nil
	})
	route("GET /api/views/{id}/export.csv", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var buf bytes.Buffer
		name, err := s.ExportCSV(r.Context(), p, r.PathValue("id"), &buf)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeName(name) + ".csv"}))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(buf.Bytes())
		return nil
	})
	route("POST /api/csv/preview", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		r.Body = http.MaxBytesReader(w, r.Body, maxCSVBytes+(1<<20))
		f, hdr, err := r.FormFile("file")
		if err != nil {
			return httpx.Fail(http.StatusBadRequest, "bad_upload", "send the file as multipart field file")
		}
		defer f.Close()
		pv, err := s.PreviewCSV(r.Context(), p, hdr.Filename, f)
		if err != nil {
			return err
		}
		return ok(w, pv)
	})
	route("POST /api/csv/import", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in ImportInput
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		id, n, err := s.ImportCSV(r.Context(), p, in)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"collection_id": id, "rows": n})
	})
	route("GET /api/views/{id}/form", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		def, err := s.Form(r.Context(), p, r.PathValue("id"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"form": def})
	})
	route("POST /api/views/{id}/form", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in Answers
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		id, err := s.Submit(r.Context(), p, r.PathValue("id"), in)
		if err != nil {
			return err
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"row_id": id})
		return nil
	})
	route("PUT /api/views/{id}/form/link", func(w http.ResponseWriter, r *http.Request, p *policy.Principal) error {
		var in struct {
			Public bool `json:"public"`
		}
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		token, err := s.SetFormLink(r.Context(), p, r.PathValue("id"), in.Public)
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"token": token})
	})
	r.HandleFunc("GET /api/public/forms/{token}", a.Public(func(w http.ResponseWriter, r *http.Request) error {
		def, err := s.PublicForm(r.Context(), r.PathValue("token"))
		if err != nil {
			return err
		}
		return ok(w, map[string]any{"form": def})
	}))
	r.HandleFunc("POST /api/public/forms/{token}", a.Public(func(w http.ResponseWriter, r *http.Request) error {
		var in Answers
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
		if err := s.PublicSubmit(r.Context(), r.PathValue("token"), httpx.ClientIP(r, m.trustProxy()), in); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return httpx.ErrNotFound
			}
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}))
	return nil
}

func safeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, s)
	if s == "" {
		return "export"
	}
	return trimText(s, 120)
}
