package scim

import (
	"database/sql"
	"net/http"
	"regexp"
	"strings"

	"github.com/nodemode-ai/form.md/internal/store"
)

type group struct {
	ID       string
	Name     string
	External string
	Created  int64
	Updated  int64
	Members  []member
}

type member struct {
	ID   string
	Name string
}

func (s *Service) group(c *call, id string, withMembers bool) (*group, error) {
	var g group
	if err := s.st.R.QueryRowContext(c.ctx, "SELECT id, name, external_id, created_at, updated_at FROM groups WHERE id = ? AND workspace_id = ?", id, c.ws).
		Scan(&g.ID, &g.Name, &g.External, &g.Created, &g.Updated); err != nil {
		return nil, fail(http.StatusNotFound, "", "no such group")
	}
	if withMembers {
		rows, err := s.st.R.QueryContext(c.ctx, "SELECT gm.user_id, u.name FROM group_members gm JOIN users u ON u.id = gm.user_id WHERE gm.group_id = ? ORDER BY u.name", id)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var m member
			if rows.Scan(&m.ID, &m.Name) == nil {
				g.Members = append(g.Members, m)
			}
		}
	}
	return &g, nil
}

func (c *call) groupResource(g *group, withMembers bool) map[string]any {
	out := map[string]any{
		"schemas":     []string{groupSchema},
		"id":          g.ID,
		"displayName": g.Name,
		"meta":        map[string]string{"resourceType": "Group", "created": iso(g.Created), "lastModified": iso(g.Updated), "location": c.base + "/Groups/" + g.ID},
	}
	if g.External != "" {
		out["externalId"] = g.External
	}
	if withMembers {
		ms := make([]map[string]string, 0, len(g.Members))
		for _, m := range g.Members {
			ms = append(ms, map[string]string{"value": m.ID, "display": m.Name, "$ref": c.base + "/Users/" + m.ID})
		}
		out["members"] = ms
	}
	return out
}

func (s *Service) groups(w http.ResponseWriter, r *http.Request, c *call, parts []string, body map[string]any) error {
	withMembers := !strings.Contains(strings.ToLower(r.URL.Query().Get("excludedAttributes")), "members")
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			return s.listGroups(w, r, c, withMembers)
		case http.MethodPost:
			return s.createGroup(w, c, body)
		}
		return fail(http.StatusMethodNotAllowed, "", "method not allowed")
	}
	id := parts[1]
	switch r.Method {
	case http.MethodGet:
		g, err := s.group(c, id, withMembers)
		if err != nil {
			return err
		}
		write(w, http.StatusOK, c.groupResource(g, withMembers))
		return nil
	case http.MethodPut:
		return s.replaceGroup(w, c, id, body)
	case http.MethodPatch:
		return s.patchGroup(w, c, id, body)
	case http.MethodDelete:
		return s.deleteGroup(w, c, id)
	}
	return fail(http.StatusMethodNotAllowed, "", "method not allowed")
}

var groupFilters = map[string]string{"displayname": "name", "externalid": "external_id", "id": "id"}

func (s *Service) listGroups(w http.ResponseWriter, r *http.Request, c *call, withMembers bool) error {
	col, val, err := parseFilter(r.URL.Query().Get("filter"), groupFilters)
	if err != nil {
		return err
	}
	where := " WHERE workspace_id = ?"
	args := []any{c.ws}
	if col != "" {
		where += " AND " + col + " = ?"
		args = append(args, val)
	}
	var total int
	if err := s.st.R.QueryRowContext(c.ctx, "SELECT COUNT(*) FROM groups"+where, args...).Scan(&total); err != nil {
		return err
	}
	start, count := paging(r)
	rows, err := s.st.R.QueryContext(c.ctx, "SELECT id FROM groups"+where+" ORDER BY created_at, id LIMIT ? OFFSET ?", append(args, count, start-1)...)
	if err != nil {
		return err
	}
	var idsList []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			idsList = append(idsList, id)
		}
	}
	rows.Close()
	var out []map[string]any
	for _, id := range idsList {
		g, err := s.group(c, id, withMembers)
		if err != nil {
			continue
		}
		out = append(out, c.groupResource(g, withMembers))
	}
	write(w, http.StatusOK, list(out, total, start))
	return nil
}

func memberIDs(v any) []string {
	var out []string
	items, _ := v.([]any)
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			if id := str(m["value"]); id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

func (s *Service) checkMembers(c *call, tx *sql.Tx, list []string) error {
	for _, id := range list {
		var n int
		if err := tx.QueryRowContext(c.ctx, "SELECT COUNT(*) FROM workspace_members WHERE workspace_id = ? AND user_id = ?", c.ws, id).Scan(&n); err != nil || n == 0 {
			return fail(http.StatusBadRequest, "invalidValue", "member "+id+" is not an active user of this workspace")
		}
	}
	return nil
}

func (s *Service) addMembers(c *call, tx *sql.Tx, groupID string, list []string) error {
	if err := s.checkMembers(c, tx, list); err != nil {
		return err
	}
	for _, id := range list {
		if _, err := tx.ExecContext(c.ctx, "INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?)", groupID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createGroup(w http.ResponseWriter, c *call, body map[string]any) error {
	name := clip(str(body["displayName"]), 100)
	if name == "" {
		return fail(http.StatusBadRequest, "invalidValue", "displayName is required")
	}
	ext := str(body["externalId"])
	id := newID()
	err := s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		now := store.NowMS()
		if _, err := tx.ExecContext(c.ctx, "INSERT INTO groups (id, workspace_id, name, external_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", id, c.ws, name, ext, now, now); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return fail(http.StatusConflict, "uniqueness", "a group with this name exists")
			}
			return err
		}
		if err := s.addMembers(c, tx, id, memberIDs(body["members"])); err != nil {
			return err
		}
		return c.record(tx, s, "scim.group_create", name)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "groups")
	g, err := s.group(c, id, true)
	if err != nil {
		return err
	}
	write(w, http.StatusCreated, c.groupResource(g, true))
	return nil
}

func (s *Service) rename(c *call, tx *sql.Tx, id, name string) error {
	name = clip(strings.TrimSpace(name), 100)
	if name == "" {
		return fail(http.StatusBadRequest, "invalidValue", "displayName cannot be empty")
	}
	if _, err := tx.ExecContext(c.ctx, "UPDATE groups SET name = ?, updated_at = ? WHERE id = ?", name, store.NowMS(), id); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fail(http.StatusConflict, "uniqueness", "a group with this name exists")
		}
		return err
	}
	return nil
}

