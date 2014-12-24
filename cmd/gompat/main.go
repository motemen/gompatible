package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/motemen/gompatible"
	"github.com/motemen/gompatible/util"

	"github.com/daviddengcn/go-colortext"
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

	dir1 := &gompatible.DirSpec{VCS: vcsType, Revision: revs[0], Path: path}
	pkgs1, err := gompatible.LoadDir(dir1, *flagRecurse)
	dieIf(err)

	dir2 := &gompatible.DirSpec{VCS: vcsType, Revision: revs[1], Path: path}
	pkgs2, err := gompatible.LoadDir(dir2, *flagRecurse)
	dieIf(err)

	diffs := map[string]gompatible.PackageChanges{}

	for _, name := range util.SortedStringSet(util.MapKeys(pkgs1), util.MapKeys(pkgs2)) {
		diffs[name] = gompatible.DiffPackages(
			pkgs1[name], pkgs2[name],
		)
	}

	var packageIndex int
	for _, name := range util.SortedStringSet(util.MapKeys(diffs)) {
		diff := diffs[name]

		var headerShown bool
		printHeader := func() {
			if *flagRecurse == false {
				return
			}

			if !headerShown {
				// FIXME strictly not a package if inspecting local import
				if packageIndex > 0 {
					fmt.Println()
				}
				fmt.Printf("package %s\n", name)
				headerShown = true
				packageIndex++
			}
		}

		for _, name := range util.SortedStringSet(util.MapKeys(diff.Funcs)) {
			change := diff.Funcs[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				printChange(change)
			}
		}

		for _, name := range util.SortedStringSet(util.MapKeys(diff.Types)) {
			change := diff.Types[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				printChange(change)
			}
		}
	}
}

type changeMark struct {
	mark  string // should have length == 2
	color ct.Color
}

var (
	markAdded      = changeMark{"+ ", ct.Green}
	markRemoved    = changeMark{"- ", ct.Red}
	markUnchanged  = changeMark{"= ", ct.Blue}
	markCompatible = changeMark{"* ", ct.Yellow}
	markBreaking   = changeMark{"! ", ct.Red}
	markConfer     = changeMark{". ", ct.None}
)

func printChange(c gompatible.Change) {
	multiline := false

	show := func(mark changeMark, s string) string {
		lines := strings.Split(s, "\n")
		for i := range lines {
			if i == 0 {
				ct.ChangeColor(mark.color, false, ct.None, false)
				fmt.Print(mark.mark)
				ct.ResetColor()
				fmt.Println(lines[i])
			} else {
				fmt.Println(" ", lines[i])
				multiline = true
			}
		}

		return strings.Join(lines, "\n")
	}

	switch c.Kind() {
	case gompatible.ChangeAdded:
		show(markAdded, c.ShowAfter())
	case gompatible.ChangeRemoved:
		show(markRemoved, c.ShowBefore())
	case gompatible.ChangeUnchanged:
		show(markUnchanged, c.ShowBefore())
	case gompatible.ChangeCompatible:
		show(markCompatible, c.ShowBefore())
		show(markConfer, c.ShowAfter())
	case gompatible.ChangeBreaking:
		fallthrough
	default:
		show(markBreaking, c.ShowBefore())
		show(markConfer, c.ShowAfter())
	}
}
