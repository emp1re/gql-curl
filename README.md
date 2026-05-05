# 🚀 gql-curl (gqc)

[![Go Version](https://img.shields.io/github/go-mod/go-version/emp1re/gql-curl)](https://golang.org)
[![License](https://img.shields.io/github/license/emp1re/gql-curl)](LICENSE)

**gql-curl** is a high-performance, interactive CLI tool designed for developers who are tired of fighting with manual JSON payloads in `curl` or waiting for heavy GUI clients to load. 

Written in Go, it bridges the gap between the speed of the command line and the intelligence of a full-blown GraphQL IDE.

---

## ✨ Key Features

* **🎮 Interactive TUI:** Type-safe terminal forms for variable input. No more JSON syntax errors.
* **📊 Professional Profiling:** Detailed network breakdown (TTFB, DNS, TLS) for every request.
* **📡 Instant Introspection:** Sync your local SDL schema from any remote endpoint in seconds.
* **🔍 Built-in Filtering:** Extract data directly with GJSON paths—no `jq` required.
* **🎨 Beautiful DX:** Syntax highlighting for queries, responses, and generated `curl` commands.
* **⚡ Single Binary:** Written in Go. Zero dependencies. Works everywhere.

---

## ✨ Features Deep Dive

### 🎮 Smart Interactive Variable Injection
Handling complex GraphQL input objects and nested variables in a standard terminal is a nightmare. 
- **Type-Aware Forms:** `gql-curl` parses your schema's AST to generate interactive forms. It knows if a field is an `Int`, `Boolean`, or a complex `InputObject`.
- **Enum Autocompletion:** No more guessing valid enum values. Select them from a visual list.
- **Recursive Input:** Easily fill out deeply nested objects without worrying about JSON syntax or escaping quotes.

### 📊 Professional Performance Profiling
Stop guessing why your request is slow. Powered by Go's `httptrace`, we provide a granular breakdown of the request lifecycle:
- **TTFB (Time To First Byte):** Isolate server-side processing time from network latency.
- **Network Overhead:** See exactly how much time is spent on DNS lookup, TCP connection, and TLS handshakes.
- **Payload Analysis:** Real-time reporting of response sizes to detect unoptimized queries or missing pagination.

### 📡 Schema Synchronization (Introspection)
Keep your local development environment in sync with the server effortlessly.
- **SDL Generation:** Automatically converts raw JSON introspection data into clean, readable `.graphql` Schema Definition Language.
- **Authenticated Fetch:** Supports custom headers (like `Authorization`) during introspection, allowing you to pull schemas from protected production or staging environments.

### 🔍 Scripting & Post-Processing
`gql-curl` is designed to be a "good citizen" in your shell environment.
- **Built-in GJSON Engine:** Use the `-q` flag to extract specific data from deep JSON paths without needing external tools like `jq`.
- **Bash-Friendly Output:** When filtering, the tool returns raw scalar values (strings, numbers) making it trivial to pipe results into other commands or environment variables.

### 🎨 Developer Experience (DX) First
- **Zero Runtime Friction:** A single statically linked binary. No `node_modules`, no Python interpreters, no JVM.
- **Silent & Verbose Modes:** Output only what you need—either a clean `curl` command for documentation or the full execution results with metrics.
- **Colorized Everything:** High-contrast syntax highlighting for both the generated queries and the server responses.

---

## 📦 Installation

```bash
go install github.com/emp1re/gql-curl/cmd/gqc@latest
```

## 🛠 Configuration
Create a graphql.curl.yaml in your project root:

```yaml
schema: "./schema.graphql"
endpoint: "http://localhost:8008/gql/query"
output: "./generated" # Target directory for saved curls

environment:
  AUTH_TOKEN: "${GQL_TOKEN}" # Loads from your .env or shell

headers:
  Authorization: "Bearer {{environment.AUTH_TOKEN}}"
  X-Custom-Header: "GQC-Client"
```

## 🚀 Quick Start
1. Fetch the Schema
Bootstrap your project by pulling the schema from your live endpoint:

```bash
gqc fetch
```

2. Generate and Execute
Generate a query, fill variables interactively, and execute it immediately:

```bash
gqc generate myMutation -i -r
```

3. Filter the Result
Need just a specific field from a massive response? Use the filter flag:

```bash
gqc g getContact -r -q "data.getContact.email"
```

## 📖 Command Reference

| Command | Alias | Description |
| --- | --- | --- |
| fetch | f | Pull SDL schema from remote endpoint via Introspection. |
| generate | g | Generate a GraphQL operation and its corresponding curl. |
| completion |  | Generate autocompletion scripts for Bash, Zsh, Fish. |

Generation Flags
* **`-i, --interactive`**: Enable TUI for variable input.

* **`-r, --run`**: Execute the request immediately after generation.

* **`-q, --filter`**: Filter the JSON response (GJSON syntax).

* **`-v, --vars`**: Pass variables as a JSON string.

## 📊 Performance Benchmarking

When using the **`--run`** flag, **`gql-curl`** provides a detailed breakdown of your request lifecycle:

```bash
📊 Performance Metrics:
  Total: 142ms  TTFB: 135ms  DNS: 2ms  TCP: 4ms  TLS: 1ms  Size: 1.45 KB
```

* **TTFB**: Time To First Byte (measures how fast your resolver/DB actually is).

* **DNS/TCP/TLS**: Helps identify network-level bottlenecks.
