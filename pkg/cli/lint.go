package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/lintnet/lintnet/pkg/config"
	"github.com/lintnet/lintnet/pkg/controller/lint"
	"github.com/lintnet/lintnet/pkg/github"
	"github.com/lintnet/lintnet/pkg/jsonnet"
	"github.com/lintnet/lintnet/pkg/log"
	"github.com/lintnet/lintnet/pkg/module"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
	"github.com/urfave/cli/v2"
)

type lintCommand struct {
	version string
	stderr io.Writer
}

func (lc *lintCommand) command() *cli.Command {
	return &cli.Command{
		Name:   "lint",
		Usage:  "Lint files",
		Action: lc.action,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "rule-base-dir",
				Aliases: []string{"d"},
			},
			&cli.StringFlag{
				Name:    "error-level",
				Aliases: []string{"e"},
				EnvVars: []string{"LINTNET_ERROR_LEVEL"},
			},
			&cli.BoolFlag{
				Name:    "output-success",
				EnvVars: []string{"LINTNET_OUTPUT_SUCCESS"},
			},
		},
	}
}

func (lc *lintCommand) action(c *cli.Context) error { //nolint:dupl
	fs := afero.NewOsFs()
	logger, err := log.New(&log.ParamNew{
		Out:     os.Stderr,
		Level:   c.String("log-level"),
		Color:   c.Bool("log-color"),
		Version: lc.version,
	})
	if err != nil {
		return fmt.Errorf("initialize a logger: %w", err)
	}
	rootDir := os.Getenv("LINTNET_ROOT_DIR")
	if rootDir == "" {
		dir, err := config.GetRootDir()
		if err != nil {
			slogerr.WithError(logger, err).Warn("get the root directory")
		}
		rootDir = dir
	}
	modInstaller := module.NewInstaller(fs, github.New(c.Context), http.DefaultClient)
	importer := jsonnet.NewImporter(c.Context, logger, &module.ParamInstall{
		BaseDir: rootDir,
	}, &jsonnet.FileImporter{
		JPaths: []string{rootDir},
	}, modInstaller)
	param := &lint.ParamController{
		Version: lc.version,
	}
	ctrl := lint.NewController(param, fs, os.Stdout, modInstaller, importer)
	return ctrl.Lint(c.Context, logger, &lint.ParamLint{ //nolint:wrapcheck
		FilePaths:      c.Args().Slice(),
		ErrorLevel:     c.String("error-level"),
		ConfigFilePath: c.String("config"),
		OutputSuccess:  c.Bool("output-success"),
		RootDir:        rootDir,
	})
}
