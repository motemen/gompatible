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
	pkg1, err := typesPackage(`package TEST
func Unchanged1(n int)
func Compatible1(n int)
func Compatible2(n int)
func Compatible3(n int) error
func Breaking1(n int)
func Breaking2(n int) []byte
func Breaking3(n int, s string)
func Breaking4(n int) string
func Removed1()
`)

	pkg2, err := typesPackage(`package TEST
func Unchanged1(n int)
func Compatible1(n int, opts ...string)
func Compatible2(n int) error
func Compatible3(m int) error
func Breaking1(n int, b bool)
func Breaking2(n int) ([]byte, error)
func Breaking3(n int)
func Breaking4(n int) []byte
func Added1()
`)

	require.NoError(t, err)

	funcs1 := map[string]*types.Func{}
	funcs2 := map[string]*types.Func{}

	for _, name := range pkg1.Scope().Names() {
		obj := pkg1.Scope().Lookup(name)
		if f, ok := obj.(*types.Func); ok {
			funcs1[f.Name()] = f
		}
	}

	for _, name := range pkg2.Scope().Names() {
		obj := pkg2.Scope().Lookup(name)
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
		change := FuncChange{
			Before: funcs1[name],
			After:  funcs2[name],
		}

		t.Log(ShowChange(change))

		if strings.HasPrefix(name, "Unchanged") {
			assert.True(t, change.IsUnchanged())
		} else if strings.HasPrefix(name, "Compatible") {
			assert.True(t, change.IsCompatible())
		} else {
			assert.False(t, change.IsCompatible())
		}
	}
}
