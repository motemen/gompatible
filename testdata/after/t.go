package testdata

func Unchanged1(n int)
func Compatible1(n int, opts ...string)
func Compatible2(n int) error
func Compatible3(m int) error
func Breaking1(n int, b bool)
func Breaking2(n int) ([]byte, error)
func Breaking3(n int)
func Breaking4(n int) []byte
func Added1()

type UnchangedT1 int
type UnchangedT2 struct {
	Foo string
}
type CompatibleT1 struct {
	Foo string
	xxx interface{}
}
type CompatibleT2 struct {
	Foo string
	Bar bool
}
type BreakingT1 struct {
	YYY int
}
type AddedT1 interface{}
