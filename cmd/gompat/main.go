package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
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
	if d.vcs == "" || d.revision == "" {
		return d.path
	}

	return fmt.Sprintf("%s:%s:%s", d.vcs, d.revision, d.path)
}

func (d dirSpec) subdir(name string) dirSpec {
	dupped := d
	dupped.path = path.Join(d.path, name) // TODO use ctx.JoinPath
	return dupped
}

func (dir dirSpec) buildContext() (*build.Context, error) {
	if dir.ctx != nil {
		return dir.ctx, nil
	}

	ctx := build.Default // copy

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
			gompatible.Debugf("OpenFile %s", path)

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
			gompatible.Debugf("ReadDir %s", path)
			return fs.ReadDir(path)
		}
	}

	dir.ctx = &ctx

	return dir.ctx, nil
}

type packageFiles struct {
	packageName string
	fset        *token.FileSet
	files       map[string]*ast.File
}

// XXX should the return value be a map from dir to files? (currently assumed importPath to files)
func listPackages(dir dirSpec, recurse bool) (map[string][]string, error) {
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

	packages := map[string][]string{}

	var mode build.ImportMode
	p, err := ctx.ImportDir(dir.path, mode)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// nop
		} else {
			return nil, fmt.Errorf("while loading %s: %s", dir, err)
		}
	} else {
		gompatible.Debugf("%+v", p)
		importPath := p.ImportPath
		if importPath == "." {
			importPath = p.Dir
		}

		// XXX something's wrong if packages[importPath] exists already
		packages[importPath] = make([]string, len(p.GoFiles))
		for i, file := range p.GoFiles {
			// TODO use ctx.JoinPath
			packages[importPath][i] = filepath.Join(dir.path, file)
		}
	}

	if recurse == false {
		return packages, nil
	}

	entries, err := readDir(dir.path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() == false {
			continue
		}

		if name := e.Name(); name[0] == '.' || name[0] == '_' {
			continue
		}

		pkgs, err := listPackages(dir.subdir(e.Name()), recurse)
		if err != nil {
			return nil, err
		}
		for path, files := range pkgs {
			packages[path] = files
		}
	}

	return packages, nil
}

func usage() {
	fmt.Printf("Usage: %s <rev1>[..<rev2>] [<path>]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var (
		flagAll     = flag.Bool("a", false, "show also unchanged APIs")
		flagRecurse = flag.Bool("r", false, "recurse into subdirectories")
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

		if strings.HasSuffix(path, "...") {
			path = strings.TrimSuffix(path, "...")
			*flagRecurse = true
		}

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

	dir1 := dirSpec{vcs: vcsType, revision: revs[0], path: path}
	ctx1, err := dir1.buildContext()
	dieIf(err)

	pkgList1, err := listPackages(dir1, *flagRecurse)
	dieIf(err)

	pkgs1, err := gompatible.LoadPackages(ctx1, pkgList1)
	dieIf(err)

	dir2 := dirSpec{vcs: vcsType, revision: revs[1], path: path}
	ctx2, err := dir2.buildContext()
	dieIf(err)

	pkgList2, err := listPackages(dir2, *flagRecurse)
	dieIf(err)

	pkgs2, err := gompatible.LoadPackages(ctx2, pkgList2)
	dieIf(err)

	diffs := map[string]gompatible.PackageChanges{}

	forEachString(pkgNames(pkgs1), pkgNames(pkgs2)).do(func(name string) {
		diffs[name] = gompatible.DiffPackages(
			pkgs1[name], pkgs2[name],
		)
	})

	for name, diff := range diffs {
		var headerShown bool
		showHeader := func() {
			if !headerShown {
				fmt.Printf("package %s\n", name)
				headerShown = true
			}
		}

		forEachString(funcNames(diff.Funcs)).do(func(name string) {
			change := diff.Funcs[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				showHeader()
				fmt.Println(gompatible.ShowChange(change))
			}
		})

		forEachString(typeNames(diff.Types)).do(func(name string) {
			change := diff.Types[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				showHeader()
				fmt.Println(gompatible.ShowChange(change))
			}
		})
	}
}
