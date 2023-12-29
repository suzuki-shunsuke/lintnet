package lint

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/lintnet/lintnet/pkg/config"
	"github.com/lintnet/lintnet/pkg/jsonnet"
	"github.com/spf13/afero"
)

//go:embed test_diff.txt
var testResultTemplateByte []byte

func (c *Controller) Test(_ context.Context, logger *slog.Logger, param *ParamLint) error { //nolint:funlen,cyclop,gocognit
	rawCfg := &config.RawConfig{}
	if err := c.findAndReadConfig(param.ConfigFilePath, rawCfg); err != nil {
		return err
	}
	cfg, err := rawCfg.Parse()
	if err != nil {
		return fmt.Errorf("parse a configuration file: %w", err)
	}

	testResultTemplate, err := template.New("_").Parse(string(testResultTemplateByte))
	if err != nil {
		return fmt.Errorf("parse the template of test result: %w", err)
	}

	targets, err := c.findFiles(cfg, param.RootDir)
	if err != nil {
		return err
	}

	pairs := c.filterTargetsWithTest(logger, targets)
	failedResults := make([]*FailedResult, 0, len(pairs))
	for _, pair := range pairs {
		testData := []*TestData{}
		if err := jsonnet.Read(c.fs, pair.TestFilePath, "{}", c.importer, &testData); err != nil {
			failedResults = append(failedResults, &FailedResult{
				LintFilePath: pair.LintFilePath,
				TestFilePath: pair.TestFilePath,
				Error:        fmt.Errorf("read a test file: %w", err).Error(),
			})
			continue
		}
		for _, td := range testData {
			if td.DataFile != "" {
				dataFilePath := filepath.Join(filepath.Dir(pair.TestFilePath), td.DataFile)
				data, err := c.parseDataFile(dataFilePath)
				if err != nil {
					failedResults = append(failedResults, &FailedResult{
						Name:         td.Name,
						LintFilePath: pair.LintFilePath,
						TestFilePath: pair.TestFilePath,
						Param:        td.Param,
						Error:        fmt.Errorf("read a data file: %w", err).Error(),
					})
					continue
				}
				if td.Param != nil && td.Param.Data != nil && td.Param.Data.FilePath != "" {
					data.Data.FilePath = td.Param.Data.FilePath
				}
				if td.Param != nil {
					data.Custom = td.Param.Custom
				}
				td.Param = data
			}
			if td.Param.Custom == nil {
				td.Param.Custom = map[string]any{}
			}
			tlaB, err := json.Marshal(td.Param)
			if err != nil {
				failedResults = append(failedResults, &FailedResult{
					Name:         td.Name,
					LintFilePath: pair.LintFilePath,
					TestFilePath: pair.TestFilePath,
					Param:        td.Param,
					Error:        fmt.Errorf("marshal param as JSON: %w", err).Error(),
				})
				continue
			}
			var result any
			if err := jsonnet.Read(c.fs, pair.LintFilePath, string(tlaB), c.importer, &result); err != nil {
				failedResults = append(failedResults, &FailedResult{
					Name:         td.Name,
					LintFilePath: pair.LintFilePath,
					TestFilePath: pair.TestFilePath,
					Param:        td.Param,
					Error:        fmt.Errorf("read a lint file: %w", err).Error(),
				})
				continue
			}
			if diff := cmp.Diff(td.Result, result); diff != "" {
				failedResults = append(failedResults, &FailedResult{
					Name:         td.Name,
					LintFilePath: pair.LintFilePath,
					TestFilePath: pair.TestFilePath,
					Wanted:       td.Result,
					Param:        td.Param,
					Got:          result,
					Diff:         diff,
				})
			}
		}
	}
	if len(failedResults) == 0 {
		return nil
	}
	if err := testResultTemplate.Execute(c.stdout, failedResults); err != nil {
		return fmt.Errorf("render the result: %w", err)
	}
	return nil
}

type TestData struct {
	Name     string           `json:"name,omitempty"`
	DataFile string           `json:"data_file,omitempty"`
	Param    *TopLevelArgment `json:"param,omitempty"`
	Result   any              `json:"result,omitempty"`
}

type TestPair struct {
	LintFilePath string
	TestFilePath string
}

type FailedResult struct {
	Name         string `json:"name,omitempty"`
	LintFilePath string `json:"lint_file_path,omitempty"`
	TestFilePath string `json:"test_file_path,omitempty"`
	Param        any    `json:"param,omitempty"`
	Wanted       any    `json:"wanted,omitempty"`
	Got          any    `json:"got,omitempty"`
	Diff         string `json:"diff,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (c *Controller) filterTargetsWithTest(logger *slog.Logger, targets []*Target) []*TestPair {
	pairs := []*TestPair{}
	for _, target := range targets {
		for _, lintFile := range target.LintFiles {
			if lintFile.Path == "" {
				continue
			}
			baseName := filepath.Base(lintFile.Path)
			ext := filepath.Ext(baseName)
			testFileName := fmt.Sprintf("%s_test%s", strings.TrimSuffix(baseName, filepath.Ext(baseName)), ext)
			testFilePath := filepath.Join(filepath.Dir(lintFile.Path), testFileName)
			f, err := afero.Exists(c.fs, testFilePath)
			if err != nil {
				logger.Warn("check if a test file exists", slog.String("error", err.Error()))
				continue
			}
			if !f {
				continue
			}
			pairs = append(pairs, &TestPair{
				LintFilePath: lintFile.Path,
				TestFilePath: testFilePath,
			})
		}
	}
	return pairs
}
