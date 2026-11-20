package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nodemode-ai/form.md/internal/agentlookup"
	"github.com/nodemode-ai/form.md/internal/claudecode"
)

func where(args []string) int {
	return whereTo(os.Stdout, os.Stderr, args)
}

func whereTo(out, errs io.Writer, args []string) int {
	if len(args) != 1 || claudecode.Overrides[args[0]] == nil {
		fmt.Fprintln(errs, "usage: form-md where claude|codex|cursor|antigravity")
		return 2
	}
	found := agentlookup.Find(args[0], agentlookup.Query{OverrideVars: claudecode.Overrides[args[0]], ShellTimeout: 10 * time.Second})
	if found.Path == "" {
		fmt.Fprintln(errs, agentlookup.Sentence(found, agentlookup.Install(found.Program, ""), claudecode.Otherwise))
		return 1
	}
	fmt.Fprintln(out, found.Path)
	return 0
}
