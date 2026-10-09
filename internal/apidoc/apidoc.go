package apidoc

import (
	"encoding/json"
	"html"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/httpx"
)

type Service struct {
	base func(r *http.Request) string

	mu     sync.Mutex
	router *httpx.Router
	cached []byte
	count  int
}

func New(base func(r *http.Request) string) *Service { return &Service{base: base} }

type route struct {
	Method string
	Path   string
}

var paramRE = regexp.MustCompile(`\{([a-zA-Z_]+)(\.\.\.)?\}`)

func (s *Service) routes() []route {
	var out []route
	seen := map[string]bool{}
	for _, p := range s.router.Patterns() {
		method, path, ok := strings.Cut(p, " ")
		if !ok {
			method, path = "POST", p
		}
		if method == "OPTIONS" || method == "HEAD" {
			continue
		}
		path = strings.ReplaceAll(path, "{$}", "")
		if !documented(path) {
			continue
		}
		path = paramRE.ReplaceAllString(path, "{$1}")
		k := method + " " + path
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, route{Method: method, Path: path})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return methodRank[out[i].Method] < methodRank[out[j].Method]
	})
	return out
}

var methodRank = map[string]int{"GET": 0, "POST": 1, "PUT": 2, "PATCH": 3, "DELETE": 4}

