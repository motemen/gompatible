package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func dieIf(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "fatal (%s:%d): %s\n", filepath.Base(file), line, err)
		os.Exit(1)
	}
}
