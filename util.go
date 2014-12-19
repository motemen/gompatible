package gompatible

import (
	"reflect"
	"sort"
)

func mapKeys(m interface{}) []string {
	v := reflect.ValueOf(m)

	refKeys := v.MapKeys()
	keys := make([]string, len(refKeys))

	for i, rk := range refKeys {
		keys[i] = rk.String()
	}

	return keys
}

func sortedStringSet(sets ...[]string) []string {
	seen := map[string]bool{}
	for _, s := range sets {
		for _, str := range s {
			seen[str] = true
		}
	}

	items := make([]string, 0, len(seen))
	for item := range seen {
		items = append(items, item)
	}

	sort.Strings(items)

	return items
}
