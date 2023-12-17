package cli

import (
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type Runner struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	LDFlags *LDFlags
	LogE    *logrus.Entry
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
		Usage:    "Lint with Jsonnet. https://github.com/suzuki-shunsuke/lintnet",
		Version:  r.LDFlags.Version + " (" + r.LDFlags.Commit + ")",
		Compiled: compiledDate,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "log level",
				EnvVars: []string{"LINTNET_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "log-color",
				Usage:   "Log color. One of 'auto' (default), 'always', 'never'",
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
				logE: r.LogE,
			}).command(),
		},
	}

	return app.RunContext(ctx, args) //nolint:wrapcheck
}
