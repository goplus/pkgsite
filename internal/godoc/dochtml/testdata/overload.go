package overload

import "fmt"

const GopPackage = true

const Gopo_Mul = "MulInt,,MulFloat"

type N struct {
}

type Foo struct {
}

const Gopo_Foo_Division = ".DivisionInt,.DivisionFoo"

// Add int
func Add__0(a int, b int) int {
	return a + b
}

// Add string
func Add__1(a string, b string) string {
	return a + b
}

// Mut int
func MulInt(a int, b int) int {
	return a * b
}

// Mut string
func Mul__1(a string, b string) string {
	return a + b
}

// Mut float
func MulFloat(a float64, b float64) float64 {
	return a * b
}

func (a *Foo) DivisionInt(b int) *Foo {
	fmt.Println("DivisionInt")
	return a
}

func (a *Foo) DivisionFoo(b *Foo) *Foo {
	fmt.Println("DivisionFoo")
	return a
}

// OnKey string && fn
func (m *N) OnKey__0(a string, fn func()) {
}

// OnKey string && fn(string)
func (m *N) OnKey__1(a string, fn func(key string)) {
}

// OnKey string[] && fn(string)
func (m *N) OnKey__2(a []string, fn func()) {
}
