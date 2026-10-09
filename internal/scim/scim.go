package scim

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/audit"
	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/ids"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
	"github.com/nodemode-ai/form.md/internal/workspace"
)

const (
	userSchema  = "urn:ietf:params:scim:schemas:core:2.0:User"
	groupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"
	listSchema  = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	errSchema   = "urn:ietf:params:scim:api:messages:2.0:Error"
	maxBody     = 1 << 20
	maxPage     = 200
)

type Service struct {
	st    *store.Store
	gate  *policy.Gate
	auth  *auth.Service
	ws    *workspace.Service
	audit *audit.Log
	log   *slog.Logger
}

func New(st *store.Store, gate *policy.Gate, a *auth.Service, ws *workspace.Service, al *audit.Log, log *slog.Logger) *Service {
	return &Service{st: st, gate: gate, auth: a, ws: ws, audit: al, log: log}
}

type Error struct {
	Status   int
	Detail   string
	ScimType string
}

func (e *Error) Error() string { return e.Detail }

func fail(status int, scimType, detail string) *Error {
	return &Error{Status: status, Detail: detail, ScimType: scimType}
}

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeErr(w http.ResponseWriter, err error) {
	var se *Error
	if !errors.As(err, &se) {
		var api *httpx.APIError
		switch {
		case errors.As(err, &api):
			se = &Error{Status: api.Status, Detail: api.Message}
		case errors.Is(err, store.ErrNotFound):
			se = &Error{Status: http.StatusNotFound, Detail: "not found"}
		default:
			se = &Error{Status: http.StatusInternalServerError, Detail: "internal error"}
		}
	}
	body := map[string]any{"schemas": []string{errSchema}, "status": strconv.Itoa(se.Status), "detail": se.Detail}
	if se.ScimType != "" {
		body["scimType"] = se.ScimType
	}
	write(w, se.Status, body)
}

type call struct {
	ctx   context.Context
	ws    string
	label string
	base  string
}

func (c *call) record(tx *sql.Tx, s *Service, action, detail string) error {
	return s.audit.Record(c.ctx, tx, nil, audit.Entry{Action: action, WorkspaceID: c.ws, Detail: detail, ActorKind: "system", ActorLabel: "SCIM " + c.label})
}

func (s *Service) resolve(ctx context.Context, r *http.Request) (string, string, error) {
	h := r.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(h, "Bearer ")
	if !ok || !strings.HasPrefix(tok, "scim_") {
		return "", "", fail(http.StatusUnauthorized, "", "a SCIM token is required")
	}
	var id, ws, label string
	if err := s.st.R.QueryRowContext(ctx, "SELECT id, workspace_id, label FROM scim_tokens WHERE token_hash = ?", secrets.Hash(strings.TrimSpace(tok))).Scan(&id, &ws, &label); err != nil {
		return "", "", fail(http.StatusUnauthorized, "", "this SCIM token is not valid")
	}
	_, _ = s.st.W.ExecContext(ctx, "UPDATE scim_tokens SET last_used_at = ? WHERE id = ?", store.NowMS(), id)
	return ws, label, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scim/v2" && !strings.HasPrefix(r.URL.Path, "/scim/v2/") {
			next.ServeHTTP(w, r)
			return
		}
		if err := s.serve(w, r); err != nil {
			writeErr(w, err)
		}
	})
}

func (s *Service) serve(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ws, label, err := s.resolve(ctx, r)
	if err != nil {
		return err
	}
	c := &call{ctx: ctx, ws: ws, label: label, base: httpx.RequestOrigin(r, s.auth.TrustProxy()) + "/scim/v2"}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/scim/v2"), "/"), "/")
	var body map[string]any
	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		raw, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
		if err != nil || len(raw) > maxBody {
			return fail(http.StatusRequestEntityTooLarge, "", "the request is too large")
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			return fail(http.StatusBadRequest, "invalidSyntax", "the body is not valid JSON")
		}
	}
	switch {
	case len(parts) == 1 && parts[0] == "ServiceProviderConfig" && r.Method == http.MethodGet:
		write(w, http.StatusOK, s.providerConfig(c))
	case len(parts) == 1 && parts[0] == "ResourceTypes" && r.Method == http.MethodGet:
		write(w, http.StatusOK, resourceTypes(c))
	case len(parts) == 1 && parts[0] == "Schemas" && r.Method == http.MethodGet:
		write(w, http.StatusOK, schemas())
	case parts[0] == "Users" && len(parts) <= 2:
		return s.users(w, r, c, parts, body)
	case parts[0] == "Groups" && len(parts) <= 2:
		return s.groups(w, r, c, parts, body)
	default:
		return fail(http.StatusNotFound, "", "no such SCIM endpoint")
	}
	return nil
}

