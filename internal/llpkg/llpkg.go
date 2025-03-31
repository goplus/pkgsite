package llpkg

import "strings"

const (
	ModulePath = "llpkg"
	GitHubRepo = "github.com/goplus/llpkg"
)

// IsLLPkgModule checks if the module path is a LLPkg module.
func IsLLPkgModule(modulePath string) bool {
	return strings.HasPrefix(modulePath, GitHubRepo)
}
