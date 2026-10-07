package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	formmd "github.com/nodemode-ai/form.md"
	"github.com/nodemode-ai/form.md/internal/app"
	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/static"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DataDir)
	if err != nil {
		log.Error("storage", "err", err)
		return 1
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

	deps := &app.Deps{
		Config:     cfg,
		Store:      st,
		Log:        log,
		Dist:       formmd.Dist(),
		Notices:    formmd.Notices(),
		InstanceID: instanceID,
	}
	modules := []app.Module{
		app.Health(),
		app.Notices(),
		static.Module(),
	}
	if err := app.Run(ctx, deps, modules); err != nil {
		log.Error("server", "err", err)
		return 1
	}
	log.Info("stopped cleanly")
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n\nUsage: form-md [command]\n\nCommands:\n", app.Product, app.Version)
	fmt.Fprintln(w, "  serve     run the server (default)")
	fmt.Fprintln(w, "  version   print the version")
	fmt.Fprintln(w, "  help      show this help")
	fmt.Fprintln(w, "\nEnvironment:")
	for _, v := range config.Vars {
		def := v.Default
		if def == "" {
			def = "unset"
		}
		fmt.Fprintf(w, "  %s%-10s %s (default: %s)\n", config.EnvPrefix, v.Name, v.Purpose, def)
	}
}
