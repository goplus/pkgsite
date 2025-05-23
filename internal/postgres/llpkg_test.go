package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	llpkgcfg "github.com/goplus/llpkgstore/config"
	"golang.org/x/pkgsite/internal/testing/sample"
)

func TestGetLLPkgConfig(t *testing.T) {
	ctx := context.Background()
	testDB, release := acquire(t)
	defer release()

	modulePath := "github.com/NEKO-CwC/llpkgstore/cjson"
	version := "v1.0.0"

	llpkgConfig := &llpkgcfg.LLPkgConfig{
		Upstream: llpkgcfg.UpstreamConfig{
			Installer: llpkgcfg.InstallerConfig{
				Name: "conan",
				Config: map[string]string{
					"shared": "True",
				},
			},
			Package: llpkgcfg.PackageConfig{
				Name:    "cjson",
				Version: "1.7.15",
			},
		},
	}

	configBytes, err := json.Marshal(llpkgConfig)
	if err != nil {
		t.Fatal(err)
	}

	m := sample.Module(modulePath, version)
	m.LLPkgConfig = configBytes

	if _, err := testDB.InsertModule(ctx, m, nil); err != nil {
		t.Fatal(err)
	}

	t.Run("valid llpkg module", func(t *testing.T) {
		got, err := testDB.GetLLPkgConfig(ctx, modulePath, version)
		if err != nil {
			t.Fatalf("GetLLPkgConfig(%q, %q) = _, %v; want _, nil", modulePath, version, err)
		}

		if diff := cmp.Diff(llpkgConfig, got); diff != "" {
			t.Errorf("GetLLPkgConfig(%q, %q) mismatch (-want +got):\n%s",
				modulePath, version, diff)
		}
	})

	t.Run("non-llpkg module", func(t *testing.T) {
		nonLLPkgPath := "golang.org/x/tools"
		_, err := testDB.GetLLPkgConfig(ctx, nonLLPkgPath, "v1.0.0")
		var notFoundErr *LLPkgConfigNotFoundError
		if err == nil || !errors.As(err, &notFoundErr) {
			t.Errorf("GetLLPkgConfig(%q, %q) = _, %v; want LLPkgConfigNotFoundError",
				nonLLPkgPath, "v1.0.0", err)
		}
	})

	t.Run("non-existent module", func(t *testing.T) {
		_, err := testDB.GetLLPkgConfig(ctx, modulePath, "v2.0.0")
		if err == nil {
			t.Errorf("GetLLPkgConfig(%q, %q) = _, nil; want error",
				modulePath, "v2.0.0")
		}
	})
}
