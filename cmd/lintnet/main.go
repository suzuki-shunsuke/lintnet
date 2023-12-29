package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lintnet/lintnet/pkg/cli"
	"github.com/m-mizutani/clog"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

var (
	version = ""
	commit  = "" //nolint:gochecknoglobals
	date    = "" //nolint:gochecknoglobals
)

func main() {
	handler := clog.New(
		clog.WithColor(true),
		clog.WithSource(true),
	)
	logger := slog.New(handler)
	if err := core(logger); err != nil {
		slogerr.WithError(logger, err).Error("lintnet failed")
	}
}

func core(logger *slog.Logger) error {
	runner := cli.Runner{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		LDFlags: &cli.LDFlags{
			Version: version,
			Commit:  commit,
			Date:    date,
		},
		Logger: logger,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runner.Run(ctx, os.Args...) //nolint:wrapcheck
}
