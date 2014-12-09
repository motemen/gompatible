package gompatible

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/types"
)

type Change interface {
	NodeBefore() types.Object
	NodeAfter() types.Object
	IsAdded() bool
	IsRemoved() bool
	IsUnchanged() bool
	IsCompatible() bool
}

var _ = Change((*FuncChange)(nil))

type FuncChange struct {
	Before *types.Func
	After  *types.Func
}

func (fc FuncChange) NodeBefore() types.Object {
	return fc.Before
}

func (fc FuncChange) NodeAfter() types.Object {
	return fc.After
}

func (fc FuncChange) IsAdded() bool {
	return fc.Before == nil
}

func (fc FuncChange) IsRemoved() bool {
	return fc.After == nil
}

func (fc FuncChange) IsUnchanged() bool {
	return fc.Before.String() == fc.After.String()
}

func ShowChange(c Change) string {
	switch {
	case c.IsAdded():
		return "+ " + c.NodeAfter().String()
	case c.IsRemoved():
		return "- " + c.NodeBefore().String()
	case c.IsUnchanged():
		return "= " + c.NodeBefore().String()
	case c.IsCompatible():
		return "* " + c.NodeBefore().String() + " -> " + c.NodeAfter().String()
	default:
		return "! " + c.NodeBefore().String() + " -> " + c.NodeAfter().String()
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

func (fc FuncChange) IsCompatible() bool {
	typeBefore, typeAfter := fc.Before.Type(), fc.After.Type()
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
