package generator

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"strings"
	"testing"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestGenerateCurlBuildsPayloadAndHeaders(t *testing.T) {
	restoreGeneratorMaxDepth(t)
	config.MaxDepth = 2

	gen, schema := newTestGenerator(t, "http://example.test/graphql", map[string]string{
		"Authorization": "Bearer token",
		"X-Trace":       "trace-id",
	})
	field := schema.Query.Fields.ForName("user")

	curl := gen.GenerateCurl("query", field, map[string]interface{}{
		"id": "42",
		"filter": map[string]interface{}{
			"active": true,
		},
	})

	for _, want := range []string{
		"curl -X POST http://example.test/graphql",
		"-H 'Authorization: Bearer token'",
		"-H 'X-Trace: trace-id'",
		"-H 'Content-Type: application/json'",
	} {
		if !strings.Contains(curl, want) {
			t.Fatalf("generated curl does not contain %q:\n%s", want, curl)
		}
	}

	payload := decodeCurlPayload(t, curl)
	if got, want := normalizeSpace(payload.Query), "query user($id: ID!, $filter: UserFilter) { user(id: $id, filter: $filter) { id name profile { bio } role } }"; got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
	if got, want := payload.Variables["id"], "42"; got != want {
		t.Fatalf("variables.id = %v, want %v", got, want)
	}

	filter, ok := payload.Variables["filter"].(map[string]interface{})
	if !ok {
		t.Fatalf("variables.filter = %#v, want object", payload.Variables["filter"])
	}
	if got, want := filter["active"], true; got != want {
		t.Fatalf("variables.filter.active = %v, want %v", got, want)
	}
}

func TestGenerateCurlDoesNotDuplicateConfiguredContentType(t *testing.T) {
	gen, schema := newTestGenerator(t, "http://example.test/graphql", map[string]string{
		"content-type": "application/graphql+json",
	})

	curl := gen.GenerateCurl("query", schema.Query.Fields.ForName("ping"), nil)
	if got, want := strings.Count(strings.ToLower(curl), "content-type:"), 1; got != want {
		t.Fatalf("Content-Type header count = %d, want %d:\n%s", got, want, curl)
	}
	if !strings.Contains(curl, "-H 'content-type: application/graphql+json'") {
		t.Fatalf("generated curl does not include configured content type:\n%s", curl)
	}
}

func TestBuildVariablesSkeletonUsesGraphQLDefaults(t *testing.T) {
	gen, schema := newTestGenerator(t, "http://example.test/graphql", nil)
	field := schema.Query.Fields.ForName("user")

	vars := gen.buildVariablesSkeleton(field)
	if got, want := vars["id"], "<ID>"; got != want {
		t.Fatalf("id default = %#v, want %#v", got, want)
	}

	filter, ok := vars["filter"].(map[string]interface{})
	if !ok {
		t.Fatalf("filter default = %#v, want object", vars["filter"])
	}

	wantFilter := map[string]interface{}{
		"active": false,
		"nested": map[string]interface{}{
			"limit": 0,
		},
		"role": "<ENUM: ADMIN>",
		"tags": []interface{}{"<String>"},
	}
	if !reflect.DeepEqual(filter, wantFilter) {
		t.Fatalf("filter default = %#v, want %#v", filter, wantFilter)
	}
}

func TestBuildPayloadIncludesOperationAndVariablesSkeleton(t *testing.T) {
	restoreGeneratorMaxDepth(t)
	config.MaxDepth = 2

	gen, schema := newTestGenerator(t, "http://example.test/graphql", nil)
	field := schema.Mutation.Fields.ForName("createUser")

	payload := gen.BuildPayload("mutation", field, nil)
	if got, want := normalizeSpace(payload.Query), "mutation createUser($input: CreateUserInput!) { createUser(input: $input) { id name profile { bio } role } }"; got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}

	input, ok := payload.Variables["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("variables.input = %#v, want object", payload.Variables["input"])
	}

	if got, want := input["name"], "<String>"; got != want {
		t.Fatalf("variables.input.name = %#v, want %#v", got, want)
	}
	if got, want := input["role"], "<ENUM: ADMIN>"; got != want {
		t.Fatalf("variables.input.role = %#v, want %#v", got, want)
	}
}

func TestGeneratePayloadJSONCanBePrettyPrinted(t *testing.T) {
	gen, schema := newTestGenerator(t, "http://example.test/graphql", nil)
	field := schema.Query.Fields.ForName("user")

	payloadJSON, err := gen.GeneratePayloadJSON("query", field, map[string]interface{}{"id": "42"}, true)
	if err != nil {
		t.Fatalf("GeneratePayloadJSON returned error: %v", err)
	}

	for _, want := range []string{
		`"query": "query user($id: ID!, $filter: UserFilter)`,
		`"variables": {`,
		`"id": "42"`,
	} {
		if !strings.Contains(payloadJSON, want) {
			t.Fatalf("payload JSON does not contain %q:\n%s", want, payloadJSON)
		}
	}
}

