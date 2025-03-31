// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frontend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/safehtml/template"
	"github.com/goplus/llpkgstore/metadata"
	"golang.org/x/pkgsite/internal/derrors"
	"golang.org/x/pkgsite/internal/llpkg"
	"golang.org/x/pkgsite/internal/log"
	"golang.org/x/pkgsite/internal/source"

	"golang.org/x/pkgsite/internal/frontend/page"
	"golang.org/x/pkgsite/internal/frontend/serrors"
	"golang.org/x/pkgsite/internal/frontend/urlinfo"
	mstats "golang.org/x/pkgsite/internal/middleware/stats"

	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/stdlib"
)

// serveDetails handles requests for package/directory/module details pages. It
// expects paths of the form "/<module-path>[@<version>?tab=<tab>]".
// stdlib module pages are handled at "/std", and requests to "/mod/std" will
// be redirected to that path.
func (s *Server) serveDetails(w http.ResponseWriter, r *http.Request, ds internal.DataSource) (err error) {
	defer mstats.Elapsed(r.Context(), "serveDetails")()

	ctx := r.Context()
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return &serrors.ServerError{Status: http.StatusMethodNotAllowed}
	}
	if r.URL.Path == "/" {
		s.serveHomepage(ctx, w, r)
		return nil
	}
	if strings.HasSuffix(r.URL.Path, "/") {
		url := *r.URL
		url.Path = strings.TrimSuffix(r.URL.Path, "/")
		http.Redirect(w, r, url.String(), http.StatusMovedPermanently)
		return
	}

	// If page statistics are enabled, use the "exp" query param to adjust
	// the active experiments.
	if s.serveStats {
		ctx = setExperimentsFromQueryParam(ctx, r)
	}

	urlInfo, err := urlinfo.ExtractURLPathInfo(r.URL.Path)
	if err != nil {
		var epage *page.ErrorPage
		if uerr := new(urlinfo.UserError); errors.As(err, &uerr) {
			epage = &page.ErrorPage{MessageData: uerr.UserMessage}
		}
		return &serrors.ServerError{
			Status: http.StatusBadRequest,
			Err:    err,
			Epage:  epage,
		}
	}
	if !urlinfo.IsSupportedVersion(urlInfo.FullPath, urlInfo.RequestedVersion) {
		return serrors.InvalidVersionError(urlInfo.FullPath, urlInfo.RequestedVersion)
	}
	if urlPath := stdlibRedirectURL(urlInfo.FullPath); urlPath != "" {
		http.Redirect(w, r, urlPath, http.StatusMovedPermanently)
		return
	}
	if urlPath := llpkgRedirectURL(urlInfo.FullPath); urlPath != "" {
		http.Redirect(w, r, urlPath, http.StatusMovedPermanently)
		return
	}
	if err := checkExcluded(ctx, ds, urlInfo.FullPath, urlInfo.RequestedVersion); err != nil {
		return err
	}
	return s.serveUnitPage(ctx, w, r, ds, urlInfo)
}

func stdlibRedirectURL(fullPath string) string {
	if !strings.HasPrefix(fullPath, stdlib.GitHubRepo) {
		return ""
	}
	if fullPath == stdlib.GitHubRepo || fullPath == stdlib.GitHubRepo+"/src" {
		return "/std"
	}
	urlPath2 := strings.TrimPrefix(strings.TrimPrefix(fullPath, stdlib.GitHubRepo+"/"), "src/")
	if fullPath == urlPath2 {
		return ""
	}
	return "/" + urlPath2
}

func llpkgRedirectURL(fullPath string) string {
	if fullPath == llpkg.GitHubRepo {
		return "/llpkg"
	}
	return ""
}

func checkExcluded(ctx context.Context, ds internal.DataSource, fullPath, version string) error {
	db, ok := ds.(internal.PostgresDB)
	if !ok {
		return nil
	}
	if db.IsExcluded(ctx, fullPath, version) {
		// Return NotFound; don't let the user know that the package was excluded.
		return &serrors.ServerError{Status: http.StatusNotFound}
	}
	return nil
}

// serveLLPkg fake the llpkg page with static content
func (s *Server) serveLLPkg(w http.ResponseWriter, r *http.Request, ds internal.DataSource) (err error) {
	defer derrors.Wrap(&err, "serveLLPkg(ctx, w, r)")
	ctx := r.Context()

	// init a base page
	basePage := s.newBasePage(r, "LLPkg")
	basePage.AllowWideContent = true
	basePage.UseResponsiveLayout = true

	// init a unit meta
	unit := &internal.UnitMeta{
		Path: "llpkg",
		Name: "",
		ModuleInfo: internal.ModuleInfo{
			ModulePath:        llpkg.GitHubRepo,
			Version:           "v0.0.0",
			HasGoMod:          true,
			IsRedistributable: true,
			SourceInfo:        source.NewInfo("https://github.com/goplus/llpkg", "src", "v0.0.0"),
			Deprecated:        false,
			Retracted:         false,
		},
	}
	breadcrumb := displayBreadcrumb(unit, "latest")

	// init directories from llpkgstore.json
	var directories []*Directory
	mgr, err := metadata.NewMetadataMgr(os.Getenv("LLPKG_METADATA_DIR")) // init metadata manager from env "LLPKG_METADATA_DIR"
	if err != nil {
		log.Warningf(ctx, "Failed to create metadata manager: %v", err)
	} else {
		metadataMap, err := mgr.AllMetadata()
		if err != nil {
			log.Warningf(ctx, "Failed to get metadata: %v", err)
		} else {
			for clibname := range metadataMap {
				var synopsis string

				latestCVer, err := mgr.LatestCVer(clibname)
				if err != nil {
					continue
				}

				latestGoVer, err := mgr.LatestGoVer(clibname)
				if err != nil {
					continue
				}

				synopsis = fmt.Sprintf("C:%s -> Go:%s", latestCVer, latestGoVer)

				directory := &Directory{
					Prefix: clibname,
					Root: &DirectoryInfo{
						Suffix:     clibname,
						URL:        fmt.Sprintf("/github.com/goplus/llpkg/%s", clibname),
						Synopsis:   synopsis,
						IsModule:   true,
						IsInternal: false,
					},
				}

				directories = append(directories, directory)
			}
		}
	}

	// init main details
	emptyHTML := template.MustParseAndExecuteToHTML("")
	mainDetails := &MainDetails{
		Directories:     directories,
		Readme:          emptyHTML,
		DocBody:         emptyHTML,
		Licenses:        []LicenseMetadata{},
		RepositoryURL:   llpkg.GitHubRepo,
		SourceURL:       llpkg.GitHubRepo,
		ModFileURL:      llpkg.GitHubRepo,
		IsPackage:       false,
		IsTaggedVersion: false,
		IsStableVersion: true,
	}

	// build the full page
	page := UnitPage{
		BasePage:         basePage,
		Unit:             unit,
		Breadcrumb:       breadcrumb,
		Title:            "LLPkg",
		URLPath:          "/llpkg",
		CanonicalURLPath: "/llpkg",
		PageType:         "llpkg",
		PageLabels:       []string{"LLPkg"},
		CanShowDetails:   true,
		SelectedTab:      unitTabLookup[tabMain],
		Details:          mainDetails,
	}

	// render the full page
	s.servePage(ctx, w, "unit/main", page)
	return nil
}
