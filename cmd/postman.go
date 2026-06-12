package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/emp1re/gql-curl/internal/generator"
	"github.com/emp1re/gql-curl/internal/parser"
	postmancol "github.com/emp1re/gql-curl/internal/postman"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/ast"
)

var (
	postmanSchema string
	postmanFile   string
	postmanOut    string
	postmanName   string
)

type postmanSourceGroup struct {
	name  string
	items []postmancol.Item
}

var postmanCmd = &cobra.Command{
	Use:     "postman [flags]",
	Aliases: []string{"pm"},
	Short:   "Generate a Postman collection from schema operations",
	Long: `Generate a Postman Collection v2.1 JSON file from configured GraphQL schemas.

By default the command includes all configured schemas from graphql.curl.yaml.
Use --schema to generate one configured schema, or --file to include only query
and mutation fields declared in one schema file such as center.graphqls.`,
	Example: `  gqc postman
  gqc postman --schema main
  gqc postman --schema main --file center.graphqls
  gqc postman --schema main --file center.graphqls --out center.postman_collection.json
  gqc postman --out -`,
	Args: noArgsWithHelp,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("graphql.curl.yaml")
		if err != nil {
			return commandError(cmd, "load config error: %v", err)
		}

		schemas, err := cfg.SelectedSchemas(postmanSchema)
		if err != nil {
			return commandError(cmd, "config error: %v", err)
		}

		items := make([]postmancol.Item, 0, len(schemas))
		totalRequests := 0
		matchedFile := postmanFile == ""

		for _, schemaCfg := range schemas {
			schemaFolder, requestCount, schemaMatchedFile, err := buildPostmanSchemaFolder(schemaCfg, postmanFile)
			if err != nil {
				return commandError(cmd, "%v", err)
			}

			if schemaMatchedFile {
				matchedFile = true
			}
			if requestCount == 0 {
				continue
			}

			items = append(items, schemaFolder)
			totalRequests += requestCount
		}

		if postmanFile != "" && !matchedFile {
			return commandError(cmd, "schema file %q was not found in selected schema paths", postmanFile)
		}
		if totalRequests == 0 {
			if postmanFile != "" {
				return commandError(cmd, "no query or mutation fields found in schema file %q", postmanFile)
			}
			return commandError(cmd, "no query or mutation fields found in selected schemas")
		}

		collectionName := postmanName
		if strings.TrimSpace(collectionName) == "" {
			collectionName = defaultPostmanCollectionName(postmanSchema)
		}

		collection := postmancol.NewCollection(collectionName, items)
		data, err := postmancol.Encode(collection)
		if err != nil {
			return commandError(cmd, "encode Postman collection failed: %v", err)
		}

		outPath := strings.TrimSpace(postmanOut)
		if outPath == "" {
			outPath = "postman_collection.json"
		}

		if outPath == "-" {
			fmt.Print(string(data))
			return nil
		}

		if err := writePostmanCollection(outPath, data); err != nil {
			return commandError(cmd, "write Postman collection failed: %v", err)
		}

		fmt.Printf("✅ Postman collection written to: %s (%d requests)\n", outPath, totalRequests)
		return nil
	},
}

func init() {
	postmanCmd.Flags().StringVarP(&postmanSchema, "schema", "s", "", "Use one schema from config.schemas instead of all schemas")
	postmanCmd.Flags().StringVarP(&postmanFile, "file", "f", "", "Include only operations declared in this schema file, for example center.graphqls")
	postmanCmd.Flags().StringVarP(&postmanOut, "out", "o", "postman_collection.json", "Output collection path, or '-' to print JSON to stdout")
	postmanCmd.Flags().StringVarP(&postmanName, "name", "n", "", "Postman collection name")
	rootCmd.AddCommand(postmanCmd)
}

