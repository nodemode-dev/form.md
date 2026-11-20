package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWherePrintsTheProgramTheFinderChose(t *testing.T) {
	name := "claude-stand-in"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FORMMD_CLAUDE_BIN", path)
	var out, errs bytes.Buffer
	if code := whereTo(&out, &errs, []string{"claude"}); code != 0 {
		t.Fatalf("exit %d, said %q", code, errs.String())
	}
	if got := strings.TrimSpace(out.String()); got != path {
		t.Fatalf("printed %q, want %q", got, path)
	}
}

func TestWhereNamesTheFourProgramsForAnythingElse(t *testing.T) {
	for _, args := range [][]string{nil, {"vim"}, {"claude", "codex"}} {
		var out, errs bytes.Buffer
		if code := whereTo(&out, &errs, args); code != 2 {
			t.Fatalf("%v: exit %d, want 2", args, code)
		}
		if out.Len() != 0 || !strings.Contains(errs.String(), "claude|codex|cursor|antigravity") {
			t.Fatalf("%v: out %q, errs %q", args, out.String(), errs.String())
		}
	}
}
