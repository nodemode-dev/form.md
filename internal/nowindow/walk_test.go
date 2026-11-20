package nowindow

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Every program the server starts goes through Command or CommandContext:
// no file of the server (cmd and internal, tests left out) calls
// exec.Command, exec.CommandContext or os.StartProcess itself.
func TestEveryStartGoesThroughTheHelper(t *testing.T) {
	root := filepath.Join("..", "..")
	found := 0
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if abs, _ := filepath.Abs(filepath.Dir(path)); abs == here(t) {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			found++
			ast.Inspect(file, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if (pkg.Name == "exec" && (sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext")) ||
					(pkg.Name == "os" && sel.Sel.Name == "StartProcess") {
					t.Errorf("%s: %s.%s starts a program without nowindow", path, pkg.Name, sel.Sel.Name)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if found < 50 {
		t.Fatalf("read %d Go files under cmd and internal, expected more", found)
	}
}

func here(t *testing.T) string {
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
