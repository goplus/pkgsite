package llpkg

import "strings"

const (
	ModulePath = "llpkg"
	GitHubRepo = "github.com/goplus/llpkg"
)

// IsOfficialLLPkgModule checks if the module path is a LLPkg module in the official repository.
func IsOfficialLLPkgModule(modulePath string) bool {
	return strings.HasPrefix(modulePath, GitHubRepo)
}
