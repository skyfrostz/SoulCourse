package httpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

type openAPIDocument struct {
	OpenAPI    string                    `yaml:"openapi"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components openAPIComponents         `yaml:"components"`
}

type openAPIComponents struct {
	Schemas map[string]map[string]any `yaml:"schemas"`
}

func TestOpenAPIContractCoversRegisteredAPIRoutes(t *testing.T) {
	doc := loadOpenAPIContract(t)
	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("openapi version = %q, want 3.1.0", doc.OpenAPI)
	}

	registeredRoutes := map[string][]string{
		"/auth/email-verification-code": {"post"},
		"/auth/forgot-password":         {"post"},
		"/auth/register":                {"post"},
		"/auth/login":                   {"post"},
		"/auth/reset-password":          {"post"},
		"/auth/logout":                  {"post"},
		"/me":                           {"get", "delete"},
		"/me/profile":                   {"get", "put"},
		"/me/sessions":                  {"get"},
		"/me/sessions/{id}":             {"delete"},
		"/uploads/images/presign":       {"post"},
		"/uploads/images/{id}/object":   {"put"},
		"/uploads/images/{id}/complete": {"post"},
		"/profiles/{name}":              {"get"},
		"/notifications":                {"get"},
		"/notifications/read-all":       {"post"},
		"/notifications/{id}/read":      {"post"},
		"/messages":                     {"get", "post"},
		"/messages/{name}":              {"get"},
		"/taxonomy":                     {"get"},
		"/telemetry/web-vitals":         {"post"},
		"/content":                      {"get"},
		"/provinces":                    {"get"},
		"/policies":                     {"get"},
		"/requirements":                 {"get"},
		"/sources/{id}":                 {"get"},
		"/insights":                     {"get"},
		"/insights/{id}":                {"get"},
		"/topics":                       {"get"},
		"/topics/{slug}":                {"get"},
		"/posts":                        {"get", "post"},
		"/posts/{id}":                   {"get", "put", "delete"},
		"/posts/{id}/comments":          {"post"},
		"/posts/{id}/report":            {"post"},
		"/posts/{id}/like":              {"post"},
		"/posts/{id}/favorite":          {"post"},
		"/authors/{name}/follow":        {"post"},
		"/ai/choice-advice":             {"post"},
		"/admin/login":                  {"post"},
		"/admin/logout":                 {"post"},
		"/admin/email-config":           {"get"},
		"/admin/email-test":             {"post"},
		"/admin/content":                {"get", "post"},
		"/admin/content/{id}":           {"put", "delete"},
		"/admin/content/{id}/workflow":  {"post"},
		"/admin/uploads/images":         {"post"},
		"/admin/content-summary":        {"get"},
		"/admin/audit-logs":             {"get"},
		"/admin/reports":                {"get"},
		"/admin/reports/{id}/moderate":  {"post"},
		"/admin/users/{id}/ban":         {"post"},
		"/admin/users/{id}/restore":     {"post"},
		"/admin/users/{id}/password":    {"put"},
	}

	for path, methods := range registeredRoutes {
		pathItem, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("OpenAPI contract is missing path %s", path)
		}
		for _, method := range methods {
			if _, ok := pathItem[method]; !ok {
				t.Fatalf("OpenAPI contract is missing %s %s", strings.ToUpper(method), path)
			}
		}
	}
}

func TestOpenAPIContractRequiresRequestIDInEnvelopes(t *testing.T) {
	doc := loadOpenAPIContract(t)

	meta := doc.Components.Schemas["Meta"]
	properties := schemaProperties(t, meta, "Meta")
	requestID, ok := properties["requestId"].(map[string]any)
	if !ok {
		t.Fatal("Meta schema must define requestId")
	}
	if requestID["type"] != "string" {
		t.Fatalf("Meta.requestId type = %v, want string", requestID["type"])
	}

	apiError := doc.Components.Schemas["Error"]
	required, ok := apiError["required"].([]any)
	if !ok {
		t.Fatal("Error schema must define required fields")
	}
	for _, field := range []string{"code", "message", "requestId"} {
		if !containsString(required, field) {
			t.Fatalf("Error schema must require %s", field)
		}
	}
}

func TestOpenAPIContractDocumentsConversationPagination(t *testing.T) {
	doc := loadOpenAPIContract(t)
	operation := doc.Paths["/messages"]["get"].(map[string]any)
	parameters, ok := operation["parameters"].([]any)
	if !ok || len(parameters) != 2 {
		t.Fatalf("GET /messages parameters = %#v, want limit and cursor", operation["parameters"])
	}
	requireOpenAPIResponse(t, doc, "/messages", "get", "400")
}

func TestOpenAPIContractDocumentsSecurityMiddlewareResponses(t *testing.T) {
	doc := loadOpenAPIContract(t)

	rateLimitedOperations := map[string][]string{
		"/auth/email-verification-code": {"post"},
		"/auth/forgot-password":         {"post"},
		"/auth/register":                {"post"},
		"/auth/login":                   {"post"},
		"/auth/reset-password":          {"post"},
		"/auth/logout":                  {"post"},
		"/me":                           {"delete"},
		"/me/sessions/{id}":             {"delete"},
		"/uploads/images/presign":       {"post"},
		"/uploads/images/{id}/object":   {"put"},
		"/uploads/images/{id}/complete": {"post"},
		"/notifications/read-all":       {"post"},
		"/notifications/{id}/read":      {"post"},
		"/messages":                     {"post"},
		"/posts":                        {"post"},
		"/posts/{id}":                   {"put", "delete"},
		"/posts/{id}/comments":          {"post"},
		"/posts/{id}/report":            {"post"},
		"/posts/{id}/like":              {"post"},
		"/posts/{id}/favorite":          {"post"},
		"/authors/{name}/follow":        {"post"},
		"/ai/choice-advice":             {"post"},
		"/admin/login":                  {"post"},
		"/admin/logout":                 {"post"},
		"/admin/email-test":             {"post"},
		"/admin/content":                {"post"},
		"/admin/content/{id}":           {"put", "delete"},
		"/admin/content/{id}/workflow":  {"post"},
		"/admin/uploads/images":         {"post"},
		"/admin/reports/{id}/moderate":  {"post"},
		"/admin/users/{id}/ban":         {"post"},
		"/admin/users/{id}/restore":     {"post"},
		"/admin/users/{id}/password":    {"put"},
	}
	for path, methods := range rateLimitedOperations {
		for _, method := range methods {
			requireOpenAPIResponse(t, doc, path, method, "429")
		}
	}

	bodyLimitedOperations := map[string][]string{
		"/auth/email-verification-code": {"post"},
		"/auth/forgot-password":         {"post"},
		"/auth/register":                {"post"},
		"/auth/login":                   {"post"},
		"/auth/reset-password":          {"post"},
		"/uploads/images/presign":       {"post"},
		"/uploads/images/{id}/object":   {"put"},
		"/uploads/images/{id}/complete": {"post"},
		"/me/profile":                   {"put"},
		"/messages":                     {"post"},
		"/posts":                        {"post"},
		"/posts/{id}":                   {"put"},
		"/posts/{id}/comments":          {"post"},
		"/posts/{id}/report":            {"post"},
		"/ai/choice-advice":             {"post"},
		"/admin/login":                  {"post"},
		"/admin/logout":                 {"post"},
		"/admin/email-test":             {"post"},
		"/admin/content":                {"post"},
		"/admin/content/{id}":           {"put"},
		"/admin/content/{id}/workflow":  {"post"},
		"/admin/uploads/images":         {"post"},
		"/admin/reports/{id}/moderate":  {"post"},
		"/admin/users/{id}/ban":         {"post"},
		"/admin/users/{id}/restore":     {"post"},
		"/admin/users/{id}/password":    {"put"},
	}
	for path, methods := range bodyLimitedOperations {
		for _, method := range methods {
			requireOpenAPIResponse(t, doc, path, method, "413")
		}
	}
}

func loadOpenAPIContract(t *testing.T) openAPIDocument {
	t.Helper()

	path := filepath.Join("..", "..", "..", "docs", "openapi", "openapi.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OpenAPI contract: %v", err)
	}

	var doc openAPIDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse OpenAPI contract: %v", err)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("OpenAPI contract has no paths")
	}
	return doc
}

func schemaProperties(t *testing.T, schema map[string]any, name string) map[string]any {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("%s schema must define properties", name)
	}
	return properties
}

func requireOpenAPIResponse(t *testing.T, doc openAPIDocument, path string, method string, status string) {
	t.Helper()
	pathItem, ok := doc.Paths[path]
	if !ok {
		t.Fatalf("OpenAPI contract is missing path %s", path)
	}
	operation, ok := pathItem[method].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI contract is missing %s %s", strings.ToUpper(method), path)
	}
	responses, ok := operation["responses"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI contract is missing responses for %s %s", strings.ToUpper(method), path)
	}
	if _, ok := responses[status]; !ok {
		t.Fatalf("OpenAPI contract is missing %s response for %s %s", status, strings.ToUpper(method), path)
	}
}

func containsString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
