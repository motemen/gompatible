package gompatible

import (
	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

var _ = Change((*TypeChange)(nil))

// TypeChange represents a change between two types.
type TypeChange struct {
	Before *Type
	After  *Type
}

func (tc TypeChange) ShowBefore() string {
	t := tc.Before
	return t.Package.showASTNode(t.Doc.Decl)
}

func (tc TypeChange) ShowAfter() string {
	t := tc.After
	return t.Package.showASTNode(t.Doc.Decl)
}

// XXX
// []rune and string -- compatible? types.Comvertible?

func (tc TypeChange) Kind() ChangeKind {
	switch {
	case tc.Before == nil && tc.After == nil:
		// XXX
		return ChangeUnchanged

	case tc.Before == nil:
		return ChangeAdded

	case tc.After == nil:
		return ChangeRemoved

	case types.Identical(tc.Before.Types.Type().Underlying(), tc.After.Types.Type().Underlying()):
		Debugf("%s -> %s", tc.Before.Types.Type().Underlying().String(), tc.After.Types.Type().Underlying().String())
		return ChangeUnchanged

	case tc.isCompatible():
		return ChangeCompatible

	default:
		return ChangeBreaking
	}
}

// TODO byte <-> uint8, rune <-> int32 compatibility
func typesCompatible(t1, t2 types.Type) bool {
	// If both types are struct, mark them comptabile
	// iff their public field types are comptabile for each their names (order insensitive)

	if s1, ok := t1.(*types.Struct); ok {
		if s2, ok := t2.(*types.Struct); ok {
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
				// The new struct type should have fields
				// which the old one had
				if !ok {
					return false
				}

				// recurse
				if typesCompatible(f1.Type().Underlying(), f2.Type().Underlying()) == false {
					return false
				}
			}

			return true
		}
	}

	return t1.String() == t2.String()
}

func (tc TypeChange) isCompatible() bool {
	return typesCompatible(tc.Before.Types.Type().Underlying(), tc.After.Types.Type().Underlying())
}
