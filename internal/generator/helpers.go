package generator

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
)

func getScalarDefault(name string) interface{} {
	switch name {
	case "String", "ID":
		return fmt.Sprintf("<%s>", name)
	case "Int", "Float":
		return 0
	case "Boolean":
		return false
	default:
		return fmt.Sprintf("<%s>", name)
	}
}

func (g *Generator) isLeafType(t *ast.Definition) bool {
	return t.Kind == ast.Scalar || t.Kind == ast.Enum
}
