package gompatible

import (
	"go/ast"
	"go/doc"
	"go/token"

	"golang.org/x/tools/go/types"
)

type Package struct {
	Doc   *doc.Package
	Types *types.Package
	Fset  *token.FileSet

	funcs map[string]*Func
	types map[string]*Type
}

type Func struct {
	Package *Package
	Doc     *doc.Func
	Types   *types.Func
}

type Type struct {
	Package *Package
	Doc     *doc.Type
	Types   *types.TypeName
}

func NewPackage(path string, fset *token.FileSet, files map[string]*ast.File) (*Package, error) {
	filesArray := make([]*ast.File, 0, len(files))
	for _, file := range files {
		filesArray = append(filesArray, file)
	}

	conf := types.Config{
		IgnoreFuncBodies: true,
	}
	typesPkg, err := conf.Check(path, fset, filesArray, nil)
	if err != nil {
		return nil, err
	}

	// Ignore (perhaps) "unresolved identifier" errors
	astPkg, _ := ast.NewPackage(fset, files, nil, nil)

	docPkg := doc.New(astPkg, path, doc.Mode(0))

	return &Package{
		Fset:  fset,
		Doc:   docPkg,
		Types: typesPkg,
	}, nil
}

func (p Package) FuncNames() []string {
	names := []string{}
	funcs := p.buildFuncs()
	for name := range funcs {
		names = append(names, name)
	}
	return names
}

func (p Package) Func(name string) *Func {
	funcs := p.buildFuncs()
	return funcs[name]
}

func (p Package) TypeNames() []string {
	names := []string{}
	funcs := p.buildTypes()
	for name := range funcs {
		names = append(names, name)
	}
	return names
}

func (p Package) Type(name string) *Type {
	funcs := p.buildTypes()
	return funcs[name]
}

func (p Package) buildFuncs() map[string]*Func {
	if p.funcs != nil {
		return p.funcs
	}

	p.funcs = map[string]*Func{}

	docFuncs := map[string]*doc.Func{}
	for _, f := range p.Doc.Funcs {
		docFuncs[f.Name] = f
	}

	typesFuncs := map[string]*types.Func{}
	scope := p.Types.Scope()
	for _, name := range scope.Names() {
		if f, ok := scope.Lookup(name).(*types.Func); ok {
			typesFuncs[name] = f
		}
	}

	for name, doc := range docFuncs {
		if types, ok := typesFuncs[name]; ok {
			p.funcs[name] = &Func{
				Package: &p,
				Doc:     doc,
				Types:   types,
			}
		}
	}

	return p.funcs
}

func (p Package) buildTypes() map[string]*Type {
	if p.types != nil {
		return p.types
	}

	p.types = map[string]*Type{}

	docTypes := map[string]*doc.Type{}
	for _, t := range p.Doc.Types {
		docTypes[t.Name] = t
	}

	typesTypes := map[string]*types.TypeName{}
	scope := p.Types.Scope()
	for _, name := range scope.Names() {
		if t, ok := scope.Lookup(name).(*types.TypeName); ok {
			typesTypes[name] = t
		}
	}

	for name, doc := range docTypes {
		if types, ok := typesTypes[name]; ok {
			p.types[name] = &Type{
				Package: &p,
				Doc:     doc,
				Types:   types,
			}
		}
	}

	return p.types
}
