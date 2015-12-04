package gompatible

import (
	"golang.org/x/tools/go/types"
)

// FuncChange represents a change between functions.
type FuncChange struct {
	Before *Func
	After  *Func
}

func (fc FuncChange) TypesObject() types.Object {
	return fc.Before.Types
}

func (fc FuncChange) ShowBefore() string {
	f := fc.Before
	if f == nil || f.Doc == nil {
		return ""
	}
	return f.Package.showASTNode(f.Doc.Decl)
}

func (fc FuncChange) ShowAfter() string {
	f := fc.After
	if f == nil || f.Doc == nil {
		return ""
	}
	return f.Package.showASTNode(f.Doc.Decl)
}

func (fc FuncChange) Kind() ChangeKind {
	switch {
	case fc.Before == nil && fc.After == nil:
		// might not happen
		return ChangeUnchanged

	case fc.Before == nil:
		return ChangeAdded

	case fc.After == nil:
		return ChangeRemoved

	// We do not use types.Identical as we want to identify functions by their signature; not by the details of
	// parameters or return types, not:
	//   case types.Identical(fc.Before.Types.Type().Underlying(), fc.After.Types.Type().Underlying()):
	case identicalSansNames(fc.Before.Types, fc.After.Types):
		return ChangeUnchanged

	case fc.isCompatible():
		return ChangeCompatible

	default:
		return ChangeBreaking
	}
}

// identicalSansNames compares two functions to check if their types are identical
// according to the names. e.g.
//   - It does not care if the names of the parameters or return values differ
//   - It does not care if the implementations of the types differ
func identicalSansNames(fa, fb *types.Func) bool {
	// must always succeed
	sigA := fa.Type().(*types.Signature)
	sigB := fb.Type().(*types.Signature)

	var (
		lenParams  = sigA.Params().Len()
		lenResults = sigA.Results().Len()
	)

	if sigB.Params().Len() != lenParams {
		return false
	}

	if sigB.Results().Len() != lenResults {
		return false
	}

	for i := 0; i < lenParams; i++ {
		if types.TypeString(sigA.Params().At(i).Type(), nil) != types.TypeString(sigB.Params().At(i).Type(), nil) {
			return false
		}
	}

	for i := 0; i < lenResults; i++ {
		if types.TypeString(sigA.Results().At(i).Type(), nil) != types.TypeString(sigB.Results().At(i).Type(), nil) {
			return false
		}
	}

	return true
}

// sigParamsCompatible determines if the parameter parts of two signatures of functions are compatible.
// They are compatible if:
// - The number of parameters equal and the types of parameters are compatible for each of them.
// - The latter parameters have exactly one extra parameter which is a variadic parameter.
func sigParamsCompatible(s1, s2 *types.Signature) bool {
	extra := tuplesCompatibleExtra(s1.Params(), s2.Params(), cmpLower)

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

	extra := tuplesCompatibleExtra(s1.Results(), s2.Results(), cmpUpper)

	switch {
	case extra == nil:
		return false
	case len(extra) == 0:
		return true
	}

	return false
}

func tuplesCompatibleExtra(p1, p2 *types.Tuple, typeDirection cmp) []*types.Var {
	len1 := p1.Len()
	len2 := p2.Len()

	if len1 > len2 {
		return nil
	}

	vars := make([]*types.Var, len2-len1)

	for i := 0; i < len2; i++ {
		if i >= len1 {
			v2 := p2.At(i)
			vars[i-len1] = v2
			continue
		}

		v1 := p1.At(i)
		v2 := p2.At(i)

		c := cmpTypes(v1.Type(), v2.Type())
		if c == cmpEqual || c == typeDirection {
			continue
		}

		return nil
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
