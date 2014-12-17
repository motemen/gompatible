package gompatible

import (
	"fmt"
	"go/build"
	"golang.org/x/tools/go/buildutil"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/hgcmd"
)

type DirSpec struct {
	VCS      string
	Revision string
	Path     string

	ctx *build.Context
}

func (dir DirSpec) String() string {
	if dir.VCS == "" || dir.Revision == "" {
		return dir.Path
	}

	return fmt.Sprintf("%s:%s:%s", dir.VCS, dir.Revision, dir.Path)
}

func (dir DirSpec) Subdir(name string) DirSpec {
	ctx, _ := dir.BuildContext() // FIXME
	dupped := dir
	dupped.Path = buildutil.JoinPath(ctx, dir.Path, name)
	return dupped
}

func (dir DirSpec) ReadDir() ([]os.FileInfo, error) {
	ctx, err := dir.BuildContext() // FIXME
	if err != nil {
		return nil, err
	}

	return buildutil.ReadDir(ctx, dir.Path)
}

func (dir DirSpec) BuildContext() (*build.Context, error) {
	if dir.ctx != nil {
		return dir.ctx, nil
	}

	ctx := build.Default // copy

	if dir.VCS != "" && dir.Revision != "" {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = dir.Path

		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		repoRoot := strings.TrimRight(string(out), "\n")
		repo, err := vcs.Open(dir.VCS, repoRoot)
		if err != nil {
			return nil, err
		}

		commit, err := repo.ResolveRevision(dir.Revision)
		if err != nil {
			return nil, err
		}

		fs, err := repo.FileSystem(commit)
		if err != nil {
			return nil, err
		}

		ctx.OpenFile = func(path string) (io.ReadCloser, error) {
			Debugf("OpenFile %s", path)

			if buildutil.IsAbsPath(&ctx, path) {
				// the path maybe outside of repository (for standard libraries)
				if strings.HasPrefix(path, repoRoot) {
					var err error
					path, err = filepath.Rel(repoRoot, path)
					if err != nil {
						return nil, err
					}
				} else {
					return os.Open(path)
				}
			}

			return fs.Open(path)
		}

		ctx.ReadDir = func(path string) ([]os.FileInfo, error) {
			Debugf("ReadDir %s", path)

			if filepath.IsAbs(path) {
				if strings.HasPrefix(path, repoRoot) {
					var err error
					path, err = filepath.Rel(repoRoot, path)
					if err != nil {
						return nil, err
					}
				} else {
					return ioutil.ReadDir(path)
				}
			}

			return fs.ReadDir(path)
		}
	}

	dir.ctx = &ctx

	return dir.ctx, nil
}
