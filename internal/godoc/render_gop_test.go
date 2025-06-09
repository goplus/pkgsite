package godoc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/pkgsite/internal"
	"golang.org/x/pkgsite/internal/godoc/dochtml"
	"golang.org/x/pkgsite/internal/source"
)

func TestGopRender(t *testing.T) {
	dochtml.LoadTemplates(templateFS)
	ctx := context.Background()
	p, err := packageForDir(filepath.Join("testdata", "gop"), false)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := p.Render(ctx, "", &source.Info{}, &dochtml.ModuleInfo{}, nil, internal.BuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	htmlDoc, err := html.Parse(strings.NewReader(parts.Body.String()))
	if err != nil {
		t.Fatal(err)
	}
	wantFuncAnchor := []string{"Mul", "Mul__1", "Mul__2", "MulFloat", "MulInt"}
	for _, anchor := range wantFuncAnchor {
		t.Run(anchor, func(t *testing.T) {
			checker := in("h4.Documentation-functionHeader#" + anchor)
			if err := checker(htmlDoc); err != nil {
				t.Fatal(err)
			}
		})
	}
	// TODO(zzy): check overload method (current htmlcheck.Checker cant select id like "foo.Division")
}
