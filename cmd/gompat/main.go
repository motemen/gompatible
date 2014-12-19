package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"
	"github.com/motemen/gompatible/sortedset"
)

func usage() {
	fmt.Printf("Usage: %s [-a] [-r] <rev1>[..<rev2>] [<import path>[...]]\n", os.Args[0])
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
	dieIf(err)

	dir2 := gompatible.DirSpec{VCS: vcsType, Revision: revs[1], Path: path}
	pkgs2, err := gompatible.LoadDir(dir2, *flagRecurse)
	dieIf(err)

	diffs := map[string]gompatible.PackageChanges{}

	sortedset.Strings(pkgNames(pkgs1), pkgNames(pkgs2)).ForEach(func(name string) {
		diffs[name] = gompatible.DiffPackages(
			pkgs1[name], pkgs2[name],
		)
	})

	for name, diff := range diffs {
		var headerShown bool
		printHeader := func() {
			if *flagRecurse == false {
				return
			}

			if !headerShown {
				// FIXME strictly not a package if inspecting local import
				fmt.Printf("package %s\n", name)
				headerShown = true
			}
		}

		sortedset.Strings(funcNames(diff.Funcs)).ForEach(func(name string) {
			change := diff.Funcs[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				fmt.Println(gompatible.ShowChange(change))
			}
		})

		sortedset.Strings(typeNames(diff.Types)).ForEach(func(name string) {
			change := diff.Types[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				fmt.Println(showChange(change))
			}
		})
	}
}

func showChange(c gompatible.Change) string {
	multiline := false

	// len(prefix) == 1
	show := func(prefix, s string) string {
		lines := strings.Split(s, "\n")
		for i := range lines {
			if i == 0 {
				lines[i] = prefix + " " + lines[i]
			} else {
				lines[i] = "  " + lines[i]
				multiline = true
			}
		}

		return strings.Join(lines, "\n")
	}

	switch c.Kind() {
	case gompatible.ChangeAdded:
		return show("+", c.ShowAfter())
	case gompatible.ChangeRemoved:
		return show("-", c.ShowBefore())
	case gompatible.ChangeUnchanged:
		return show("=", c.ShowBefore())
	case gompatible.ChangeCompatible:
		var (
			before = show("*", c.ShowBefore())
			after  = show(" ", c.ShowAfter())
		)
		sep := " "
		if multiline {
			sep = "\n"
		}

		return before + sep + "->" + sep + after
	case gompatible.ChangeBreaking:
		fallthrough
	default:
		var (
			before = show("!", c.ShowBefore())
			after  = show(" ", c.ShowAfter())
		)
		sep := " "
		if multiline {
			sep = "\n"
		}

		return before + sep + "->" + sep + after
	}
}
