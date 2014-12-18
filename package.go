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
	TypesPkg *types.Package
	DocPkg   *doc.Package
	Fset     *token.FileSet

	funcs map[string]*Func
	types map[string]*Type
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
}

// XXX should the return value be a map from dir to files? (currently assumed importPath to files)
func listDirFiles(dir DirSpec, recurse bool) (map[string][]string, error) {
	ctx, err := dir.BuildContext()
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
		Debugf("%+v", p)
		importPath := p.ImportPath
		if importPath == "." {
			importPath = p.Dir
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

func LoadDir(dir DirSpec, recurse bool) (map[string]*Package, error) {
	ctx, err := dir.BuildContext()
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
		SourceImports:       true, // TODO should be controllable by flags
	}
	for path, files := range filepaths {
		Debugf("CreateFromFilenames %s %v", path, files)
		err := conf.CreateFromFilenames(path, files...)
		if err != nil {
			Debugf("ERR %+v", err)
			return nil, err
		}
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
	// Ignore (perhaps) "unresolved identifier" errors
	files := map[string]*ast.File{}
	for _, f := range pkgInfo.Files {
		files[prog.Fset.File(f.Pos()).Name()] = f

	}
	astPkg, _ := ast.NewPackage(prog.Fset, files, nil, nil)

	var mode doc.Mode
	docPkg := doc.New(astPkg, pkgInfo.String(), mode)

	return &Package{
		Fset:     prog.Fset,
		DocPkg:   docPkg,
		TypesPkg: pkgInfo.Pkg,
	}
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

	for _, docF := range p.DocPkg.Funcs {
		name := docF.Name
		if typesF, ok := p.TypesPkg.Scope().Lookup(name).(*types.Func); ok {
			p.funcs[name] = &Func{
				Package: &p,
				Doc:     docF,
				Types:   typesF,
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

	for _, docT := range p.DocPkg.Types {
		name := docT.Name
		if typesT, ok := p.TypesPkg.Scope().Lookup(name).(*types.TypeName); ok {
			p.types[name] = &Type{
				Package: &p,
				Doc:     docT,
				Types:   typesT,
			}
		}
	}

	return p.types
}

func (p Package) showASTNode(node interface{}) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, p.Fset, node)
	return buf.String()
}
