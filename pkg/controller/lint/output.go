package lint

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"

	"github.com/lintnet/lintnet/pkg/config"
	"github.com/lintnet/lintnet/pkg/errlevel"
	"github.com/lintnet/lintnet/pkg/jsonnet"
	"github.com/lintnet/lintnet/pkg/render"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/logrus-error/logerr"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

func (c *Controller) Output(logger *slog.Logger, errLevel errlevel.Level, results map[string]*FileResult, outputters []Outputter, outputSuccess bool) error {
	fes := c.formatResultToOutput(results)
	failed, err := isFailed(fes.Errors, errLevel)
	if err != nil {
		return err
	}
	if !outputSuccess && !failed {
		return nil
	}
	for _, outputter := range outputters {
		if err := outputter.Output(fes); err != nil {
			slogerr.WithError(logger, err).Error("output errors")
		}
	}
	if failed {
		return errors.New("lint failed")
	}
	return nil
}

func (c *Controller) getOutputs(cfg *config.Config, outputIDs []string) ([]*config.Output, error) {
	outputList := cfg.Outputs
	if len(outputList) == 0 {
		outputList = []*config.Output{
			{
				ID:       "stdout",
				Type:     "stdout",
				Renderer: "jsonnet",
			},
		}
	}
	if len(outputIDs) == 0 {
		outputIDs = []string{
			"stdout",
		}
	}
	outputs := make([]*config.Output, len(outputIDs))
	outputMap := make(map[string]*config.Output, len(outputList))
	for _, output := range outputList {
		outputMap[output.ID] = output
	}
	for i, outputID := range outputIDs {
		output, ok := outputMap[outputID]
		if !ok {
			return nil, logerr.WithFields(errors.New("unknown output id"), logrus.Fields{ //nolint:wrapcheck
				"output_id": outputID,
			})
		}
		outputs[i] = output
	}
	return outputs, nil
}

type jsonnetOutputter struct {
	fs     afero.Fs
	stdout io.Writer
	output *config.Output
	node   jsonnet.Node
}

func newJsonnetOutputter(fs afero.Fs, stdout io.Writer, output *config.Output) (*jsonnetOutputter, error) {
	var node jsonnet.Node
	if output.Template != "" {
		n, err := jsonnet.ReadToNode(fs, output.Template)
		if err != nil {
			return nil, fmt.Errorf("read a template as Jsonnet: %w", err)
		}
		node = n
	}
	return &jsonnetOutputter{
		fs:     fs,
		stdout: stdout,
		output: output,
		node:   node,
	}, nil
}

func (o *jsonnetOutputter) Output(result *Output) error {
	out := o.stdout
	if o.output.Type == "file" {
		f, err := o.fs.Create(o.output.Path)
		if err != nil {
			return fmt.Errorf("create a file: %w", err)
		}
		defer f.Close()
		out = f
	}
	if o.output.Template != "" {
		b, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal results as JSON: %w", err)
		}
		vm := jsonnet.MakeVM()
		vm.TLACode("param", string(b))
		jsonnet.SetNativeFunctions(vm)
		result, err := vm.Evaluate(o.node)
		if err != nil {
			return fmt.Errorf("evaluate a Jsonnet: %w", err)
		}
		fmt.Fprintln(out, result)
		return nil
	}
	return outputJSON(out, result)
}

func outputJSON(w io.Writer, result any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode the result as JSON: %w", err)
	}
	return nil
}

type Outputter interface {
	Output(result *Output) error
}

type templateOutputter struct {
	stdout   io.Writer
	fs       afero.Fs
	output   *config.Output
	template render.Template
}

func newTemplateOutputter(stdout io.Writer, fs afero.Fs, renderer render.TemplateRenderer, output *config.Output) (*templateOutputter, error) {
	if output.Template == "" {
		return nil, errors.New("template is required")
	}
	b, err := afero.ReadFile(fs, output.Template)
	if err != nil {
		return nil, fmt.Errorf("read a template: %w", err)
	}
	tpl, err := renderer.Compile(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse a template: %w", err)
	}
	return &templateOutputter{
		stdout:   stdout,
		fs:       fs,
		output:   output,
		template: tpl,
	}, nil
}

func (o *templateOutputter) Output(result *Output) error {
	out := o.stdout
	if o.output.Type == "file" {
		f, err := o.fs.Create(o.output.Path)
		if err != nil {
			return fmt.Errorf("create a file: %w", err)
		}
		defer f.Close()
		out = f
	}
	if err := o.template.Execute(out, map[string]any{
		"result": result,
	}); err != nil {
		return fmt.Errorf("render a template: %w", err)
	}
	return nil
}

func (c *Controller) getOutputter(output *config.Output) (Outputter, error) {
	switch output.Renderer {
	case "jsonnet":
		return newJsonnetOutputter(c.fs, c.stdout, output)
	case "text/template":
		return newTemplateOutputter(c.stdout, c.fs, &render.TextTemplateRenderer{}, output)
	case "html/template":
		return newTemplateOutputter(c.stdout, c.fs, &render.HTMLTemplateRenderer{}, output)
	}
	return nil, errors.New("unknown renderer")
}

type FlatError struct {
	RuleName     string `json:"rule,omitempty"`
	Level        string `json:"level,omitempty"`
	Message      string `json:"message,omitempty"`
	LintFilePath string `json:"lint_file,omitempty"`
	DataFilePath string `json:"data_file,omitempty"`
	Location     any    `json:"location,omitempty"`
	Custom       any    `json:"custom,omitempty"`
}

func (e *FlatError) Failed(errLevel errlevel.Level) (bool, error) {
	level := errlevel.Error
	if e.Level != "" {
		feErrLevel, err := errlevel.New(e.Level)
		if err != nil {
			return false, fmt.Errorf("verify the error level of a result: %w", err)
		}
		level = feErrLevel
	}
	return level >= errLevel, nil
}

type Output struct {
	LintnetVersion string       `json:"lintnet_version"`
	Env            string       `json:"env"`
	Errors         []*FlatError `json:"errors,omitempty"`
}

func (c *Controller) formatResultToOutput(results map[string]*FileResult) *Output {
	list := make([]*FlatError, 0, len(results))
	for dataFilePath, fileResult := range results {
		list = append(list, fileResult.flattenError(dataFilePath)...)
	}
	return &Output{
		LintnetVersion: c.param.Version,
		Env:            fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Errors:         list,
	}
}
