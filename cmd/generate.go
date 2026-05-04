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

var run bool

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
				fmt.Printf("\n# Operation: %s %s\n", op.OpType, curl)

				if run {
					fmt.Printf("🚀 Executing against endpoint: %s\n", cfg.Endpoint)
					result, err := gen.ExecuteQuery(op.OpType, field)
					if err != nil {
						fmt.Printf("❌ Error: %v\n", err)
					} else {
						fmt.Println(result)
					}
				} else {
					// Якщо флаг --run не вказано, просто виводимо curl
					curl := gen.GenerateCurl(op.OpType, field)
					fmt.Println(curl)
				}
			}
		}

		// If a target operation was specified but not found in the schema, log an error
		if targetOp != "" && !found {
			log.Fatalf("❌ Operation '%s' not found in schema", targetOp)
		}

	},
}

func init() {
	// Expose the --run flag to allow users to execute the generated query directly against the endpoint
	generateCmd.Flags().BoolVarP(&run, "run", "r", false, "Connect to the endpoint and execute the generated query, printing the response")
	rootCmd.AddCommand(generateCmd)
}
