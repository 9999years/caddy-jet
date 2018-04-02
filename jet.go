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

// Package jettemplates implements CloudyKit Jet template execution for files
// to be dynamically rendered for the client.
package jet

import (
	"bytes"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"fmt"

	"github.com/mholt/caddy/caddyhttp/httpserver"

	"github.com/CloudyKit/jet"
)

// JetTemplates is middleware to render templated files as the HTTP response.
type JetTemplates struct {
	Next     httpserver.Handler // next handler
	SiteRoot string
	Rules    []Rule
	BufPool  *sync.Pool // docs: "A Pool must not be copied after first use."
}

// Rule represents a jet rule. A template will only execute
// with this rule if the request path matches the Root path specified
// and requests a resource with one of the extensions specified.
type Rule struct {
	// root path for the rule; nothing outside of here is loaded
	Root       string
	// extensions to render
	Extensions []string
	// index files to try
	IndexFiles []string
	// template loader, essentially
	View       jet.Set
}

// ServeHTTP implements the httpserver.Handler interface.
func (t JetTemplates) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	// iterate rules, to find first one that matches the request path
	for _, rule := range t.Rules {
		if !httpserver.Path(r.URL.Path).Matches(rule.Root) {
			continue
		}

		fpath := r.URL.Path

		fmt.Printf("rule %v matches path %v\n", rule.Root, fpath)

		// get a buffer from the pool and make a response recorder
		buf := t.BufPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer t.BufPool.Put(buf)

		// only buffer the response when we want to execute a template
		shouldBuf := func(status int, header http.Header) bool {
			// see if this request matches a template extension
			reqExt := filepath.Ext(fpath)
			for _, ext := range rule.Extensions {
				if reqExt == "" {
					// request has no extension, so check response Content-Type
					ct := mime.TypeByExtension(ext)
					if ct != "" && strings.Contains(header.Get("Content-Type"), ct) {
						return true
					}
				} else if reqExt == ext {
					return true
				}
			}
			return false
		}

		// prepare a buffer to hold the response, if applicable
		rb := httpserver.NewResponseBuffer(buf, w, shouldBuf)

		// pass request up the chain to let another middleware provide us the template
		//println("passing req. down chain")
		//code, err := t.Next.ServeHTTP(rb, r)
		//fmt.Printf("%v", rb.Buffer)
		//if !rb.Buffered() || code >= 300 || err != nil {
			//return code, err
		//}
		//println("continuing")

		// create a new template
		templatePath := filepath.ToSlash(httpserver.SafePath(t.SiteRoot, fpath))
		fmt.Printf("\nDIAG: templatePath = %v\n\t(= %v + %v)\n",
			templatePath, t.SiteRoot, fpath)
		fmt.Printf("diagnostic: rendering jet tpl!\n" +
			"root path: %s\nreq path: %s\nloading %s\n",
			t.SiteRoot, fpath, templatePath)
		tpl, err := rule.View.GetTemplate(templatePath)
		println("template gotten")
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// add custom functions
		//tpl.Funcs(httpserver.TemplateFuncs)

		// parse the template
		//parsedTpl, err := tpl.Parse(rb.Buffer.String())
		//if err != nil {
			//return http.StatusInternalServerError, err
		//}

		// create execution context for the template template
		ctx := httpserver.NewContextWithHeader(w.Header())
		ctx.Root = http.Dir(httpserver.SafePath(t.SiteRoot, rule.Root))
		ctx.Req = r
		ctx.URL = r.URL

		// execute the template
		buf.Reset()
		// TODO: vars
		err = tpl.Execute(buf, nil, ctx)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// copy the buffered header into the real ResponseWriter
		rb.CopyHeader()

		// set the actual content length now that the template was executed
		w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

		// get the modification time in preparation for http.ServeContent
		modTime, _ := time.Parse(http.TimeFormat, w.Header().Get("Last-Modified"))

		// at last, write the rendered template to the response; make sure to use
		// use the proper status code, since ServeContent hard-codes 2xx codes...
		http.ServeContent(rb.StatusCodeWriter(w), r, templatePath, modTime, bytes.NewReader(buf.Bytes()))

		return 0, nil
	}

	return t.Next.ServeHTTP(w, r)
}
