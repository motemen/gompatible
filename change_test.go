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

type RemovedT1 bool
type UnchangedT1 int
type UnchangedT2 struct {
	Foo string
}
type CompatibleT1 struct {
	Foo string
}
type CompatibleT2 struct {
	Foo string
}
type BreakingT1 struct {
	XXX string
}
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

type UnchangedT1 int
type UnchangedT2 struct {
	Foo string
}
type CompatibleT1 struct {
	Foo string
	xxx interface{}
}
type CompatibleT2 struct {
	Foo string
	Bar bool
}
type BreakingT1 struct {
	YYY int
}
type AddedT1 interface{}
`)

	require.NoError(t, err)

	funcs1 := map[string]*types.Func{}
	funcs2 := map[string]*types.Func{}

	types1 := map[string]*types.TypeName{}
	types2 := map[string]*types.TypeName{}

	for _, name := range pkg1.Scope().Names() {
		obj := pkg1.Scope().Lookup(name)
		switch o := obj.(type) {
		case *types.Func:
			funcs1[o.Name()] = o
		case *types.TypeName:
			types1[o.Name()] = o
		}
	}

	for _, name := range pkg2.Scope().Names() {
		obj := pkg2.Scope().Lookup(name)
		switch o := obj.(type) {
		case *types.Func:
			funcs2[o.Name()] = o
		case *types.TypeName:
			types2[o.Name()] = o
		}
	}

	funcNames := map[string]interface{}{}
	for name := range funcs1 {
		funcNames[name] = nil
	}
	for name := range funcs2 {
		funcNames[name] = nil
	}

	for name := range funcNames {
		change := FuncChange{
			Before: funcs1[name],
			After:  funcs2[name],
		}

		t.Log(ShowChange(change))

		if strings.HasPrefix(name, "Unchanged") {
			assert.Equal(t, change.Kind(), ChangeUnchanged)
		} else if strings.HasPrefix(name, "Compatible") {
			assert.Equal(t, change.Kind(), ChangeCompatible)
		} else if strings.HasPrefix(name, "Added") {
			assert.Equal(t, change.Kind(), ChangeAdded)
		} else if strings.HasPrefix(name, "Removed") {
			assert.Equal(t, change.Kind(), ChangeRemoved)
		} else {
			assert.Equal(t, change.Kind(), ChangeBreaking)
		}
	}

	typeNames := map[string]interface{}{}
	for name := range types1 {
		typeNames[name] = nil
	}
	for name := range types2 {
		typeNames[name] = nil
	}

	for name := range typeNames {
		change := TypeChange{
			Before: types1[name],
			After:  types2[name],
		}

		t.Log(ShowChange(change))

		var expected ChangeKind
		name = strings.TrimPrefix(name, "TEST.")
		if strings.HasPrefix(name, "Unchanged") {
			expected = ChangeUnchanged
		} else if strings.HasPrefix(name, "Compatible") {
			expected = ChangeCompatible
		} else if strings.HasPrefix(name, "Added") {
			expected = ChangeAdded
		} else if strings.HasPrefix(name, "Removed") {
			expected = ChangeRemoved
		} else {
			expected = ChangeBreaking
		}
		assert.Equal(t, expected, change.Kind(), name)
	}
}
