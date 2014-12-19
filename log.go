package gompatible

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var Debug bool = false

func Debugf(pat string, args ...interface{}) {
	if Debug == false {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	fmt.Fprintf(os.Stderr, "--> %s:%d "+pat+"\n", append([]interface{}{filepath.Base(file), line}, args...)...)
}
