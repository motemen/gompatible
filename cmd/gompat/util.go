package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/motemen/gompatible"
)

type funcNames map[string]gompatible.FuncChange

func (b funcNames) Yield(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

type typeNames map[string]gompatible.TypeChange

func (b typeNames) Yield(yield func(string)) {
	for name := range b {
		yield(name)
	}
}

type pkgNames map[string]*gompatible.Package

func (b pkgNames) Yield(yield func(string)) {
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
