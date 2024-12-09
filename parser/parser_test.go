// parser/parser_test.go
package parser

import (
	"testing"

	"github.com/pablolagos/jdocgen/utils"
)

func TestParseGenericType(t *testing.T) {
	baseType, typeArgs := utils.ParseGenericType("Pagination[ReportItem]")
	if baseType != "Pagination" {
		t.Errorf("Expected baseType 'Pagination', got '%s'", baseType)
	}
	if len(typeArgs) != 1 || typeArgs[0] != "ReportItem" {
		t.Errorf("Expected typeArgs ['ReportItem'], got %v", typeArgs)
	}

	baseType, typeArgs = utils.ParseGenericType("Map[string, int]")
	if baseType != "Map" {
		t.Errorf("Expected baseType 'Map', got '%s'", baseType)
	}
	if len(typeArgs) != 2 || typeArgs[0] != "string" || typeArgs[1] != "int" {
		t.Errorf("Expected typeArgs ['string', 'int'], got %v", typeArgs)
	}

	baseType, typeArgs = utils.ParseGenericType("NonGenericType")
	if baseType != "NonGenericType" {
		t.Errorf("Expected baseType 'NonGenericType', got '%s'", baseType)
	}
	if len(typeArgs) != 0 {
		t.Errorf("Expected typeArgs [], got %v", typeArgs)
	}
}
