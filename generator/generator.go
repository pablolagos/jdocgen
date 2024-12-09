// generator/generator.go
package generator

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/pablolagos/jdocgen/models"
	"github.com/pablolagos/jdocgen/utils"
)

// GenerateDocumentation generates markdown documentation for API endpoints.
// It documents only the structs referenced by API functions with @Command annotations.
// includeRFC determines whether to include JSON-RPC 2.0 specification information.
func GenerateDocumentation(apiFunctions []models.APIFunction, structDefinitions map[models.StructKey]models.StructDefinition, projectInfo models.ProjectInfo, outFile string, includeRFC bool) error {
	// Step 1: Collect the subset of structs to document
	// Note: structsToDocument was previously declared here but is not used in the current implementation.
	// Therefore, it's removed to prevent compilation errors.

	// Step 2: Create the output file
	file, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Step 3: Write Project Info at the top
	fmt.Fprintf(writer, "# %s\n\n", projectInfo.Title)
	fmt.Fprintf(writer, "Version: %s\n\n", projectInfo.Version)
	if projectInfo.Description != "" {
		fmt.Fprintf(writer, "%s\n\n", projectInfo.Description)
	}

	// Optional: Add other project info like Author, License, etc.
	if projectInfo.Author != "" {
		fmt.Fprintf(writer, "**Author:** %s\n\n", projectInfo.Author)
	}
	if projectInfo.License != "" {
		fmt.Fprintf(writer, "**License:** %s\n\n", projectInfo.License)
	}
	if len(projectInfo.Tags) > 0 {
		fmt.Fprintf(writer, "**Tags:** %s\n\n", strings.Join(projectInfo.Tags, ", "))
	}

	// Include JSON-RPC 2.0 specification information if not omitted
	if includeRFC {
		fmt.Fprintf(writer, "## JSON-RPC 2.0 Specification\n\n")
		fmt.Fprintf(writer, "This API adheres to the [JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification).\n\n")
	}

	// Sort API functions for consistent order
	sort.Slice(apiFunctions, func(i, j int) bool {
		return apiFunctions[i].Command < apiFunctions[j].Command
	})

	// Iterate over each API function and write its documentation
	for _, apiFunc := range apiFunctions {
		log.Printf("Documenting API Command: %s", apiFunc.Command)

		// Write Command as a header
		fmt.Fprintf(writer, "## %s\n\n", apiFunc.Command)

		// Write Description
		if apiFunc.Description != "" {
			fmt.Fprintf(writer, "%s\n\n", apiFunc.Description)
		}

		// Write Parameters section
		if len(apiFunc.Parameters) > 0 {
			fmt.Fprintf(writer, "### Parameters:\n\n")
			fmt.Fprintf(writer, "| Name | Type | Description | Required |\n")
			fmt.Fprintf(writer, "|------|------|-------------|----------|\n")
			for _, param := range apiFunc.Parameters {
				required := "Yes"
				if !param.Required {
					required = "No"
				}
				// Escape pipe characters in descriptions to prevent markdown table issues
				description := strings.ReplaceAll(param.Description, "|", "\\|")
				fmt.Fprintf(writer, "| %s | %s | %s | %s |\n", param.Name, param.Type, description, required)
			}
			fmt.Fprintf(writer, "\n")
		}

		// Write Results section
		if len(apiFunc.Results) > 0 {
			fmt.Fprintf(writer, "### Results:\n\n")
			fmt.Fprintf(writer, "| Name | Type | Description |\n")
			fmt.Fprintf(writer, "|------|------|-------------|\n")
			for _, result := range apiFunc.Results {
				// Escape pipe characters in descriptions to prevent markdown table issues
				description := strings.ReplaceAll(result.Description, "|", "\\|")
				fmt.Fprintf(writer, "| %s | %s | %s |\n", result.Name, result.Type, description)
			}
			fmt.Fprintf(writer, "\n")

			// Include detailed struct definitions for result types that are structs
			for _, result := range apiFunc.Results {
				// Check if the result type is a struct (excluding basic types)
				baseType, _ := utils.ParseGenericType(result.Type)
				if !utils.IsBasicType(baseType) {
					// Attempt to resolve the struct
					// For generic types, handle accordingly
					baseTypeName, _ := utils.ParseGenericType(result.Type)
					structKey := models.StructKey{
						Package: "", // To be determined
						Name:    baseTypeName,
					}

					// Find the struct in structDefinitions
					var found bool
					var resolvedKey models.StructKey
					for key := range structDefinitions {
						if key.Name == structKey.Name {
							resolvedKey = key
							found = true
							break
						}
					}

					if found {
						structDef := structDefinitions[resolvedKey]
						// Write struct details
						fmt.Fprintf(writer, "#### %s.%s\n\n", resolvedKey.Package, structDef.Name)
						if structDef.Description != "" {
							fmt.Fprintf(writer, "%s\n\n", structDef.Description)
						}
						if len(structDef.Fields) > 0 {
							fmt.Fprintf(writer, "| Name | Type | Description | JSON Name |\n")
							fmt.Fprintf(writer, "|------|------|-------------|-----------|\n")
							for _, field := range structDef.Fields {
								description := strings.ReplaceAll(field.Description, "|", "\\|")
								jsonName := field.JSONName
								if jsonName == "-" {
									jsonName = "omitempty"
								}
								fmt.Fprintf(writer, "| %s | %s | %s | %s |\n", field.Name, field.Type, description, jsonName)
							}
							fmt.Fprintf(writer, "\n")
						} else {
							fmt.Fprintf(writer, "_No fields defined._\n\n")
						}
					} else {
						log.Printf("Warning: Struct '%s' not found for result '%s'", baseTypeName, result.Name)
					}
				}
			}
		}

		// Optional: Write Errors section if there are any errors
		if len(apiFunc.Errors) > 0 {
			fmt.Fprintf(writer, "### Errors:\n\n")
			fmt.Fprintf(writer, "| Code | Description |\n")
			fmt.Fprintf(writer, "|------|-------------|\n")
			for _, apiError := range apiFunc.Errors {
				fmt.Fprintf(writer, "| %d | %s |\n", apiError.Code, apiError.Description)
			}
			fmt.Fprintf(writer, "\n")
		}

		// Add a horizontal rule to separate API functions
		fmt.Fprintf(writer, "---\n\n")
	}

	// Flush the buffer to ensure all content is written
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}

	log.Printf("Documentation successfully generated at %s", outFile)
	return nil
}

