package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	llpkgcfg "github.com/goplus/llpkgstore/config"
	"golang.org/x/pkgsite/internal/llpkg"
)

// GetLLPkgConfig retrieves the LLPkgConfig file for a given module path and version.
func (db *DB) GetLLPkgConfig(ctx context.Context, modulePath, version string) (*llpkgcfg.LLPkgConfig, error) {
	var err error

	if llpkg.IsOfficialLLPkgModule(modulePath) {
		return nil, fmt.Errorf("%s@%s is not a LLPkg module", modulePath, version)
	}

	var llpkgConfig *llpkgcfg.LLPkgConfig
	var content json.RawMessage
	err = db.db.QueryRow(ctx, `
        SELECT llpkg_config
        FROM modules
        WHERE module_path = $1 AND version = $2`,
		modulePath, version).Scan(&content)

	err = json.Unmarshal(content, &llpkgConfig)

	return llpkgConfig, err
}
