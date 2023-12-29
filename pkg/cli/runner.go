package cli

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/urfave/cli/v2"
)

type Runner struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	LDFlags *LDFlags
	Logger  *slog.Logger
}

type LDFlags struct {
	Version string
	Commit  string
	Date    string
}

func (r *Runner) Run(ctx context.Context, args ...string) error {
	compiledDate, err := time.Parse(time.RFC3339, r.LDFlags.Date)
	if err != nil {
		compiledDate = time.Now()
	}
	app := cli.App{
		Name:     "lintnet",
		Usage:    "Lint with Jsonnet. https://github.com/lintnet/lintnet",
		Version:  r.LDFlags.Version + " (" + r.LDFlags.Commit + ")",
		Compiled: compiledDate,
		// DefaultCommand: "lint",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "log level",
				EnvVars: []string{"LINTNET_LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "log-color",
				Usage:   "Log color",
				Value:   true,
				EnvVars: []string{"LINTNET_LOG_COLOR"},
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Configuration file path",
				EnvVars: []string{"LINTNET_CONFIG"},
			},
		},
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			(&versionCommand{}).command(),
			(&lintCommand{
				version: r.LDFlags.Version,
				stderr:  r.Stderr,
			}).command(),
			(&initCommand{
				version: r.LDFlags.Version,
				stderr:  r.Stderr,
			}).command(),
			(&testCommand{
				version: r.LDFlags.Version,
				stderr:  r.Stderr,
			}).command(),
		},
	}

	return app.RunContext(ctx, args) //nolint:wrapcheck
}
