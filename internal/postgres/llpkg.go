package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	llpkgcfg "github.com/goplus/llpkgstore/config"
	"golang.org/x/pkgsite/internal/llpkg"
)

type LLPkgConfigNotFoundError struct {
	ModulePath string
	Version    string
}

func (e *LLPkgConfigNotFoundError) Error() string {
	return fmt.Sprintf("%s@%s is not a LLPkg module", e.ModulePath, e.Version)
}

// GetLLPkgConfig retrieves the LLPkgConfig file for a given module path and version.
func (db *DB) GetLLPkgConfig(ctx context.Context, modulePath, version string) (*llpkgcfg.LLPkgConfig, error) {
	if !llpkg.IsOfficialLLPkgModule(modulePath) {
		return nil, &LLPkgConfigNotFoundError{modulePath, version}
	}

	var llpkgConfig *llpkgcfg.LLPkgConfig
	var content json.RawMessage
	err := db.db.QueryRow(ctx, `
        SELECT llpkg_config
        FROM modules
        WHERE module_path = $1 AND version = $2`,
		modulePath, version).Scan(&content)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(content, &llpkgConfig); err != nil {
		return nil, fmt.Errorf("unmarshal llpkg config: %v", err)
	}

	if content == nil {
		return nil, &LLPkgConfigNotFoundError{modulePath, version}
	}

	return llpkgConfig, nil
}
