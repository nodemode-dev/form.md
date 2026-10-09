package scim

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/nodemode-ai/form.md/internal/auth"
	"github.com/nodemode-ai/form.md/internal/secrets"
	"github.com/nodemode-ai/form.md/internal/store"
)

type user struct {
	ID       string
	Email    string
	Name     string
	External string
	Active   bool
	Linked   bool
	Created  int64
	Updated  int64
}

const userSelect = `SELECT u.id, u.email, u.name, u.created_at, u.updated_at, COALESCE(su.external_id, ''),
	EXISTS (SELECT 1 FROM workspace_members m WHERE m.workspace_id = ? AND m.user_id = u.id), su.user_id IS NOT NULL
	FROM users u LEFT JOIN scim_users su ON su.workspace_id = ? AND su.user_id = u.id`

const inScope = `(su.user_id IS NOT NULL OR EXISTS (SELECT 1 FROM workspace_members m WHERE m.workspace_id = ? AND m.user_id = u.id))`

func scanUser(row interface{ Scan(...any) error }) (*user, error) {
	var u user
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Created, &u.Updated, &u.External, &u.Active, &u.Linked); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) user(c *call, id string) (*user, error) {
	u, err := scanUser(s.st.R.QueryRowContext(c.ctx, userSelect+` WHERE u.id = ? AND `+inScope, c.ws, c.ws, id, c.ws))
	if err != nil {
		return nil, fail(http.StatusNotFound, "", "no such user")
	}
	return u, nil
}

func splitName(name string) (string, string) {
	given, family, _ := strings.Cut(strings.TrimSpace(name), " ")
	return given, strings.TrimSpace(family)
}

func (c *call) userResource(u *user) map[string]any {
	given, family := splitName(u.Name)
	out := map[string]any{
		"schemas":     []string{userSchema},
		"id":          u.ID,
		"userName":    u.Email,
		"displayName": u.Name,
		"name":        map[string]string{"formatted": u.Name, "givenName": given, "familyName": family},
		"emails":      []map[string]any{{"value": u.Email, "primary": true, "type": "work"}},
		"active":      u.Active,
		"meta":        map[string]string{"resourceType": "User", "created": iso(u.Created), "lastModified": iso(u.Updated), "location": c.base + "/Users/" + u.ID},
	}
	if u.External != "" {
		out["externalId"] = u.External
	}
	return out
}

func (s *Service) users(w http.ResponseWriter, r *http.Request, c *call, parts []string, body map[string]any) error {
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return s.listUsers(w, r, c)
		case http.MethodPost:
			return s.createUser(w, c, body)
		}
		return fail(http.StatusMethodNotAllowed, "", "method not allowed")
	}
	id := parts[1]
	switch r.Method {
	case http.MethodGet:
		u, err := s.user(c, id)
		if err != nil {
			return err
		}
		write(w, http.StatusOK, c.userResource(u))
		return nil
	case http.MethodPut:
		return s.replaceUser(w, c, id, body)
	case http.MethodPatch:
		return s.patchUser(w, c, id, body)
	case http.MethodDelete:
		return s.deleteUser(w, c, id)
	}
	return fail(http.StatusMethodNotAllowed, "", "method not allowed")
}

var userFilters = map[string]string{"username": "u.email", "emails.value": "u.email", "emails": "u.email", "externalid": "su.external_id", "id": "u.id", "displayname": "u.name"}

func (s *Service) listUsers(w http.ResponseWriter, r *http.Request, c *call) error {
	col, val, err := parseFilter(r.URL.Query().Get("filter"), userFilters)
	if err != nil {
		return err
	}
	where := ` WHERE ` + inScope
	args := []any{c.ws, c.ws, c.ws}
	if col != "" {
		if col == "u.email" {
			where += ` AND u.email = ? COLLATE NOCASE`
		} else {
			where += ` AND ` + col + ` = ?`
		}
		args = append(args, val)
	}
	var total int
	if err := s.st.R.QueryRowContext(c.ctx, `SELECT COUNT(*) FROM users u LEFT JOIN scim_users su ON su.workspace_id = ? AND su.user_id = u.id`+where, args[1:]...).Scan(&total); err != nil {
		return err
	}
	start, count := paging(r)
	rows, err := s.st.R.QueryContext(c.ctx, userSelect+where+` ORDER BY u.created_at, u.id LIMIT ? OFFSET ?`, append(args, count, start-1)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return err
		}
		out = append(out, c.userResource(u))
	}
	write(w, http.StatusOK, list(out, total, start))
	return rows.Err()
}

