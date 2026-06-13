package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/suessflorian/gqlfetch"

	"github.com/emp1re/gql-curl/internal/config"
)

var fetchSchema string

var fetchCmd = &cobra.Command{
	Use:     "fetch",
	Aliases: []string{"f", "pull"},
	Short:   "Fetch a GraphQL schema with introspection",
	Long: `Fetch GraphQL schema SDL from configured endpoints using introspection.

The command reads endpoint and headers from graphql.curl.yaml. If a schema path
points to a directory, the fetched schema is saved as schema.graphql inside it.`,
	Example: `  gqc fetch
  gqc fetch --schema main
  gqc pull -s api`,
	Args: noArgsWithHelp,
	RunE: func(cmd *cobra.Command, args []string) error {
		infoColor := color.New(color.FgCyan).SprintFunc()
		successColor := color.New(color.FgGreen, color.Bold).SprintFunc()
		errorColor := color.New(color.FgRed, color.Bold).SprintFunc()

		// 1. Load configuration
		cfg, err := config.LoadConfig("graphql.curl.yaml")
		if err != nil {
			return commandError(cmd, "%s %v", errorColor("load config error:"), err)
		}

		schemas, err := cfg.SelectedSchemas(fetchSchema)
		if err != nil {
			return commandError(cmd, "%s %v", errorColor("config error:"), err)
		}

		for _, schemaCfg := range schemas {
			fmt.Printf("📡 %s %s (%s)\n", infoColor("Requesting schema from endpoint:"), schemaCfg.Config.Endpoint, schemaCfg.Name)

			// 2. Form headers from config
			headers := make(http.Header)
			for k, v := range schemaCfg.Config.Headers {
				headers.Set(k, v)
			}

			// 3. Fetch schema with timeout context
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			schemaData, err := gqlfetch.BuildClientSchemaWithHeaders(ctx, schemaCfg.Config.Endpoint, headers, true)
			cancel()
			if err != nil {
				return commandError(cmd, "%s %v\nCheck if the endpoint is correct and accessible, and if the required headers are set in the configuration.", errorColor("failed to fetch schema:"), err)
			}

			// 4. Determine output path
			outPaths := []string(schemaCfg.Config.Path)
			if len(outPaths) == 0 {
				outPaths = []string{schemaCfg.Name + ".graphql"}
			}

			for _, outPath := range outPaths {
				fileInfo, err := os.Stat(outPath)

				if err == nil && fileInfo.IsDir() {
					outPath = filepath.Join(outPath, "schema.graphql")
				} else if outPath == "" {
					outPath = schemaCfg.Name + ".graphql"
				}

				err = os.WriteFile(outPath, []byte(schemaData), 0644)
				if err != nil {
					return commandError(cmd, "%s %v", errorColor("failed to save schema to file:"), err)
				}

				fmt.Printf("✅ %s %s\n", successColor("Schema successfully saved to:"), outPath)
			}
		}

		return nil
	},
}

func init() {
	fetchCmd.Flags().StringVarP(&fetchSchema, "schema", "s", "", "Use one schema from config.schemas instead of all schemas")
	registerSchemaFlagCompletion(fetchCmd, "schema")
	rootCmd.AddCommand(fetchCmd)
}
