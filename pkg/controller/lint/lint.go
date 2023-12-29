package lint

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lintnet/lintnet/pkg/config"
	"github.com/lintnet/lintnet/pkg/errlevel"
	"github.com/lintnet/lintnet/pkg/module"
)

type ParamLint struct {
	ErrorLevel     string
	RootDir        string
	ConfigFilePath string
	FilePaths      []string
	Outputs        []string
	OutputSuccess  bool
}

// func debug(data any) {
// 	encoder := json.NewEncoder(os.Stdout)
// 	encoder.SetIndent("", "  ")
// 	if err := encoder.Encode(data); err != nil {
// 		fmt.Println("ERROR", err)
// 	}
// }

func (c *Controller) Lint(ctx context.Context, logger *slog.Logger, param *ParamLint) error {
	rawCfg := &config.RawConfig{}
	if err := c.findAndReadConfig(param.ConfigFilePath, rawCfg); err != nil {
		return err
	}
	cfg, err := rawCfg.Parse()
	if err != nil {
		return fmt.Errorf("parse a configuration file: %w", err)
	}

	outputters, err := c.getOutputters(cfg, param.Outputs)
	if err != nil {
		return err
	}

	errLevel, err := c.getErrorLevel(cfg, param)
	if err != nil {
		return err
	}

	if err := c.installModules(ctx, logger, &module.ParamInstall{
		BaseDir: param.RootDir,
	}, cfg.ModuleArchives); err != nil {
		return err
	}

	targets, err := c.findFiles(cfg, param.RootDir)
	if err != nil {
		return err
	}

	if len(param.FilePaths) > 0 {
		logger.Debug("filtering targets by given files")
		targets = filterTargets(targets, param.FilePaths)
	}

	results, err := c.getResults(targets)
	if err != nil {
		return err
	}
	logger.Debug("linted", slog.Any("results", results))

	return c.Output(logger, errLevel, results, outputters, param.OutputSuccess)
}

func (c *Controller) getOutputters(cfg *config.Config, outputIDs []string) ([]Outputter, error) {
	outputs, err := c.getOutputs(cfg, outputIDs)
	if err != nil {
		return nil, err
	}
	outputters := make([]Outputter, len(outputs))
	for i, output := range outputs {
		o, err := c.getOutputter(output)
		if err != nil {
			return nil, err
		}
		outputters[i] = o
	}
	return outputters, nil
}

func (c *Controller) getResults(targets []*Target) (map[string]*FileResult, error) {
	results := make(map[string]*FileResult, len(targets))
	for _, target := range targets {
		if err := c.lintTarget(target, results); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (c *Controller) lintTarget(target *Target, results map[string]*FileResult) error {
	lintFiles, err := c.parseLintFiles(target.LintFiles)
	if err != nil {
		return err
	}
	for _, dataFile := range target.DataFiles {
		rs, err := c.lint(dataFile, lintFiles)
		if err != nil {
			results[dataFile] = &FileResult{
				Error: err.Error(),
			}
			continue
		}
		results[dataFile] = &FileResult{
			Results: rs,
		}
	}
	return nil
}

func (c *Controller) getErrorLevel(cfg *config.Config, param *ParamLint) (errlevel.Level, error) {
	if param.ErrorLevel == "" {
		return cfg.ErrorLevel, nil
	}
	ll, err := errlevel.New(param.ErrorLevel)
	if err != nil {
		return ll, err //nolint:wrapcheck
	}
	return ll, nil
}

func (c *Controller) lint(dataFile string, jsonnetAsts []*Node) ([]*Result, error) {
	tla, err := c.parseDataFile(dataFile)
	if err != nil {
		return nil, err
	}

	results := c.evaluate(tla.Data, jsonnetAsts)
	rs := make([]*Result, len(results))

	for i, result := range results {
		rs[i] = c.parseResult(result)
	}
	return rs, nil
}
