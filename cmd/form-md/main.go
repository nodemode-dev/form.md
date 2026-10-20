package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	formmd "github.com/nodemode-ai/form.md"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/crypt"
	"github.com/nodemode-ai/form.md/internal/i18n"
	"github.com/nodemode-ai/form.md/internal/ops"
	"github.com/nodemode-ai/form.md/internal/server"
	"github.com/nodemode-ai/form.md/internal/settings"
	"github.com/nodemode-ai/form.md/internal/store"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "serve":
		return serve()
	case "version", "--version", "-v":
		fmt.Println(app.Version)
		return 0
	case "keygen", "encrypt", "backup", "restore", "doctor", "admin":
		return tool(cmd, args[1:])
	case "mcp":
		return bridge(args[1:])
	case "cli":
		return cli(args[1:])
	case "help", "--help", "-h":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "form-md: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}
}

func serve() int {
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		return 2
	}
	if err := i18n.LoadFS(formmd.Catalogs()); err != nil {
		log.Error("catalogs", "err", err)
		return 1
	}
	if limit := ops.SizeMemory(); limit > 0 {
		log.Info("memory limit set from the container", "soft_limit_mb", limit>>20)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.Desktop() {
		go func() {
			_, _ = io.Copy(io.Discard, os.Stdin)
			log.Info("the desktop app closed its pipe; stopping")
			stop()
		}()
	}

	key, err := crypt.Load(cfg.DataDir, crypt.KeyPath(cfg.KeyFile), store.HasData(cfg.DataDir))
	if err != nil {
		log.Error("encryption", "err", err)
		return 2
	}
	unlock, err := ops.Lock(cfg.DataDir)
	if err != nil {
		log.Error("data directory", "err", err)
		return 1
	}
	defer unlock()
	st, err := store.Open(ctx, cfg.DataDir, key)
	if err != nil {
		log.Error("storage", "err", err)
		return 1
	}
	if key != nil {
		log.Info("at-rest encryption on", "database", "xts-aes-256", "files", "aes-256-gcm")
	}
	defer func() {
		if err := st.Close(); err != nil {
			log.Error("storage close", "err", err)
		}
	}()

	instanceID, err := st.Bootstrap(ctx, app.Version)
	if err != nil {
		log.Error("bootstrap", "err", err)
		return 1
	}

	srv, err := server.Build(ctx, cfg, log, st, formmd.Dist(), formmd.ContextSkill(), formmd.Notices(), instanceID)
	if err != nil {
		log.Error("startup", "err", err)
		return 1
	}
	ops.OnFlushRequest(ctx, cfg.DataDir, srv.Docs.FlushAll)
	if err := srv.Run(ctx); err != nil {
		log.Error("server", "err", err)
		return 1
	}
	log.Info("stopped cleanly")
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n\nUsage: form-md [command]\n\nCommands:\n", app.Product, app.Version)
	fmt.Fprintln(w, "  serve                 run the server (default)")
	fmt.Fprintln(w, "  backup FILE           write a consistent backup archive while the server runs")
	fmt.Fprintln(w, "  restore FILE          replace the data directory with a backup (server stopped)")
	fmt.Fprintln(w, "  doctor                check the data directory, database, files and settings")
	fmt.Fprintln(w, "  admin reset-owner     set a new password for the instance owner and print it")
	fmt.Fprintln(w, "  admin set KEY VALUE   change an instance setting, for example https.auto false")
	fmt.Fprintln(w, "  keygen                print a new at-rest encryption key")
	fmt.Fprintln(w, "  encrypt               encrypt a plain data directory with FORMMD_KEY_FILE (server stopped)")
	fmt.Fprintln(w, "  mcp                   bridge MCP between standard input and output and a form.md server, for Codex and other clients that start a command; a server on this computer that is not running is started")
	fmt.Fprintln(w, "  cli list|describe|run  the command catalogue of the open window: list [text], describe ID, run ID key=value ...")
	fmt.Fprintln(w, "  version               print the version")
	fmt.Fprintln(w, "  help                  show this help")
	fmt.Fprintln(w, "\nEnvironment:")
	for _, v := range config.Vars {
		def := v.Default
		if def == "" {
			def = "unset"
		}
		fmt.Fprintf(w, "  %s%-10s %s (default: %s)\n", config.EnvPrefix, v.Name, v.Purpose, def)
	}
	fmt.Fprintln(w, "\nFor form-md mcp:")
	fmt.Fprintf(w, "  %-19s server address (default: http://127.0.0.1:8750), or --url\n", bridgeURLEnv)
	fmt.Fprintf(w, "  %-19s personal API token, or --token\n", bridgeTokenEnv)
	fmt.Fprintf(w, "  %-19s set to 0 to stop form-md mcp from starting a local server that is not running\n", "FORMMD_MCP_AUTOSTART")
	fmt.Fprintln(w, "\nInstance setting defaults (a value saved in Instance settings wins):")
	for _, d := range settings.Defs {
		if d.Env != "" {
			fmt.Fprintf(w, "  %-19s setting %s (default: %s)\n", d.Env, d.Key, d.Default)
		}
	}
}
