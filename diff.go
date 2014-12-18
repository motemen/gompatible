package gompatible

import (
	"github.com/motemen/gompatible/sortedset"
)

type PackageChanges struct {
	Before *Package
	After  *Package
	Funcs  map[string]FuncChange
	Types  map[string]TypeChange
}

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
	sortedset.Strings(funcNames(pkg1.Funcs), funcNames(pkg2.Funcs)).ForEach(func(name string) {
		Debugf("%q", name)
		diff.Funcs[name] = FuncChange{
			Before: pkg1.Funcs[name],
			After:  pkg2.Funcs[name],
		}
	})

	sortedset.Strings(typeNames(pkg1.Types), typeNames(pkg2.Types)).ForEach(func(name string) {
		diff.Types[name] = TypeChange{
			Before: pkg1.Types[name],
			After:  pkg2.Types[name],
		}
	})

	return diff
}