func TestGenerateVariablesJSONReturnsEmptyObjectWithoutArguments(t *testing.T) {
	gen, schema := newTestGenerator(t, "http://example.test/graphql", nil)
	field := schema.Query.Fields.ForName("ping")

	variablesJSON, err := gen.GenerateVariablesJSON(field, nil, true)
	if err != nil {
		t.Fatalf("GenerateVariablesJSON returned error: %v", err)
	}
	if got, want := variablesJSON, "{}"; got != want {
		t.Fatalf("variables JSON = %q, want %q", got, want)
	}
}

func TestBuildQueryRespectsMaxDepth(t *testing.T) {
	restoreGeneratorMaxDepth(t)
	gen, schema := newTestGenerator(t, "http://example.test/graphql", nil)
	field := schema.Query.Fields.ForName("user")

	config.MaxDepth = 0
	if got, want := normalizeSpace(gen.BuildQuery("query", field)), "query { user { id name role } }"; got != want {
		t.Fatalf("BuildQuery depth 0 = %q, want %q", got, want)
	}

	config.MaxDepth = 2
	if got, want := normalizeSpace(gen.BuildQuery("query", field)), "query { user { id name profile { bio } role } }"; got != want {
		t.Fatalf("BuildQuery depth 2 = %q, want %q", got, want)
	}
}

func TestExecuteQuerySendsRequestAndReturnsMetrics(t *testing.T) {
	restoreGeneratorMaxDepth(t)
	config.MaxDepth = 2

	const responseBody = `{"data":{"user":{"id":"42"}}}`

	gen, schema := newTestGenerator(t, "", map[string]string{"Authorization": "Bearer token"})
	gen.Endpoint = "http://example.test/graphql"

	previousTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			t.Fatalf("Content-Type = %q, want %q", got, want)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		var payload graphQLPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body %q: %v", string(body), err)
		}
		if !strings.Contains(payload.Query, "query user($id: ID!") {
			t.Fatalf("query = %q, want user query with id variable", payload.Query)
		}
		if got, want := payload.Variables["id"], "42"; got != want {
			t.Fatalf("variables.id = %v, want %v", got, want)
		}

		if trace := httptrace.ContextClientTrace(r.Context()); trace != nil && trace.GotFirstResponseByte != nil {
			trace.GotFirstResponseByte()
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() {
		http.DefaultTransport = previousTransport
	})

	result, metrics, err := gen.ExecuteQuery("query", schema.Query.Fields.ForName("user"), map[string]interface{}{"id": "42"})
	if err != nil {
		t.Fatalf("ExecuteQuery returned error: %v", err)
	}
	if result != responseBody {
		t.Fatalf("response = %q, want %q", result, responseBody)
	}
	if metrics == nil {
		t.Fatalf("metrics is nil")
	}
	if got, want := metrics.Size, int64(len(responseBody)); got != want {
		t.Fatalf("metrics.Size = %d, want %d", got, want)
	}
	if metrics.Total <= 0 {
		t.Fatalf("metrics.Total = %s, want positive duration", metrics.Total)
	}
}

type graphQLPayload struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func newTestGenerator(t *testing.T, endpoint string, headers map[string]string) (*Generator, *ast.Schema) {
	t.Helper()

	schema := loadTestSchema(t)
	return NewGenerator(schema, endpoint, headers), schema
}

func loadTestSchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := gqlparser.LoadSchema(&ast.Source{
		Name: "schema.graphql",
		Input: `
enum Role {
  ADMIN
  USER
}

input NestedInput {
  limit: Int
}

input UserFilter {
  active: Boolean
  nested: NestedInput
  role: Role
  tags: [String!]
}

input CreateUserInput {
  name: String!
  role: Role
}

type Profile {
  bio: String
}

type User {
  id: ID!
  name: String
  profile: Profile
  role: Role
}

type Query {
  user(id: ID!, filter: UserFilter): User
  ping: String
}

type Mutation {
  createUser(input: CreateUserInput!): User
}
`,
	})
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	return schema
}

func decodeCurlPayload(t *testing.T, curl string) graphQLPayload {
	t.Helper()

	const marker = "--data-raw '"
	idx := strings.Index(curl, marker)
	if idx == -1 {
		t.Fatalf("curl payload marker %q not found in:\n%s", marker, curl)
	}

	payloadJSON := strings.TrimSuffix(curl[idx+len(marker):], "'")
	var payload graphQLPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode payload %q: %v", payloadJSON, err)
	}

	return payload
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func restoreGeneratorMaxDepth(t *testing.T) {
	t.Helper()

	previous := config.MaxDepth
	t.Cleanup(func() {
		config.MaxDepth = previous
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