type userInput struct {
	email    string
	name     string
	external *string
	active   *bool
}

func readUser(body map[string]any) userInput {
	var in userInput
	in.email = str(body["userName"])
	if emails, ok := body["emails"].([]any); ok && !strings.Contains(in.email, "@") {
		first, primary := "", ""
		for _, e := range emails {
			m, _ := e.(map[string]any)
			v := str(m["value"])
			if first == "" {
				first = v
			}
			if p, _ := boolOf(m["primary"]); p && primary == "" {
				primary = v
			}
		}
		if primary == "" {
			primary = first
		}
		if primary != "" {
			in.email = primary
		}
	}
	in.name = str(body["displayName"])
	if n, ok := body["name"].(map[string]any); ok && in.name == "" {
		in.name = str(n["formatted"])
		if in.name == "" {
			in.name = strings.TrimSpace(str(n["givenName"]) + " " + str(n["familyName"]))
		}
	}
	if v, ok := body["externalId"]; ok {
		x := str(v)
		in.external = &x
	}
	if v, ok := body["active"]; ok {
		if b, ok := boolOf(v); ok {
			in.active = &b
		}
	}
	return in
}

func (s *Service) setActive(c *call, tx *sql.Tx, id string, on, wasMember bool) error {
	switch {
	case on && !wasMember:
		if _, err := s.ws.SetAccess(c.ctx, tx, c.ws, id, "member"); err != nil {
			return err
		}
		return c.record(tx, s, "scim.user_activate", id)
	case !on && wasMember:
		if _, err := s.ws.SetAccess(c.ctx, tx, c.ws, id, ""); err != nil {
			return fail(http.StatusBadRequest, "mutability", "this user is the last admin of the workspace")
		}
		return c.record(tx, s, "scim.user_deactivate", id)
	}
	return nil
}

func (s *Service) createUser(w http.ResponseWriter, c *call, body map[string]any) error {
	in := readUser(body)
	email, err := auth.NormalizeEmail(in.email)
	if err != nil {
		return fail(http.StatusBadRequest, "invalidValue", "userName must be an email address")
	}
	if in.name == "" {
		in.name, _, _ = strings.Cut(email, "@")
	}
	active := in.active == nil || *in.active
	var id string
	err = s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		var linked, member int
		err := tx.QueryRowContext(c.ctx, `SELECT u.id, EXISTS (SELECT 1 FROM scim_users su WHERE su.workspace_id = ? AND su.user_id = u.id),
			EXISTS (SELECT 1 FROM workspace_members m WHERE m.workspace_id = ? AND m.user_id = u.id) FROM users u WHERE u.email = ? COLLATE NOCASE`, c.ws, c.ws, email).Scan(&id, &linked, &member)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if id, err = s.auth.CreateAccount(c.ctx, tx, email, in.name, secrets.Token("", 24), "member"); err != nil {
				return err
			}
			if err := c.record(tx, s, "scim.user_create", id); err != nil {
				return err
			}
		case err != nil:
			return err
		case linked == 1:
			return fail(http.StatusConflict, "uniqueness", "this user is already provisioned")
		default:
			if err := c.record(tx, s, "scim.user_link", id); err != nil {
				return err
			}
		}
		ext := ""
		if in.external != nil {
			ext = *in.external
		}
		now := store.NowMS()
		if _, err := tx.ExecContext(c.ctx, `INSERT INTO scim_users (workspace_id, user_id, external_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (workspace_id, user_id) DO UPDATE SET external_id = excluded.external_id, updated_at = excluded.updated_at`, c.ws, id, ext, now, now); err != nil {
			return err
		}
		return s.setActive(c, tx, id, active, member == 1)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "members", id)
	u, err := s.user(c, id)
	if err != nil {
		return err
	}
	write(w, http.StatusCreated, c.userResource(u))
	return nil
}

