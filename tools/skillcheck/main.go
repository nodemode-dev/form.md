package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	formmd "github.com/nodemode-ai/form.md"
	"github.com/nodemode-ai/form.md/internal/contextpack"
)

func main() {
	skill := formmd.ContextSkill()
	var problems []string
	if _, err := contextpack.LoadPolicy(skill); err != nil {
		problems = append(problems, err.Error())
	}
	router, _ := fs.ReadFile(skill, "SKILL.md")
	files := 0
	_ = fs.WalkDir(skill, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		files++
		raw, err := fs.ReadFile(skill, p)
		if err != nil {
			problems = append(problems, p+": "+err.Error())
			return nil
		}
		text := string(raw)
		if strings.ContainsRune(text, '—') {
			problems = append(problems, p+": contains an em dash")
		}
		name := path.Base(p)
		if name != "SKILL.md" && name != "SUB-SKILL.md" {
			return nil
		}
		want := "contextual-compression"
		if name == "SUB-SKILL.md" {
			want = path.Base(path.Dir(p))
			if !strings.Contains(string(router), "sub-skills/"+want+"/") {
				problems = append(problems, p+": the router SKILL.md never routes to it")
			}
		}
		meta := map[string]string{}
		if rest, ok := strings.CutPrefix(text, "---\n"); ok {
			if head, _, ok := strings.Cut(rest, "\n---\n"); ok {
				for _, line := range strings.Split(head, "\n") {
					if k, v, ok := strings.Cut(line, ":"); ok {
						meta[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"`)
					}
				}
			}
		}
		if meta["name"] != want {
			problems = append(problems, fmt.Sprintf("%s: name is %q, want %q", p, meta["name"], want))
		}
		if n := utf8.RuneCountInString(meta["description"]); n == 0 || n > 1024 {
			problems = append(problems, fmt.Sprintf("%s: description has %d characters, want 1 to 1024", p, n))
		}
		return nil
	})
	for _, p := range problems {
		fmt.Println(p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Printf("skillcheck: contextual-compression has %d files and a valid policy.json\n", files)
}
