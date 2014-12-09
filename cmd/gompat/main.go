package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

//import "github.com/k0kubun/pp"

func noTestFilter(fi os.FileInfo) bool {
	return strings.HasSuffix(fi.Name(), "_test.go") == false
}

func listGoSource(path string) ([]string, error) {
	fd, err := os.Open(".")
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}

	ctx := build.Context{}

	files := []string{}

	for _, d := range list {
		match, err := ctx.MatchFile(path, d.Name())
		if err != nil {
			return nil, err
		}

		if match {
			filename := filepath.Join(path, d.Name())
			files = append(files, filename)
		}
	}

	return files, nil
}

func dieIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func typesPkg(dir string) (*types.Package, error) {
	fset := token.NewFileSet()

	conf := types.Config{
		IgnoreFuncBodies: true,
		Error: func(err error) {
			log.Println(err)
		},
	}

	filepaths, err := listGoSource(dir)
	if err != nil {
		return nil, err
	}

	astFiles := []*ast.File{}
	for _, filepath := range filepaths {
		f, err := parser.ParseFile(fset, filepath, nil, 0)
		if err != nil {
			log.Fatal(err)
		}
		astFiles = append(astFiles, f)
	}

	return conf.Check(dir, fset, astFiles, nil)
}

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	dieIf(exec.Command("git", "checkout", before).Run())

	pkgBefore, err := typesPkg(".")
	dieIf(err)

	dieIf(exec.Command("git", "checkout", after).Run())
	pkgAfter, err := typesPkg(".")
	dieIf(err)

	log.Println(pkgBefore, pkgAfter)

	if false {
		pkgsBefore, err := parser.ParseDir(token.NewFileSet(), ".", noTestFilter, parser.ParseComments)

		err = exec.Command("git", "checkout", after).Run()
		if err != nil {
			log.Fatal(err)
		}

		pkgsAfter, err := parser.ParseDir(token.NewFileSet(), ".", noTestFilter, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}

		for name := range pkgsAfter {
			fmt.Println("package %s", name)

			var importPath = "XXX stub"
			var mode doc.Mode
			var (
				docA = doc.New(pkgsBefore[name], importPath, mode)
				docB = doc.New(pkgsAfter[name], importPath, mode)
			)

			funcsA := map[string]*doc.Func{}
			funcsB := map[string]*doc.Func{}

			for _, f := range docA.Funcs {
				funcsA[f.Name] = f
			}
			for _, f := range docB.Funcs {
				funcsB[f.Name] = f
			}

			for name := range funcsA {
				fc := gompatible.FuncChange{
					Before: funcsA[name],
					After:  funcsB[name],
				}
				fmt.Println(gompatible.ShowChange(fc))
			}
		}
	}
}
