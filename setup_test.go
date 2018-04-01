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
	"fmt"
	"testing"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func TestSetup(t *testing.T) {
	c := caddy.NewTestController("http", `templates`)
	err := setup(c)
	if err != nil {
		t.Errorf("Expected no errors, got: %v", err)
	}
	mids := httpserver.GetConfig(c).Middleware()
	if len(mids) == 0 {
		t.Fatal("Expected middleware, got 0 instead")
	}

	handler := mids[0](httpserver.EmptyNext)
	myHandler, ok := handler.(JetTemplates)

	if !ok {
		t.Fatalf("Expected handler to be type JetTemplates, got: %#v", handler)
	}

	if myHandler.Rules[0].Root != defaultJetPath {
		t.Errorf("Expected / as the default Root")
	}
	if fmt.Sprint(myHandler.Rules[0].Extensions) != fmt.Sprint(defaultJetExtensions) {
		t.Errorf("Expected %v to be the Default Extensions", defaultJetExtensions)
	}
	var indexFiles []string
	for _, extension := range defaultJetExtensions {
		indexFiles = append(indexFiles, "index"+extension)
	}
	if fmt.Sprint(myHandler.Rules[0].IndexFiles) != fmt.Sprint(indexFiles) {
		t.Errorf("Expected %v to be the Default Index files", indexFiles)
	}
}

func TestTemplatesParse(t *testing.T) {
	tests := []struct {
		inputTemplateConfig    string
		shouldErr              bool
		expectedTemplateConfig []Rule
	}{
		{`jet`, false, []Rule{{
			Root:       defaultJetPath,
			Extensions: defaultJetExtensions,
		}}},
		{`jet /api1`, false, []Rule{{
			Root:       "/api1",
			Extensions: defaultJetExtensions,
		}}},
		{`jet /api2 .txt .htm`, false, []Rule{{
			Root:       "/api2",
			Extensions: []string{".txt", ".htm"},
		}}},

		{`jet /api3 .htm .html
		  jet /api4 .txt .tpl `, false, []Rule{{
			Root:       "/api3",
			Extensions: []string{".htm", ".html"},
		}, {
			Root:       "/api4",
			Extensions: []string{".txt", ".tpl"},
		}}},
		{`jet {
				path /api5
				ext .html
			}`, false, []Rule{{
			Root:       "/api5",
			Extensions: []string{".html"},
		}}},
	}
	for i, test := range tests {
		c := caddy.NewTestController("http", test.inputTemplateConfig)
		actualTemplateConfigs, err := jetParse(c)

		if err == nil && test.shouldErr {
			t.Errorf("Test %d didn't error, but it should have", i)
		} else if err != nil && !test.shouldErr {
			t.Errorf("Test %d errored, but it shouldn't have; got '%v'", i, err)
		}
		if len(actualTemplateConfigs) != len(test.expectedTemplateConfig) {
			t.Fatalf("Test %d expected %d no of JetTemplate configs, but got %d ",
				i, len(test.expectedTemplateConfig), len(actualTemplateConfigs))
		}
		for j, actualTemplateConfig := range actualTemplateConfigs {
			if actualTemplateConfig.Root != test.expectedTemplateConfig[j].Root {
				t.Errorf("Test %d expected %dth JetTemplate Config Root to be  %s  , but got %s",
					i, j, test.expectedTemplateConfig[j].Root, actualTemplateConfig.Root)
			}
			if fmt.Sprint(actualTemplateConfig.Extensions) != fmt.Sprint(test.expectedTemplateConfig[j].Extensions) {
				t.Errorf("Expected %v to be the  Extensions , but got %v instead", test.expectedTemplateConfig[j].Extensions, actualTemplateConfig.Extensions)
			}
		}
	}

}
