package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerateCommandUsesSelectedNamedSchema(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQC(t, workspace, "generate", "--schema", "main", "getUser")

	for _, want := range []string{
		"Schema:",
		"main",
		"Operation:",
		"query",
		"Field:",
		"getUser",
		"curl -X POST http://main.test/graphql",
		"Authorization: Bearer main-token",
		"query getUser($id: ID!)",
		"getUser(id: $id)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}

	for _, notWant := range []string{"apiPing", "http://api.test/graphql", "api-key"} {
		if strings.Contains(output, notWant) {
			t.Fatalf("output contains %q but selected schema was main:\n%s", notWant, output)
		}
	}
}

func TestGenerateCommandProcessesAllNamedSchemasByDefault(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQC(t, workspace, "generate")

	for _, want := range []string{
		"main",
		"getUser",
		"http://main.test/graphql",
		"api",
		"apiPing",
		"http://api.test/graphql",
		"X-API-Key: api-key",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func runGQC(t *testing.T, workspace string, args ...string) string {
	t.Helper()

	binaryPath := buildGQC(t)
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(),
		"MAIN_AUTH_TOKEN=main-token",
		"API_AUTH_TOKEN=api-token",
		"API_KEY=api-key",
		"NO_COLOR=1",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gqc %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}

	return string(output)
}

func buildGQC(t *testing.T) string {
	t.Helper()

	repoRoot := repoRoot(t)
	binaryPath := filepath.Join(t.TempDir(), "gqc")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/gqc")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOCACHE="+goCacheDir(t))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build gqc failed: %v\n%s", err, string(output))
	}

	return binaryPath
}

func goCacheDir(t *testing.T) string {
	t.Helper()

	if cacheDir := os.Getenv("GOCACHE"); cacheDir != "" {
		return cacheDir
	}

	cacheDir := filepath.Join(os.TempDir(), "gql-curl-test-gocache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir go cache %s: %v", cacheDir, err)
	}

	return cacheDir
}

func writeCLIWorkspace(t *testing.T) string {
	t.Helper()

	workspace := t.TempDir()
	mainSchemaDir := filepath.Join(workspace, "main-schema")
	apiSchemaDir := filepath.Join(workspace, "api-schema")
	mkdirAll(t, mainSchemaDir)
	mkdirAll(t, apiSchemaDir)

	writeFile(t, filepath.Join(mainSchemaDir, "schema.graphql"), `
type Query {
  getUser(id: ID!): User
}

type User {
  id: ID!
  name: String
}
`)
	writeFile(t, filepath.Join(apiSchemaDir, "schema.graphql"), `
type Query {
  apiPing: String
}
`)

	writeFile(t, filepath.Join(workspace, "graphql.curl.yaml"), `
schemas:
  main:
    path: "`+mainSchemaDir+`"
    endpoint: "http://main.test/graphql"
    auth_token: ${MAIN_AUTH_TOKEN}
    headers:
      Authorization: "Bearer {{auth_token}}"
  api:
    path: "`+apiSchemaDir+`"
    endpoint: "http://api.test/graphql"
    auth_token: ${API_AUTH_TOKEN}
    headers:
      Authorization: "Bearer {{auth_token}}"
      X-API-Key: ${API_KEY}
document_extensions: [".graphql", ".graphqls", ".gql"]
environment:
  MAX_DEPTH: 2
`)

	return workspace
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
