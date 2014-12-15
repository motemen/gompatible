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

	"github.com/motemen/go-vcs-fs/git"
	"github.com/motemen/gompatible"
	_ "golang.org/x/tools/go/gcimporter"
)

var rxGitVirtDir = regexp.MustCompile(`^git:([^:]+):(.+)$`)

func dieIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func buildPackage(path string) (build.Context, *build.Package, error) {
	ctx := build.Default

	m := rxGitVirtDir.FindStringSubmatch(path)
	if m != nil {
		repo, err := git.NewRepository(m[1], "")
		if err != nil {
			return ctx, nil, err
		}

		path = m[2]
		ctx.OpenFile = func(path string) (io.ReadCloser, error) {
			r, err := repo.Open(path)
			return r, err
		}
		ctx.ReadDir = func(path string) ([]os.FileInfo, error) {
			ff, err := repo.ReadDir(path)
			return ff, err
		}
	}

	var mode build.ImportMode
	bPkg, err := ctx.ImportDir(path, mode)
	return ctx, bPkg, err
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

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	path1, fset1, files1, err := parseDir(before)
	dieIf(err)

	pkg1, err := gompatible.NewPackage(path1, fset1, files1)
	dieIf(err)

	path2, fset2, files2, err := parseDir(after)
	dieIf(err)

	pkg2, err := gompatible.NewPackage(path2, fset2, files2)
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
