package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emp1re/gql-curl/internal/config"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type GQLParser struct {
	Schema *ast.Schema
}

func NewParserFromDir(rootPath string) (*GQLParser, error) {
	return NewParserFromPaths([]string{rootPath})
}

func NewParserFromPaths(rootPaths []string) (*GQLParser, error) {
	sources, searchedPaths, err := collectSourcesFromPaths(rootPaths)
	if err != nil {
		return nil, err
	}

	return newParserFromSources(sources, searchedPaths)
}

func collectSourcesFromPaths(rootPaths []string) ([]*ast.Source, []string, error) {
	var sources []*ast.Source
	searchedPaths := normalizeRootPaths(rootPaths)
	if len(searchedPaths) == 0 {
		return nil, nil, fmt.Errorf("schema path is not configured")
	}

	for _, rootPath := range searchedPaths {
		if err := collectSourcesFromPath(rootPath, &sources); err != nil {
			return nil, nil, err
		}
	}

	return sources, searchedPaths, nil
}

func newParserFromSources(sources []*ast.Source, searchedPaths []string) (*GQLParser, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no %s files found in %s, set 'document_extensions' in config if your schema files have different extensions", strings.Join(config.DocumentExtensions, ", "), strings.Join(searchedPaths, ", "))
	}

	// Loading schema from collected sources
	// If there are multiple files, gqlparser will merge them into a single schema
	schema, gqlErr := gqlparser.LoadSchema(sources...)
	if gqlErr != nil {
		return nil, fmt.Errorf("schema validation error: %s", gqlErr.Error())
	}

	return &GQLParser{Schema: schema}, nil
}

func normalizeRootPaths(rootPaths []string) []string {
	var normalized []string
	for _, rootPath := range rootPaths {
		rootPath = strings.TrimSpace(rootPath)
		if rootPath != "" {
			normalized = append(normalized, rootPath)
		}
	}

	return normalized
}

func collectSourcesFromPath(rootPath string, sources *[]*ast.Source) error {
	info, err := os.Stat(rootPath)
	if err != nil {
		return fmt.Errorf("access schema path %s failed: %w", rootPath, err)
	}

	if !info.IsDir() {
		return appendSource(rootPath, sources)
	}

	return filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("access path %s failed: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}

		return appendSource(path, sources)
	})
}

func appendSource(path string, sources *[]*ast.Source) error {
	// Check file extension case-insensitively.
	ext := strings.ToLower(filepath.Ext(path))
	if !slices.Contains(config.DocumentExtensions, ext) {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file %s failed: %w", path, err)
	}

	*sources = append(*sources, &ast.Source{
		Name:  path,
		Input: string(content),
	})

	return nil
}

func (p *GQLParser) GetOperations() map[string]*ast.Definition {
	ops := make(map[string]*ast.Definition)

	if p.Schema.Query != nil {
		ops["query"] = p.Schema.Query
	}
	if p.Schema.Mutation != nil {
		ops["mutation"] = p.Schema.Mutation
	}

	return ops
}
