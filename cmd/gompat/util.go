package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/motemen/gompatible"
)

func forEachName(gen namesYielder, f func(string)) {
	names := []string{}
	gen.yieldNames(func(name string) {
		names = append(names, name)
	})
	sort.Strings(names)

	for _, name := range names {
		f(name)
	}
}

type namesYielder interface {
	yieldNames(func(string))
}

type byFuncName map[string]gompatible.FuncChange

func (b byFuncName) yieldNames(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

type byTypeName map[string]gompatible.TypeChange

func (b byTypeName) yieldNames(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s", err)
		os.Exit(1)
	}
}
