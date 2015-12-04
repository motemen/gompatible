package testdata

import (
	"bytes"
	"io"
)

func Unchanged1(n int)
func Unchanged2(n int) (err error)
func Unchanged3(m int) error

func Compatible1(n int, opts ...string)
func Compatible2(n int) error

func Breaking1(n int, b bool)
func Breaking2(n int) ([]byte, error)
func Breaking3(n int)
func Breaking4(n int) []byte
func Added1()

type UnchangedT1 int
type UnchangedT2 struct {
	Foo string
}
type UnchangedT3 struct {
	Foo string
}

// Should not care unexported fields
type UnchangedT4 struct {
	Foo string
	xxx interface{}
}
type CompatibleT1 struct {
	Foo string
	Bar bool
}
type BreakingT1 struct {
	YYY int
}
type AddedT1 interface{}

var UnchangedV1 int

var BreakingV1 bool

const BreakingV2 int = 0

var CompatibleV1 struct {
	Foo int
	Bar int
}

type AuxInt int

var CompatibleV2 string

var BreakingV3 AuxInt

var AddedV1 int

type CompatibleT3 uint8

type CompatibleT4 io.Writer

type CompatibleT5 <-chan struct{}

func Compatible3(b io.Reader)

func Compatible4() *bytes.Buffer
