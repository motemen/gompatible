package gompatible

import (
	"golang.org/x/tools/go/types"
)

type PackageChanges struct {
	Before *types.Package
	After  *types.Package
	Funcs  map[string]FuncChange
	Types  map[string]TypeChange
}

func DiffPackages(pkg1, pkg2 *types.Package) PackageChanges {
	pc := PackageChanges{
		Before: pkg1,
		After:  pkg2,
		Funcs:  map[string]FuncChange{},
		Types:  map[string]TypeChange{},
	}

	funcs1 := map[string]*types.Func{}
	funcs2 := map[string]*types.Func{}

	types1 := map[string]*types.TypeName{}
	types2 := map[string]*types.TypeName{}

	for _, name := range pkg1.Scope().Names() {
		obj := pkg1.Scope().Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch o := obj.(type) {
		case *types.Func:
			funcs1[o.Name()] = o
		case *types.TypeName:
			types1[o.Name()] = o
		}
	}

	for _, name := range pkg2.Scope().Names() {
		obj := pkg2.Scope().Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch o := obj.(type) {
		case *types.Func:
			funcs2[o.Name()] = o
		case *types.TypeName:
			types2[o.Name()] = o
		}
	}

	funcNames := map[string]interface{}{}
	for name := range funcs1 {
		funcNames[name] = nil
	}
	for name := range funcs2 {
		funcNames[name] = nil
	}

	for name := range funcNames {
		pc.Funcs[name] = FuncChange{
			Before: funcs1[name],
			After:  funcs2[name],
		}
	}

	typeNames := map[string]interface{}{}
	for name := range types1 {
		typeNames[name] = nil
	}
	for name := range types2 {
		typeNames[name] = nil
	}

	for name := range typeNames {
		pc.Types[name] = TypeChange{
			Before: types1[name],
			After:  types2[name],
		}
	}

	return pc
}
