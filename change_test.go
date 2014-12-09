package gompatible

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func docPackage(source string) (*doc.Package, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	pkg, _ := ast.NewPackage(fset, map[string]*ast.File{"x.go": file}, nil, nil)

	return doc.New(pkg, "TODO: importPath", doc.Mode(0)), nil
}

func TestFuncChange(t *testing.T) {
	pkg, _ := docPackage(`package t
func Compatible1_A(n int)
func Compatible1_B(n int)

func Compatible2_A(n int)
func Compatible2_B(n int, opts ...string)

func Compatible3_A(n int)
func Compatible3_B(n int) error

func Breaking1_A(n int)
func Breaking1_B(n int, b bool)

func Breaking2_A(n int) []bytes
func Breaking2_B(n int) ([]bytes, error)
`)

	funcs := map[string]*doc.Func{}
	for _, f := range pkg.Funcs {
		funcs[f.Name] = f
	}

	compat1 := FuncChange{
		Before: funcs["Compatible1_A"],
		After:  funcs["Compatible1_B"],
	}

	assert.True(t, compat1.IsCompatible())

	compat2 := FuncChange{
		Before: funcs["Compatible2_A"],
		After:  funcs["Compatible2_B"],
	}

	assert.True(t, compat2.IsCompatible())

	compat3 := FuncChange{
		Before: funcs["Compatible3_A"],
		After:  funcs["Compatible3_B"],
	}

	assert.True(t, compat3.IsCompatible())

	breaking1 := FuncChange{
		Before: funcs["Breaking1_A"],
		After:  funcs["Breaking1_B"],
	}

	assert.True(t, !breaking1.IsCompatible())

	breaking2 := FuncChange{
		Before: funcs["Breaking2_A"],
		After:  funcs["Breaking2_B"],
	}

	assert.True(t, !breaking2.IsCompatible(), "Breaking2")
}
