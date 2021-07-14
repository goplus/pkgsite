// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/database"
	"golang.org/x/pkgsite/internal/derrors"
	"golang.org/x/pkgsite/internal/experiment"
)

func upsertSymbolSearchDocuments(ctx context.Context, tx *database.DB,
	modulePath, v string) (err error) {
	defer derrors.Wrap(&err, "upsertSymbolSearchDocuments(ctx, ddb, %q, %q)", modulePath, v)

	if !experiment.IsActive(ctx, internal.ExperimentInsertSymbolSearchDocuments) {
		return nil
	}

	// If a user is looking for the symbol "DB.Begin", from package
	// database/sql, we want them to be able to find this by searching for
	// "DB.Begin" and "sql.DB.Begin". Searching for "sql.DB", "DB", "Begin" or
	// "sql.DB" will not return "DB.Begin".
	// If a user is looking for the symbol "DB.Begin", from package
	// database/sql, we want them to be able to find this by searching for
	// "DB.Begin", "Begin", and "sql.DB.Begin". Searching for "sql.DB" or
	// "DB" will not return "DB.Begin".
	q := `
		INSERT INTO symbol_search_documents (
			package_path_id,
			symbol_name_id,
			unit_id,
			package_symbol_id,
			goos,
			goarch
		)
		SELECT DISTINCT ON (sd.package_path_id, ps.symbol_name_id)
			sd.package_path_id,
			ps.symbol_name_id,
			sd.unit_id,
			ps.id AS package_symbol_id,
			d.goos,
			d.goarch
		FROM search_documents sd
		INNER JOIN units u
			ON sd.unit_id = u.id
		INNER JOIN documentation d
			ON d.unit_id = sd.unit_id
		INNER JOIN documentation_symbols ds
			ON d.id = ds.documentation_id
		INNER JOIN package_symbols ps
			ON ps.id = ds.package_symbol_id
		WHERE
			sd.module_path = $1 AND sd.version = $2
			AND u.name != 'main' -- do not insert data for commands
		ORDER BY
			sd.package_path_id,
			ps.symbol_name_id,
			-- Order should match internal.BuildContexts.
			CASE WHEN d.goos = 'all' THEN 0
			WHEN d.goos = 'linux' THEN 1
			WHEN d.goos = 'windows' THEN 2
			WHEN d.goos = 'darwin' THEN 3
			WHEN d.goos = 'js' THEN 4
			END
		ON CONFLICT (package_path_id, symbol_name_id)
		DO UPDATE SET
			unit_id = excluded.unit_id,
			package_symbol_id = excluded.package_symbol_id,
			goos = excluded.goos,
			goarch = excluded.goarch;`
	_, err = tx.Exec(ctx, q, modulePath, v)
	return err
}

// symbolSearch searches all symbols in the symbol_search_documents table for
// the query.
//
// TODO(https://golang.org/issue/44142): factor out common code between
// symbolSearch and deepSearch.
func (db *DB) symbolSearch(ctx context.Context, q string, limit, offset, maxResultCount int) searchResponse {
	query := fmt.Sprintf(`
		SELECT
			package_path,
			module_path,
			version,
			name,
			synopsis,
			license_types,
			commit_time,
			imported_by_count,
			symbol_name,
		    type,
		    symbol_synopsis,
		    goos,
		    goarch,
			COUNT(*) OVER() AS total
		FROM (
			SELECT
				DISTINCT ON (s.name, sd.package_path)
				s.name AS symbol_name,
				sd.package_path,
				sd.module_path,
				sd.version,
			    sd.name,
			    sd.synopsis,
			    sd.license_types,
				sd.commit_time,
				sd.imported_by_count,
				ps.type,
				ps.synopsis AS symbol_synopsis,
				d.goos,
				d.goarch,
				(%s) AS score
			FROM symbol_search_documents ssd
			INNER JOIN symbol_names s ON s.id = ssd.symbol_name_id
			INNER JOIN units u ON u.id = ssd.unit_id
			INNER JOIN search_documents sd ON sd.unit_id = u.id
			INNER JOIN documentation d ON d.unit_id = u.id
			INNER JOIN documentation_symbols ds ON ds.documentation_id = d.id
			INNER JOIN package_symbols ps
				ON ps.id = ds.package_symbol_id
				AND ps.symbol_name_id = ssd.symbol_name_id
			WHERE
				ssd.tsv_symbol_tokens @@ `+symbolToTSQuery+
		`ORDER BY
				symbol_name,
				package_path,
				CASE WHEN d.goos = 'all' THEN 0
					 WHEN d.goos = 'linux' THEN 1
					 WHEN d.goos = 'windows' THEN 2
					 WHEN d.goos = 'darwin' THEN 3
					 WHEN d.goos = 'js' THEN 4
					 END
		) r
		WHERE r.score > 0.1
		ORDER BY
			score DESC,
			commit_time DESC,
			symbol_name,
			package_path
		LIMIT $2
		OFFSET $3`, symbolScoreExpr)

	var results []*SearchResult
	collect := func(rows *sql.Rows) error {
		var r SearchResult
		if err := rows.Scan(
			&r.PackagePath,
			&r.ModulePath,
			&r.Version,
			&r.Name,
			&r.Synopsis,
			pq.Array(&r.Licenses),
			&r.CommitTime,
			&r.NumImportedBy,
			&r.SymbolName,
			&r.SymbolKind,
			&r.SymbolSynopsis,
			&r.SymbolGOOS,
			&r.SymbolGOARCH,
			&r.NumResults); err != nil {
			return fmt.Errorf("symbolSearch: rows.Scan(): %v", err)
		}
		results = append(results, &r)
		return nil
	}

	// Search for an OR of the terms, so that if the user searches for
	// "db begin", queries matching "db" and "begin" will be returned.
	q = strings.Join(strings.Split(q, " "), " | ")

	err := db.db.RunQuery(ctx, query, collect, q, limit, offset)
	if err != nil {
		results = nil
	}
	if len(results) > 0 && results[0].NumResults > uint64(maxResultCount) {
		for _, r := range results {
			r.NumResults = uint64(maxResultCount)
		}
	}
	return searchResponse{
		source:  "symbol",
		results: results,
		err:     err,
	}
}

// symbolTextSearchConfiguration is the search configuration that is used for
// indexing and searching for symbols.
const symbolTextSearchConfiguration = "symbols"

// processSymbol converts a symbol with underscores to slashes (for example,
// "A_B" -> "A/B"). This is because the postgres parser treats underscores as
// slashes, but we want a search for "A" to rank "A_B" lower than just "A". We
// also want to be able to search specificially for "A_B".
func processSymbol(s string) string {
	return fmt.Sprintf("replace(%s, '_', '/')", s)
}

var symbolToTSQuery = fmt.Sprintf("to_tsquery('%s', %s)", symbolTextSearchConfiguration, processSymbol("$1"))

var symbolScoreExpr = fmt.Sprintf(`
		ts_rank('{0.1, 0.2, 1.0, 1.0}', ssd.tsv_symbol_tokens, `+symbolToTSQuery+`) *
		ln(exp(1)+imported_by_count) *
		CASE WHEN u.redistributable THEN 1 ELSE %f END *
		CASE WHEN COALESCE(has_go_mod, true) THEN 1 ELSE %f END
	`, nonRedistributablePenalty, noGoModPenalty)
