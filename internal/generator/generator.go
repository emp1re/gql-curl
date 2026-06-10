package generator

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/vektah/gqlparser/v2/ast"
)

type Metrics struct {
	DNS   time.Duration
	TCP   time.Duration
	TLS   time.Duration
	TTFB  time.Duration // Time To First Byte
	Total time.Duration
	Size  int64 // Response size in bytes
}

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

func (g *Generator) GenerateCurl(opType string, field *ast.FieldDefinition, customVars map[string]interface{}) string {
	query := g.buildOperationString(opType, field)

	// Prepare the JSON payload for the GraphQL request
	var vars map[string]interface{}
	if customVars != nil {
		vars = customVars
	} else {
		vars = g.buildVariablesSkeleton(field)
	}

	payloadMap := map[string]interface{}{
		"query": query,
	}
	if vars != nil && len(vars) > 0 {
		payloadMap["variables"] = vars
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payloadMap); err != nil {
		return fmt.Sprintf("# Generation error JSON: %v", err)
	}
	payloadString := strings.TrimSpace(buf.String())

	var sb strings.Builder
	sb.WriteString("curl -X POST ")
	sb.WriteString(fmt.Sprintf("%s \\\n", g.Endpoint))

	hasContentType := false
	for k, v := range g.Headers {
		sb.WriteString(fmt.Sprintf("  -H '%s: %s' \\\n", k, v))
		if strings.ToLower(k) == "content-type" {
			hasContentType = true
		}
	}
	if !hasContentType {
		sb.WriteString("  -H 'Content-Type: application/json' \\\n")
	}

	sb.WriteString(fmt.Sprintf("  --data-raw '%s'", payloadString))

	return sb.String()
}

// buildOperationString forms the complete GraphQL operation string, including variable definitions and field arguments, based on the provided operation type and field definition.
func (g *Generator) buildOperationString(opType string, field *ast.FieldDefinition) string {
	var varDefs []string
	var fieldArgs []string

	// Review the field's arguments to build variable definitions and field arguments
	for _, arg := range field.Arguments {
		// For variable definitions: $id: ID!
		varDefs = append(varDefs, fmt.Sprintf("$%s: %s", arg.Name, arg.Type.String()))
		// For field arguments: id: $id
		fieldArgs = append(fieldArgs, fmt.Sprintf("%s: $%s", arg.Name, arg.Name))
	}

	varDefStr := ""
	if len(varDefs) > 0 {
		varDefStr = "(" + strings.Join(varDefs, ", ") + ")"
	}

	fieldArgsStr := ""
	if len(fieldArgs) > 0 {
		fieldArgsStr = "(" + strings.Join(fieldArgs, ", ") + ")"
	}

	selection := g.expandType(field.Type, 0)

	// Form the full operation string, e.g. "query getUser($id: ID!) { getUser(id: $id) { id name } }"
	return fmt.Sprintf("%s %s%s { %s%s %s }", opType, field.Name, varDefStr, field.Name, fieldArgsStr, selection)
}

// buildVariablesSkeleton generates a skeleton of variables for the given field's arguments, assigning default values based on their types.
// This is used to create a complete JSON payload for the GraphQL request, even if the actual values are placeholders.
func (g *Generator) buildVariablesSkeleton(field *ast.FieldDefinition) map[string]interface{} {
	if len(field.Arguments) == 0 {
		return nil
	}

	vars := make(map[string]interface{})
	for _, arg := range field.Arguments {
		vars[arg.Name] = g.getDefaultValueForType(arg.Type, 0)
	}
	return vars
}

func (g *Generator) getDefaultValueForType(typ *ast.Type, depth int) interface{} {
	if depth > config.MaxDepth {
		return nil
	}

	// Recursively handle list types by returning an array with a single default element
	if typ.Elem != nil {
		// Return an array with one default value for the element type
		return []interface{}{g.getDefaultValueForType(typ.Elem, depth+1)}
	}

	typeName := typ.Name()
	def, ok := g.Schema.Types[typeName]

	// If the type definition is not found in the schema, return a generic default value based on the type name (assuming it's a scalar or custom type)
	if !ok {
		return getScalarDefault(typeName)
	}

	switch def.Kind {
	case ast.Scalar:
		return getScalarDefault(typeName)
	case ast.Enum:
		if len(def.EnumValues) > 0 {
			return fmt.Sprintf("<ENUM: %s>", def.EnumValues[0].Name)
		}
		return "<ENUM>"
	case ast.InputObject:
		obj := make(map[string]interface{})
		for _, f := range def.Fields {
			obj[f.Name] = g.getDefaultValueForType(f.Type, depth+1)
		}
		return obj
	}

	return nil
}

