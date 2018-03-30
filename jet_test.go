// Copyright 2015 Light Code Labs, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jettemplates

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"path/filepath"

	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/mholt/caddy/caddyhttp/staticfiles"

	"github.com/CloudyKit/jet"
)

func TestTemplates(t *testing.T) {
	siteRoot := "./testdata"
	tmpl := JetTemplates{
		Next: staticfiles.FileServer{Root: http.Dir(siteRoot)},
		Rules: []Rule{
			{
				Extensions: []string{".html"},
				IndexFiles: []string{"index.html"},
				Path:       "/photos",
				View:       jet.NewHTMLSet(filepath.Join(siteRoot, "/photos")),
			},
			{
				Extensions: []string{".html", ".htm"},
				IndexFiles: []string{"index.html", "index.htm"},
				Path:       "/images",
				View:       jet.NewHTMLSet(filepath.Join(siteRoot, "/images")),
			},
		},
		Root:    siteRoot,
		FileSys: http.Dir(siteRoot),
		BufPool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
	}

	tmplroot := JetTemplates{
		Next: staticfiles.FileServer{Root: http.Dir(siteRoot)},
		Rules: []Rule{
			{
				Extensions: []string{".html"},
				IndexFiles: []string{"index.html"},
				Path:       "/",
				View:       jet.NewHTMLSet(filepath.Join(siteRoot, "/")),
			},
		},
		Root:    siteRoot,
		FileSys: http.Dir(siteRoot),
		BufPool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
	}

	// register custom function which is used in template
	httpserver.TemplateFuncs["root"] = func() string { return "root" }

	for _, c := range []struct {
		tpl      JetTemplates
		req      string
		respCode int
		res      string
	}{
		{
			tpl:      tmpl,
			req:      "/photos/test.html",
			respCode: http.StatusOK,
			res: `<!DOCTYPE html><html><head><title>example title</title></head><body>body</body></html>
`,
		},

		{
			tpl:      tmpl,
			req:      "/images/img.htm",
			respCode: http.StatusOK,
			res: `<!DOCTYPE html><html><head><title>img</title></head><body><h1>Header title</h1>
</body></html>
`,
		},

		{
			tpl:      tmplroot,
			req:      "/root.html",
			respCode: http.StatusOK,
			res: `<!DOCTYPE html><html><head><title>root</title></head><body><h1>Header title</h1>
</body></html>
`,
		},

		{
			tpl:      tmplroot,
			req:      "/malformed.html",
			respCode: http.StatusInternalServerError,
			res: ``,
		},

		{
			tpl:      tmplroot,
			req:      "/syntax_error.html",
			respCode: http.StatusInternalServerError,
			res: ``,
		},

		// test extension filter
		{
			tpl:      tmplroot,
			req:      "/as_it_is.txt",
			respCode: http.StatusOK,
			res: `<!DOCTYPE html><html><head><title>as it is</title></head><body>{{include "header.html"}}</body></html>
`,
		},
	} {
		c := c
		t.Run("", func(t *testing.T) {
			req, err := http.NewRequest("GET", c.req, nil)
			if err != nil {
				t.Fatalf("Test: Could not create HTTP request: %v", err)
			}
			req = req.WithContext(context.WithValue(req.Context(), httpserver.OriginalURLCtxKey, *req.URL))

			rec := httptest.NewRecorder()
			println("response: ", rec.Body.String())

			c.tpl.ServeHTTP(rec, req)

			if rec.Code != c.respCode {
				t.Fatalf("Test: Wrong response code for request %v: %d, should be %d", c.req, rec.Code, c.respCode)
			}

			respBody := rec.Body.String()
			if respBody != c.res {
				t.Fatalf("Test %v: the expected body %v is different from the response one: %v", c.req, c.res, respBody)
			}
		})
	}
}
