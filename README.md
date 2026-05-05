# GraphQL-Curl (`gqc`)

`gqc` is a Go CLI that reads your GraphQL schema and helps you either:

- generate ready-to-run `curl` requests for top-level `query` and `mutation` fields, or
- execute those generated operations directly against your endpoint.

It also supports schema fetching via GraphQL introspection.

## What's New

- `--interactive` mode to fill variables from a terminal form.
- `--run` mode to execute generated operations immediately.
- Request performance metrics in `--run` mode (Total, TTFB, DNS, TCP, TLS, Size).
- `--filter` support (gjson syntax) to print only part of a response.
- Variables from inline JSON (`--vars`) or JSON file (`--var-file`).
- Header interpolation using `{{environment.KEY}}` values.
- Configurable query expansion depth via `environment.MAX_DEPTH`.
- Configurable schema file extensions via `document_extensions`.

## Requirements

- Go `1.25.5` (from `go.mod`).

## Install

### Option 1: Install from module

```bash
go install github.com/emp1re/gql-curl/cmd/gqc@latest
```

### Option 2: Build from source

```bash
git clone https://github.com/emp1re/gql-curl
cd gql-curl
go build -o gqc ./cmd/gqc
```

## Quick Start

1. Create `graphql.curl.yaml` in your working directory.
2. Point `schema` to your GraphQL schema directory.
3. Run `gqc generate`.

```bash
gqc generate
```

Generate one operation only:

```bash
gqc generate getUser
```

## Configuration (`graphql.curl.yaml`)

The CLI loads `graphql.curl.yaml` from the current directory.
It also calls `.env` loading automatically (via `godotenv`).

```yaml
schema: "./schema"
document_extensions: [".graphql", ".graphqls", ".gql"]
endpoint: "http://localhost:8080/graphql"

environment:
  GQL_AUTH_TOKEN: ${GQL_AUTH_TOKEN}
  MAX_DEPTH: "3"

headers:
  Authorization: "Bearer {{environment.GQL_AUTH_TOKEN}}"
```

### Field Reference

- `schema` (string): path to schema directory (used by `generate`) or output file/dir target (used by `fetch`).
- `document_extensions` ([]string): schema file extensions to parse (for example `.graphql`, `.graphqls`, `.gql`).
- `endpoint` (string): GraphQL server URL.
- `environment` (map): values used for interpolation and runtime settings (like `MAX_DEPTH`).
- `headers` (map): HTTP headers for generated/executed requests.
- `output` (string): present in config struct, currently not used by commands.

## Commands

### `generate`

Generate `curl` commands for all root operations:

```bash
gqc generate
```

Generate for one operation:

```bash
gqc generate getUser || gqc g getUser
```

Use inline variables:

```bash
gqc generate getUser --vars '{"id":"123"}' || gqc g getUser --vars '{"id":"123"}'
```

Use variables from file:

```bash
gqc generate getUser --var-file ./vars.json || gqc g getUser --var-file ./vars.json
```

Interactive variable input:

```bash
gqc generate createUser --interactive || gqc g createUser -i
```

Execute request immediately:

```bash
gqc generate getUser --run
```

Execute and filter output (gjson path):

```bash
gqc generate getUser --run --filter 'data.getUser.name'
```

Run with variables and still see performance metrics:

```bash
gqc generate getUser --run --vars '{"id":"123"}'
```

> Note: `--vars` and `--var-file` are mutually exclusive.

### `fetch`

Fetch schema using introspection and save it to `schema` path from config:

```bash
gqc fetch
```

If `schema` points to a directory, output is saved as `schema.graphql` in that directory.

## Generated Output Example

```bash
# Operation: query | Field: getUser
curl -X POST http://localhost:8080/graphql \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query":"query getUser($id: ID!) { getUser(id: $id) { id name } }","variables":{"id":"<ID>"}}'
```

## Runtime Response Behavior (`--run`)

- JSON object/array responses are colorized and pretty-printed.
- With `--filter`, scalar results are printed as raw values (useful for scripts).
- If filtered path does not exist, a warning is shown.

## Metrics

When you use `gqc generate ... --run`, the CLI prints a performance block after the response:

- `Total`: full request time (send request + receive/read response body).
- `TTFB`: time to first byte from the server.
- `DNS`: DNS lookup duration (can be zero on cached/reused connections).
- `TCP`: TCP connect duration (can be zero on keep-alive reuse).
- `TLS`: TLS handshake duration (can be zero for plain HTTP or reused TLS session).
- `Size`: response body size.

Example:

```text
📊 Performance Metrics:
  Total: 123ms  TTFB: 47ms  DNS: 2ms  TCP: 4ms  TLS: 0ms  Size: 3.21 KB
```

This is useful for quick endpoint latency checks without external tooling.

## Help

```bash
gqc --help
gqc generate --help
gqc fetch --help
```

## Development

Run without installing:

```bash
go run ./cmd/gqc --help
go run ./cmd/gqc generate --help
go run ./cmd/gqc fetch --help
```
