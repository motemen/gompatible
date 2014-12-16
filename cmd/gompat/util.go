package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/motemen/gompatible"
)

type forEachStringHolder struct {
	yielders []stringsYielder
}

func (h forEachStringHolder) do(f func(string)) {
	seen := map[string]bool{}
	for _, y := range h.yielders {
		y.yieldStrings(func(name string) {
			seen[name] = true
		})
	}

	names := []string{}
	for name := range seen {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		f(name)
	}
}

func forEachString(gen ...stringsYielder) forEachStringHolder {
	return forEachStringHolder{gen}
}

type stringsYielder interface {
	yieldStrings(func(string))
}

type funcNames map[string]gompatible.FuncChange

func (b funcNames) yieldStrings(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

type typeNames map[string]gompatible.TypeChange

func (b typeNames) yieldStrings(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

func dieIf(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "fatal (%s:%d): %s\n", filepath.Base(file), line, err)
		os.Exit(1)
	}
}
