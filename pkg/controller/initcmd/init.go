package initcmd

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/spf13/afero"
)

//go:embed init.jsonnet
var cfgTemplate []byte

func (c *Controller) Init(_ context.Context, _ *slog.Logger) error {
	paths := []string{
		"lintnet.jsonnet",
		".lintnet.jsonnet",
	}
	for _, path := range paths {
		f, err := afero.Exists(c.fs, path)
		if err != nil {
			return fmt.Errorf("check if a configuration file exists: %w", err)
		}
		if f {
			return nil
		}
	}
	f, err := c.fs.Create("lintnet.jsonnet")
	if err != nil {
		return fmt.Errorf("create a configuration file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(cfgTemplate); err != nil {
		return fmt.Errorf("write a configuration file: %w", err)
	}
	return nil
}
