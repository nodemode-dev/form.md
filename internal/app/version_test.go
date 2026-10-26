package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func firstGroup(t *testing.T, rel, pattern string) string {
	t.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(repoFile(t, rel))
	if m == nil {
		t.Fatalf("%s: no match for %s", rel, pattern)
	}
	return m[1]
}

func jsonVersion(t *testing.T, rel string, path ...string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(repoFile(t, rel)), &v); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("%s: no %v", rel, path)
		}
		v = m[k]
	}
	s, _ := v.(string)
	if s == "" {
		t.Fatalf("%s: no version at %v", rel, path)
	}
	return s
}

func TestEveryVersionSiteHoldsTheSameRelease(t *testing.T) {
	want := Version
	if !regexp.MustCompile(`^v\d+\.\d+\.\d$`).MatchString(want) {
		t.Fatalf("internal/app/version.go: %q is not vX.Y.Z with a one-digit patch", want)
	}
	sites := map[string]string{
		"Makefile":                             firstGroup(t, "Makefile", `(?m)^VERSION := (v\S+)$`),
		"README.md badge":                      firstGroup(t, "README.md", `badge/version-(v[\d.]+)-`),
		"README.md alt":                        firstGroup(t, "README.md", `alt="version (v[\d.]+)"`),
		".docs/wiki/Home.md":                   firstGroup(t, ".docs/wiki/Home.md", `\*\*Current version:\*\* (v[\d.]+)`),
		"src-tauri/Cargo.toml":                 "v" + firstGroup(t, "src-tauri/Cargo.toml", `(?m)^\[package\][^\[]*?^version = "([\d.]+)"`),
		"src-tauri/Cargo.lock":                 "v" + firstGroup(t, "src-tauri/Cargo.lock", `name = "form-md-desktop"\nversion = "([\d.]+)"`),
		"src-tauri/tauri.conf.json":            "v" + jsonVersion(t, "src-tauri/tauri.conf.json", "version"),
		"web/package.json":                     "v" + jsonVersion(t, "web/package.json", "version"),
		"web/package-lock.json":                "v" + jsonVersion(t, "web/package-lock.json", "version"),
		"web/package-lock.json packages[\"\"]": "v" + jsonVersion(t, "web/package-lock.json", "packages", "", "version"),
	}
	for site, got := range sites {
		if got != want {
			t.Errorf("%s holds %s, internal/app/version.go holds %s", site, got, want)
		}
	}
}

func TestDesktopReleaseProfileKeepsItsSymbols(t *testing.T) {
	section := ""
	seen := map[string]bool{}
	for _, line := range strings.Split(repoFile(t, "src-tauri/Cargo.toml"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = strings.Trim(line, "[]")
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "strip" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"`)
		if val != "false" && val != "none" {
			t.Errorf("[%s] sets strip = %s; macOS builds fail on a stripped binary", section, val)
		}
		seen[section] = true
	}
	for _, want := range []string{"profile.release", "profile.release.build-override"} {
		if !seen[want] {
			t.Errorf("[%s] does not set strip = false", want)
		}
	}
}
