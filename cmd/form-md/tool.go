package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/ops"
	"github.com/nodemode-ai/form.md/internal/settings"
)

func tool(cmd string, args []string) int {
	if cmd == "keygen" {
		fmt.Println(crypt.Generate())
		return 0
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "form-md:", err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fail := func(err error) int {
		fmt.Fprintln(os.Stderr, "form-md "+cmd+":", err)
		return 1
	}
	switch cmd {
	case "backup":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "usage: form-md backup FILE")
			return 2
		}
		m, err := ops.Backup(ctx, cfg.DataDir, cfg.KeyFile, args[0], func() bool { return ops.RequestFlush(cfg.DataDir) })
		if err != nil {
			return fail(err)
		}
		state := "plain"
		if m.Encrypted {
			state = "encrypted, restore needs the same key"
		}
		if m.Live {
			state += "; the running server saved open documents first"
		}
		fmt.Printf("backup written to %s: database and %d files, %d bytes before compression (%s)\n", args[0], m.Files, m.Bytes, state)
	case "restore":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "usage: form-md restore FILE")
			return 2
		}
		old, m, err := ops.Restore(ctx, cfg.DataDir, cfg.KeyFile, args[0])
		if err != nil {
			return fail(err)
		}
		fmt.Printf("restored a backup from version %s (%d files) into %s\n", m.Version, m.Files, cfg.DataDir)
		if old != "" {
			fmt.Printf("the previous data directory was kept at %s; delete it once the restore is checked\n", old)
		}
	case "doctor":
		if !ops.Print(os.Stdout, ops.Doctor(ctx, cfg)) {
			return 1
		}
	case "encrypt":
		rep, err := ops.Encrypt(ctx, cfg.DataDir, cfg.KeyFile)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("encrypted the database and %d files (%d already encrypted, %d unfinished temporary files removed); start the server with the same FORMMD_KEY_FILE\n", rep.Files, rep.Skipped, rep.Removed)
	case "admin":
		if len(args) == 3 && args[0] == "set" {
			v, err := ops.SetSetting(ctx, cfg.DataDir, cfg.KeyFile, args[1], args[2])
			if err != nil {
				return fail(err)
			}
			if settings.IsSecret(args[1]) {
				v = "(saved)"
			}
			fmt.Printf("%s = %s; a running server picks this up at its next start\n", args[1], v)
			return 0
		}
		if len(args) != 1 || args[0] != "reset-owner" {
			fmt.Fprintln(os.Stderr, "usage: form-md admin reset-owner | form-md admin set KEY VALUE")
			return 2
		}
		email, password, err := ops.ResetOwner(ctx, cfg.DataDir, cfg.KeyFile)
		if err != nil {
			return fail(err)
		}
		fmt.Printf("owner %s: new password %s\ntwo-step sign-in was turned off and every session signed out; set a new password after signing in\n", email, password)
	}
	return 0
}
