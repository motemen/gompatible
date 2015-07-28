package gompatible

import (
	"bytes"
	"fmt"

	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/types"
)

// Package represents a parsed, type-checked and documented package.
type Package struct {
	// The types aspect of the package
	TypesPkg *types.Package
	// The docs aspect of the package
	DocPkg *doc.Package

	Funcs map[string]*Func
	Types map[string]*Type

	Fset *token.FileSet
}

// Func is a parsed, type-checked and documented function.
type Func struct {
	Package *Package
	Types   *types.Func
	Doc     *doc.Func
}

// Type is a parsed, type-checked and documented type declaration.
type Type struct {
	Package *Package
	Types   *types.TypeName
	Doc     *doc.Type
	Funcs   map[string]*Func
	Methods map[string]*Func
}

// XXX should the return value be a map from dir to files? (currently assumed importPath to files)
func listDirFiles(dir *DirSpec, recurse bool) (map[string][]string, error) {
	ctx, err := dir.buildContext()
	if err != nil {
		return nil, err
	}

	packages := map[string][]string{}

	var mode build.ImportMode
	p, err := ctx.ImportDir(dir.Path, mode)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// nop
		} else {
			return nil, fmt.Errorf("while loading %s: %s", dir, err)
		}
	} else {
		importPath := p.ImportPath
		if importPath == "." {
			importPath = p.Dir
		}
		if dir.pkgOverride != "" {
			importPath = dir.pkgOverride
		}

		// XXX something's wrong if packages[importPath] exists already
		packages[importPath] = make([]string, len(p.GoFiles))
		for i, file := range p.GoFiles {
			packages[importPath][i] = buildutil.JoinPath(ctx, dir.Path, file)
		}
	}

	if recurse == false {
		return packages, nil
	}

	entries, err := dir.ReadDir()
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() == false {
			continue
		}

		if name := e.Name(); name[0] == '.' || name[0] == '_' {
			continue
		}

		pkgs, err := listDirFiles(dir.Subdir(e.Name()), recurse)
		if err != nil {
			return nil, err
		}
		for path, files := range pkgs {
			packages[path] = files
		}
	}

	return packages, nil
}

func LoadDir(dir *DirSpec, recurse bool) (map[string]*Package, error) {
	ctx, err := dir.buildContext()
	if err != nil {
		return nil, err
	}

	files, err := listDirFiles(dir, recurse)
	if err != nil {
		return nil, err
	}

	return LoadPackages(ctx, files)
}

func LoadPackages(ctx *build.Context, filepaths map[string][]string) (map[string]*Package, error) {
	conf := &loader.Config{
		Build:               ctx,
		ParserMode:          parser.ParseComments,
		TypeCheckFuncBodies: func(_ string) bool { return false },
	}
	for path, files := range filepaths {
		Debugf("CreateFromFilenames %s %v", path, files)
		conf.CreateFromFilenames(path, files...)
	}

	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	packages := map[string]*Package{}
	for _, pkg := range prog.Created {
		packages[pkg.String()] = packageFromInfo(prog, pkg)
	}

	return packages, nil
}

func packageFromInfo(prog *loader.Program, pkgInfo *loader.PackageInfo) *Package {
	files := map[string]*ast.File{}
	for _, f := range pkgInfo.Files {
		files[prog.Fset.File(f.Pos()).Name()] = f

	}

	// Ignore (perhaps) "unresolved identifier" errors
	astPkg, _ := ast.NewPackage(prog.Fset, files, nil, nil)

	var mode doc.Mode
	docPkg := doc.New(astPkg, pkgInfo.String(), mode)

	return NewPackage(prog.Fset, docPkg, pkgInfo.Pkg)
}

// NewPackage builds a Package from one from doc and types package.
func NewPackage(fset *token.FileSet, doc *doc.Package, types *types.Package) *Package {
	pkg := &Package{
		Fset:     fset,
		DocPkg:   doc,
		TypesPkg: types,
	}
	pkg.buildFuncs()
	pkg.buildTypes()

	return pkg
}

func (p *Package) buildFuncs() map[string]*Func {
	if p.Funcs != nil {
		return p.Funcs
	}

	p.Funcs = map[string]*Func{}

	for _, docF := range p.DocPkg.Funcs {
		name := docF.Name
		if typesF, ok := p.TypesPkg.Scope().Lookup(name).(*types.Func); ok {
			p.Funcs[name] = &Func{
				Package: p,
				Doc:     docF,
				Types:   typesF,
			}
		}
	}

	return p.Funcs
}

func (p *Package) buildTypes() map[string]*Type {
	if p.Types != nil {
		return p.Types
	}

	p.Types = map[string]*Type{}

	for _, docT := range p.DocPkg.Types {
		name := docT.Name
		if typesT, ok := p.TypesPkg.Scope().Lookup(name).(*types.TypeName); ok {
			funcs := make(map[string]*Func, len(docT.Funcs))
			for _, f := range docT.Funcs {
				funcs[f.Name] = &Func{
					Package: p,
					Doc:     f,
					Types:   p.TypesPkg.Scope().Lookup(f.Name).(*types.Func),
				}
			}

			methods := make(map[string]*Func, len(docT.Methods))
			for _, m := range docT.Methods {
				// Must be found and be *types.Func
				obj, _, _ := types.LookupFieldOrMethod(typesT.Type(), true, p.TypesPkg, m.Name)
				methods[m.Name] = &Func{
					Package: p,
					Doc:     m,
					Types:   obj.(*types.Func),
				}
			}

			p.Types[name] = &Type{
				Package: p,
				Doc:     docT,
				Types:   typesT,
				Funcs:   funcs,
				Methods: methods,
			}
		}
	}

	return p.Types
}

// showASTNode takes an AST node to return its string presentation.
func (p Package) showASTNode(node interface{}) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.Fset, node)
	return buf.String()
}
