package gompatible

import (
	"bytes"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/loader"
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

func LoadPackage(ctx *build.Context, path string, filepaths []string) (*Package, error) {
	conf := &loader.Config{
		Build:      ctx,
		ParserMode: parser.ParseComments,
		// TypeChecker: types.Config{
		// 	Import: func(imports map[string]*types.Package, path string) (*types.Package, error) {
		// 		if stdLibs[path] {
		// 			return gcimporter.Import(imports, path)
		// 		}

		// 		/*
		// 			// TODO localImport
		// 			bPkg, err := ctx.Import(path, ".", build.FindOnly|build.AllowBinary)
		// 			id := strings.TrimSuffix(bPkg.PkgObj, ".a")
		// 			fmt.Println("Import", "id", id)
		// 			return gcimporter.ImportData(imports, filename, id, data)
		// 		*/
		// 	},
		// },
		TypeCheckFuncBodies: func(_ string) bool { return false },
		SourceImports:       true,
	}
	err := conf.CreateFromFilenames("", filepaths...)
	if err != nil {
		return nil, err
	}
	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	pkgInfo := prog.Created[0]

	// Ignore (perhaps) "unresolved identifier" errors
	files := map[string]*ast.File{}
	for _, f := range pkgInfo.Files {
		files[prog.Fset.File(f.Pos()).Name()] = f

	}
	astPkg, _ := ast.NewPackage(prog.Fset, files, nil, nil)

	var mode doc.Mode
	docPkg := doc.New(astPkg, path, mode)

	return &Package{
		Fset:  prog.Fset,
		Doc:   docPkg,
		Types: pkgInfo.Pkg,
	}, nil
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
		Fset:  prog.Fset,
		Doc:   docPkg,
		Types: pkgInfo.Pkg,
	}
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

	for _, docF := range p.Doc.Funcs {
		name := docF.Name
		if typesF, ok := p.Types.Scope().Lookup(name).(*types.Func); ok {
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

	for _, docT := range p.Doc.Types {
		name := docT.Name
		if typesT, ok := p.Types.Scope().Lookup(name).(*types.TypeName); ok {
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