func iso(ms int64) string { return time.UnixMilli(ms).UTC().Format(time.RFC3339) }

func (s *Service) providerConfig(c *call) map[string]any {
	return map[string]any{
		"schemas":          []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"documentationUri": "https://github.com/nodemode-ai/form.md",
		"patch":            map[string]bool{"supported": true},
		"bulk":             map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":           map[string]any{"supported": true, "maxResults": maxPage},
		"changePassword":   map[string]bool{"supported": false},
		"sort":             map[string]bool{"supported": false},
		"etag":             map[string]bool{"supported": false},
		"authenticationSchemes": []map[string]any{{
			"type": "oauthbearertoken", "name": "Bearer token", "description": "A SCIM token created in workspace settings", "primary": true,
		}},
		"meta": map[string]string{"resourceType": "ServiceProviderConfig", "location": c.base + "/ServiceProviderConfig"},
	}
}

func resourceTypes(c *call) map[string]any {
	types := []map[string]any{
		{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"}, "id": "User", "name": "User", "endpoint": "/Users", "schema": userSchema, "meta": map[string]string{"resourceType": "ResourceType", "location": c.base + "/ResourceTypes/User"}},
		{"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"}, "id": "Group", "name": "Group", "endpoint": "/Groups", "schema": groupSchema, "meta": map[string]string{"resourceType": "ResourceType", "location": c.base + "/ResourceTypes/Group"}},
	}
	return map[string]any{"schemas": []string{listSchema}, "totalResults": len(types), "itemsPerPage": len(types), "startIndex": 1, "Resources": types}
}

func attr(name, kind string, required bool) map[string]any {
	return map[string]any{"name": name, "type": kind, "multiValued": false, "required": required, "mutability": "readWrite", "returned": "default", "uniqueness": "none"}
}

func schemas() map[string]any {
	user := map[string]any{"id": userSchema, "name": "User", "attributes": []map[string]any{attr("userName", "string", true), attr("displayName", "string", false), attr("active", "boolean", false), attr("externalId", "string", false)}}
	group := map[string]any{"id": groupSchema, "name": "Group", "attributes": []map[string]any{attr("displayName", "string", true), attr("externalId", "string", false)}}
	return map[string]any{"schemas": []string{listSchema}, "totalResults": 2, "itemsPerPage": 2, "startIndex": 1, "Resources": []any{user, group}}
}

var filterRE = regexp.MustCompile(`(?i)^\s*([a-z.]+)\s+eq\s+"((?:[^"\\]|\\.)*)"\s*$`)

func parseFilter(f string, allowed map[string]string) (string, string, error) {
	if strings.TrimSpace(f) == "" {
		return "", "", nil
	}
	m := filterRE.FindStringSubmatch(f)
	if m == nil {
		return "", "", fail(http.StatusBadRequest, "invalidFilter", "only `attribute eq \"value\"` filters are supported")
	}
	col, ok := allowed[strings.ToLower(m[1])]
	if !ok {
		return "", "", fail(http.StatusBadRequest, "invalidFilter", "filtering on "+m[1]+" is not supported")
	}
	return col, strings.ReplaceAll(m[2], `\"`, `"`), nil
}

func paging(r *http.Request) (int, int) {
	start, _ := strconv.Atoi(r.URL.Query().Get("startIndex"))
	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if start < 1 {
		start = 1
	}
	if err != nil || count > maxPage {
		count = maxPage
	}
	if count < 0 {
		count = 0
	}
	return start, count
}

func list(resources []map[string]any, total, start int) map[string]any {
	if resources == nil {
		resources = []map[string]any{}
	}
	return map[string]any{"schemas": []string{listSchema}, "totalResults": total, "itemsPerPage": len(resources), "startIndex": start, "Resources": resources}
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolOf(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func newID() string { return ids.New() }
