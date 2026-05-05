package tui

import (
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/vektah/gqlparser/v2/ast"
)

// FillVariablesInteractive builds an interactive form based on the provided GraphQL argument definitions and schema.
func FillVariablesInteractive(schema *ast.Schema, args ast.ArgumentDefinitionList) (map[string]interface{}, error) {
	if len(args) == 0 {
		return nil, nil
	}

	result := make(map[string]interface{})
	var fields []huh.Field

	// Recursively build fields from the AST definitions, supporting nested input objects
	fields = buildFieldsFromAST(schema, "", args, result)

	if len(fields) == 0 {
		return nil, nil
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	err := form.Run()
	if err != nil {
		return nil, err
	}

	return finalizeResult(result), nil
}

func buildFieldsFromAST(schema *ast.Schema, prefix string, args ast.ArgumentDefinitionList, result map[string]interface{}) []huh.Field {
	var fields []huh.Field

	for _, arg := range args {
		label := arg.Name
		if prefix != "" {
			label = prefix + "." + arg.Name
		}

		// Get the named type (unwrap NonNull and List)
		namedType := arg.Type.Name()
		def, ok := schema.Types[namedType]
		if !ok {
			// If the type definition is not found in the schema, treat it as a scalar with a generic input
			fields = append(fields, huh.NewInput().
				Title(label).
				Value(createStringPointer(result, arg.Name)))
			continue
		}

		switch def.Kind {
		case ast.Scalar:
			fields = append(fields, buildScalarField(label, namedType, arg.Name, result))

		case ast.Enum:
			// List enum values as options in a select field
			var options []huh.Option[string]
			for _, val := range def.EnumValues {
				options = append(options, huh.NewOption(val.Name, val.Name))
			}
			fields = append(fields, huh.NewSelect[string]().
				Title(label).
				Options(options...).
				Value(createStringPointer(result, arg.Name)))

		case ast.InputObject:
			// Recursively build fields for nested input objects
			subResult := make(map[string]interface{})
			result[arg.Name] = subResult
			// Convert InputObject fields to ArgumentDefinitionList for recursive processing
			subArgs := convertFieldsToArgs(def.Fields)
			fields = append(fields, buildFieldsFromAST(schema, label, subArgs, subResult)...)
		}
	}
	return fields
}

func buildScalarField(label, typeName, key string, result map[string]interface{}) huh.Field {
	switch typeName {
	case "Boolean":
		return huh.NewConfirm().Title(label).Value(createBoolPointer(result, key))
	case "Int", "Float":
		return huh.NewInput().
			Title(label).
			Validate(func(s string) error {
				_, err := strconv.ParseFloat(s, 64)
				return err
			}).
			Value(createStringPointer(result, key))
	default:
		return huh.NewInput().Title(label).Value(createStringPointer(result, key))
	}
}

// Helper function to convert a list of ast.FieldDefinition to ast.ArgumentDefinitionList for recursive processing of nested input objects
func convertFieldsToArgs(fields ast.FieldList) ast.ArgumentDefinitionList {
	var args ast.ArgumentDefinitionList
	for _, f := range fields {
		args = append(args, &ast.ArgumentDefinition{
			Name: f.Name,
			Type: f.Type,
		})
	}
	return args
}

func createStringPointer(m map[string]interface{}, key string) *string {
	s := ""
	m[key] = &s
	return &s
}

func createBoolPointer(m map[string]interface{}, key string) *bool {
	b := false
	m[key] = &b
	return &b
}

// finalizeResult finalizes the result map by dereferencing any pointer values and recursively processing nested maps,
// returning a clean map[string]interface{} with actual values instead of pointers.
func finalizeResult(m map[string]interface{}) map[string]interface{} {
	final := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case *string:
			final[k] = *val
		case *bool:
			final[k] = *val
		case map[string]interface{}:
			final[k] = finalizeResult(val)
		}
	}
	return final
}
