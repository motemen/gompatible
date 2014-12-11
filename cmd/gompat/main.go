package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/motemen/gompatible"
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

//import "github.com/k0kubun/pp"

func listGoSource(path string) ([]string, error) {
	buildPkg, err := build.Default.ImportDir(path, build.ImportMode(0))
	if err != nil {
		return nil, err
	}

	goFiles := buildPkg.GoFiles
	files := make([]string, len(goFiles))
	for i, file := range goFiles {
		files[i] = filepath.Join(buildPkg.Dir, file)
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
		f, err := parser.ParseFile(fset, filepath, nil, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}
		astFiles = append(astFiles, f)
	}

	return conf.Check("", fset, astFiles, nil)
}

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	dir := "."
	if len(os.Args) > 3 {
		dir = os.Args[3]
	}

	dieIf(exec.Command("git", "checkout", before).Run())

	pkg1, err := typesPkg(dir)
	dieIf(err)

	dieIf(exec.Command("git", "checkout", after).Run())

	pkg2, err := typesPkg(dir)
	dieIf(err)

	funcs1 := map[string]*types.Func{}
	funcs2 := map[string]*types.Func{}

	for _, name := range pkg1.Scope().Names() {
		obj := pkg1.Scope().Lookup(name)
		if obj.Exported() == false {
			continue
		}
		if f, ok := obj.(*types.Func); ok {
			funcs1[f.Name()] = f
		}
	}

	for _, name := range pkg2.Scope().Names() {
		obj := pkg2.Scope().Lookup(name)
		if obj.Exported() == false {
			continue
		}
		if f, ok := obj.(*types.Func); ok {
			funcs2[f.Name()] = f
		}
	}

	names := map[string]interface{}{}
	for name := range funcs1 {
		names[name] = nil
	}
	for name := range funcs2 {
		names[name] = nil
	}

	for name := range names {
		change := gompatible.FuncChange{
			Before: funcs1[name],
			After:  funcs2[name],
		}

		fmt.Println(gompatible.ShowChange(change))
	}
}
