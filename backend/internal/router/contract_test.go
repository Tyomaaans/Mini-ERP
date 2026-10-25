package router_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"mini-erp/docs"
)

type responseContract struct {
	Method  string
	Path    string
	Match   *regexp.Regexp
	Schemas map[int]*jsonschema.Schema
}

type apiContract struct{ Operations []responseContract }

func pointer(value string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(value)
}

func newContract(t *testing.T) *apiContract {
	t.Helper()
	var document map[string]any
	if err := yaml.Unmarshal(docs.Spec, &document); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	if document["openapi"] != "3.1.0" {
		t.Fatal("API contract must use OpenAPI 3.1")
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	resource := "https://example.test/openapi.json"
	if err := compiler.AddResource(resource, document); err != nil {
		t.Fatal(err)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	for name := range components {
		if _, err := compiler.Compile(resource + "#/components/schemas/" + pointer(name)); err != nil {
			t.Fatalf("schema %s: %v", name, err)
		}
	}
	contract := &apiContract{}
	for path, definition := range document["paths"].(map[string]any) {
		for method, value := range definition.(map[string]any) {
			operation := value.(map[string]any)
			pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(path), "\\{id\\}", "[^/]+") + "$"
			entry := responseContract{
				Method:  strings.ToUpper(method),
				Path:    path,
				Match:   regexp.MustCompile(pattern),
				Schemas: map[int]*jsonschema.Schema{},
			}
			for code, value := range operation["responses"].(map[string]any) {
				response := value.(map[string]any)
				content, ok := response["content"].(map[string]any)
				if !ok || content["application/json"] == nil {
					continue
				}
				var status int
				if _, err := fmt.Sscanf(code, "%d", &status); err != nil {
					t.Fatal(err)
				}
				schema, err := compiler.Compile(resource + "#/paths/" + pointer(path) + "/" + method + "/responses/" + code + "/content/application~1json/schema")
				if err != nil {
					t.Fatalf("%s %s %s: %v", method, path, code, err)
				}
				entry.Schemas[status] = schema
			}
			contract.Operations = append(contract.Operations, entry)
		}
	}
	return contract
}

func (c *apiContract) Validate(method, path string, status int, body any) error {
	path = strings.SplitN(path, "?", 2)[0]
	for _, operation := range c.Operations {
		if operation.Method == method && operation.Match.MatchString(path) {
			if schema := operation.Schemas[status]; schema != nil {
				if err := schema.Validate(body); err != nil {
					return fmt.Errorf("%s %s response %d: %w", method, path, status, err)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("%s %s has no documented operation", method, path)
}

func (c *apiContract) RequireRoutes(t *testing.T, engine *gin.Engine) {
	t.Helper()
	actual := map[string]bool{}
	for _, route := range engine.Routes() {
		actual[route.Method+" "+strings.ReplaceAll(strings.TrimPrefix(route.Path, "/api/v1"), ":id", "{id}")] = true
	}
	for _, operation := range c.Operations {
		key := operation.Method + " " + operation.Path
		if !actual[key] {
			t.Errorf("documented operation has no route: %s", key)
		}
		delete(actual, key)
	}
	for key := range actual {
		t.Errorf("route has no OpenAPI operation: %s", key)
	}
}

func TestOpenAPIContractCompiles(t *testing.T) {
	_ = newContract(t)
}
