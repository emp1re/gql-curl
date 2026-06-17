package cmd

import "github.com/vektah/gqlparser/v2/ast"

type rootOperationDefinition struct {
	OpType string
	Def    *ast.Definition
}

func rootOperationDefinitions(schema *ast.Schema) []rootOperationDefinition {
	if schema == nil {
		return nil
	}

	return []rootOperationDefinition{
		{OpType: "query", Def: schema.Query},
		{OpType: "mutation", Def: schema.Mutation},
	}
}
