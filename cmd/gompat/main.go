package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"
)

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
	pkgs1, err := gompatible.LoadDir(dir1, *flagRecurse)

	dir2 := gompatible.DirSpec{VCS: vcsType, Revision: revs[1], Path: path}
	pkgs2, err := gompatible.LoadDir(dir2, *flagRecurse)
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
