package postgres

import (
	"context"
	"encoding/json"

	llpkgcfg "github.com/goplus/llpkgstore/config"
)

// GetLLPkgConfig retrieves the LLPkgConfig file for a given module path and version.
func (db *DB) GetLLPkgConfig(ctx context.Context, modulePath, version string) (*llpkgcfg.LLPkgConfig, error) {
	var err error

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
