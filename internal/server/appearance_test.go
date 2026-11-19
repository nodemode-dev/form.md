package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAmoledThemeIsStoredAsTheDarkProfile(t *testing.T) {
	e := newEnv(t, "")
	owner := e.setupOwner()
	look := func(dark string) map[string]any {
		return map[string]any{"prefs": map[string]any{"appearance": map[string]any{"mode": "dark", "dark": dark, "at": 1}}}
	}
	if r := owner.do("PATCH", "/api/me", look("amoled")); r.Status != http.StatusOK {
		t.Fatalf("save amoled: %d %s", r.Status, r.Body)
	}
	r := owner.do("GET", "/api/me", nil)
	var me struct {
		User struct {
			Prefs struct {
				Appearance struct {
					Mode string `json:"mode"`
					Dark string `json:"dark"`
				} `json:"appearance"`
			} `json:"prefs"`
		} `json:"user"`
	}
	if err := json.Unmarshal(r.Body, &me); err != nil {
		t.Fatal(err)
	}
	if a := me.User.Prefs.Appearance; a.Mode != "dark" || a.Dark != "amoled" {
		t.Fatalf("stored appearance: %+v", a)
	}
	if r := owner.do("PATCH", "/api/me", look("amoled-black")); r.Status != http.StatusBadRequest {
		t.Fatalf("a profile name outside the rule was stored: %d %s", r.Status, r.Body)
	}
}