func documented(path string) bool {
	for _, p := range []string{"/api/", "/mcp", "/oauth/", "/.well-known/", "/scim/", "/files/", "/healthz", "/notices"} {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func public(r route) bool {
	p := r.Path
	switch {
	case strings.HasPrefix(p, "/api/public/"), strings.HasPrefix(p, "/oauth/"), strings.HasPrefix(p, "/.well-known/"),
		p == "/healthz", p == "/notices", p == "/api/openapi.json", p == "/api/docs", p == "/api/setup",
		strings.HasPrefix(p, "/api/auth/"), strings.HasPrefix(p, "/api/invites/"):
		return !strings.HasPrefix(p, "/api/auth/sessions") && p != "/api/auth/logout"
	}
	return false
}

func tagOf(path string) string {
	switch {
	case strings.HasPrefix(path, "/mcp"):
		return "mcp"
	case strings.HasPrefix(path, "/oauth/"), strings.HasPrefix(path, "/.well-known/"):
		return "oauth"
	case strings.HasPrefix(path, "/scim/"):
		return "scim"
	case strings.HasPrefix(path, "/files/"):
		return "files"
	case !strings.HasPrefix(path, "/api/"):
		return "system"
	}
	seg := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
	t := seg[0]
	if t == "workspaces" && len(seg) >= 3 && !strings.HasPrefix(seg[2], "{") {
		t = seg[2]
	}
	if a, ok := tagAlias[t]; ok {
		return a
	}
	return t
}

var singular = map[string]string{
	"pages": "page", "collections": "database", "views": "view", "rows": "row", "workspaces": "workspace", "comments": "comment",
	"files": "file", "users": "user", "tokens": "token", "groups": "group", "teamspaces": "teamspace", "shares": "share",
	"links": "link", "sites": "site", "webhooks": "webhook", "automations": "automation", "agents": "agent", "chats": "chat",
	"imports": "import job", "invites": "invitation", "members": "member", "threads": "thread", "versions": "version", "agent-runs": "agent run",
	"providers": "provider", "templates": "template", "reminders": "reminder", "favorites": "favorite", "suggestions": "suggestion",
}

var actions = map[string]bool{
	"duplicate": true, "restore": true, "move": true, "run": true, "query": true, "mark": true, "publish": true, "unpublish": true,
	"resolve": true, "reopen": true, "accept": true, "reject": true, "revoke": true, "test": true, "chart": true, "consent": true,
	"login": true, "logout": true, "signup": true, "undo": true, "use": true, "preview": true, "import": true, "transcribe": true,
	"stop": true, "reset": true, "unlock": true, "verify": true, "rotate": true, "pause": true, "resume": true, "fill": true,
}

func words(seg string) string {
	return strings.ReplaceAll(strings.ReplaceAll(seg, "-", " "), "_", " ")
}

func summarize(r route) string {
	if s, ok := overrides[r.Method+" "+r.Path]; ok {
		return s.summary
	}
	trimmed := strings.TrimPrefix(r.Path, "/api/")
	seg := strings.Split(strings.Trim(trimmed, "/"), "/")
	var statics []string
	lastParam := false
	for _, s := range seg {
		if strings.HasPrefix(s, "{") {
			lastParam = true
			continue
		}
		statics = append(statics, s)
		lastParam = false
	}
	if len(statics) == 0 {
		return r.Method + " " + r.Path
	}
	last := strings.TrimSuffix(statics[len(statics)-1], ".csv")
	owner, ownerNoun, prev := "", "", ""
	if len(statics) >= 2 {
		prev = statics[len(statics)-2]
		if o, ok := singular[prev]; ok {
			owner, ownerNoun = " of "+art(o), o
		}
	}
	noun := words(last)
	one := singular[last]
	if one == "" {
		one = strings.TrimSuffix(noun, "s")
	}
	switch r.Method {
	case "GET":
		if lastParam {
			return "Get " + art(one)
		}
		if strings.HasSuffix(last, "s") {
			return "List " + noun + owner
		}
		return "Get the " + noun + owner
	case "POST":
		if actions[last] {
			switch {
			case ownerNoun != "":
				return capital(noun) + " " + art(ownerNoun)
			case prev != "" && !generic[prev]:
				return capital(noun) + " " + words(prev)
			}
			return capital(noun)
		}
		if lastParam {
			return "Act on " + art(one)
		}
		return "Create " + art(one) + owner
	case "PUT":
		if lastParam {
			return "Set " + art(one)
		}
		return "Set the " + noun + owner
	case "PATCH":
		return "Update " + art(one) + owner
	case "DELETE":
		if lastParam {
			return "Delete " + art(one)
		}
		return "Remove the " + noun + owner
	}
	return r.Method + " " + r.Path
}

var generic = map[string]bool{"auth": true, "admin": true, "me": true, "public": true, "api": true}

func art(n string) string {
	if n == "" {
		return n
	}
	if strings.ContainsRune("aeiou", rune(n[0])) {
		return "an " + n
	}
	return "a " + n
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

type override struct {
	summary string
	query   []string
	body    string
	result  string
}

var overrides = map[string]override{
	"GET /api/me":                      {summary: "Get the signed-in person, their workspaces and preferences"},
	"GET /api/workspaces/{id}/pages":   {summary: "List the pages a credential can read in a workspace", query: []string{"limit", "cursor"}, result: "PageList"},
	"POST /api/pages":                  {summary: "Create a page or a database", body: "PageInput", result: "PageResult"},
	"GET /api/pages/{id}":              {summary: "Get a page with its path and access level", result: "PageResult"},
	"PATCH /api/pages/{id}":            {summary: "Update a page's title, icon, cover, description, tags or template flag", result: "PageResult"},
	"GET /api/pages/{id}/markdown":     {summary: "Read a page body as Markdown"},
	"PUT /api/pages/{id}/markdown":     {summary: "Write Markdown into a page body (replace, append, prepend or after a block); merges with live editing", body: "MarkdownWrite"},
	"GET /api/pages/{id}/export":       {summary: "Export a page, optionally with everything inside, as Markdown and CSV (zip), HTML or PDF", query: []string{"format", "subpages"}},
	"GET /api/workspaces/{id}/export":  {summary: "Export every page a credential can read in a workspace as a zip", query: []string{"format"}},
	"POST /api/imports":                {summary: "Start an import from uploaded files (multipart); returns a job to poll", result: "ImportJobResult"},
	"GET /api/imports/{id}":            {summary: "Get the progress and result of an import", result: "ImportJobResult"},
	"POST /api/collections/{id}/query": {summary: "Query a database's rows with filters, sorts and groups; paginate with cursor", body: "QueryInput", result: "QueryResult"},
	"POST /api/collections/{id}/rows":  {summary: "Add a row to a database", result: "RowResult"},
	"PATCH /api/rows/{id}":             {summary: "Set property values on a row", result: "RowResult"},
	"GET /api/collections/{id}":        {summary: "Get a database's properties, views and templates"},
	"POST /api/collections/{id}/chart": {summary: "Compute a chart's figures on the server"},
	"GET /api/search":                  {summary: "Search titles, bodies and properties; paginate with cursor", query: []string{"q", "workspace_id", "limit", "cursor"}},
	"GET /api/workspaces/{id}/audit":   {summary: "Read the audit log with filters; paginate with cursor", query: []string{"actor", "action", "page", "from", "to", "limit", "cursor"}},
	"POST /api/me/tokens":              {summary: "Create a personal API token (read or write, optionally limited to workspaces)"},
	"GET /api/openapi.json":            {summary: "This document"},
	"GET /api/docs":                    {summary: "This document as a readable page"},
	"POST /mcp":                        {summary: "Model Context Protocol endpoint for agents (Streamable HTTP, stateless JSON)"},
	"GET /healthz":                     {summary: "Health and version"},
	"GET /oauth/authorize":             {summary: "OAuth authorization endpoint (PKCE S256 required); resource is /mcp or /api"},
	"POST /oauth/token":                {summary: "OAuth token endpoint: authorization code and refresh token grants"},
}

func init() {
	for k, v := range map[string]string{
		"GET /.well-known/oauth-authorization-server":   "OAuth 2.1 authorization server metadata",
		"GET /.well-known/oauth-protected-resource/api": "Protected resource metadata for the REST API",
		"DELETE /api/pages/{id}/forever":                "Delete a trashed page permanently",
		"GET /api/views/{id}/export.csv":                "Export a view as CSV",
		"POST /api/public/forms/{token}":                "Submit a public form",
		"PATCH /api/me":                                 "Update the signed-in account",
		"POST /oauth/register":                          "Register an OAuth client",
		"POST /api/auth/logout":                         "Sign out",
		"POST /api/auth/login":                          "Sign in with email and password",
		"GET /api/events":                               "Live event stream (server-sent events)",
		"POST /api/rows/{id}/buttons/{prop}":            "Press a row's button property",
		"POST /api/buttons/run":                         "Run a button block's server actions",
		"DELETE /api/workspaces/{id}/trash":             "Empty a workspace's trash",
		"GET /api/workspaces/{id}/graph":                "Get the link graph of a workspace",
		"POST /api/views/{id}/calendar-feed":            "Get or renew a private calendar subscription address",
		"GET /api/public/calendar/{file}":               "Calendar subscription feed (iCalendar)",
		"POST /api/setup":                               "Create the first owner account on a new instance",
		"POST /api/collections/{id}/rows/delete":        "Delete rows of a database",
		"POST /api/ai/answer":                           "Answer a question from the workspace's pages (server-sent events)",
		"POST /api/collections/{id}/feed":               "Read a feed view's posts with reactions and comment counts",
		"POST /api/admin/stranded/{id}/adopt":           "Adopt a workspace left without an admin",
		"POST /api/me/email":                            "Change the account's email address",
	} {
		overrides[k] = override{summary: v}
	}
}

var tagAlias = map[string]string{
	"audit.csv": "audit", "automation-key": "automations", "buttons": "automations", "export": "transfer", "imports": "transfer",
	"openapi.json": "docs", "view-history": "analytics", "scim-tokens": "scim", "agent-runs": "agents", "agent": "agents",
	"deliveries": "webhooks", "guest-invites": "sharing", "shares": "sharing", "links": "sharing", "shared": "sharing",
	"members": "workspaces", "rules": "workspaces", "emergency": "admin", "me": "account", "setup": "auth", "csv": "collections",
	"views": "collections", "rows": "collections", "tags": "pages", "trash": "pages", "recent": "pages", "favorites": "pages",
	"versions": "pages", "graph": "pages", "publishing": "sites", "threads": "comments", "suggestions": "comments", "unfurl": "files",
	"events": "system", "groups": "teamspaces", "invites": "workspaces", "ai": "ai", "public": "public",
}

var queryParams = map[string]map[string]any{
	"limit":        {"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 500}, "description": "Page size."},
	"cursor":       {"name": "cursor", "in": "query", "schema": map[string]any{"type": "string"}, "description": "Opaque cursor from next_cursor of the previous page."},
	"format":       {"name": "format", "in": "query", "schema": map[string]any{"type": "string", "enum": []string{"markdown", "html", "pdf"}}},
	"subpages":     {"name": "subpages", "in": "query", "schema": map[string]any{"type": "string", "enum": []string{"0", "1"}}, "description": "Include pages and rows inside."},
	"q":            {"name": "q", "in": "query", "schema": map[string]any{"type": "string"}},
	"workspace_id": {"name": "workspace_id", "in": "query", "schema": map[string]any{"type": "string"}},
	"actor":        {"name": "actor", "in": "query", "schema": map[string]any{"type": "string"}},
	"action":       {"name": "action", "in": "query", "schema": map[string]any{"type": "string"}},
	"page":         {"name": "page", "in": "query", "schema": map[string]any{"type": "string"}},
	"from":         {"name": "from", "in": "query", "schema": map[string]any{"type": "integer"}, "description": "Unix milliseconds."},
	"to":           {"name": "to", "in": "query", "schema": map[string]any{"type": "integer"}, "description": "Unix milliseconds."},
}

func ref(name string) map[string]any { return map[string]any{"$ref": "#/components/schemas/" + name} }

func schemas() map[string]any {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "integer"}
	boolean := map[string]any{"type": "boolean"}
	strs := map[string]any{"type": "array", "items": str}
	obj := func(props map[string]any, req ...string) map[string]any {
		o := map[string]any{"type": "object", "properties": props}
		if len(req) > 0 {
			o["required"] = req
		}
		return o
	}
	page := obj(map[string]any{"id": str, "workspace_id": str, "parent_id": str, "kind": map[string]any{"type": "string", "enum": []string{"doc", "collection"}}, "title": str, "icon": str, "cover": str, "description": str, "tags": strs, "visibility": str, "is_template": boolean, "created_at": num, "updated_at": num, "access": str, "row": boolean})
	return map[string]any{
		"Error":           obj(map[string]any{"error": obj(map[string]any{"code": str, "message": str, "data": map[string]any{"type": "object"}}, "code", "message")}, "error"),
		"Page":            page,
		"PageResult":      obj(map[string]any{"page": ref("Page")}),
		"PageList":        obj(map[string]any{"pages": map[string]any{"type": "array", "items": ref("Page")}, "next_cursor": str}),
		"PageInput":       obj(map[string]any{"workspace_id": str, "parent_id": str, "after_id": str, "title": str, "icon": str, "kind": map[string]any{"type": "string", "enum": []string{"doc", "collection"}}, "teamspace_id": str}, "workspace_id"),
		"MarkdownWrite":   obj(map[string]any{"markdown": str, "mode": map[string]any{"type": "string", "enum": []string{"replace", "append", "prepend", "after"}}, "after": str}, "markdown"),
		"QueryInput":      obj(map[string]any{"view_id": str, "filter": map[string]any{"type": "object"}, "sorts": map[string]any{"type": "array", "items": map[string]any{"type": "object"}}, "search": str, "limit": num, "cursor": str, "no_groups": boolean}),
		"Row":             obj(map[string]any{"id": str, "title": str, "values": map[string]any{"type": "object"}, "created_at": num, "updated_at": num}),
		"RowResult":       obj(map[string]any{"row": ref("Row")}),
		"QueryResult":     obj(map[string]any{"total": num, "rows": map[string]any{"type": "array", "items": ref("Row")}, "offset": num, "next_cursor": str}),
		"ImportJob":       obj(map[string]any{"id": str, "state": map[string]any{"type": "string", "enum": []string{"queued", "reading", "writing", "done", "failed", "cancelled"}}, "format": str, "done": num, "total": num, "pages": num, "rows": num, "files": num, "root_ids": strs, "warnings": strs, "error": str}),
		"ImportJobResult": obj(map[string]any{"job": ref("ImportJob")}),
	}
}

func (s *Service) build(r *http.Request) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	rs := s.routes()
	if s.cached != nil && s.count == len(rs) {
		return s.cached
	}
	paths := map[string]map[string]any{}
	tags := map[string]bool{}
	for _, rt := range rs {
		op := map[string]any{
			"summary":     summarize(rt),
			"operationId": opID(rt),
			"tags":        []string{tagOf(rt.Path)},
			"responses": map[string]any{
				"200":     map[string]any{"description": "Success"},
				"default": map[string]any{"description": "An error", "content": map[string]any{"application/json": map[string]any{"schema": ref("Error")}}},
			},
		}
		tags[tagOf(rt.Path)] = true
		var params []any
		for _, m := range paramRE.FindAllStringSubmatch(rt.Path, -1) {
			params = append(params, map[string]any{"name": m[1], "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
		}
		o, has := overrides[rt.Method+" "+rt.Path]
		if has {
			for _, q := range o.query {
				if p, ok := queryParams[q]; ok {
					params = append(params, p)
				}
			}
			if o.body != "" {
				op["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": ref(o.body)}}}
			}
			if o.result != "" {
				op["responses"].(map[string]any)["200"] = map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{"schema": ref(o.result)}}}
			}
		}
		if len(params) > 0 {
			op["parameters"] = params
		}
		if public(rt) {
			op["security"] = []any{}
		} else {
			op["responses"].(map[string]any)["401"] = map[string]any{"description": "Not signed in or the credential is unknown"}
			op["responses"].(map[string]any)["429"] = map[string]any{"description": "Rate limited (tokens): 600 requests a minute with bursts of 120", "headers": map[string]any{"Retry-After": map[string]any{"schema": map[string]any{"type": "integer"}}, "RateLimit-Remaining": map[string]any{"schema": map[string]any{"type": "integer"}}}}
		}
		if paths[rt.Path] == nil {
			paths[rt.Path] = map[string]any{}
		}
		paths[rt.Path][strings.ToLower(rt.Method)] = op
	}
	var tagList []any
	names := make([]string, 0, len(tags))
	for t := range tags {
		names = append(names, t)
	}
	sort.Strings(names)
	for _, t := range names {
		tagList = append(tagList, map[string]any{"name": t})
	}
	base := s.base(r)
	doc := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "form.md API",
			"version":     app.Version,
			"description": "Every route this server answers. Authenticate with a session cookie (the web app), a personal API token (Authorization: Bearer fmd_..., made in account settings) or an OAuth 2.1 access token issued for the resource " + base + "/api. Lists that can be long take limit and an opaque cursor and return next_cursor. Tokens are limited to 600 requests a minute with bursts of 120; responses carry RateLimit-Limit and RateLimit-Remaining, and a 429 carries Retry-After. Errors are JSON with error.code and error.message.",
		},
		"servers": []any{map[string]any{"url": base}},
		"tags":    tagList,
		"paths":   paths,
		"components": map[string]any{
			"schemas": schemas(),
			"securitySchemes": map[string]any{
				"session": map[string]any{"type": "apiKey", "in": "cookie", "name": "formmd_session"},
				"token":   map[string]any{"type": "http", "scheme": "bearer", "description": "Personal API token"},
				"oauth": map[string]any{"type": "oauth2", "flows": map[string]any{"authorizationCode": map[string]any{
					"authorizationUrl": base + "/oauth/authorize", "tokenUrl": base + "/oauth/token", "refreshUrl": base + "/oauth/token",
					"scopes": map[string]any{"read": "Read pages and databases", "write": "Change pages and databases"},
				}}},
			},
		},
		"security": []any{map[string]any{"session": []any{}}, map[string]any{"token": []any{}}, map[string]any{"oauth": []any{"read"}}},
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	s.cached, s.count = b, len(rs)
	return b
}

