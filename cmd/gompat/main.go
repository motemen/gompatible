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
	_ "os/exec"
	"path"
	"regexp"

	"github.com/motemen/go-vcs-fs/git"
	"github.com/motemen/gompatible"
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
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

func typesPackage(dir string) (*types.Package, error) {
	fset := token.NewFileSet()

	conf := types.Config{
		IgnoreFuncBodies: true,
		Error: func(err error) {
			log.Println(err)
		},
	}

	ctx, bPkg, err := buildPackage(dir)
	if err != nil {
		return nil, err
	}

	astFiles := make([]*ast.File, len(bPkg.GoFiles))
	for i, file := range bPkg.GoFiles {
		filepath := path.Join(bPkg.Dir, file)

		var r io.Reader
		if ctx.OpenFile != nil {
			r, err = ctx.OpenFile(filepath)
		} else {
			r, err = os.Open(filepath)
		}
		if err != nil {
			return nil, err
		}

		astFile, err := parser.ParseFile(fset, filepath, r, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		astFiles[i] = astFile
	}

	return conf.Check(bPkg.Name, fset, astFiles, nil)
}

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	pkg1, err := typesPackage(before)
	dieIf(err)

	pkg2, err := typesPackage(after)
	dieIf(err)

	diff := gompatible.DiffPackages(pkg1, pkg2)

	for _, change := range diff.Funcs {
		fmt.Println(gompatible.ShowChange(change))
	}

	for _, change := range diff.Types {
		fmt.Println(gompatible.ShowChange(change))
	}
}