func (s *Service) applyUser(c *call, id string, in userInput) error {
	u, err := s.user(c, id)
	if err != nil {
		return err
	}
	err = s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		if in.name != "" && in.name != u.Name {
			if _, err := tx.ExecContext(c.ctx, "UPDATE users SET name = ?, updated_at = ? WHERE id = ?", clip(in.name, 80), now, id); err != nil {
				return err
			}
		}
		ext := u.External
		if in.external != nil {
			ext = *in.external
		}
		if _, err := tx.ExecContext(c.ctx, `INSERT INTO scim_users (workspace_id, user_id, external_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (workspace_id, user_id) DO UPDATE SET external_id = excluded.external_id, updated_at = excluded.updated_at`, c.ws, id, ext, now, now); err != nil {
			return err
		}
		if in.active != nil {
			if err := s.setActive(c, tx, id, *in.active, u.Active); err != nil {
				return err
			}
		}
		return c.record(tx, s, "scim.user_update", id)
	})
	if err == nil {
		s.ws.Changed(c.ws, "members", id)
	}
	return err
}

func (s *Service) replaceUser(w http.ResponseWriter, c *call, id string, body map[string]any) error {
	in := readUser(body)
	if in.active == nil {
		t := true
		in.active = &t
	}
	if err := s.applyUser(c, id, in); err != nil {
		return err
	}
	u, err := s.user(c, id)
	if err != nil {
		return err
	}
	write(w, http.StatusOK, c.userResource(u))
	return nil
}

func ops(body map[string]any) ([]map[string]any, error) {
	raw, ok := body["Operations"].([]any)
	if !ok {
		raw, ok = body["operations"].([]any)
	}
	if !ok {
		return nil, fail(http.StatusBadRequest, "invalidSyntax", "Operations is required")
	}
	out := make([]map[string]any, 0, len(raw))
	for _, o := range raw {
		m, ok := o.(map[string]any)
		if !ok {
			return nil, fail(http.StatusBadRequest, "invalidSyntax", "each operation is an object")
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) patchUser(w http.ResponseWriter, c *call, id string, body map[string]any) error {
	list, err := ops(body)
	if err != nil {
		return err
	}
	var in userInput
	set := func(path string, v any) {
		switch strings.ToLower(path) {
		case "active":
			if b, ok := boolOf(v); ok {
				in.active = &b
			}
		case "displayname", "name.formatted":
			in.name = str(v)
		case "name.givenname":
			if in.name == "" {
				in.name = str(v)
			}
		case "externalid":
			x := str(v)
			in.external = &x
		}
	}
	for _, o := range list {
		op := strings.ToLower(str(o["op"]))
		if op != "replace" && op != "add" {
			continue
		}
		if path := str(o["path"]); path != "" {
			set(path, o["value"])
			continue
		}
		if m, ok := o["value"].(map[string]any); ok {
			for k, v := range m {
				if k == "name" {
					if nm, ok := v.(map[string]any); ok {
						if f := str(nm["formatted"]); f != "" {
							in.name = f
						} else if g := strings.TrimSpace(str(nm["givenName"]) + " " + str(nm["familyName"])); g != "" {
							in.name = g
						}
					}
					continue
				}
				set(k, v)
			}
		}
	}
	if err := s.applyUser(c, id, in); err != nil {
		return err
	}
	u, err := s.user(c, id)
	if err != nil {
		return err
	}
	write(w, http.StatusOK, c.userResource(u))
	return nil
}

func (s *Service) deleteUser(w http.ResponseWriter, c *call, id string) error {
	u, err := s.user(c, id)
	if err != nil {
		return err
	}
	err = s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		if err := s.setActive(c, tx, id, false, u.Active); err != nil {
			return err
		}
		if _, err := tx.ExecContext(c.ctx, "DELETE FROM scim_users WHERE workspace_id = ? AND user_id = ?", c.ws, id); err != nil {
			return err
		}
		return c.record(tx, s, "scim.user_delete", id)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "members", id)
	write(w, http.StatusNoContent, nil)
	return nil
}

func clip(v string, n int) string {
	if r := []rune(v); len(r) > n {
		return string(r[:n])
	}
	return v
}
