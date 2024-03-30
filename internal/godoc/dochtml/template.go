// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dochtml

import (
	"fmt"
	"go/doc"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/safehtml"
	"github.com/google/safehtml/template"
	"golang.org/x/pkgsite/internal/godoc/dochtml/internal/render"
)

var (
	loadOnce sync.Once

	// TODO(golang.org/issue/5060): finalize URL scheme and design for notes,
	// then it becomes more viable to factor out inline CSS style.
	bodyTemplate, outlineTemplate, sidenavTemplate *template.Template
)
var (
	overloadFuncIndexPattern = regexp.MustCompile(`^overload_func_index:(\d+)\n`)
)

func Templates() []*template.Template {
	return []*template.Template{bodyTemplate, outlineTemplate, sidenavTemplate}
}

// LoadTemplates reads and parses the templates used to generate documentation.
func LoadTemplates(fsys template.TrustedFS) {
	const dir = "doc"
	loadOnce.Do(func() {
		bodyTemplate = template.Must(template.New("body.tmpl").
			Funcs(tmpl).
			ParseFS(fsys,
				path.Join(dir, "body.tmpl"),
				path.Join(dir, "declaration.tmpl"),
				path.Join(dir, "example.tmpl")))
		outlineTemplate = template.Must(template.New("outline.tmpl").
			Funcs(tmpl).
			ParseFS(fsys, path.Join(dir, "outline.tmpl")))
		sidenavTemplate = template.Must(template.New("sidenav-mobile.tmpl").
			Funcs(tmpl).
			ParseFS(fsys, path.Join(dir, "sidenav-mobile.tmpl")))
	})
}

func overloadFuncIndex(item *item) int {
	if item.Kind != "method" && item.Kind != "function" {
		return -1
	}
	match := overloadFuncIndexPattern.FindStringSubmatch(item.Doc)
	if len(match) == 2 {
		valueStr := match[1]
		if intValue, err := strconv.Atoi(valueStr); err == nil {
			return intValue
		}
	}
	return -1
}

var tmpl = map[string]any{
	"ternary": func(q, a, b any) any {
		v := reflect.ValueOf(q)
		vz := reflect.New(v.Type()).Elem()
		if reflect.DeepEqual(v.Interface(), vz.Interface()) {
			return b
		}
		return a
	},
	// These are just placeholders, for parsing. The actual functions
	// are in dochtml.go.
	"render_short_synopsis":    (*render.Renderer)(nil).ShortSynopsis,
	"render_synopsis":          (*render.Renderer)(nil).Synopsis,
	"render_doc":               (*render.Renderer)(nil).DocHTML,
	"render_doc_extract_links": (*render.Renderer)(nil).DocHTMLExtractLinks,
	"render_decl":              (*render.Renderer)(nil).DeclHTML,
	"render_code":              (*render.Renderer)(nil).CodeHTML,
	"file_link":                func() string { return "" },
	"source_link":              func(string, any) string { return "" },
	"since_version":            func(string) safehtml.HTML { return safehtml.HTML{} },
	"play_url":                 func(*doc.Example) string { return "" },
	"safe_id":                  render.SafeGoID,
	"render_function_id": func(item *item) string {
		fullName := item.Name
		overloadIndex := overloadFuncIndex(item)
		if item.FullName != "" {
			fullName = item.FullName
		}
		if overloadIndex != -1 {
			return fmt.Sprintf("%s__%d", fullName, overloadIndex)
		} else {
			return fullName
		}
	},
	"render_actual_doc": func(item *item) string {
		if item.Kind != "method" && item.Kind != "function" {
			return item.Doc
		}
		doc := item.Doc
		doc = overloadFuncIndexPattern.ReplaceAllString(doc, "")
		doc = strings.TrimSpace(doc)
		return doc
	},
}
