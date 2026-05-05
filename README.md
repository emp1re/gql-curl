gqc
 gqcis a CLI that reads a GraphQL schema and prints ready-to-run `curl` commands for your top-level `query` and `mutation` fields.

## Features

- Parses `.graphql` and `.graphqls` files from a schema directory.
- Generates one `curl` command per top-level field.
- Supports optional filtering by operation name.
- Injects header values from `credentials` placeholders like `{{credentials.token}}`.

## Requirements

- Go `1.25.x` (based on `go.mod`).

## Install

### Option 1: Install with `go install`

```bash
go install github.com/emp1re/gql-curl/cmd/gqc@latest
```

This installs the binary into your Go bin path (usually `$GOPATH/bin` or `$HOME/go/bin`).

### Option 2: Build from source

```bash
git clone https://github.com/emp1re/gql-curl
cd gqc
go build -o gqc ./cmd/gqc
```

## Quick Start

1. Create a config file named `graphql.curl.yaml` in your working directory.
2. Point `schema` to a directory containing your GraphQL schema files.
3. Run `generate` to print curl commands.

Example:

```bash
gqc generate
```

Generate only one operation:

```bash
gqc generate getUser
```

## Configuration (`graphql.curl.yaml`)

The CLI always reads `graphql.curl.yaml` from the current directory.

```yaml
schema: "./schema"
output: "./generated_curls"
endpoint: "http://localhost:8080/graphql"
credentials:
  token: "your-token"
headers:
  Authorization: "Bearer {{credentials.token}}"
```

### Fields

- `schema`: Directory containing `.graphql` / `.graphqls` files.
- `output`: Currently not used by the CLI (commands are printed to stdout).
- `endpoint`: GraphQL HTTP endpoint.
- `environment`: Optional environment variables from ENV or other sources.
- `headers`: HTTP headers included in generated `curl` command.

## Usage

Show help:

```bash
gqc --help
gqc generate --help
```

Generate commands for all top-level operations:

```bash
gqc generate
```

Generate command for one operation only:

```bash
gqc generate [operationName]
```

Run the generated `curl` command in your terminal to execute the GraphQL query/mutation.

```bash
gqc generate operationName [--run || -r]
````

## Example Output

```bash
# Operation: getUser
curl -X POST http://localhost:8080/graphql \
  -H 'Authorization: Bearer {$GQL_AUTH_TOKEN}' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query": "query { getUser { id name } }"}'
```

## Notes

- The tool inspects schema types and auto-expands nested fields (up to depth 3).
- If an operation name is provided but not found, the command exits with an error.
- Credential values are used as plain strings; environment-variable expansion is not automatic.

## Development

Run directly without building:

```bash
go run . --help
go run . generate --help
```

