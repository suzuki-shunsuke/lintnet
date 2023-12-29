package module

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/go-github/v57/github"
	"github.com/lintnet/lintnet/pkg/osfile"
	"github.com/mholt/archiver/v3"
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/slog-error/slogerr"
)

type ParamInstall struct {
	BaseDir string
}

type GitHub interface {
	GetArchiveLink(ctx context.Context, owner, repo string, archiveformat github.ArchiveFormat, opts *github.RepositoryContentGetOptions, maxRedirects int) (*url.URL, *github.Response, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Installer struct {
	fs   afero.Fs
	gh   GitHub
	http HTTPClient
}

func NewInstaller(fs afero.Fs, gh GitHub, httpClient HTTPClient) *Installer {
	return &Installer{
		fs:   fs,
		gh:   gh,
		http: httpClient,
	}
}

func (mi *Installer) Install(ctx context.Context, logger *slog.Logger, param *ParamInstall, mod *Archive) error { //nolint:funlen,cyclop
	// Check if the module is already downloaded
	dest := filepath.Join(param.BaseDir, filepath.FromSlash(mod.ID))
	f, err := afero.DirExists(mi.fs, dest)
	if err != nil {
		return fmt.Errorf("check if the module is already installed: %w", err)
	}
	if f {
		return nil
	}
	if err := osfile.MkdirAll(mi.fs, filepath.Dir(dest)); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}
	// Download Module
	u, _, err := mi.gh.GetArchiveLink(ctx, mod.RepoOwner, mod.RepoName, github.Tarball, &github.RepositoryContentGetOptions{
		Ref: mod.Ref,
	}, 5) //nolint:gomnd
	if err != nil {
		return fmt.Errorf("get an archive link by GitHub API: %w",
			slogerr.WithAttrs(err,
				slog.String("moduel_repo_owner", mod.RepoOwner),
				slog.String("moduel_repo_name", mod.RepoName),
				slog.String("moduel_ref", mod.Ref),
			))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create a HTTP request: %w", err)
	}
	resp, err := mi.http.Do(req)
	if err != nil {
		return fmt.Errorf("send a HTTP request: %w", err)
	}
	if resp.StatusCode >= 300 { //nolint:gomnd
		return errors.New("HTTP status code >= 300")
	}
	defer resp.Body.Close()
	tempDir, err := afero.TempDir(mi.fs, "", "")
	if err != nil {
		return fmt.Errorf("create a temporal directory: %w", err)
	}
	defer func() {
		if err := mi.fs.RemoveAll(tempDir); err != nil {
			logger.Warn("delete a temporal directory", slog.String("error", err.Error()))
		}
	}()
	tempDest := filepath.Join(tempDir, "module.tar.gz")
	tempFile, err := mi.fs.Create(tempDest)
	if err != nil {
		return fmt.Errorf("create a temporal file: %w", err)
	}
	defer tempFile.Close()
	logger.Info("downloading a module")
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("download a module on a temporal directory: %w", err)
	}
	tarGz := archiver.NewTarGz()
	unarchiveDest := filepath.Join(tempDir, "unarchived_dir")
	if err := tarGz.Unarchive(tempDest, unarchiveDest); err != nil {
		return fmt.Errorf("unarchive a tarball: %w", err)
	}
	dirs, err := os.ReadDir(unarchiveDest)
	if err != nil {
		return fmt.Errorf("read a directory: %w", err)
	}
	if len(dirs) != 1 {
		return fmt.Errorf("the number of sub directories must be one")
	}
	if err := osfile.Copy(mi.fs, filepath.Join(unarchiveDest, dirs[0].Name()), dest); err != nil {
		return fmt.Errorf("copy a module from a teporal directory: %w", err)
	}
	return nil
}
