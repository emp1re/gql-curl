package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type GQLParser struct {
	Schema *ast.Schema
}

func NewParserFromDir(rootPath string) (*GQLParser, error) {
	var sources []*ast.Source

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("access path %s failed: %w", path, err)
		}
		if d.IsDir() {
			return nil // skip directories
		}

		// Check file extension (case-insensitive)
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".graphql" || ext == ".graphqls" {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read file %s failed: %w", path, err)
			}

			sources = append(sources, &ast.Source{
				Name:  path,
				Input: string(content),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no .graphql or .graphqls files found in %s", rootPath)
	}

	// Loading schema from collected sources
	// If there are multiple files, gqlparser will merge them into a single schema
	schema, gqlErr := gqlparser.LoadSchema(sources...)
	if gqlErr != nil {
		return nil, fmt.Errorf("schema validation error: %s", gqlErr.Error())
	}

	return &GQLParser{Schema: schema}, nil
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
