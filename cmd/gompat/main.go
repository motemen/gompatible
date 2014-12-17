package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"
)

type packageFiles struct {
	packageName string
	fset        *token.FileSet
	files       map[string]*ast.File
}

// XXX should the return value be a map from dir to files? (currently assumed importPath to files)
func listPackages(dir gompatible.DirSpec, recurse bool) (map[string][]string, error) {
	ctx, err := dir.BuildContext()
	if err != nil {
		return nil, err
	}

	var readDir func(string) ([]os.FileInfo, error)
	if ctx.ReadDir != nil {
		readDir = ctx.ReadDir
	} else {
		readDir = ioutil.ReadDir
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
		gompatible.Debugf("%+v", p)
		importPath := p.ImportPath
		if importPath == "." {
			importPath = p.Dir
		}

		// XXX something's wrong if packages[importPath] exists already
		packages[importPath] = make([]string, len(p.GoFiles))
		for i, file := range p.GoFiles {
			// TODO use ctx.JoinPath
			packages[importPath][i] = filepath.Join(dir.Path, file)
		}
	}

	if recurse == false {
		return packages, nil
	}

	entries, err := readDir(dir.Path)
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

		pkgs, err := listPackages(dir.Subdir(e.Name()), recurse)
		if err != nil {
			return nil, err
		}
		for path, files := range pkgs {
			packages[path] = files
		}
	}

	return packages, nil
}

func usage() {
	fmt.Printf("Usage: %s <rev1>[..<rev2>] [<path>]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var (
		flagAll     = flag.Bool("a", false, "show also unchanged APIs")
		flagRecurse = flag.Bool("r", false, "recurse into subdirectories")
	)
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	vcsType := "git" // TODO auto-detect

	revs := strings.Split(args[0], "..")
	if len(revs) < 2 || revs[1] == "" {
		revs = []string{revs[0], ""}
	}

	path := "."
	if len(args) >= 2 {
		path = args[1]

		if strings.HasSuffix(path, "...") {
			path = strings.TrimSuffix(path, "...")
			*flagRecurse = true
		}

		if build.IsLocalImport(path) == false {
			for _, srcDir := range build.Default.SrcDirs() {
				pkgPath := filepath.Join(srcDir, path)
				if _, err := os.Stat(pkgPath); err == nil {
					path = pkgPath
					break
				}
			}
		}
	}

	dir1 := gompatible.DirSpec{VCS: vcsType, Revision: revs[0], Path: path}
	ctx1, err := dir1.BuildContext()
	dieIf(err)

	pkgList1, err := listPackages(dir1, *flagRecurse)
	dieIf(err)

	pkgs1, err := gompatible.LoadPackages(ctx1, pkgList1)
	dieIf(err)

	dir2 := gompatible.DirSpec{VCS: vcsType, Revision: revs[1], Path: path}
	ctx2, err := dir2.BuildContext()
	dieIf(err)

	pkgList2, err := listPackages(dir2, *flagRecurse)
	dieIf(err)

	pkgs2, err := gompatible.LoadPackages(ctx2, pkgList2)
	dieIf(err)

	diffs := map[string]gompatible.PackageChanges{}

	forEachString(pkgNames(pkgs1), pkgNames(pkgs2)).do(func(name string) {
		diffs[name] = gompatible.DiffPackages(
			pkgs1[name], pkgs2[name],
		)
	})

	for name, diff := range diffs {
		var headerShown bool
		showHeader := func() {
			if !headerShown {
				fmt.Printf("package %s\n", name)
				headerShown = true
			}
		}

		forEachString(funcNames(diff.Funcs)).do(func(name string) {
			change := diff.Funcs[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				showHeader()
				fmt.Println(gompatible.ShowChange(change))
			}
		})

		forEachString(typeNames(diff.Types)).do(func(name string) {
			change := diff.Types[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				showHeader()
				fmt.Println(gompatible.ShowChange(change))
			}
		})
	}
}
