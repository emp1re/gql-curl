package generator

import "fmt"

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
