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
	require.NoError(t, err)

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

	diff := DiffPackages(pkg1, pkg2)
	for name, change := range diff.Funcs {
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

	for name, change := range diff.Types {
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
