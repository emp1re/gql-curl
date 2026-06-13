package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TylerBrock/colorjson"
	"github.com/emp1re/gql-curl/internal/config"
	"github.com/emp1re/gql-curl/internal/generator"
	"github.com/emp1re/gql-curl/internal/parser"
	"github.com/emp1re/gql-curl/internal/tui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	run         bool
	varsStr     string
	varsFile    string
	interactive bool
	filterStr   string
	genSchema   string
	genFormat   string
)

type generateOutputFormat string

const (
	generateFormatCurl       generateOutputFormat = "curl"
	generateFormatPayload    generateOutputFormat = "payload"
	generateFormatPlayground generateOutputFormat = "playground"
)

// parseVariables is a helper function that takes either a raw JSON string or a file path to a JSON file, and parses it into a map[string]interface{}.
// It ensures that both inputs are not provided at the same time and provides detailed error messages for various failure scenarios.
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
	Short:   "Generate GraphQL requests from schema operations",
	Long: `Generate ready-to-copy requests for top-level GraphQL query and mutation fields.

By default the command prints curl requests. Use --format postman for a raw JSON
request body, or --format playground for separate query and variables blocks.`,
	Example: `  gqc generate
  gqc generate getUser
  gqc generate getUser --schema main
  gqc generate getUser --format postman
  gqc generate createUser --format playground
  gqc generate getUser --vars '{"id":"123"}'
  gqc generate createUser --interactive
  gqc generate getUser --run --filter 'data.getUser.name'`,
	Args: maximumNArgsWithHelp(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := config.LoadConfig("graphql.curl.yaml")
		if err != nil {
			return commandError(cmd, "load config error: %v", err)
		}

		// Retrieve the target operation name from the command-line arguments, if provided
		targetOp := ""
		if len(args) > 0 {
			targetOp = args[0]
		}

		outputFormat, err := normalizeGenerateFormat(genFormat)
		if err != nil {
			return commandError(cmd, "%v", err)
		}

		schemas, err := cfg.SelectedSchemas(genSchema)
		if err != nil {
			return commandError(cmd, "config error: %v", err)
		}

		successColor := color.New(color.FgGreen, color.Bold).SprintFunc()
		errorColor := color.New(color.FgRed, color.Bold).SprintFunc()
		infoColor := color.New(color.FgCyan).SprintFunc()
		cmdColor := color.New(color.FgYellow).SprintFunc()

		found := false

		for _, schemaCfg := range schemas {
			// Read and parse GraphQL schema sources for the current configured schema.
			gql, err := parser.NewParserFromPaths([]string(schemaCfg.Config.Path))
			if err != nil {
				return commandError(cmd, "parse schema %q error: %v", schemaCfg.Name, err)
			}

			gen := generator.NewGenerator(gql.Schema, schemaCfg.Config.Endpoint, schemaCfg.Config.Headers)

			operations := []struct {
				OpType string
				Def    *ast.Definition
			}{
				{"query", gql.Schema.Query},
				{"mutation", gql.Schema.Mutation},
			}

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
							return commandError(cmd, "input error: %v", err)
						}
					} else if varsStr != "" || varsFile != "" {
						finalVars, err = parseVariables(varsStr, varsFile)
						if err != nil {
							return commandError(cmd, "%v", err)
						}
					}

					found = true

					// If the --run flag is set, execute the generated query against the endpoint and print the response
					if run {
						fmt.Printf("\n🚀 %s %s.%s...\n", infoColor("Execute request:"), successColor(schemaCfg.Name), successColor(field.Name))

						generatedOutput, err := buildGenerateOutput(gen, op.OpType, field, finalVars, outputFormat)
						if err != nil {
							return commandError(cmd, "generation error: %v", err)
						}
						fmt.Printf("%s\n\n", colorGeneratedOutput(generatedOutput, outputFormat, cmdColor))

						resultRaw, metrics, err := gen.ExecuteQuery(op.OpType, field, finalVars)
						if err != nil {
							return commandError(cmd, "%s %v", errorColor("execution error:"), err)
						} else {
							fmt.Printf("✅ %s\n", successColor("Server request:"))

							// Filter the response using gjson if a filter string is provided; otherwise, print the entire response colorized
							if filterStr != "" {
								parsed := gjson.Get(resultRaw, filterStr)

								if !parsed.Exists() {
									fmt.Printf("⚠️ %s Path '%s' not found in response\n", errorColor("Attention:"), filterStr)
								} else {
									// If the filtered result is an object or array, print it colorized; otherwise, print it as a raw string (useful for bash scripts)
									if parsed.IsObject() || parsed.IsArray() {
										printColorized(parsed.Raw)
									} else {
										// Raw string output for non-object/array results, which is useful for command-line usage (e.g., in bash scripts)
										fmt.Println(parsed.String())
									}
								}
							} else {
								printColorized(resultRaw)
							}
							if metrics != nil {
								metricColor := color.New(color.FgHiMagenta).SprintFunc()
								valColor := color.New(color.FgWhite, color.Bold).SprintFunc()

								fmt.Printf("\n%s\n", metricColor("📊 Performance Metrics:"))
								fmt.Printf("  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s\n",
									metricColor("Total:"), valColor(metrics.Total.Round(time.Millisecond)),
									metricColor("TTFB:"), valColor(metrics.TTFB.Round(time.Millisecond)),
									metricColor("DNS:"), valColor(metrics.DNS.Round(time.Millisecond)),
									metricColor("TCP:"), valColor(metrics.TCP.Round(time.Millisecond)),
									metricColor("TLS:"), valColor(metrics.TLS.Round(time.Millisecond)),
									metricColor("Size:"), valColor(formatBytes(metrics.Size)),
								)
							}
						}
					} else {
						generatedOutput, err := buildGenerateOutput(gen, op.OpType, field, finalVars, outputFormat)
						if err != nil {
							return commandError(cmd, "generation error: %v", err)
						}

						fmt.Printf("\n# Schema: %s | Operation: %s | Field: %s\n%s\n",
							successColor(schemaCfg.Name),
							infoColor(op.OpType),
							successColor(field.Name),
							colorGeneratedOutput(generatedOutput, outputFormat, cmdColor),
						)
					}
				}
			}
		}

		// If a target operation was specified but not found in the schema, log an error
		if targetOp != "" && !found {
			return commandError(cmd, "operation %q not found in schema", targetOp)
		}

		return nil
	},
}

