# GraphQL-Curl (gqc)

A powerful Go CLI tool that generates ready-to-run `curl` commands from your GraphQL schema. Parse your schema, generate curl commands for every query and mutation, pass variables, and optionally execute them directly against your endpoint.

## Features

- **Schema Parsing**: Reads `.graphql` and `.graphqls` files from a schema directory (you can specify in your config)
- **Automatic curl Generation**: Generates one `curl` command per top-level `query` and `mutation` field  
- **Variable Support**: Pass variables inline (`--vars`) or from a JSON file (`--var-file`)
- **Smart Defaults**: Auto-generates variable placeholders based on field arguments and types
- **Direct Execution**: Use `--run` flag to execute queries directly against your endpoint and see formatted responses
- **Environment Interpolation**: Use `{{environment.KEY}}` placeholders in headers and config
- **Custom Depth Control**: Configure how deep nested types are expanded (via `MAX_DEPTH` environment variable)
- **Custom Document Extensions**: Define which file extensions to parse (`.graphql`, `.graphqls`, or custom)
- **Pretty-printed Responses**: JSON responses are automatically formatted for readability
- **Operation Filtering**: Generate commands for a single operation or all operations

## Requirements


- Go `1.25.x` (based on `go.mod`)

## Install

### Option 1: Install with `go install`

```bash
go install github.com/emp1re/gql-curl/cmd/gqc@latest
```

This installs the binary into your Go bin path (usually `$GOPATH/bin` or `$HOME/go/bin`).

### Option 2: Build from source

```bash
git clone https://github.com/emp1re/gql-curl
cd gql-curl
go build -o gqc ./cmd/gqc
```

## Quick Start

1. Create a config file named `graphql.curl.yaml` in your working directory
2. Point `schema` to a directory containing your GraphQL schema files
3. Run `generate` to print curl commands

Example:

```bash
gqc generate
```

## Configuration (`graphql.curl.yaml`)

The CLI always reads `graphql.curl.yaml` from the current directory.

```yaml
schema: "./schema"
endpoint: "http://localhost:8080/graphql"
document_extensions:
  - ".graphql"
  - ".graphqls"
headers:
  Authorization: "Bearer {{environment.GQL_TOKEN}}"
  X-Custom-Header: "value"
environment:
  GQL_TOKEN: "my-secret-token"
  MAX_DEPTH: 3
```

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema` | string | ✓ | Directory path containing GraphQL schema files |
| `endpoint` | string | ✓ | GraphQL HTTP endpoint URL |
| `headers` | map | | HTTP headers to include in curl commands. Supports `{{environment.KEY}}` interpolation |
| `environment` | map | | Environment variables used for header interpolation and configuration |
| `document_extensions` | list | | File extensions to parse (e.g., `.graphql`, `.graphqls`). Defaults to both if not specified |

## Usage

### Show help

```bash
gqc --help
gqc generate --help
```

### Generate curl commands for all operations

```bash
gqc generate
```

### Generate curl for a specific operation

```bash
gqc generate getUser
```

### Generate with inline variables

```bash
gqc generate getUser --vars '{"id": "123"}'
```

### Generate with variables from a file

```bash
gqc generate getUser --var-file ./variables.json
```

**variables.json:**
```json
{
  "id": "123",
  "name": "John"
}
```

### Execute a query directly against the endpoint

```bash
gqc generate getUser --run
```

This generates the curl command, executes it, and displays the formatted JSON response.

### Combine flags

```bash
gqc generate createUser --var-file ./user.json --run
```

## Example Output

### Generated curl command

```bash
# Operation: query | Field: getUser
curl -X POST http://localhost:8080/graphql \
  -H 'Authorization: Bearer my-secret-token' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query":"query getUser($id: ID!) {\n  getUser(id: $id) {\n    id\n    name\n    email\n  }\n}","variables":{"id":"<placeholder>"}}'
```

### With --run flag

```bash
🚀 Execute query: getUser...

# Operation: query | Field: getUser
curl -X POST http://localhost:8080/graphql \
  -H 'Authorization: Bearer my-secret-token' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query":"query getUser($id: ID!) {\n  getUser(id: $id) {\n    id\n    name\n    email\n  }\n}","variables":{"id":"<placeholder>"}}'

✅ Server response:
{
  "data": {
    "getUser": {
      "id": "123",
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

## Advanced Features

### Variable Type Defaults

When you generate a command without providing variables, the tool automatically creates placeholder values based on GraphQL types:

- **Scalars**: Type name in angle brackets (e.g., `<string>`, `<ID>`)
- **Enums**: First enum value prefixed with `<ENUM: >`
- **Input Objects**: Nested structure with default values for each field
- **Lists**: Array with one element of the list type

### Custom Depth Control

To control how deeply nested types are expanded in the selection set, use the `MAX_DEPTH` environment variable:

```yaml
environment:
  MAX_DEPTH: "2"  # Limit expansion to 2 levels
```

This prevents overly large selection sets for deeply nested schemas.

### Environment Variable Interpolation

Use `{{environment.KEY}}` in headers to reference environment variables from your config:

```yaml
environment:
  GQL_TOKEN: "my-secret-token"
  API_KEY: "secret-api-key"
headers:
  Authorization: "Bearer {{environment.GQL_TOKEN}}"
  X-API-Key: "{{environment.API_KEY}}"
```

## Error Handling

The CLI provides detailed error messages with emoji indicators:

- ❌ Error messages for invalid operations, configuration issues, or execution failures
- ✅ Success indicator when queries execute successfully

Example errors:
```
❌ Operation 'unknownField' not found in schema
❌ You cannot use both --vars and --var-file flags at the same time. Please choose one.
❌ Error reading variables file: no such file or directory
```

## Notes

- The tool auto-expands nested fields up to a configurable depth (default: 3)
- If an operation name is specified but not found in the schema, the command exits with an error
- Variables can be passed inline (JSON string) or from a file (JSON file)
- The `--run` flag requires a valid endpoint configured in your config file
- Response bodies are automatically pretty-printed as formatted JSON

## Development

Run directly without building:

```bash
go run ./cmd/gqc --help
go run ./cmd/gqc generate --help
```

Run tests (if available):

```bash
go test ./...
```
