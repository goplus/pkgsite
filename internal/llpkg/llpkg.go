package llpkg

import (
	"strings"
)

const (
	ModulePath = "llpkg"
	GitHubRepo = "github.com/goplus/llpkg"
)

// IsOfficialLLPkgModule checks if the module path is a LLPkg module in the official repository.
func IsOfficialLLPkgModule(modulePath string) bool {
	paths := strings.Split(modulePath, "/")
	if len(paths) < 2 {
		return false
	}
	if paths[0] != "github.com" {
		return false
	}
	if paths[1] != "goplus" {
		return false
	}
	if paths[2] != "llpkg" {
		return false
	}
	return true
}