func opID(r route) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(r.Method))
	for _, seg := range strings.Split(strings.TrimPrefix(r.Path, "/api"), "/") {
		seg = strings.Trim(seg, "{}.")
		for _, part := range strings.FieldsFunc(seg, func(c rune) bool { return c == '-' || c == '_' || c == '.' }) {
			b.WriteString(capital(part))
		}
	}
	return b.String()
}

func (s *Service) page(r *http.Request) string {
	var doc struct {
		Info struct {
			Description string `json:"description"`
			Version     string `json:"version"`
		} `json:"info"`
		Paths map[string]map[string]struct {
			Summary  string   `json:"summary"`
			Tags     []string `json:"tags"`
			Security *[]any   `json:"security"`
		} `json:"paths"`
	}
	_ = json.Unmarshal(s.build(r), &doc)
	type line struct {
		method, path, summary string
		open                  bool
	}
	groups := map[string][]line{}
	total := 0
	for path, ops := range doc.Paths {
		for m, op := range ops {
			tag := "other"
			if len(op.Tags) > 0 {
				tag = op.Tags[0]
			}
			groups[tag] = append(groups[tag], line{strings.ToUpper(m), path, op.Summary, op.Security != nil})
			total++
		}
	}
	names := make([]string, 0, len(groups))
	for t := range groups {
		names = append(names, t)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="color-scheme" content="dark light"><title>form.md API</title><style>`)
	b.WriteString(docCSS)
	b.WriteString(`</style></head><body><main><h1>form.md API <small>` + html.EscapeString(doc.Info.Version) + `</small></h1><p>` + html.EscapeString(doc.Info.Description) + `</p><p><a href="/api/openapi.json">OpenAPI 3.1 document</a> · ` + itoa(total) + ` operations</p><nav>`)
	for _, t := range names {
		b.WriteString(`<a href="#` + html.EscapeString(t) + `">` + html.EscapeString(t) + `</a>`)
	}
	b.WriteString(`</nav>`)
	for _, t := range names {
		list := groups[t]
		sort.Slice(list, func(i, j int) bool {
			if list[i].path != list[j].path {
				return list[i].path < list[j].path
			}
			return methodRank[list[i].method] < methodRank[list[j].method]
		})
		b.WriteString(`<section id="` + html.EscapeString(t) + `"><h2>` + html.EscapeString(t) + `</h2><table>`)
		for _, l := range list {
			auth := ""
			if l.open {
				auth = `<span class="open">public</span>`
			}
			b.WriteString(`<tr><td><span class="m m-` + strings.ToLower(l.method) + `">` + l.method + `</span></td><td><code>` + html.EscapeString(l.path) + `</code> ` + auth + `<div>` + html.EscapeString(l.summary) + `</div></td></tr>`)
		}
		b.WriteString(`</table></section>`)
	}
	b.WriteString(`</main></body></html>`)
	return b.String()
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

const docCSS = `:root{color-scheme:light dark;--bg:#fff;--fg:#1d1d22;--mute:#6b6b76;--line:#e6e6ea;--acc:#5b4bd5}@media (prefers-color-scheme:dark){:root{--bg:#121214;--fg:#ececf0;--mute:#9a9aa6;--line:#2a2a30;--acc:#9a8cff}}body{margin:0;background:var(--bg);color:var(--fg);font:15px/1.55 system-ui,sans-serif}main{max-width:960px;margin:0 auto;padding:32px 16px 64px}h1 small{font-size:14px;color:var(--mute);font-weight:400}a{color:var(--acc)}nav{display:flex;flex-wrap:wrap;gap:6px 12px;margin:16px 0 24px;font-size:14px}h2{margin-top:32px;font-size:18px;text-transform:capitalize}table{width:100%;border-collapse:collapse}td{padding:8px 6px;border-top:1px solid var(--line);vertical-align:top}td:first-child{width:72px}code{font:13px ui-monospace,monospace;word-break:break-all}td div{color:var(--mute);font-size:14px}.m{display:inline-block;min-width:56px;padding:1px 6px;border-radius:4px;font:600 11px ui-monospace,monospace;text-align:center;background:var(--line)}.m-get{color:#2f9e6b}.m-post{color:#3b7dd8}.m-put,.m-patch{color:#c2810c}.m-delete{color:#d04a4a}.open{font-size:11px;color:var(--mute);border:1px solid var(--line);border-radius:4px;padding:0 4px}`

func (s *Service) Module() app.Module { return module{s} }

type module struct{ s *Service }

func (module) ID() string { return "apidoc" }

func (m module) Mount(r *httpx.Router, _ *app.Deps) error {
	s := m.s
	s.router = r
	r.HandleFunc("GET /api/openapi.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_, _ = w.Write(s.build(req))
	})
	r.HandleFunc("GET /api/docs", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(s.page(req)))
	})
	return nil
}
