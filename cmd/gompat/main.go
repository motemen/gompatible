package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"sort"

	"github.com/motemen/gompatible"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
	_ "sourcegraph.com/sourcegraph/go-vcs/vcs/hgcmd"
)

var rxVCSDir = regexp.MustCompile(`^(git|hg):([^:]+):(.+)$`)

func dieIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func buildPackage(path string) (*build.Context, *build.Package, error) {
	ctx := build.Default

	m := rxVCSDir.FindStringSubmatch(path)
	if m != nil {
		vcsType := m[1]
		rev := m[2]
		path = m[3] // this should not be :=

		repo, err := vcs.Open(vcsType, ".")
		if err != nil {
			return nil, nil, err
		}

		commit, err := repo.ResolveRevision(rev)
		if err != nil {
			return nil, nil, err
		}

		fs, err := repo.FileSystem(commit)
		if err != nil {
			return nil, nil, err
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

	var mode build.ImportMode
	bPkg, err := ctx.ImportDir(path, mode)
	return &ctx, bPkg, err
}

func parseDir(dir string) (string, *token.FileSet, map[string]*ast.File, error) {
	fset := token.NewFileSet()

	ctx, bPkg, err := buildPackage(dir)
	if err != nil {
		return "", nil, nil, err
	}

	files := map[string]*ast.File{}
	for _, file := range bPkg.GoFiles {
		filepath := path.Join(bPkg.Dir, file)

		var r io.Reader
		if ctx.OpenFile != nil {
			r, err = ctx.OpenFile(filepath)
		} else {
			r, err = os.Open(filepath)
		}
		if err != nil {
			return "", nil, nil, err
		}

		files[file], err = parser.ParseFile(fset, filepath, r, parser.ParseComments)
		if err != nil {
			return "", nil, nil, err
		}
	}

	return bPkg.Name, fset, files, nil
}

func parseDirToPackage(dir string) (*gompatible.Package, error) {
	path, fset, files, err := parseDir(dir)
	if err != nil {
		return nil, err
	}

	return gompatible.NewPackage(path, fset, files)
}

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	pkg1, err := parseDirToPackage(before)
	dieIf(err)

	pkg2, err := parseDirToPackage(after)
	dieIf(err)

	diff := gompatible.DiffPackages(pkg1, pkg2)

	forEachName(byFuncName(diff.Funcs), func(name string) {
		fmt.Println(gompatible.ShowChange(diff.Funcs[name]))
	})

	forEachName(byTypeName(diff.Types), func(name string) {
		fmt.Println(gompatible.ShowChange(diff.Types[name]))
	})

}

func forEachName(gen namesYielder, f func(string)) {
	names := []string{}
	gen.yieldNames(func(name string) {
		names = append(names, name)
	})
	sort.Strings(names)

	for _, name := range names {
		f(name)
	}
}

type namesYielder interface {
	yieldNames(func(string))
}

type byFuncName map[string]gompatible.FuncChange

func (b byFuncName) yieldNames(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

type byTypeName map[string]gompatible.TypeChange

func (b byTypeName) yieldNames(yield func(string)) {
	for name := range b {
		yield(name)
	}
}