func buildPostmanSchemaFolder(schemaCfg config.NamedSchema, fileFilter string) (postmancol.Item, int, bool, error) {
	schemaPaths := []string(schemaCfg.Config.Path)
	gql, err := parser.NewParserFromPaths(schemaPaths)
	if err != nil {
		return postmancol.Item{}, 0, false, fmt.Errorf("parse schema %q error: %w", schemaCfg.Name, err)
	}

	gen := generator.NewGenerator(gql.Schema, schemaCfg.Config.Endpoint, schemaCfg.Config.Headers)
	headers := buildPostmanHeaders(schemaCfg.Config.Headers)
	groups := make(map[string]*postmanSourceGroup)
	sourceOrder := make([]string, 0, len(gql.Sources))
	matchedFile := fileFilter == ""

	for _, source := range gql.Sources {
		if fileFilter != "" && !sourceMatchesFileFilter(source.Name, schemaPaths, fileFilter) {
			continue
		}

		matchedFile = true
		addPostmanSourceGroup(groups, &sourceOrder, source.Name, sourceDisplayName(source.Name, schemaPaths))
	}

	appendOperation := func(opType string, def *ast.Definition) error {
		if def == nil {
			return nil
		}

		for _, field := range def.Fields {
			sourceName := fieldSourceName(field)
			if sourceName == "" {
				sourceName = schemaCfg.Name
			}
			if fileFilter != "" && !sourceMatchesFileFilter(sourceName, schemaPaths, fileFilter) {
				continue
			}

			group := addPostmanSourceGroup(groups, &sourceOrder, sourceName, sourceDisplayName(sourceName, schemaPaths))
			variablesJSON, err := gen.GenerateVariablesJSON(field, nil, true)
			if err != nil {
				return fmt.Errorf("generate variables for %s %s failed: %w", opType, field.Name, err)
			}

			requestName := fmt.Sprintf("%s %s", opType, field.Name)
			group.items = append(group.items, postmancol.NewGraphQLRequestItem(
				requestName,
				schemaCfg.Config.Endpoint,
				headers,
				gen.BuildOperation(opType, field),
				variablesJSON,
			))
		}

		return nil
	}

	if err := appendOperation("query", gql.Schema.Query); err != nil {
		return postmancol.Item{}, 0, matchedFile, err
	}
	if err := appendOperation("mutation", gql.Schema.Mutation); err != nil {
		return postmancol.Item{}, 0, matchedFile, err
	}

	fileFolders := make([]postmancol.Item, 0, len(sourceOrder))
	requestCount := 0
	for _, sourceName := range sourceOrder {
		group := groups[sourceName]
		if group == nil || len(group.items) == 0 {
			continue
		}

		requestCount += len(group.items)
		fileFolders = append(fileFolders, postmancol.NewFolder(group.name, group.items))
	}

	return postmancol.NewFolder(schemaCfg.Name, fileFolders), requestCount, matchedFile, nil
}

func addPostmanSourceGroup(groups map[string]*postmanSourceGroup, sourceOrder *[]string, sourceName, folderName string) *postmanSourceGroup {
	if group, ok := groups[sourceName]; ok {
		return group
	}

	group := &postmanSourceGroup{name: folderName}
	groups[sourceName] = group
	*sourceOrder = append(*sourceOrder, sourceName)
	return group
}

func buildPostmanHeaders(headers map[string]string) []postmancol.Header {
	keys := make([]string, 0, len(headers))
	hasContentType := false
	for key := range headers {
		keys = append(keys, key)
		if strings.EqualFold(key, "Content-Type") {
			hasContentType = true
		}
	}
	sort.Strings(keys)

	result := make([]postmancol.Header, 0, len(keys)+1)
	for _, key := range keys {
		result = append(result, postmancol.Header{
			Key:   key,
			Value: headers[key],
			Type:  "text",
		})
	}

	if !hasContentType {
		result = append(result, postmancol.Header{
			Key:   "Content-Type",
			Value: "application/json",
			Type:  "text",
		})
	}

	return result
}

func fieldSourceName(field *ast.FieldDefinition) string {
	if field == nil || field.Position == nil || field.Position.Src == nil {
		return ""
	}

	return field.Position.Src.Name
}

func sourceMatchesFileFilter(sourceName string, schemaPaths []string, fileFilter string) bool {
	fileFilter = strings.TrimSpace(fileFilter)
	if fileFilter == "" {
		return true
	}

	if normalizePathForMatch(sourceName) == normalizePathForMatch(fileFilter) {
		return true
	}

	if !strings.Contains(fileFilter, "/") && !strings.Contains(fileFilter, string(filepath.Separator)) && filepath.Base(sourceName) == fileFilter {
		return true
	}

	displayName := sourceDisplayName(sourceName, schemaPaths)
	if normalizePathForMatch(displayName) == normalizePathForMatch(fileFilter) {
		return true
	}

	absFilter, err := filepath.Abs(fileFilter)
	if err == nil {
		absSource, sourceErr := filepath.Abs(sourceName)
		if sourceErr == nil && normalizePathForMatch(absSource) == normalizePathForMatch(absFilter) {
			return true
		}
	}

	return false
}

func sourceDisplayName(sourceName string, schemaPaths []string) string {
	sourceAbs, err := filepath.Abs(sourceName)
	if err != nil {
		return filepath.ToSlash(filepath.Base(sourceName))
	}

	best := filepath.Base(sourceName)
	bestLen := len(best)

	for _, schemaPath := range schemaPaths {
		schemaPath = strings.TrimSpace(schemaPath)
		if schemaPath == "" {
			continue
		}

		rootPath := schemaPath
		info, err := os.Stat(rootPath)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			rootPath = filepath.Dir(rootPath)
		}

		rootAbs, err := filepath.Abs(rootPath)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(rootAbs, sourceAbs)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if bestLen == 0 || len(rel) < bestLen || best == filepath.Base(sourceName) {
			best = rel
			bestLen = len(rel)
		}
	}

	return filepath.ToSlash(best)
}

func normalizePathForMatch(path string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
}

func defaultPostmanCollectionName(schemaName string) string {
	if strings.TrimSpace(schemaName) != "" {
		return schemaName + " GraphQL"
	}

	return "GraphQL APIs"
}

func writePostmanCollection(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, data, 0644)
}
