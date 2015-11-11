package gompatible

import (
	"log"
	"strings"
	"testing"

	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/types"

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

func TestDiffPackages(t *testing.T) {
	pkgs1, err := LoadDir(&DirSpec{Path: "testdata/before", pkgOverride: "testdata"}, false)
	require.NoError(t, err)
	pkgs2, err := LoadDir(&DirSpec{Path: "testdata/after", pkgOverride: "testdata"}, false)
	require.NoError(t, err)

	diff := DiffPackages(pkgs1["testdata"], pkgs2["testdata"])
	assert.NotEmpty(t, diff.Funcs())
	assert.NotEmpty(t, diff.Types())

	for name, change := range diff.Funcs() {
		expected := ChangeBreaking

		if strings.HasPrefix(name, "Unchanged") {
			expected = ChangeUnchanged
		} else if strings.HasPrefix(name, "Compatible") {
			expected = ChangeCompatible
		} else if strings.HasPrefix(name, "Added") {
			expected = ChangeAdded
		} else if strings.HasPrefix(name, "Removed") {
			expected = ChangeRemoved
		}

		assert.Equal(t, expected.String(), change.Kind().String(), ShowChange(change))
	}

	for name, change := range diff.Types() {
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
		} else if strings.HasPrefix(name, "Breaking") {
			expected = ChangeBreaking
		} else {
			t.Fatalf("unexpected name: %q", name)
		}

		assert.Equal(t, expected.String(), change.Kind().String(), ShowChange(change))
	}
}
