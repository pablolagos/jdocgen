// utils/utils.go
package utils

import (
	"go/ast"
	"strings"

	"github.com/pablolagos/jdocgen/models"
)

// ExprToString converts an AST expression to its string representation.
func ExprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + ExprToString(e.X)
	case *ast.ArrayType:
		return "[]" + ExprToString(e.Elt)
	case *ast.SelectorExpr:
		return ExprToString(e.X) + "." + e.Sel.Name
	case *ast.MapType:
		return "map[" + ExprToString(e.Key) + "]" + ExprToString(e.Value)
	case *ast.FuncType:
		return "func" // Simplified
	case *ast.InterfaceType:
		return "interface{}" // Simplified
	case *ast.ChanType:
		return "chan " + ExprToString(e.Value)
	case *ast.Ellipsis:
		return "..." + ExprToString(e.Elt)
	case *ast.BasicLit:
		return e.Value
	case *ast.IndexExpr:
		return ExprToString(e.X) + "[" + ExprToString(e.Index) + "]"
	default:
		return ""
	}
}

// ExtractJSONTag extracts the JSON tag from a struct field tag.
// If no JSON tag is found, it defaults to the field name.
func ExtractJSONTag(tag string, fieldName string) string {
	// Remove backticks
	tag = strings.Trim(tag, "`")
	// Split by space to separate tags
	tags := strings.Split(tag, " ")
	for _, t := range tags {
		if strings.HasPrefix(t, "json:") {
			// Extract value within quotes
			jsonTag := strings.TrimPrefix(t, "json:")
			jsonTag = strings.Trim(jsonTag, `"`)
			// Handle omitempty and other options
			jsonParts := strings.Split(jsonTag, ",")
			if len(jsonParts) > 0 && jsonParts[0] != "" {
				return jsonParts[0]
			}
			break
		}
	}
	return fieldName
}

// IsBasicType checks if a given type is a basic Go type.
func IsBasicType(typ string) bool {
	basicTypes := []string{
		"bool", "string",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune",
		"float32", "float64",
		"complex64", "complex128",
	}
	for _, bt := range basicTypes {
		if typ == bt {
			return true
		}
	}
	return false
}

// ResolveType extracts the base type and package from a given type string.
// For example, "reports.ReportItem" returns ("ReportItem", "reports")
func ResolveType(typ string) (baseType string, pkg string) {
	if strings.Contains(typ, ".") {
		parts := strings.Split(typ, ".")
		if len(parts) != 2 {
			return "", ""
		}
		return parts[1], parts[0]
	}
	return typ, ""
}

// ParseGenericType parses a generic type string and returns the base type and type arguments.
// For example, "Pagination[ReportItem]" returns ("Pagination", ["ReportItem"])
func ParseGenericType(typ string) (string, []string) {
	start := strings.Index(typ, "[")
	end := strings.LastIndex(typ, "]")
	if start == -1 || end == -1 || end <= start+1 {
		return typ, nil
	}
	baseType := strings.TrimSpace(typ[:start])
	argsStr := typ[start+1 : end]
	typeArgs := splitTypeArguments(argsStr)
	return baseType, typeArgs
}

// splitTypeArguments splits type arguments considering nested generics.
// For example, "ReportItem, Pair[Details, Info]" returns ["ReportItem", "Pair[Details, Info]"]
func splitTypeArguments(argsStr string) []string {
	var args []string
	var current strings.Builder
	depth := 0
	for _, r := range argsStr {
		switch r {
		case '[':
			depth++
			current.WriteRune(r)
		case ']':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}
	return args
}

// ReplaceTypeParams replaces type parameters in a type string with concrete types.
// For example, replacing "T" with "ReportItem" in "[]T" returns "[]ReportItem"
func ReplaceTypeParams(typ string, typeParams []models.TypeParam, concreteTypes []string) string {
	if len(typeParams) != len(concreteTypes) {
		// Mismatch in type parameters and concrete types
		return typ
	}
	for i, param := range typeParams {
		typ = strings.ReplaceAll(typ, param.Name, concreteTypes[i])
	}
	return typ
}

// SplitQualifiedName splits a fully qualified name like "package.structname" into its package and struct name.
// Returns empty strings if the input is not qualified.
func SplitQualifiedName(qualifiedName string) (pkg string, structName string) {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", qualifiedName // If not qualified, return the structName only
}
