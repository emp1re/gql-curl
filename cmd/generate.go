package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/emp1re/gql-curl/internal/generator"
	"github.com/emp1re/gql-curl/internal/parser"
	"github.com/emp1re/gql-curl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	run         bool
	varsStr     string
	varsFile    string
	interactive bool
)

// parseVariables відповідає за читання JSON з рядка або файлу
func parseVariables(vStr, vFile string) (map[string]interface{}, error) {
	if vStr != "" && vFile != "" {
		return nil, fmt.Errorf("❌ You cannot use both --vars and --vars-file flags at the same time. Please choose one.")
	}

	var data []byte
	var err error

	if vFile != "" {
		data, err = os.ReadFile(vFile)
		if err != nil {
			return nil, fmt.Errorf("❌ Error reading variables file: %w", err)
		}
	} else if vStr != "" {
		data = []byte(vStr)
	} else {
		// Якщо прапорці не передані, повертаємо nil
		return nil, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {

		return nil, fmt.Errorf("❌ Error parsing variables JSON: %w\nPlease ensure the JSON is valid and properly formatted.", err)
	}

	return result, nil
}

var generateCmd = &cobra.Command{
	Use:     "generate [operationName] [flags]",
	Aliases: []string{"g", "gen"},
	Short:   "Generate cURL commands for GraphQL operations defined in the schema",
	Args:    cobra.MaximumNArgs(1), // Only one optional argument for operation name
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

		successColor := color.New(color.FgGreen, color.Bold).SprintFunc()
		errorColor := color.New(color.FgRed, color.Bold).SprintFunc()
		infoColor := color.New(color.FgCyan).SprintFunc()
		cmdColor := color.New(color.FgHiYellow).SprintFunc()

		found := false

		for _, op := range operations {
			if op.Def == nil {
				continue
			}

			for _, field := range op.Def.Fields {
				// Filter by target operation name if specified
				if targetOp != "" && field.Name != targetOp {
					continue
				}

				var finalVars map[string]interface{}
				var err error

				if interactive && len(field.Arguments) > 0 {
					// Create an interactive form for filling in variables based on the field's arguments
					finalVars, err = tui.FillVariablesInteractive(gql.Schema, field.Arguments)
					if err != nil {
						log.Fatalf("❌ Помилка вводу: %v", err)
					}
				} else if varsStr != "" || varsFile != "" {
					finalVars, err = parseVariables(varsStr, varsFile)
				}

				found = true

				// If the --run flag is set, execute the generated query against the endpoint and print the response
				if run {
					fmt.Printf("\n🚀 %s %s...\n", infoColor("Execute request:"), successColor(field.Name))

					result, err := gen.ExecuteQuery(op.OpType, field, finalVars)
					if err != nil {
						fmt.Printf("❌ %s %v\n", errorColor("Execution error:"), err)
					} else {
						fmt.Printf("✅ %s\n%s\n", successColor("Server response:"), result)
					}
				} else {
					curl := gen.GenerateCurl(op.OpType, field, finalVars)

					fmt.Printf("\n# Operation: %s | Field: %s\n%s\n",
						infoColor(op.OpType),
						successColor(field.Name),
						cmdColor(curl),
					)
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
	generateCmd.Flags().StringVarP(&varsStr, "vars", "v", "", "JSON raw with variables (exam. '{\"id\": 1}')")
	generateCmd.Flags().StringVarP(&varsFile, "var-file", "f", "", "Path to a JSON file containing variables (exam. './vars.json')")
	generateCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactively fill in variable")
	// Expose the --run flag to allow users to execute the generated query directly against the endpoint
	generateCmd.Flags().BoolVarP(&run, "run", "r", false, "Connect to the endpoint and execute the generated query, printing the response")
	rootCmd.AddCommand(generateCmd)
}
