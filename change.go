package gompatible

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/types"
)

type Change interface {
	NodeBefore() ast.Node
	NodeAfter() ast.Node
	IsAdded() bool
	IsRemoved() bool
	IsUnchanged() bool
	IsCompatible() bool
}

var _ = Change((*FuncChange)(nil))

type FuncChange struct {
	Before *doc.Func
	After  *doc.Func
}

func (fc FuncChange) NodeBefore() ast.Node {
	return fc.Before.Decl
}

func (fc FuncChange) NodeAfter() ast.Node {
	return fc.After.Decl
}

func showNode(node ast.Node) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()

	err := printer.Fprint(&buf, fset, node)
	if err != nil {
		return fmt.Sprintf("<!ERR %s>", err)
	}

	return string(buf.Bytes())
}

func (fc FuncChange) IsAdded() bool {
	return fc.Before == nil
}

func (fc FuncChange) IsRemoved() bool {
	return fc.After == nil
}

func (fc FuncChange) IsUnchanged() bool {
	return showNode(fc.Before.Decl) == showNode(fc.After.Decl)
}

func ShowChange(c Change) string {
	switch {
	case c.IsAdded():
		return "+ " + showNode(c.NodeAfter())
	case c.IsRemoved():
		return "+ " + showNode(c.NodeBefore())
	case c.IsUnchanged():
		return "= " + showNode(c.NodeBefore())
	case c.IsCompatible():
		return "* " + showNode(c.NodeBefore()) + " -> " + showNode(c.NodeAfter())
	default:
		return "! " + showNode(c.NodeBefore()) + " -> " + showNode(c.NodeAfter())
	}
}

func paramsCompatible(p1, p2 *ast.FieldList) bool {
	extra := fieldListCompatibleExtra(p1, p2)

	switch {
	case extra == nil:
		return false
	case len(extra) == 0:
		return true
	case len(extra) == 1:
		if _, ok := extra[0].Type.(*ast.Ellipsis); ok {
			return ok
		}
	}

	return false
}

func resultsCompatible(p1, p2 *ast.FieldList) bool {
	if p1.NumFields() == 0 {
		return true
	}

	extra := fieldListCompatibleExtra(p1, p2)

	switch {
	case extra == nil:
		return false
	case len(extra) == 0:
		return true
	}

	return false
}

func (fc FuncChange) IsCompatible() bool {
	if paramsCompatible(fc.Before.Decl.Type.Params, fc.After.Decl.Type.Params) == false {
		return false
	}

	if resultsCompatible(fc.Before.Decl.Type.Results, fc.After.Decl.Type.Results) == false {
		return false
	}

	return true
}

func fieldListCompatibleExtra(fl1, fl2 *ast.FieldList) []*ast.Field {
	numBefore := fl1.NumFields()

	if fl2.NumFields() < numBefore {
		return nil
	}

	for i := range fl1.List {
		if isFieldCompatible(fl1.List[i], fl2.List[i]) == false {
			return nil
		}
	}

	// Types match so far

	return fl2.List[numBefore:]
}

func isFieldCompatible(f1, f2 *ast.Field) bool {
	return isTypeCompatible(f1.Type, f2.Type)
}

func isTypeCompatible(t1, t2 ast.Expr) bool {
	return showNode(t1) == showNode(t2)
	// switch t1 := t1.(type) {
	// case *ast.StructType:
	// case *ast.InterfaceType:
	// default:
	// }
}
