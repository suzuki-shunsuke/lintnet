package cli

import (
	"fmt"
	"io"

	"github.com/lintnet/lintnet/pkg/controller/initcmd"
	"github.com/lintnet/lintnet/pkg/log"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

type initCommand struct {
	version string
	stderr  io.Writer
}

func (lc *initCommand) command() *cli.Command {
	return &cli.Command{
		Name:   "init",
		Usage:  "Scaffold configuration file",
		Action: lc.action,
	}
}

func (lc *initCommand) action(c *cli.Context) error {
	ctrl := initcmd.NewController(afero.NewOsFs())
	logger, err := log.New(&log.ParamNew{
		Level:   c.String("log-level"),
		Color:   c.Bool("log-color"),
		Version: lc.version,
	})
	if err != nil {
		return fmt.Errorf("initialize a logger: %w", err)
	}
	return ctrl.Init(c.Context, logger) //nolint:wrapcheck
}