func normalizeGenerateFormat(format string) (generateOutputFormat, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "curl":
		return generateFormatCurl, nil
	case "payload", "json", "postman":
		return generateFormatPayload, nil
	case "playground", "pg":
		return generateFormatPlayground, nil
	default:
		return "", fmt.Errorf("unknown output format %q, expected curl, payload/json/postman, or playground", format)
	}
}

func buildGenerateOutput(gen *generator.Generator, opType string, field *ast.FieldDefinition, vars map[string]interface{}, format generateOutputFormat) (string, error) {
	switch format {
	case generateFormatCurl:
		return gen.GenerateCurl(opType, field, vars), nil
	case generateFormatPayload:
		return gen.GeneratePayloadJSON(opType, field, vars, true)
	case generateFormatPlayground:
		variablesJSON, err := gen.GenerateVariablesJSON(field, vars, true)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("# Query\n%s\n\n# Variables\n%s", gen.BuildOperation(opType, field), variablesJSON), nil
	default:
		return "", fmt.Errorf("unsupported output format %q", format)
	}
}

func colorGeneratedOutput(output string, format generateOutputFormat, colorize func(a ...interface{}) string) string {
	switch format {
	case generateFormatCurl:
		return colorize(output)
	case generateFormatPlayground:
		return colorPlaygroundOutput(output)
	default:
		return output
	}
}

func colorPlaygroundOutput(output string) string {
	if color.NoColor {
		return output
	}

	const separator = "\n\n# Variables\n"
	parts := strings.SplitN(output, separator, 2)
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "# Query\n") {
		return output
	}

	headerColor := color.New(color.FgCyan, color.Bold).SprintFunc()
	queryColor := color.New(color.FgYellow).SprintFunc()

	query := strings.TrimPrefix(parts[0], "# Query\n")
	return fmt.Sprintf("%s\n%s\n\n%s\n%s",
		headerColor("# Query"),
		queryColor(query),
		headerColor("# Variables"),
		colorizeJSONString(parts[1]),
	)
}

func printColorized(rawJSON string) {
	var obj interface{}
	if err := json.Unmarshal([]byte(rawJSON), &obj); err != nil {
		fmt.Println(rawJSON)
		return
	}

	f := colorjson.NewFormatter()
	f.Indent = 2
	colored, _ := f.Marshal(obj)
	fmt.Println(string(colored))
}

func colorizeJSONString(rawJSON string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(rawJSON), &obj); err != nil {
		return rawJSON
	}

	f := colorjson.NewFormatter()
	f.Indent = 2
	colored, err := f.Marshal(obj)
	if err != nil {
		return rawJSON
	}

	return string(colored)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
