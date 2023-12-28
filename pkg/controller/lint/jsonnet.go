package lint

import (
	"encoding/json"
	"fmt"

	"github.com/lintnet/lintnet/pkg/jsonnet"
	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/logrus-error/logerr"
)

func (c *Controller) parseLintFiles(lintFiles []*LintFile) ([]*Node, error) {
	jsonnetAsts := make([]*Node, 0, len(lintFiles))
	for _, lintFile := range lintFiles {
		node, err := c.parseLintFile(lintFile)
		if err != nil {
			return nil, logerr.WithFields(err, logrus.Fields{ //nolint:wrapcheck
				"file_path": lintFile.Path,
			})
		}
		jsonnetAsts = append(jsonnetAsts, node)
	}
	return jsonnetAsts, nil
}

func (c *Controller) parseLintFile(lintFile *LintFile) (*Node, error) {
	ja, err := jsonnet.ReadToNode(c.fs, lintFile.Path)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	key := lintFile.Path
	if lintFile.ModulePath != "" {
		key = lintFile.ModulePath
	}
	return &Node{
		Node:   ja,
		Key:    key,
		Custom: lintFile.Param,
	}, nil
}

type Node struct {
	Node   jsonnet.Node
	Custom map[string]any
	Key    string
}

func (c *Controller) evaluateNode(data *Data, ja *Node) *JsonnetEvaluateResult {
	tla := &TopLevelArgment{
		Data:   data,
		Custom: ja.Custom,
	}
	if tla.Custom == nil {
		tla.Custom = map[string]any{}
	}
	tlaB, err := json.Marshal(tla)
	if err != nil {
		return &JsonnetEvaluateResult{
			Key:   ja.Key,
			Error: fmt.Errorf("marshal a top level argument as JSON: %w", err).Error(),
		}
	}
	vm := jsonnet.NewVM(string(tlaB), c.importer)
	result, err := vm.Evaluate(ja.Node)
	if err != nil {
		return &JsonnetEvaluateResult{
			Key:   ja.Key,
			Error: err.Error(),
		}
	}
	return &JsonnetEvaluateResult{
		Key:    ja.Key,
		Result: result,
	}
}

func (c *Controller) evaluate(data *Data, jsonnetAsts []*Node) []*JsonnetEvaluateResult {
	results := make([]*JsonnetEvaluateResult, len(jsonnetAsts))
	for i, ja := range jsonnetAsts {
		results[i] = c.evaluateNode(data, ja)
	}
	return results
}
