package gompatible

import (
	"github.com/motemen/gompatible/util"
)

// PackageChanges represent changes between two packages.
type PackageChanges struct {
	Before *Package
	After  *Package
	Funcs  map[string]FuncChange
	Types  map[string]TypeChange
}

func (pc PackageChanges) Path() string {
	if pc.Before != nil {
		return pc.Before.TypesPkg.Path()
	}

	return pc.After.TypesPkg.Path()
}

// DiffPackages takes two packages to produce the changes between them.
func DiffPackages(pkg1, pkg2 *Package) PackageChanges {
	diff := PackageChanges{
		Before: pkg1,
		After:  pkg2,
		Funcs:  map[string]FuncChange{},
		Types:  map[string]TypeChange{},
	}

	// FIXME
	if pkg1 == nil {
		pkg1 = &Package{
			Funcs: map[string]*Func{},
			Types: map[string]*Type{},
		}
	}
	if pkg2 == nil {
		pkg2 = &Package{
			Funcs: map[string]*Func{},
			Types: map[string]*Type{},
		}
	}

	Debugf("%+v %+v", pkg1, pkg2)
	for _, name := range util.SortedStringSet(util.MapKeys(pkg1.Funcs), util.MapKeys(pkg2.Funcs)) {
		Debugf("%q", name)
		diff.Funcs[name] = FuncChange{
			Before: pkg1.Funcs[name],
			After:  pkg2.Funcs[name],
		}
	}

	for _, name := range util.SortedStringSet(util.MapKeys(pkg1.Types), util.MapKeys(pkg2.Types)) {
		type1 := pkg1.Types[name]
		type2 := pkg2.Types[name]

		diff.Types[name] = TypeChange{
			Before: type1,
			After:  type2,
		}

		if type1 != nil && type2 != nil {
			for _, fname := range util.SortedStringSet(util.MapKeys(type1.Funcs), util.MapKeys(type2.Funcs)) {
				diff.Funcs[fname] = FuncChange{
					Before: type1.Funcs[fname],
					After:  type2.Funcs[fname],
				}
			}

			for _, mname := range util.SortedStringSet(util.MapKeys(type1.Methods), util.MapKeys(type2.Methods)) {
				diff.Funcs[name+"."+mname] = FuncChange{
					Before: type1.Methods[mname],
					After:  type2.Methods[mname],
				}
			}
		}
	}

	return diff
}
