package gompatible

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go/build"
	"golang.org/x/tools/go/buildutil"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/hgcmd"

	// _ "sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	_ "github.com/motemen/go-vcs-gitcmd-fastopen"
)

// DirSpec represents a virtual directory which may point to a source tree of a
// specific Revision controlled under a VCS.
type DirSpec struct {
	VCS      string
	Revision string
	Path     string

	// vcs root directory
	root string

	pkgOverride string

	ctx *build.Context
}

// NewDirSpec creates a virtual directory which may point to a source tree of a
// specific revision controlled under a vcs.
// The path may be one of:
//   - An absolute path
//   - An import path
//   - A relative import path
func NewDirSpec(path, vcs, revision string) (*DirSpec, error) {
	if _, err := os.Stat(path); err != nil {
		bPkg, err := build.Default.Import(path, ".", build.FindOnly)
		if err != nil {
			return nil, err
		}

		path = bPkg.Dir
	}

	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	dir := &DirSpec{
		VCS:      vcs,
		Revision: revision,
		Path:     path,
	}

	if _, err := dir.buildContext(); err != nil {
		return nil, err
	}

	return dir, nil
}

func (dir *DirSpec) String() string {
	if dir.VCS == "" || dir.Revision == "" {
		return dir.Path
	}

	return fmt.Sprintf("%s:%s:%s", dir.VCS, dir.Revision, dir.Path)
}

func (dir *DirSpec) ReadDir() ([]os.FileInfo, error) {
	ctx, err := dir.buildContext()
	if err != nil {
		return nil, err
	}

	return buildutil.ReadDir(ctx, dir.Path)
}

func (dir *DirSpec) buildContext() (*build.Context, error) {
	if dir.ctx != nil {
		return dir.ctx, nil
	}

	ctx := build.Default // copy

	if dir.VCS != "" && dir.Revision != "" {
		if dir.root == "" {
			cmd := exec.Command("git", "rev-parse", "--show-toplevel")
			cmd.Dir = dir.Path

			out, err := cmd.Output()
			if err != nil {
				return nil, err
			}

			dir.root = strings.TrimRight(string(out), "\n")
		}

		repo, err := vcs.Open(dir.VCS, dir.root)
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

		ctx.IsDir = func(path string) bool {
			if buildutil.IsAbsPath(&ctx, path) {
				if strings.HasPrefix(path, dir.root) {
					var err error
					path, err = filepath.Rel(dir.root, path)
					if err != nil {
						return false
					}
				} else {
					fi, err := os.Stat(path)
					return err == nil && fi.IsDir()
				}
			}

			fi, err := fs.Stat(path)
			return err == nil && fi.IsDir()
		}

		ctx.OpenFile = func(path string) (io.ReadCloser, error) {
			if buildutil.IsAbsPath(&ctx, path) {
				// the path maybe outside of repository (for standard libraries)
				if strings.HasPrefix(path, dir.root) {
					var err error
					path, err = filepath.Rel(dir.root, path)
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
				if strings.HasPrefix(path, dir.root) {
					var err error
					path, err = filepath.Rel(dir.root, path)
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
