package gompatible

import (
	"bytes"
	"go/printer"
	"go/token"
	"golang.org/x/tools/go/types"
)

var _ = Change((*FuncChange)(nil))

type FuncChange struct {
	Before *Func
	After  *Func
}

func showASTNode(node interface{}, fset *token.FileSet) string {
	if fset == nil {
		fset = token.NewFileSet()
	}
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, node)
	return buf.String()
}

func (fc FuncChange) ShowBefore() string {
	f := fc.Before
	return showASTNode(f.Doc.Decl, f.Package.Fset)
}

func (fc FuncChange) ShowAfter() string {
	f := fc.After
	return showASTNode(f.Doc.Decl, f.Package.Fset)
}

func (fc FuncChange) Kind() ChangeKind {
	switch {
	case fc.Before == nil && fc.After == nil:
		// XXX
		return ChangeUnchanged

	case fc.Before == nil:
		return ChangeAdded

	case fc.After == nil:
		return ChangeRemoved

	case fc.Before.Types.String() == fc.After.Types.String():
		return ChangeUnchanged

	case fc.isCompatible():
		return ChangeCompatible

	default:
		return ChangeBreaking
	}
}

func sigParamsCompatible(s1, s2 *types.Signature) bool {
	extra := tuplesCompatibleExtra(s1.Params(), s2.Params())

	switch {
	case extra == nil:
		// s2 params is incompatible with s1 params
		return false

	case len(extra) == 0:
		// s2 params is compatible with s1 params
		return true

	case len(extra) == 1:
		// s2 params is compatible with s1 params with an extra variadic arg
		if s1.Variadic() == false && s2.Variadic() == true {
			return true
		}
	}

	return false
}

func sigResultsCompatible(s1, s2 *types.Signature) bool {
	if s1.Results().Len() == 0 {
		return true
	}

	extra := tuplesCompatibleExtra(s1.Results(), s2.Results())

	switch {
	case extra == nil:
		return false
	case len(extra) == 0:
		return true
	}

	return false
}

func tuplesCompatibleExtra(p1, p2 *types.Tuple) []*types.Var {
	len1 := p1.Len()
	len2 := p2.Len()

	if len1 > len2 {
		return nil
	}

	vars := make([]*types.Var, len2-len1)

	for i := 0; i < len2; i++ {
		if i < len1 {
			v1 := p1.At(i)
			v2 := p2.At(i)

			if v1.Type().String() != v2.Type().String() {
				return nil
			}
		} else {
			v2 := p2.At(i)
			vars[i-len1] = v2
		}
	}

	return vars
}

func (fc FuncChange) isCompatible() bool {
	if fc.Before == nil || fc.After == nil {
		return false
	}

	typeBefore, typeAfter := fc.Before.Types.Type(), fc.After.Types.Type()
	if typeBefore == nil || typeAfter == nil {
		return false
	}

	sigBefore, sigAfter := typeBefore.(*types.Signature), typeAfter.(*types.Signature)

	if sigParamsCompatible(sigBefore, sigAfter) == false {
		return false
	}

	if sigResultsCompatible(sigBefore, sigAfter) == false {
		return false
	}

	return true
}
