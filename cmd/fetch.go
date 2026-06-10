package cmd

import (
	"context"
	"fmt"
	"log"
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
	Short:   "Fetch the GraphQL schema using Introspection and save it to a file",
	Run: func(cmd *cobra.Command, args []string) {
		infoColor := color.New(color.FgCyan).SprintFunc()
		successColor := color.New(color.FgGreen, color.Bold).SprintFunc()
		errorColor := color.New(color.FgRed, color.Bold).SprintFunc()

		// 1. Load configuration
		cfg, err := config.LoadConfig("graphql.curl.yaml")
		if err != nil {
			log.Fatalf("❌ %s %v", errorColor("Load config error:"), err)
		}

		schemas, err := cfg.SelectedSchemas(fetchSchema)
		if err != nil {
			log.Fatalf("❌ %s %v", errorColor("Config error:"), err)
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
				log.Fatalf("❌ %s %v\nCheck if the endpoint is correct and accessible, and if the required headers are set in the configuration.", errorColor("Failed to fetch schema:"), err)
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
					log.Fatalf("❌ %s %v", errorColor("Failed to save schema to file:"), err)
				}

				fmt.Printf("✅ %s %s\n", successColor("Schema successfully saved to:"), outPath)
			}
		}
	},
}

func init() {
	fetchCmd.Flags().StringVarP(&fetchSchema, "schema", "s", "", "Schema name from config.schemas to use (default: all)")
	rootCmd.AddCommand(fetchCmd)
}
