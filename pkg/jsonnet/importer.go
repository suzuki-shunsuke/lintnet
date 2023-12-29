package jsonnet

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/lintnet/lintnet/pkg/module"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type FileImporter = jsonnet.FileImporter

type Importer struct {
	ctx             context.Context //nolint:containedctx
	logger          *slog.Logger
	param           *module.ParamInstall
	importer        jsonnet.Importer
	moduleInstaller *module.Installer
}

func NewImporter(ctx context.Context, logger *slog.Logger, param *module.ParamInstall, importer jsonnet.Importer, installer *module.Installer) *Importer {
	return &Importer{
		ctx:             ctx,
		logger:          logger,
		param:           param,
		importer:        importer,
		moduleInstaller: installer,
	}
}

func (ip *Importer) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	contents, foundAt, err := ip.importer.Import(importedFrom, importedPath)
	if err == nil {
		return contents, foundAt, nil
	}
	if !strings.HasPrefix(importedPath, "github.com/") {
		return contents, foundAt, err //nolint:wrapcheck
	}
	mod, err := module.ParseModuleLine(importedPath)
	if err != nil {
		return contents, foundAt, fmt.Errorf("parse a module import path: %w", err)
	}
	if err := ip.moduleInstaller.Install(ip.ctx, ip.logger, ip.param, mod.Archive); err != nil {
		return contents, foundAt, fmt.Errorf("install a module: %w",
			slogerr.WithAttrs(err, slog.String("module_id", mod.Archive.ID), slog.String("import", importedPath)))
	}
	return ip.importer.Import(importedFrom, mod.SlashPath) //nolint:wrapcheck
}
