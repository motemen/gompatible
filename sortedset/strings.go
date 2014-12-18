package sortedset

import (
	"sort"
)

func Strings(gen ...StringsYielder) stringsHolder {
	seen := map[string]bool{}
	for _, g := range gen {
		g.Yield(func(item string) {
			seen[item] = true
		})
	}

	items := []string{}
	for item := range seen {
		items = append(items, item)
	}

	sort.Strings(items)

	return stringsHolder(items)
}

func StringSlices(slices ...[]string) stringsHolder {
	array := []string{}
	for _, s := range slices {
		array = append(array, s...)
	}
	return stringsHolder(array)
}

type StringsYielder interface {
	Yield(func(string))
}

type stringsHolder []string

func (h stringsHolder) ForEach(f func(string)) {
	for _, item := range h {
		f(item)
	}
}
