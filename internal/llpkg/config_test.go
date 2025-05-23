package llpkg

import (
	"testing"
)

func TestIsOfficialLLPkgModule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		modulePath string
		want       bool
	}{
		{
			name:       "official llpkg module",
			modulePath: "github.com/goplus/llpkg",
			want:       true,
		},
		{
			name:       "official llpkg submodule",
			modulePath: "github.com/goplus/llpkg/cjson",
			want:       true,
		},
		{
			name:       "different repo with similar name",
			modulePath: "github.com/goplus/llpkg-fork",
			want:       false,
		},
		{
			name:       "different owner",
			modulePath: "github.com/someone/llpkg",
			want:       false,
		},
		{
			name:       "empty path",
			modulePath: "",
			want:       false,
		},
		{
			name:       "llpkg module path",
			modulePath: "llpkg",
			want:       false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := IsOfficialLLPkgModule(test.modulePath)
			if got != test.want {
				t.Errorf("IsOfficialLLPkgModule(%q) = %v, want %v",
					test.modulePath, got, test.want)
			}
		})
	}
}
