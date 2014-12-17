package gompatible

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func Debugf(pat string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "--> %s:%d "+pat+"\n", append([]interface{}{filepath.Base(file), line}, args...)...)
}

func Stub() error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("stub %s:%d", filepath.Base(file), line)
}
