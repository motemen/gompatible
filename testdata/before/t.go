package testdata

import (
	"bytes"
	"io"
)

func Unchanged1(n int)
func Unchanged2(n int) error
func Unchanged3(n int) error

func Compatible1(n int)
func Compatible2(n int)

func Breaking1(n int)
func Breaking2(n int) []byte
func Breaking3(n int, s string)
func Breaking4(n int) string
func Removed1()

type RemovedT1 bool
type UnchangedT1 int
type UnchangedT2 struct {
	Foo string
}
type UnchangedT3 struct {
	// Foo is a foo
	Foo string
}

// Should not care unexported fields
type UnchangedT4 struct {
	Foo string
}
type CompatibleT1 struct {
	Foo string
}
type BreakingT1 struct {
	XXX string
}

var UnchangedV1 int

var BreakingV1 []string

var BreakingV2 int

var CompatibleV1 struct {
	Foo int
}

const CompatibleV2 = ""

var BreakingV3 int

var RemovedV1 int

type CompatibleT3 byte

type CompatibleT4 *bytes.Buffer

type CompatibleT5 chan struct{}

func Compatible3(b *bytes.Buffer)

func Compatible4() io.Reader
