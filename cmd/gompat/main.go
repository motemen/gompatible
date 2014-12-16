package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
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
}

func (d dirSpec) String() string {
	return fmt.Sprint("%s:%s:%s", d.vcs, d.revision, d.path)
}

type packageFiles struct {
	packageName string
	fset        *token.FileSet
	files       map[string]*ast.File
}

func buildContext(dir dirSpec) (*build.Context, error) {
	ctx := build.Default

	if dir.vcs != "" && dir.revision != "" {
		repo, err := vcs.Open(dir.vcs, ".")
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
			r, err := fs.Open(path)
			return r, err
		}
		ctx.ReadDir = func(path string) ([]os.FileInfo, error) {
			ff, err := fs.ReadDir(path)
			return ff, err
		}
	}

	return &ctx, nil
}

func parseDir(dir dirSpec) (*packageFiles, error) {
	fset := token.NewFileSet()

	ctx, err := buildContext(dir)
	if err != nil {
		return nil, err
	}

	var mode build.ImportMode
	pkg, err := ctx.ImportDir(dir.path, mode)
	if err != nil {
		return nil, err
	}

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

	return &packageFiles{
		packageName: pkg.Name,
		fset:        fset,
		files:       files,
	}, nil
}

func parseDirToPackage(dir dirSpec) (*gompatible.Package, error) {
	pkgFiles, err := parseDir(dir)
	if err != nil {
		return nil, err
	}

	return gompatible.NewPackage(pkgFiles.packageName, pkgFiles.fset, pkgFiles.files)
}

func usage() {
	fmt.Printf("Usage: %s <rev1>[..<rev2>] [<path>]\n", os.Args[0])
	os.Exit(1)
}

// gompat <rev1>[..<rev2>] <path>
func main() {
	flagAll := flag.Bool("a", false, "show also unchanged APIs")
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
	}

	pkg1, err := parseDirToPackage(dirSpec{vcs: vcsType, revision: revs[0], path: path})
	dieIf(err)

	pkg2, err := parseDirToPackage(dirSpec{vcs: vcsType, revision: revs[1], path: path})
	dieIf(err)

	diff := gompatible.DiffPackages(pkg1, pkg2)

	forEachName(byFuncName(diff.Funcs), func(name string) {
		change := diff.Funcs[name]
		if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
			fmt.Println(gompatible.ShowChange(change))
		}
	})

	forEachName(byTypeName(diff.Types), func(name string) {
		change := diff.Types[name]
		if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
			fmt.Println(gompatible.ShowChange(change))
		}
	})
}
