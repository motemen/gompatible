package main

import (
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/motemen/gompatible"
)

//import "github.com/k0kubun/pp"

func noTestFilter(fi os.FileInfo) bool {
	return strings.HasSuffix(fi.Name(), "_test.go") == false
}

func main() {
	var err error

	var (
		before = os.Args[1]
		after  = os.Args[2]
	)

	err = exec.Command("git", "checkout", before).Run()
	if err != nil {
		log.Fatal(err)
	}

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
