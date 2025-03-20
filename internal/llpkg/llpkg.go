package llpkg

import "strings"

const (
	ModulePath = "llpkg"
	GitHubRepo = "github.com/goplus/llpkg"
)

// TODO: 实现 Check 是否为llpkg的函数
func CheckURL(path string) bool {
	return strings.Contains(path, GitHubRepo)
}
