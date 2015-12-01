package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/motemen/gompatible"
	"github.com/motemen/gompatible/internal/util"

	"github.com/daviddengcn/go-colortext"
)

func usage() {
	fmt.Printf("Usage: %s [-a] [-r] <rev1>[..<rev2>] [<import path>[...]]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	var (
		flagAll     = flag.Bool("a", false, "show also unchanged APIs")
		flagRecurse = flag.Bool("r", false, "recurse into subdirectories")
		flagDiff    = flag.Bool("d", false, "run `diff` on multi-line changes")
	)
	flag.Parse()
	flag.Usage = usage

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	vcsType := "git" // TODO auto-detect

	revs := strings.SplitN(args[0], "..", 2)
	if len(revs) == 1 {
		revs = []string{revs[0] + "~1", revs[0]}
	} else if revs[1] == "" {
		revs = []string{revs[0], ""}
	}

	path := "."
	if len(args) >= 2 {
		path = args[1]

		if strings.HasSuffix(path, "...") {
			path = strings.TrimSuffix(path, "...")
			*flagRecurse = true
		}
	}

	dir1, err := gompatible.NewDirSpec(path, vcsType, revs[0])
	dieIf(err)

	pkgs1, err := gompatible.LoadDir(dir1, *flagRecurse)
	dieIf(err)

	dir2, err := gompatible.NewDirSpec(path, vcsType, revs[1])
	dieIf(err)

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

		funcs := diff.Funcs()
		for _, name := range util.SortedStringSet(util.MapKeys(funcs)) {
			change := funcs[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				printChange(change, *flagDiff)
			}
		}

		types := diff.Types()
		for _, name := range util.SortedStringSet(util.MapKeys(types)) {
			change := types[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				printChange(change, *flagDiff)
			}
		}

		values := diff.Values()
		for _, name := range util.SortedStringSet(util.MapKeys(values)) {
			change := values[name]
			if *flagAll || change.Kind() != gompatible.ChangeUnchanged {
				printHeader()
				printChange(change, *flagDiff)
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

var rxDiffThunkStart = regexp.MustCompile(`^(?:\x1b\[\d+m)?@@ `)

func printChange(c gompatible.Change, doDiff bool) {
	show := func(mark changeMark, s string) {
		lines := strings.Split(s, "\n")
		for i := range lines {
			if i == 0 {
				ct.ChangeColor(mark.color, false, ct.None, false)
				fmt.Print(mark.mark)
				ct.ResetColor()
				fmt.Println(lines[i])
			} else {
				fmt.Println(" ", lines[i])
			}
		}
	}

	switch c.Kind() {
	case gompatible.ChangeAdded:
		show(markAdded, c.ShowAfter())
	case gompatible.ChangeRemoved:
		show(markRemoved, c.ShowBefore())
	case gompatible.ChangeUnchanged:
		show(markUnchanged, c.ShowBefore())
	case gompatible.ChangeCompatible:
		showCompare(markCompatible, c, show, doDiff)
	case gompatible.ChangeBreaking:
		showCompare(markBreaking, c, show, doDiff)
	}
}

func showCompare(mark changeMark, c gompatible.Change, show func(changeMark, string), doDiff bool) {
	if doDiff == false {
		show(mark, c.ShowBefore())
		show(markConfer, c.ShowAfter())
		return
	}

	d, err := diff([]byte(c.ShowBefore()), []byte(c.ShowAfter()))
	dieIf(err)

	ct.ChangeColor(mark.color, false, ct.None, false)
	fmt.Print(mark.mark)
	ct.ResetColor()

	fmt.Println(typesObjectString(c.TypesObject()))

	lines := strings.Split(string(d), "\n")
	inHeader := true
	for _, line := range lines {
		if inHeader {
			if rxDiffThunkStart.MatchString(line) {
				inHeader = false
			} else {
				continue
			}
		}
		fmt.Println("  " + line)
	}
}
