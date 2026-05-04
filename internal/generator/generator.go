package generator

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// MaxDepth defines how deep the generator will expand nested types when building the selection set.
const MaxDepth = 3

type Generator struct {
	Schema   *ast.Schema
	Endpoint string
	Headers  map[string]string
}

func NewGenerator(schema *ast.Schema, endpoint string, headers map[string]string) *Generator {
	return &Generator{
		Schema:   schema,
		Endpoint: endpoint,
		Headers:  headers,
	}
}

func (g *Generator) GenerateCurl(opType string, field *ast.FieldDefinition) string {
	query := g.BuildQuery(opType, field)

	var sb strings.Builder
	sb.WriteString("curl -X POST ")
	sb.WriteString(fmt.Sprintf("%s \\\n", g.Endpoint))

	for k, v := range g.Headers {
		sb.WriteString(fmt.Sprintf("  -H '%s: %s' \\\n", k, v))
	}

	sb.WriteString("  -H 'Content-Type: application/json' \\\n")

	// Escape newlines and double quotes in the query string for safe inclusion in the JSON payload
	cleanQuery := strings.ReplaceAll(query, "\n", " ")
	cleanQuery = strings.ReplaceAll(cleanQuery, `"`, `\"`)

	payload := fmt.Sprintf(`{"query": "%s"}`, cleanQuery)
	sb.WriteString(fmt.Sprintf("  --data-raw '%s'", payload))

	return sb.String()
}

func (g *Generator) BuildQuery(opType string, field *ast.FieldDefinition) string {
	// Generate the selection set for the given field's return type
	selection := g.expandType(field.Type, 0)

	// Form the full query string with the operation type, field name, and selection set
	return fmt.Sprintf("%s { %s %s }", opType, field.Name, selection)
}

// expandType recursively builds the selection set for a given GraphQL type, respecting the maximum depth to avoid infinite recursion.
func (g *Generator) expandType(typ *ast.Type, depth int) string {
	if depth > MaxDepth {
		return ""
	}

	typeName := typ.Name()

	// Get the type definition from the schema
	def, ok := g.Schema.Types[typeName]
	if !ok {
		return ""
	}

	// If it's a Scalar or Enum type, we don't need to expand further
	if def.Kind == ast.Scalar || def.Kind == ast.Enum {
		return ""
	}

	// For Object, Interface, or Union types, we need to expand their fields
	var sb strings.Builder
	sb.WriteString("{ ")

	for _, f := range def.Fields {
		subSelection := g.expandType(f.Type, depth+1)

		sb.WriteString(f.Name)
		if subSelection != "" {
			sb.WriteString(" " + subSelection)
		}
		sb.WriteString(" ")
	}

	sb.WriteString("}")
	return sb.String()
}