// collectStructsToDocument determines which structs should be documented based on API functions.
// It includes all structs referenced by API functions' Parameters and Results, recursively.
func collectStructsToDocument(apiFunctions []models.APIFunction, structDefinitions map[models.StructKey]models.StructDefinition) map[models.StructKey]struct{} {
	// Previously, structsToDocument was used here, but since it's not utilized in GenerateDocumentation,
	// this function can be removed or left as is if you plan to use it in the future.
	// For now, it's retained but not used to avoid affecting other parts of the code.
	structsToDocument := make(map[models.StructKey]struct{})
	visited := make(map[models.StructKey]struct{})

	for _, apiFunc := range apiFunctions {
		// Collect types from Parameters
		for _, param := range apiFunc.Parameters {
			collectStructsFromType(param.Type, apiFunc.PackageName, apiFunc.ImportAliases, structDefinitions, structsToDocument, visited)
		}

		// Collect types from Results
		for _, result := range apiFunc.Results {
			collectStructsFromType(result.Type, apiFunc.PackageName, apiFunc.ImportAliases, structDefinitions, structsToDocument, visited)
		}
	}

	return structsToDocument
}

// collectStructsFromType recursively collects structs referenced by a given type.
func collectStructsFromType(typ string, currentPackage string, importAliases map[string]string, structDefinitions map[models.StructKey]models.StructDefinition, structsToDocument map[models.StructKey]struct{}, visited map[models.StructKey]struct{}) {
	baseType, typeArgs := utils.ParseGenericType(typ)

	// Skip basic types
	if utils.IsBasicType(baseType) {
		return
	}

	pkg, typeName := resolvePackageAndType(baseType, currentPackage, importAliases, structDefinitions)
	if typeName == "" {
		// Cannot resolve type, skip
		log.Printf("Warning: Cannot resolve type '%s'", baseType)
		return
	}

	key := models.StructKey{
		Package: pkg,
		Name:    typeName,
	}

	if _, exists := structsToDocument[key]; !exists {
		structsToDocument[key] = struct{}{}
	}

	// If generic, process type arguments
	if len(typeArgs) > 0 {
		for _, arg := range typeArgs {
			collectStructsFromType(arg, currentPackage, importAliases, structDefinitions, structsToDocument, visited)
		}
	}

	// Now, traverse fields of this struct to collect referenced structs
	if _, seen := visited[key]; seen {
		return
	}
	visited[key] = struct{}{}

	structDef, exists := structDefinitions[key]
	if !exists {
		log.Printf("Warning: Struct '%s.%s' not found in definitions", key.Package, key.Name)
		return
	}

	for _, field := range structDef.Fields {
		collectStructsFromType(field.Type, key.Package, map[string]string{}, structDefinitions, structsToDocument, visited)
	}
}

// resolvePackageAndType resolves the package and type name for a given type.
// It handles fully qualified types and uses import aliases.
// If the type is unqualified, it assigns it to the current package if it exists there.
func resolvePackageAndType(typ string, currentPackage string, importAliases map[string]string, structDefinitions map[models.StructKey]models.StructDefinition) (pkg string, typeName string) {
	if strings.Contains(typ, ".") {
		// Type is fully qualified
		parts := strings.Split(typ, ".")
		if len(parts) != 2 {
			return "", ""
		}
		alias := parts[0]
		typeName = parts[1]
		pkgAlias, exists := importAliases[alias]
		if exists {
			return pkgAlias, typeName
		}
		// If alias not found, assume alias is the package name
		pkgAlias = alias
		return pkgAlias, typeName
	}

	// Type is unqualified; check if it exists in the current package
	key := models.StructKey{
		Package: currentPackage,
		Name:    typ,
	}
	if _, exists := structDefinitions[key]; exists {
		return currentPackage, typ
	}

	// If not found in the current package, it's likely from another package without a prefix
	// Log a warning and return empty strings
	log.Printf("Type '%s' not found in package '%s'. Ensure it is imported or fully qualified.", typ, currentPackage)
	return "", ""
}
