package cmd

import (
	"fmt"
	"log"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/emp1re/gql-curl/internal/generator"
	"github.com/emp1re/gql-curl/internal/parser"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

var generateCmd = &cobra.Command{
	Use:   "generate [operationName]",
	Short: "Generate cURL commands for GraphQL operations defined in the schema",
	Args:  cobra.MaximumNArgs(1), // Only one optional argument for operation name
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig("graphql.curl.yaml")
		if err != nil {
			log.Fatalf("❌ Load config error: %v", err)
		}

		// Read and parse the GraphQL schema from the specified directory
		gql, err := parser.NewParserFromDir(cfg.Schema)
		if err != nil {
			log.Fatalf("❌ Parse schema error: %v", err)
		}

		// Retrieve the target operation name from the command-line arguments, if provided
		targetOp := ""
		if len(args) > 0 {
			targetOp = args[0]
		}

		gen := generator.NewGenerator(gql.Schema, cfg.Endpoint, cfg.Headers)

		operations := []struct {
			OpType string
			Def    *ast.Definition
		}{
			{"query", gql.Schema.Query},
			{"mutation", gql.Schema.Mutation},
		}

		found := false

		for _, op := range operations {
			if op.Def == nil {
				continue
			}

			for _, field := range op.Def.Fields {
				// if the user specified an operation name, skip fields that don't match
				if targetOp != "" && field.Name != targetOp {
					continue
				}

				found = true
				curl := gen.GenerateCurl(op.OpType, field)
				fmt.Printf("\n# Field: %s\n%s\n", field.Name, curl)
			}
		}

		// If a target operation was specified but not found in the schema, log an error
		if targetOp != "" && !found {
			log.Fatalf("❌ Operation '%s' not found in schema", targetOp)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
