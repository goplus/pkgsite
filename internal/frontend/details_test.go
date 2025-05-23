// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frontend

import (
	"testing"
)

func TestStdlibRedirectURL(t *testing.T) {
	for _, test := range []struct {
		path string
		want string
	}{
		{"std", ""},
		{"cmd/go", ""},
		{"github.com/golang/go", "/std"},
		{"github.com/golang/go/src", "/std"},
		{"github.com/golang/go/src", "/std"},
		{"github.com/golang/go/cmd/go", "/cmd/go"},
		{"github.com/golang/go/src/cmd/go", "/cmd/go"},
		{"github.com/golang/gofrontend", ""},
		{"github.com/golang/gofrontend/libgo/misc/cgo/frontend/libgo/misc", ""},
	} {
		if got := stdlibRedirectURL(test.path); got != test.want {
			t.Errorf("stdlibRedirectURL(%q) = %q; want = %q", test.path, got, test.want)
		}
	}
}

func TestLLPkgRedirectURL(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		fullPath string
		want     string
	}{
		{
			fullPath: "github.com/NEKO-CwC/llpkgstore",
			want:     "/llpkg",
		},
		{
			fullPath: "github.com/NEKO-CwC/llpkgstore/cjson",
			want:     "",
		},
		{
			fullPath: "github.com/golang/go",
			want:     "",
		},
		{
			fullPath: "llpkg",
			want:     "",
		},
		{
			fullPath: "",
			want:     "",
		},
	} {
		t.Run(test.fullPath, func(t *testing.T) {
			got := llpkgRedirectURL(test.fullPath)
			if got != test.want {
				t.Errorf("llpkgRedirectURL(%q) = %q, want %q",
					test.fullPath, got, test.want)
			}
		})
	}
}
