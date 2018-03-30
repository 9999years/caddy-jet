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
	"net/http"
	"sync"
	"path/filepath"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"

	ckjet "github.com/CloudyKit/jet"
)

func init() {
	caddy.RegisterPlugin("jet", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// setup configures a new Templates middleware instance.
func setup(c *caddy.Controller) error {
	rules, err := jetParse(c)
	if err != nil {
		return err
	}

	cfg := httpserver.GetConfig(c)

	tmpls := JetTemplates{
		Rules:   rules,
		Root:    cfg.Root,
		FileSys: http.Dir(cfg.Root),
		BufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}

	cfg.AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		tmpls.Next = next
		return tmpls
	})

	return nil
}

func (r Rule) initView(cfg *httpserver.SiteConfig) *ckjet.Set {
	return ckjet.NewHTMLSet(filepath.Join(cfg.Root, r.Path))
}

func jetParse(c *caddy.Controller) ([]Rule, error) {
	var rules []Rule
	cfg := httpserver.GetConfig(c)

	for c.Next() {
		var rule Rule

		rule.Path = defaultJetPath
		rule.Extensions = defaultJetExtensions

		args := c.RemainingArgs()

		switch len(args) {
		case 0:
			// Optional block
			for c.NextBlock() {
				switch c.Val() {
				case "path":
					args := c.RemainingArgs()
					if len(args) != 1 {
						return nil, c.ArgErr()
					}
					rule.Path = args[0]

				case "ext":
					args := c.RemainingArgs()
					if len(args) == 0 {
						return nil, c.ArgErr()
					}
					rule.Extensions = args
				}
			}
		default:
			// First argument would be the path
			rule.Path = args[0]

			// Any remaining arguments are extensions
			rule.Extensions = args[1:]
			if len(rule.Extensions) == 0 {
				rule.Extensions = defaultJetExtensions
			}
		}

		for _, ext := range rule.Extensions {
			rule.IndexFiles = append(rule.IndexFiles, "index"+ext)
		}

		rule.initView(cfg)

		rules = append(rules, rule)
	}
	return rules, nil
}

const defaultJetPath = "/"
var defaultJetExtensions = []string{".html", ".htm", ".jet"}
