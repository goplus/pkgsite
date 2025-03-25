// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frontend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/safehtml/template"
	"golang.org/x/pkgsite/internal/derrors"
	"golang.org/x/pkgsite/internal/llpkg"

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

// serveLLPkg 处理 /llpkg 路径的请求，显示 LLPkg 包的目录列表
func (s *Server) serveLLPkg(w http.ResponseWriter, r *http.Request, ds internal.DataSource) (err error) {
	defer derrors.Wrap(&err, "serveLLPkg(ctx, w, r)")

	// 1. 创建基础页面
	basePage := s.newBasePage(r, "LLPkg")
	basePage.AllowWideContent = true

	// 2. 构建面包屑导航，添加 LLPkg 部分
	breadcrumb := breadcrumb{
		Links: []link{
			{Href: "/", Body: "Discover Packages"},
		},
		Current:  "LLPkg",
		CopyData: "llpkg",
	}

	// 3. 定义 LLPkg 数据
	llpkgData := map[string]struct {
		Versions []struct {
			C  string   `json:"c"`
			Go []string `json:"go"`
		} `json:"versions"`
	}{
		"cjson": {
			Versions: []struct {
				C  string   `json:"c"`
				Go []string `json:"go"`
			}{
				{C: "1.3", Go: []string{"v0.1.0", "v0.1.1"}},
				{C: "1.3.1", Go: []string{"v1.1.0"}},
			},
		},
		"bjson": {
			Versions: []struct {
				C  string   `json:"c"`
				Go []string `json:"go"`
			}{
				{C: "1.2", Go: []string{"v0.1.0", "v0.1.1"}},
				{C: "1.2.1", Go: []string{"v1.1.0"}},
			},
		},
	}

	// 4. 构建目录信息
	var subdirectories []*DirectoryInfo
	for packageName, packageInfo := range llpkgData {
		// 获取最新版本信息
		latestCVersion := ""
		latestGoVersion := ""
		for _, v := range packageInfo.Versions {
			for _, goVer := range v.Go {
				if compareVersions(goVer, latestGoVersion) > 0 {
					latestGoVersion = goVer
				}
			}
		}
		for _, v := range packageInfo.Versions {
			if compareVersions(v.C, latestCVersion) > 0 {
				latestCVersion = v.C
			}
		}

		// 构建目录项
		subdirectories = append(subdirectories, &DirectoryInfo{
			Suffix:   packageName,
			URL:      "/github.com/goplus/llpkg/" + packageName,
			Synopsis: fmt.Sprintf("C:%s -> Go:%s", latestCVersion, latestGoVersion),
			IsModule: false,
		})
	}

	// 排序目录项以确保显示顺序一致
	sort.Slice(subdirectories, func(i, j int) bool {
		return subdirectories[i].Suffix < subdirectories[j].Suffix
	})

	// 5. 创建目录结构
	directories := []*Directory{
		{
			Prefix:         "",
			Root:           nil,
			Subdirectories: subdirectories,
		},
	}

	// 6. 创建详情对象，只包含目录部分
	emptyHTML := template.MustParseAndExecuteToHTML("")
	mainDetails := &MainDetails{
		Directories: directories,
		Readme:      emptyHTML,
		DocBody:     emptyHTML,
	}

	// 7. 创建完整页面
	page := UnitPage{
		BasePage:         basePage,
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

	// 8. 渲染页面
	s.servePage(r.Context(), w, "unit", page)
	return nil
}

// 辅助函数: 找出最新版本
func findLatestVersion(versions []struct {
	C  string
	Go []string
},
	getVersion func(v struct {
		C  string
		Go []string
	}) string) string {

	if len(versions) == 0 {
		return ""
	}

	latest := getVersion(versions[0])
	for _, v := range versions {
		ver := getVersion(v)
		if compareVersions(ver, latest) > 0 {
			latest = ver
		}
	}
	return latest
}

// 辅助函数: 比较版本号
func compareVersions(v1, v2 string) int {
	if v2 == "" {
		return 1 // 任何版本都大于空版本
	}

	// 去掉版本前缀 'v'
	v1Clean := strings.TrimPrefix(v1, "v")
	v2Clean := strings.TrimPrefix(v2, "v")

	// 分割版本号
	parts1 := strings.Split(v1Clean, ".")
	parts2 := strings.Split(v2Clean, ".")

	// 比较每个部分
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}
