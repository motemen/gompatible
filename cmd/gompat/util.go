package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/tools/go/types"
)

func dieIf(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Fprintf(os.Stderr, "fatal (%s:%d): %s\n", filepath.Base(file), line, err)
		os.Exit(1)
	}
}

func diff(a, b []byte) ([]byte, error) {
	f1, err := ioutil.TempFile("", "gompat")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "gompat")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(append(a, '\n'))
	f2.Write(append(b, '\n'))

	out, err := exec.Command("git", "diff", "--no-index", "--color", f1.Name(), f2.Name()).Output()
	if len(out) > 0 {
		err = nil
	}
	if err != nil {
		return nil, err
	}

	return out, nil
}

func typesObjectString(obj types.Object) string {
	var prefix string

	switch obj.(type) {
	case *types.Builtin:
		prefix = "builtin"
	case *types.Func:
		prefix = "func"
	case *types.Const:
		prefix = "const"
	case *types.PkgName:
		prefix = "package"
	case *types.Var:
		prefix = "var"
	case *types.Label:
		prefix = "label"
	case *types.Nil:
		return "nil"
	case *types.TypeName:
		prefix = "type"
	default:
		panic(fmt.Sprintf("unexpected type: %T", obj))
	}

	return prefix + " " + obj.Name()
}
