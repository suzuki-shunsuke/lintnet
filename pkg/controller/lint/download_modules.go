package lint

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/go-github/v57/github"
	"github.com/lintnet/lintnet/pkg/module"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type ParamDownloadModule struct {
	BaseDir string
}

type GitHub interface {
	GetArchiveLink(ctx context.Context, owner, repo string, archiveformat github.ArchiveFormat, opts *github.RepositoryContentGetOptions, maxRedirects int) (*url.URL, *github.Response, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (c *Controller) installModules(ctx context.Context, logger *slog.Logger, param *module.ParamInstall, modules map[string]*module.Archive) error {
	for _, mod := range modules {
		logE := logger.With(slog.String("module_id", mod.ID))
		if err := c.moduleInstaller.Install(ctx, logE, param, mod); err != nil {
			return fmt.Errorf("install a module: %w",
				slogerr.WithAttrs(err, slog.String("module_id", mod.ID)))
		}
	}
	return nil
}
