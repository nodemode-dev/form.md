package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/settings"
)

var (
	envRef     = regexp.MustCompile(`FORMMD_[A-Z_]+[A-Z]`)
	cmdRef     = regexp.MustCompile("(?m)(?:^|[`\\s])form-md ([a-z][a-z-]+)")
	caseRef    = regexp.MustCompile(`case ((?:"[a-z-]+",? ?)+):`)
	linkRef    = regexp.MustCompile(`\[\[(?:[^|\]]*\|)?([^\]]+)\]\]`)
	routeRef   = regexp.MustCompile("`(?:GET |POST |PUT |PATCH |DELETE )?(/(?:api|mcp|oauth|offline|files|healthz)[A-Za-z0-9_./{}-]*)`")
	quoted     = regexp.MustCompile(`"([a-z-]+)"`)
	notAnEnv   = map[string]bool{"FORMMD_": true}
	extraWords = map[string]bool{"serve": true, "keygen": true, "help": true, "version": true, "reset-owner": true}
)

func main() {
	known := map[string]bool{}
	for _, v := range config.Vars {
		known[config.EnvPrefix+v.Name] = true
	}
	for _, d := range settings.Defs {
		if d.Env != "" {
			known[d.Env] = true
		}
	}
	install, _ := os.ReadFile("install.sh")
	for _, m := range envRef.FindAllString(string(install), -1) {
		known[m] = true
	}
	commands := map[string]bool{}
	for _, f := range []string{"cmd/form-md/main.go", "cmd/form-md/tool.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			fail("read %s: %v", f, err)
		}
		for _, c := range caseRef.FindAllStringSubmatch(string(src), -1) {
			for _, q := range quoted.FindAllStringSubmatch(c[1], -1) {
				commands[q[1]] = true
			}
		}
	}
	for w := range extraWords {
		commands[w] = true
	}
	code := gather()
	pages := map[string]bool{}
	files, _ := filepath.Glob(".docs/wiki/*.md")
	for _, f := range files {
		pages[strings.TrimSuffix(filepath.Base(f), ".md")] = true
	}
	docs := append(files, "README.md", "BRANCHING.md")
	var problems []string
	for _, f := range docs {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		text := string(raw)
		for _, m := range envRef.FindAllString(text, -1) {
			if !known[m] && !notAnEnv[m] {
				problems = append(problems, fmt.Sprintf("%s: %s is not a variable the server or installer reads", f, m))
			}
		}
		for _, m := range cmdRef.FindAllStringSubmatch(text, -1) {
			if !commands[m[1]] && !strings.HasPrefix(m[1], "http") {
				problems = append(problems, fmt.Sprintf("%s: form-md %s is not a command", f, m[1]))
			}
		}
		for _, m := range linkRef.FindAllStringSubmatch(text, -1) {
			if !pages[strings.ReplaceAll(strings.TrimSpace(m[1]), " ", "-")] {
				problems = append(problems, fmt.Sprintf("%s: [[%s]] has no page", f, m[1]))
			}
		}
		for _, m := range routeRef.FindAllStringSubmatch(text, -1) {
			if !strings.Contains(code, routeStem(m[1])) {
				problems = append(problems, fmt.Sprintf("%s: route %s is not in the code", f, m[1]))
			}
		}
		if strings.ContainsRune(text, '—') {
			problems = append(problems, f+": contains an em dash")
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Printf("wikicheck: %d pages agree with the code\n", len(docs))
}

func routeStem(route string) string {
	if i := strings.IndexAny(route, "{"); i > 0 {
		route = route[:i]
	}
	return strings.TrimRight(route, "/.")
}

func gather() string {
	var b strings.Builder
	for _, root := range []string{"internal", "cmd", "web/public", "web/src"} {
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (strings.HasSuffix(p, ".go") || strings.HasSuffix(p, ".ts") || strings.HasSuffix(p, ".js")) {
				if raw, err := os.ReadFile(p); err == nil {
					b.Write(raw)
				}
			}
			return nil
		})
	}
	return b.String()
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "wikicheck: "+format+"\n", args...)
	os.Exit(2)
}
