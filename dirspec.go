package gompatible

import (
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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
	dupped := dir
	dupped.Path = path.Join(dir.Path, name) // TODO use ctx.JoinPath
	return dupped
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

			// TODO use ctx.IsAbsPath
			if filepath.IsAbs(path) {
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
			Debugf("ReadDir %s", path)
			return fs.ReadDir(path)
		}
	}

	dir.ctx = &ctx

	return dir.ctx, nil
}
