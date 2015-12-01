package gompatible

import (
	"golang.org/x/tools/go/types"
)

type ValueChange struct {
	Before *Value
	After  *Value
}

func (vc ValueChange) TypesObject() types.Object {
	return vc.Before.Types
}

func (vc ValueChange) ShowBefore() string {
	v := vc.Before
	if v == nil || v.Doc == nil {
		return ""
	}
	return v.Package.showASTNode(v.Doc.Decl) // or dig into .Specs?
}

func (vc ValueChange) ShowAfter() string {
	v := vc.After
	if v == nil || v.Doc == nil {
		return ""
	}
	return v.Package.showASTNode(v.Doc.Decl) // or dig into .Specs?
}

func (vc ValueChange) Kind() ChangeKind {
	switch {
	case vc.Before == nil && vc.After == nil:
		// might not happen
		return ChangeUnchanged

	case vc.Before == nil:
		return ChangeAdded

	case vc.After == nil:
		return ChangeRemoved
	}

	// i) const -> var:   compatible
	// i) var -> const:   breaking (or weak compatible)
	// i) const -> const: identical
	// i) var -> var:     identical
	// ii) types identical:  identical
	// ii) types compatible: compatible
	// ii) types breaking:   breaking

	var k ChangeKind
	if vc.Before.IsConst == false && vc.After.IsConst == true {
		// var -> const: breaking (or weak compatible)
		return ChangeBreaking
	} else if vc.Before.IsConst == true && vc.After.IsConst == false {
		// const -> var: compatible
		k = ChangeCompatible
	} else {
		k = ChangeUnchanged
	}

	switch compareTypes(vc.Before.Types.Type(), vc.After.Types.Type()) {
	case compIncompatible:
		return ChangeBreaking
	case compCompatible:
		return ChangeCompatible
	case compIdentical:
		return k
	}

	panic("could not happen here")
}