func (g *Generator) BuildQuery(opType string, field *ast.FieldDefinition) string {
	// Generate the selection set for the given field's return type
	selection := g.expandType(field.Type, 0)

	// Form the full query string with the operation type, field name, and selection set
	return fmt.Sprintf("%s { %s %s }", opType, field.Name, selection)
}

// expandType recursively builds the selection set for a given GraphQL type, respecting the maximum depth to avoid infinite recursion.
func (g *Generator) expandType(typ *ast.Type, depth int) string {
	if depth > config.MaxDepth {
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

	var sb strings.Builder
	sb.WriteString("{ ")

	hasFields := false
	for _, f := range def.Fields {
		baseType := f.Type.Name()
		typeDef := g.Schema.Types[baseType]

		if typeDef == nil {
			continue
		}

		// Check if the field's type is a leaf (Scalar or Enum) or an Object/Interface that requires further expansion
		isLeaf := typeDef.Kind == ast.Scalar || typeDef.Kind == ast.Enum

		if isLeaf {
			// SCALAR or ENUM: Add the field name directly to the selection set
			sb.WriteString(f.Name + " ")
			hasFields = true
		} else {
			// OBJECT or INTERFACE: Attempt to expand further
			subSelection := g.expandType(f.Type, depth+1)

			// CRITICAL: Add the object field ONLY if we were able to generate nested fields for it.
			// This prevents the "Field must have a selection of subfields" error.
			if subSelection != "" {
				sb.WriteString(fmt.Sprintf("%s %s ", f.Name, subSelection))
				hasFields = true
			}
		}
	}

	sb.WriteString("}")

	// If no valid fields were found inside the block (e.g., due to depth limit),
	// return an empty string to prevent the parent field from being rendered invalidly.
	if !hasFields {
		return ""
	}
	return sb.String()
}

// ExecuteQuery sends the generated GraphQL query to the specified endpoint and returns the pretty-printed JSON response.
func (g *Generator) ExecuteQuery(opType string, field *ast.FieldDefinition, customVars map[string]interface{}) (string, *Metrics, error) {
	query := g.buildOperationString(opType, field)

	var vars map[string]interface{}
	if customVars != nil {
		vars = customVars
	} else {
		vars = g.buildVariablesSkeleton(field)
	}

	payloadMap := map[string]interface{}{
		"query": query,
	}
	if vars != nil && len(vars) > 0 {
		payloadMap["variables"] = vars
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payloadMap); err != nil {
		return "", nil, fmt.Errorf("encode payload failed: %w", err)
	}

	req, err := http.NewRequest("POST", g.Endpoint, &buf)
	if err != nil {
		return "", nil, err
	}

	// Headers
	for k, v := range g.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	// <-- HTTP Trace for metrics -->
	var t0, t1, t2, t3, t4 time.Time
	var dnsStart, tcpStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { t1 = time.Now() },
		ConnectStart: func(_, _ string) {
			if dnsStart.IsZero() {
				t1 = time.Now()
			}
			tcpStart = time.Now()
		},
		ConnectDone:          func(net, addr string, err error) { t2 = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { t3 = time.Now() },
		GotFirstResponseByte: func() { t4 = time.Now() },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Send the request with a timeout
	client := &http.Client{Timeout: 10 * time.Second}
	t0 = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	t5 := time.Now()

	// Calculate metrics based on the recorded timestamps. We check if each timestamp is set before calculating the duration to avoid zero values.
	metrics := &Metrics{
		Total: t5.Sub(t0),
		Size:  int64(len(respBody)),
	}

	if !t1.IsZero() && !dnsStart.IsZero() {
		metrics.DNS = t1.Sub(dnsStart)
	}
	if !t2.IsZero() && !tcpStart.IsZero() {
		metrics.TCP = t2.Sub(tcpStart)
	}
	if !t3.IsZero() && !tlsStart.IsZero() {
		metrics.TLS = t3.Sub(tlsStart)
	}
	if !t4.IsZero() {
		// TTFB = time from the start of the request to the moment we receive the first byte of the response
		metrics.TTFB = t4.Sub(t0)
	}

	return string(respBody), metrics, nil
}
