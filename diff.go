package gompatible

type PackageChanges struct {
	Before *Package
	After  *Package
	Funcs  map[string]FuncChange
	Types  map[string]TypeChange
}

func DiffPackages(pkg1, pkg2 *Package) PackageChanges {
	diff := PackageChanges{
		Before: pkg1,
		After:  pkg2,
		Funcs:  map[string]FuncChange{},
		Types:  map[string]TypeChange{},
	}

	for _, name := range union(pkg1.FuncNames(), pkg2.FuncNames()) {
		diff.Funcs[name] = FuncChange{
			Before: pkg1.Func(name),
			After:  pkg2.Func(name),
		}
	}

	for _, name := range union(pkg1.TypeNames(), pkg2.TypeNames()) {
		diff.Types[name] = TypeChange{
			Before: pkg1.Type(name),
			After:  pkg2.Type(name),
		}
	}

	return diff
}

func union(ss ...[]string) []string {
	union := []string{}
	seen := map[string]bool{}

	for _, s := range ss {
		for _, str := range s {
			if seen[str] == false {
				seen[str] = true
				union = append(union, str)
			}
		}
	}

	return union
}
