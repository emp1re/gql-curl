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

func TestGenerateCommandPrintsPostmanPayloadFormat(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQC(t, workspace, "generate", "--schema", "main", "getUser", "--format", "postman")

	for _, want := range []string{
		"Schema:",
		"main",
		`"query": "query getUser($id: ID!) {\n  getUser(id: $id) {\n    id\n    name\n  }\n}"`,
		`"variables": {`,
		`"id": "<ID>"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}

	if strings.Contains(output, "curl -X POST") {
		t.Fatalf("payload format should not print curl command:\n%s", output)
	}
}

func TestGenerateCommandPrintsPlaygroundFormatForMutation(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQC(t, workspace, "generate", "--schema", "main", "createUser", "--format", "playground")

	for _, want := range []string{
		"# Mutation",
		"mutation createUser($input: CreateUserInput!)",
		"createUser(input: $input)",
		"  createUser(input: $input) {",
		"    id",
		"    name",
		"# Variables",
		`"input": {`,
		`"name": "<String>"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}

	for _, notWant := range []string{"curl -X POST", `"query":`} {
		if strings.Contains(output, notWant) {
			t.Fatalf("playground format should not contain %q:\n%s", notWant, output)
		}
	}
}

func TestRootHelpIncludesCommonExamples(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQC(t, workspace, "--help")

	for _, want := range []string{
		"ready-to-copy requests for curl, Postman, and GraphQL Playground",
		"Examples:",
		"gqc generate createUser --format playground",
		"gqc generate getUser --format postman --vars",
		"gqc postman --schema main --file center.graphqls",
		"Available Commands:",
		"fetch",
		"generate",
		"postman",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestGenerateCommandErrorPrintsHelp(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQCExpectError(t, workspace, "generate", "--format", "unknown")

	for _, want := range []string{
		`Error: unknown output format "unknown"`,
		"Usage:",
		"gqc generate [operationName] [flags]",
		"Examples:",
		"gqc generate getUser --format postman",
		"Flags:",
		"--format string",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestFetchCommandErrorPrintsHelp(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQCExpectError(t, workspace, "fetch", "unexpected")

	for _, want := range []string{
		`Error: unknown argument "unexpected"`,
		"Usage:",
		"gqc fetch [flags]",
		"Examples:",
		"gqc fetch --schema main",
		"Flags:",
		"--schema string",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestPostmanCommandWritesSelectedSchemaFileCollection(t *testing.T) {
	workspace := writeCLIWorkspace(t)
	outPath := filepath.Join(workspace, "center.postman_collection.json")

	output := runGQC(t, workspace, "postman", "--schema", "main", "--file", "center.graphqls", "--out", outPath)
	if !strings.Contains(output, "Postman collection written to: "+outPath) {
		t.Fatalf("output does not report collection path:\n%s", output)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read collection: %v", err)
	}
	collection := string(data)

	for _, want := range []string{
		`"name": "main"`,
		`"name": "center.graphqls"`,
		`"name": "query getUser"`,
		`"name": "mutation createUser"`,
		`"url": "http://main.test/graphql"`,
		`"key": "Authorization"`,
		`"value": "Bearer main-token"`,
		`"key": "Content-Type"`,
		`"mode": "graphql"`,
		`"query": "query getUser($id: ID!)`,
		`\n  getUser(id: $id) {\n    id\n    name\n  }\n}`,
		`"variables": "{\n  \"id\": \"<ID>\"\n}"`,
	} {
		if !strings.Contains(collection, want) {
			t.Fatalf("collection does not contain %q:\n%s", want, collection)
		}
	}

	for _, notWant := range []string{"apiPing", "http://api.test/graphql", "api-key"} {
		if strings.Contains(collection, notWant) {
			t.Fatalf("collection contains %q but selected schema file was main/center.graphqls:\n%s", notWant, collection)
		}
	}
}

func TestPostmanCommandGeneratesAllSchemasByDefault(t *testing.T) {
	workspace := writeCLIWorkspace(t)
	outPath := filepath.Join(workspace, "all.postman_collection.json")

	runGQC(t, workspace, "postman", "--out", outPath)

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read collection: %v", err)
	}
	collection := string(data)

	for _, want := range []string{
		`"name": "GraphQL APIs"`,
		`"name": "main"`,
		`"name": "center.graphqls"`,
		`"name": "api"`,
		`"name": "api.graphql"`,
		`"name": "query apiPing"`,
		`"url": "http://api.test/graphql"`,
		`"key": "X-API-Key"`,
		`"value": "api-key"`,
	} {
		if !strings.Contains(collection, want) {
			t.Fatalf("collection does not contain %q:\n%s", want, collection)
		}
	}
}

func TestPostmanCommandErrorPrintsHelp(t *testing.T) {
	workspace := writeCLIWorkspace(t)

	output := runGQCExpectError(t, workspace, "postman", "--file", "missing.graphqls")

	for _, want := range []string{
		`Error: schema file "missing.graphqls" was not found in selected schema paths`,
		"Usage:",
		"gqc postman [flags]",
		"Examples:",
		"gqc postman --schema main --file center.graphqls",
		"Flags:",
		"--file string",
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
	cmd.Env = cliEnv()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gqc %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}

	return string(output)
}

func runGQCExpectError(t *testing.T, workspace string, args ...string) string {
	t.Helper()

	binaryPath := buildGQC(t)
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workspace
	cmd.Env = cliEnv()

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("gqc %s succeeded, want error\n%s", strings.Join(args, " "), string(output))
	}

	return string(output)
}

func cliEnv() []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		key := strings.SplitN(item, "=", 2)[0]
		switch key {
		case "MAIN_AUTH_TOKEN", "API_AUTH_TOKEN", "API_KEY":
			continue
		default:
			env = append(env, item)
		}
	}

	return append(env, "NO_COLOR=1")
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

	writeFile(t, filepath.Join(mainSchemaDir, "types.graphqls"), `
input CreateUserInput {
  name: String!
}

type User {
  id: ID!
  name: String
}
`)
	writeFile(t, filepath.Join(mainSchemaDir, "center.graphqls"), `
type Query {
  getUser(id: ID!): User
}

type Mutation {
  createUser(input: CreateUserInput!): User
}
`)
	writeFile(t, filepath.Join(apiSchemaDir, "api.graphql"), `
type Query {
  apiPing: String
}
`)
	writeFile(t, filepath.Join(workspace, ".env"), `
MAIN_AUTH_TOKEN=main-token
API_AUTH_TOKEN=api-token
API_KEY=api-key
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
