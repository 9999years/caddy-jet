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

package jet

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"path/filepath"

	"github.com/mholt/caddy"
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
				Root:       "/photos",
				View:       *jet.NewHTMLSet(filepath.Join(siteRoot, "/photos")),
			},
			{
				Extensions: []string{".html", ".htm"},
				IndexFiles: []string{"index.html", "index.htm"},
				Root:       "/images",
				View:       *jet.NewHTMLSet(filepath.Join(siteRoot, "/images")),
			},
		},
		SiteRoot:    siteRoot,
		BufPool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
	}

	tmplroot := JetTemplates{
		Next: staticfiles.FileServer{Root: http.Dir(siteRoot)},
		Rules: []Rule{
			{
				Extensions: []string{".html"},
				IndexFiles: []string{"index.html"},
				Root:       "/",
				View:       *jet.NewHTMLSet(filepath.Join(siteRoot, "/")),
			},
		},
		SiteRoot:    siteRoot,
		BufPool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
	}

	integration := JetTemplates{
		Next: staticfiles.FileServer{Root: http.Dir(siteRoot)},
		Rules: []Rule{
			{
				Extensions: defaultJetExtensions,
				IndexFiles: []string{"index.html", "index.jet"},
				Root:       "/",
				View:       *jet.NewHTMLSet(filepath.Join(siteRoot, "/")),
			},
		},
		SiteRoot: siteRoot,
		BufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
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

		{
			tpl:      integration,
			req:      "/root.html",
			respCode: http.StatusOK,
			res: `<!DOCTYPE html><html><head><title>root</title></head><body><h1>Header title</h1>
</body></html>
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

type ReqTest struct {
	t *testing.T
	rule string
	req string
	expect string
	code int
}

func testReq(test ReqTest) {
	req, err := http.NewRequest("GET", test.req, nil)
	if err != nil {
		test.t.Fatalf("Test: Could not create HTTP request to %v: %v",
			test.req, err)
	}
	controller := caddy.NewTestController("http", `localhost {
		root ./testdata
		errors stderr` +
		test.rule +
	"\n}")
	jetTemplates, err := NewJetTemplates(controller)
	if err != nil {
		test.t.Fatalf("Error setting up templates: %v", err)
	}
	resp := httptest.NewRecorder()

	jetTemplates.ServeHTTP(resp, req)

	res := resp.Result()

	if test.code != res.StatusCode {
		test.t.Fatalf("Test: Wrong response code for request %v: %d, should be %d", test.req, test.code, res.StatusCode)
	}

	respBody := resp.Body.String()
	if respBody != test.expect {
		test.t.Fatalf("Test %v: the expected body %v is different from the response one: %v", test.req, test.expect, respBody)
	}
}

func TestErr(t *testing.T) {
	testReq(ReqTest{
		t: t,
		rule: "jet",
		req: "/root.html",
		expect: `<!DOCTYPE html><html><head><title>root</title></head><body><h1>Header title</h1>
</body></html>
`,
		code: http.StatusOK,
	})
}
