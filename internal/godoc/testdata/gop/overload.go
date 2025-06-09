package foo

const GopPackage = true

const Gopo_Mul = "MulInt,,MulFloat"

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
