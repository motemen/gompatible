package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/hgcmd"
)

type dirSpec struct {
	vcs      string
	revision string
	path     string

	ctx *build.Context
}

func (d dirSpec) String() string {
	return fmt.Sprint("%s:%s:%s", d.vcs, d.revision, d.path)
}

func (d dirSpec) subdir(name string) dirSpec {
	dupped := d
	dupped.path = path.Join(d.path, name) // TODO use ctx.JoinPath
	return dupped
}

func (d dirSpec) buildContext() (*build.Context, error) {
	if d.ctx != nil {
		return d.ctx, nil
	}

	var err error
	d.ctx, err = buildContext(d)

	return d.ctx, err
}

type packageFiles struct {
	packageName string
	fset        *token.FileSet
	files       map[string]*ast.File
}

func buildContext(dir dirSpec) (*build.Context, error) {
	ctx := build.Default

	if dir.vcs != "" && dir.revision != "" {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = dir.path
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		repoRoot := strings.TrimRight(string(out), "\n")
		repo, err := vcs.Open(dir.vcs, repoRoot)
		if err != nil {
			return nil, err
		}

		commit, err := repo.ResolveRevision(dir.revision)
		if err != nil {
			return nil, err
		}

		fs, err := repo.FileSystem(commit)
		if err != nil {
			return nil, err
		}

		ctx.OpenFile = func(path string) (io.ReadCloser, error) {
			// TODO use ctx.IsAbsPath
			if filepath.IsAbs(path) {
				var err error
				path, err = filepath.Rel(repoRoot, path)
				if err != nil {
					return nil, err
				}
			}
			return fs.Open(path)
		}
		ctx.ReadDir = func(path string) ([]os.FileInfo, error) {
			if filepath.IsAbs(path) {
				var err error
				path, err = filepath.Rel(repoRoot, path)
				if err != nil {
					return nil, err
				}
			}
			return fs.ReadDir(path)
		}
	}

	return &ctx, nil
}

func loadPackagesRecurse(dir dirSpec) (map[string]*gompatible.Package, error) {
	ctx, err := dir.buildContext()
	if err != nil {
		return nil, err
	}

	var readDir func(string) ([]os.FileInfo, error)
	if ctx.ReadDir != nil {
		readDir = ctx.ReadDir
	} else {
		readDir = ioutil.ReadDir
	}

	packages := map[string]*gompatible.Package{}

	p, err := loadPackage(dir)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// nop
		} else {
			return nil, err
		}
	} else {
		packages[p.Types.Path()] = p
	}

	entries, err := readDir(dir.path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() == false {
			continue
		}

		pkgs, err := loadPackagesRecurse(dir.subdir(e.Name()))
		if err != nil {
			return nil, err
		}
		for name, p := range pkgs {
			packages[name] = p
		}
	}

	return packages, nil
}

// loadPackage parses .go sources under the dirSpec dir (not recursively)
// and returns a *gompatible.Package.
func loadPackage(dir dirSpec) (*gompatible.Package, error) {
	ctx, err := dir.buildContext()
	if err != nil {
		return nil, err
	}

	var mode build.ImportMode
	pkg, err := ctx.ImportDir(dir.path, mode)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for _, file := range pkg.GoFiles {
		filepath := path.Join(pkg.Dir, file)

		var r io.Reader
		if ctx.OpenFile != nil {
			r, err = ctx.OpenFile(filepath)
		} else {
			r, err = os.Open(filepath)
		}
		if err != nil {
			return nil, err
		}

		files[file], err = parser.ParseFile(fset, filepath, r, parser.ParseComments)
		if err != nil {
			return nil, err
		}
	}

	return gompatible.NewPackage(pkg.Name, fset, files)
}

func usage() {
	fmt.Printf("Usage: %s <rev1>[..<rev2>] [<path>]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var (
		flagAll = flag.Bool("a", false, "show also unchanged APIs")
		// flagRecurse = flag.Bool("r", false, "recurse into subdirectories")
	)
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	vcsType := "git" // TODO auto-detect

	revs := strings.Split(args[0], "..")
	if len(revs) < 2 || revs[1] == "" {
		revs = []string{revs[0], ""}
	}

	path := "."
	if len(args) >= 2 {
		path = args[1]

		if build.IsLocalImport(path) == false {
			for _, srcDir := range build.Default.SrcDirs() {
				pkgPath := filepath.Join(srcDir, path)
				if _, err := os.Stat(pkgPath); err == nil {
					path = pkgPath
					break
				}
			}
		}
	}

	pkg1, err := loadPackage(dirSpec{vcs: vcsType, revision: revs[0], path: path})
	dieIf(err)

	pkg2, err := loadPackage(dirSpec{vcs: vcsType, revision: revs[1], path: path})
	dieIf(err)

	diff := gompatible.DiffPackages(pkg1, pkg2)

	forEachString(funcNames(diff.Funcs)).do(func(name string) {
		change := diff.Funcs[name]
		if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
			fmt.Println(gompatible.ShowChange(change))
		}
	})

	forEachString(typeNames(diff.Types)).do(func(name string) {
		change := diff.Types[name]
		if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
			fmt.Println(gompatible.ShowChange(change))
		}
	})
}
