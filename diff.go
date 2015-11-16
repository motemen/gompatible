package gompatible

import (
	"github.com/motemen/gompatible/internal/util"
)

type ObjectCategory string

const (
	ObjectCategoryFunc  ObjectCategory = "func"
	ObjectCategoryType  ObjectCategory = "type"
	ObjectCategoryValue ObjectCategory = "value"
)

// PackageChanges represent changes between two packages.
type PackageChanges struct {
	Before  *Package
	After   *Package
	Changes map[ObjectCategory]map[string]Change
}

func (pc PackageChanges) Path() string {
	if pc.Before != nil {
		return pc.Before.TypesPkg.Path()
	}

	return pc.After.TypesPkg.Path()
}

func (pc PackageChanges) Funcs() map[string]FuncChange {
	changes := pc.Changes[ObjectCategoryFunc]
	m := make(map[string]FuncChange, len(changes))
	for k, c := range changes {
		m[k] = c.(FuncChange)
	}
	return m
}

func (pc PackageChanges) Types() map[string]TypeChange {
	changes := pc.Changes[ObjectCategoryType]
	m := make(map[string]TypeChange, len(changes))
	for k, c := range changes {
		m[k] = c.(TypeChange)
	}
	return m
}

func (pc PackageChanges) Values() map[string]ValueChange {
	changes := pc.Changes[ObjectCategoryValue]
	m := make(map[string]ValueChange, len(changes))
	for k, c := range changes {
		m[k] = c.(ValueChange)
	}
	return m
}

// DiffPackages takes two packages to produce the changes between them.
func DiffPackages(pkg1, pkg2 *Package) PackageChanges {
	diff := PackageChanges{
		Before: pkg1,
		After:  pkg2,
		Changes: map[ObjectCategory]map[string]Change{
			ObjectCategoryFunc:  {},
			ObjectCategoryType:  {},
			ObjectCategoryValue: {},
		},
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
		diff.Changes[ObjectCategoryFunc][name] = FuncChange{
			Before: pkg1.Funcs[name],
			After:  pkg2.Funcs[name],
		}
	}

	for _, name := range util.SortedStringSet(util.MapKeys(pkg1.Types), util.MapKeys(pkg2.Types)) {
		type1 := pkg1.Types[name]
		type2 := pkg2.Types[name]

		diff.Changes[ObjectCategoryType][name] = TypeChange{
			Before: pkg1.Types[name],
			After:  pkg2.Types[name],
		}

		if type1 != nil && type2 != nil {
			for _, fname := range util.SortedStringSet(util.MapKeys(type1.Funcs), util.MapKeys(type2.Funcs)) {
				diff.Changes[ObjectCategoryFunc][fname] = FuncChange{
					Before: type1.Funcs[fname],
					After:  type2.Funcs[fname],
				}
			}

			for _, mname := range util.SortedStringSet(util.MapKeys(type1.Methods), util.MapKeys(type2.Methods)) {
				diff.Changes[ObjectCategoryFunc][name+"."+mname] = FuncChange{
					Before: type1.Methods[mname],
					After:  type2.Methods[mname],
				}
			}
		}
	}

	for _, name := range util.SortedStringSet(util.MapKeys(pkg1.Values), util.MapKeys(pkg2.Values)) {
		Debugf("%q", name)
		diff.Changes[ObjectCategoryValue][name] = ValueChange{
			Before: pkg1.Values[name],
			After:  pkg2.Values[name],
		}
	}

	return diff
}
