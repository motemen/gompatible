package gompatible

import (
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/types"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func typesPackage(source string) (*types.Package, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	conf := types.Config{
		IgnoreFuncBodies: true,
		Error: func(err error) {
			log.Println(err)
		},
	}
	return conf.Check("TEST", fset, []*ast.File{file}, nil)
}

func TestFuncChange_IsCompatible(t *testing.T) {
	pkg, err := typesPackage(`package TEST
func Compatible1(n int)
func Compatible1_(n int)

func Compatible2(n int)
func Compatible2_(n int, opts ...string)

func Compatible3(n int)
func Compatible3_(n int) error

func Compatible4(n int) error
func Compatible4_(m int) error

func Breaking1(n int)
func Breaking1_(n int, b bool)

func Breaking2(n int) []byte
func Breaking2_(n int) ([]byte, error)

func Breaking3(n int, s string)
func Breaking3_(n int)

func Breaking4(n int) string
func Breaking4_(n int) []byte
`)
	require.NoError(t, err)

	funcs := map[string]*types.Func{}
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		if f, ok := obj.(*types.Func); ok {
			funcs[f.Name()] = f
		}
	}

	for name := range funcs {
		if strings.HasSuffix(name, "_") {
			continue
		}

		change := FuncChange{
			Before: funcs[name],
			After:  funcs[name+"_"],
		}

		t.Log(ShowChange(change))

		if strings.HasPrefix(name, "Compatible") {
			assert.True(t, change.IsCompatible())
		} else {
			assert.False(t, change.IsCompatible())
		}
	}
}
