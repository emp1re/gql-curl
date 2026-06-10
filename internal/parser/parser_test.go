package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emp1re/gql-curl/internal/config"
)

func TestNewParserFromPathsLoadsMultipleDirectoriesAndFiles(t *testing.T) {
	restoreParserDocumentExtensions(t)
	config.DocumentExtensions = []string{".graphql", ".graphqls"}

	root := t.TempDir()
	queryDir := filepath.Join(root, "query")
	mutationDir := filepath.Join(root, "mutation")

	mkdirAll(t, queryDir)
	mkdirAll(t, mutationDir)

	writeFile(t, filepath.Join(queryDir, "query.graphql"), `
type Query {
  ping: String
}
`)
	writeFile(t, filepath.Join(mutationDir, "mutation.graphqls"), `
type Mutation {
  createUser(input: CreateUserInput!): Boolean
}

input CreateUserInput {
  id: ID!
}
`)
	writeFile(t, filepath.Join(mutationDir, "ignored.txt"), `this is not GraphQL SDL`)

	gql, err := NewParserFromPaths([]string{
		queryDir,
		filepath.Join(mutationDir, "mutation.graphqls"),
	})
	if err != nil {
		t.Fatalf("NewParserFromPaths returned error: %v", err)
	}

	if gql.Schema.Query == nil || gql.Schema.Query.Fields.ForName("ping") == nil {
		t.Fatalf("query field ping was not loaded")
	}
	if gql.Schema.Mutation == nil || gql.Schema.Mutation.Fields.ForName("createUser") == nil {
		t.Fatalf("mutation field createUser was not loaded")
	}
}

func TestNewParserFromPathsRejectsEmptyPaths(t *testing.T) {
	if _, err := NewParserFromPaths(nil); err == nil {
		t.Fatalf("NewParserFromPaths(nil) returned nil error")
	}
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

func restoreParserDocumentExtensions(t *testing.T) {
	t.Helper()

	previous := append([]string(nil), config.DocumentExtensions...)
	t.Cleanup(func() {
		config.DocumentExtensions = previous
	})
}
