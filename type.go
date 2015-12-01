package gompatible

import (
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

// TypeChange represents a change between two types.
type TypeChange struct {
	Before *Type
	After  *Type
}

func (tc TypeChange) TypesObject() types.Object {
	return tc.Before.Types
}

func (tc TypeChange) ShowBefore() string {
	t := tc.Before
	if t == nil || t.Doc == nil {
		return ""
	}
	return t.Package.showASTNode(t.Doc.Decl)
}

func (tc TypeChange) ShowAfter() string {
	t := tc.After
	if t == nil || t.Doc == nil {
		return ""
	}
	return t.Package.showASTNode(t.Doc.Decl)
}

func (tc TypeChange) Kind() ChangeKind {
	switch {
	case tc.Before == nil && tc.After == nil:
		// might not happen
		return ChangeUnchanged

	case tc.Before == nil:
		return ChangeAdded

	case tc.After == nil:
		return ChangeRemoved

	case types.ObjectString(tc.Before.Types, nil) == types.ObjectString(tc.After.Types, nil):
		return ChangeUnchanged
	}

	switch tc.compatibility() {
	case compIdentical:
		return ChangeUnchanged

	case compCompatible:
		return ChangeCompatible

	default:
		return ChangeBreaking
	}

	return ChangeBreaking
}

// FIXME: acutally the type compatibility has direction,
// namely more specific to more general (eg. struct to interface) and the opposite.
// In function parameters the former case will be compatible,
// while in function results the latter case will.

type compatibility int

const (
	compIncompatible compatibility = iota
	compCompatible
	compIdentical
)

func compareTypes(t1, t2 types.Type) compatibility {
	// If both types are struct, mark them comptabile
	// iff their public field types are comptabile for each their names (order insensitive)

	if s1, ok := t1.(*types.Struct); ok {
		if s2, ok := t2.(*types.Struct); ok {
			return compareStructs(s1, s2)
		}
	}

	// TODO: is it really ok?
	if types.TypeString(t1, nil) == types.TypeString(t2, nil) {
		return compIdentical
	}

	if bt1, ok := t1.(*types.Basic); ok {
		if bt2, ok := t2.(*types.Basic); ok {
			// eg. untyped string -> string
			if bt1.Info()&types.IsUntyped != 0 {
				if bt1.Info()&bt2.Info() == bt1.Info()^types.IsUntyped {
					return compCompatible
				}
			}

			// Names differ, but the basic kind is the same
			// eg. uint8 vs byte
			if bt1.Kind() == bt2.Kind() {
				return compCompatible
			}
		}
	}

	return compIncompatible
}

func compareStructs(s1, s2 *types.Struct) compatibility {
	identical := true

	fields1 := map[string]*types.Var{}
	fields2 := map[string]*types.Var{}

	for i := 0; i < s1.NumFields(); i++ {
		f := s1.Field(i)
		if f.Exported() {
			fields1[f.Name()] = f
		}
	}
	for i := 0; i < s2.NumFields(); i++ {
		f := s2.Field(i)
		if f.Exported() {
			fields2[f.Name()] = f
		}
	}

	for name, f1 := range fields1 {
		f2, ok := fields2[name]
		// For two types to be compatible,
		// the new struct type should have fields
		// which the old one had
		if !ok {
			return compIncompatible
		}

		// recurse
		switch compareTypes(f1.Type().Underlying(), f2.Type().Underlying()) {
		case compIdentical:

		case compCompatible:
			identical = false

		case compIncompatible:
			return compIncompatible
		}
	}

	for name := range fields2 {
		// If the newer type has a new field,
		// two types must not be identical
		// (yet have a change to be compatible)
		if _, ok := fields1[name]; !ok {
			identical = false
		}
	}

	if identical {
		return compIdentical
	} else {
		return compCompatible
	}
}

func (tc TypeChange) compatibility() compatibility {
	return compareTypes(tc.Before.Types.Type().Underlying(), tc.After.Types.Type().Underlying())
}
