package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigAcceptsNamedSchemas(t *testing.T) {
	restoreGlobals(t)
	t.Setenv("MAIN_AUTH_TOKEN", "main-token")
	t.Setenv("API_AUTH_TOKEN", "api-token")
	t.Setenv("API_KEY", "api-key")

	configPath := writeConfig(t, `
schemas:
  main:
    path: "/gql"
    endpoint: "http://localhost:8080/gql/query"
    auth_token: ${MAIN_AUTH_TOKEN}
    headers:
      Authorization: "Bearer {{auth_token}}"
  api:
    path: "./api/gql/"
    endpoint: "http://api.service:8080/query"
    auth_token: ${API_AUTH_TOKEN}
    headers:
      Authorization: "Bearer {{auth_token}}"
      X-API-Key: ${API_KEY}
document_extensions: [".graphql", ".graphqls", ".gql"]
environment:
  MAX_DEPTH: 3
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if got, want := cfg.SchemaNames(), []string{"api", "main"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SchemaNames = %v, want %v", got, want)
	}
	if got, want := []string(cfg.Schemas["main"].Path), []string{"/gql"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("main.path = %v, want %v", got, want)
	}
	if got, want := cfg.Schemas["main"].Headers["Authorization"], "Bearer main-token"; got != want {
		t.Fatalf("main Authorization = %q, want %q", got, want)
	}
	if got, want := cfg.Schemas["api"].Headers["Authorization"], "Bearer api-token"; got != want {
		t.Fatalf("api Authorization = %q, want %q", got, want)
	}
	if got, want := cfg.Schemas["api"].Headers["X-API-Key"], "api-key"; got != want {
		t.Fatalf("api X-API-Key = %q, want %q", got, want)
	}
	if got, want := DocumentExtensions, []string{".graphql", ".graphqls", ".gql"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DocumentExtensions = %v, want %v", got, want)
	}
	if got, want := MaxDepth, 3; got != want {
		t.Fatalf("MaxDepth = %d, want %d", got, want)
	}
}

func TestLoadConfigRequiresSchemas(t *testing.T) {
	restoreGlobals(t)

	configPath := writeConfig(t, `
document_extensions: [".graphql"]
`)

	if _, err := LoadConfig(configPath); err == nil {
		t.Fatalf("LoadConfig returned nil error")
	}
}

func TestLoadConfigAcceptsSchemaPathListAndDefaultExtensions(t *testing.T) {
	restoreGlobals(t)

	configPath := writeConfig(t, `
schemas:
  main:
    path:
      - "./schema"
      - "./shared/schema.graphql"
    endpoint: "http://localhost:8080/gql/query"
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if got, want := []string(cfg.Schemas["main"].Path), []string{"./schema", "./shared/schema.graphql"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("main.path = %v, want %v", got, want)
	}
	if got, want := DocumentExtensions, DefaultDocumentExtensions; !reflect.DeepEqual(got, want) {
		t.Fatalf("DocumentExtensions = %v, want %v", got, want)
	}
}

func TestSelectedSchemasCanPickOneSchema(t *testing.T) {
	cfg := &Config{
		Schemas: map[string]SchemaConfig{
			"main": {Path: StringList{"/gql"}, Endpoint: "http://localhost:8080/gql/query"},
			"api":  {Path: StringList{"./api/gql/"}, Endpoint: "http://api.service:8080/query"},
		},
	}

	selected, err := cfg.SelectedSchemas("main")
	if err != nil {
		t.Fatalf("SelectedSchemas returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].Name != "main" {
		t.Fatalf("SelectedSchemas returned %v, want only main", selected)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "graphql.curl.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return path
}

func restoreGlobals(t *testing.T) {
	t.Helper()

	previousExtensions := append([]string(nil), DocumentExtensions...)
	previousMaxDepth := MaxDepth
	t.Cleanup(func() {
		DocumentExtensions = previousExtensions
		MaxDepth = previousMaxDepth
	})
}