func (s *Service) replaceGroup(w http.ResponseWriter, c *call, id string, body map[string]any) error {
	if _, err := s.group(c, id, false); err != nil {
		return err
	}
	err := s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		if name := str(body["displayName"]); name != "" {
			if err := s.rename(c, tx, id, name); err != nil {
				return err
			}
		}
		if v, ok := body["externalId"]; ok {
			if _, err := tx.ExecContext(c.ctx, "UPDATE groups SET external_id = ? WHERE id = ?", str(v), id); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ?", id); err != nil {
			return err
		}
		if err := s.addMembers(c, tx, id, memberIDs(body["members"])); err != nil {
			return err
		}
		return c.record(tx, s, "scim.group_update", id)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "groups")
	s.ws.Changed(c.ws, "pages")
	g, err := s.group(c, id, true)
	if err != nil {
		return err
	}
	write(w, http.StatusOK, c.groupResource(g, true))
	return nil
}

var memberPath = regexp.MustCompile(`(?i)^members\[value eq "([^"]+)"\]$`)

func (s *Service) patchGroup(w http.ResponseWriter, c *call, id string, body map[string]any) error {
	if _, err := s.group(c, id, false); err != nil {
		return err
	}
	list, err := ops(body)
	if err != nil {
		return err
	}
	err = s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		for _, o := range list {
			op := strings.ToLower(str(o["op"]))
			path := str(o["path"])
			lp := strings.ToLower(path)
			switch {
			case lp == "members" && op == "add":
				if err := s.addMembers(c, tx, id, memberIDs(o["value"])); err != nil {
					return err
				}
			case lp == "members" && op == "remove":
				targets := memberIDs(o["value"])
				if o["value"] == nil {
					if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ?", id); err != nil {
						return err
					}
				}
				for _, u := range targets {
					if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ? AND user_id = ?", id, u); err != nil {
						return err
					}
				}
			case memberPath.MatchString(path) && op == "remove":
				u := memberPath.FindStringSubmatch(path)[1]
				if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ? AND user_id = ?", id, u); err != nil {
					return err
				}
			case lp == "members" && op == "replace":
				if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ?", id); err != nil {
					return err
				}
				if err := s.addMembers(c, tx, id, memberIDs(o["value"])); err != nil {
					return err
				}
			case lp == "displayname" && (op == "replace" || op == "add"):
				if err := s.rename(c, tx, id, str(o["value"])); err != nil {
					return err
				}
			case lp == "externalid" && (op == "replace" || op == "add"):
				if _, err := tx.ExecContext(c.ctx, "UPDATE groups SET external_id = ? WHERE id = ?", str(o["value"]), id); err != nil {
					return err
				}
			case path == "" && (op == "replace" || op == "add"):
				m, _ := o["value"].(map[string]any)
				if name := str(m["displayName"]); name != "" {
					if err := s.rename(c, tx, id, name); err != nil {
						return err
					}
				}
				if v, ok := m["externalId"]; ok {
					if _, err := tx.ExecContext(c.ctx, "UPDATE groups SET external_id = ? WHERE id = ?", str(v), id); err != nil {
						return err
					}
				}
				if v, ok := m["members"]; ok {
					if op == "replace" {
						if _, err := tx.ExecContext(c.ctx, "DELETE FROM group_members WHERE group_id = ?", id); err != nil {
							return err
						}
					}
					if err := s.addMembers(c, tx, id, memberIDs(v)); err != nil {
						return err
					}
				}
			default:
				return fail(http.StatusBadRequest, "invalidPath", "unsupported operation "+op+" "+path)
			}
		}
		if _, err := tx.ExecContext(c.ctx, "UPDATE groups SET updated_at = ? WHERE id = ?", store.NowMS(), id); err != nil {
			return err
		}
		return c.record(tx, s, "scim.group_update", id)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "groups")
	s.ws.Changed(c.ws, "pages")
	g, err := s.group(c, id, true)
	if err != nil {
		return err
	}
	write(w, http.StatusOK, c.groupResource(g, true))
	return nil
}

func (s *Service) deleteGroup(w http.ResponseWriter, c *call, id string) error {
	g, err := s.group(c, id, false)
	if err != nil {
		return err
	}
	err = s.st.Tx(c.ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(c.ctx, "DELETE FROM page_shares WHERE subject_kind = 'group' AND subject_id = ?", id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(c.ctx, "DELETE FROM groups WHERE id = ?", id); err != nil {
			return err
		}
		return c.record(tx, s, "scim.group_delete", g.Name)
	})
	if err != nil {
		return err
	}
	s.ws.Changed(c.ws, "groups")
	s.ws.Changed(c.ws, "pages")
	write(w, http.StatusNoContent, nil)
	return nil
}
