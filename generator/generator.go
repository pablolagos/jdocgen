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
// Now, we print structs inline, immediately after referencing them in the results section.
// If a struct references other structs, we document those inline as well.
func GenerateDocumentation(apiFunctions []models.APIFunction, structDefinitions map[models.StructKey]models.StructDefinition, projectInfo models.ProjectInfo, outFile string, includeRFC bool) error {
	file, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write Project Info at the top
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

	visited := make(map[models.StructKey]bool) // Keep track of visited structs to avoid duplicates

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
				description := strings.ReplaceAll(result.Description, "|", "\\|")
				fmt.Fprintf(writer, "| %s | %s | %s |\n", result.Name, result.Type, description)
			}
			fmt.Fprintf(writer, "\n")

			// For each result, if it's a struct type (not basic), print it inline along with its referenced structs
			for _, result := range apiFunc.Results {
				baseType, typeArgs := utils.ParseGenericType(result.Type)
				if !utils.IsBasicType(baseType) {
					// Try to find the concrete struct definition
					concreteType := result.Type

					// Find the struct in structDefinitions
					var found bool
					var resolvedKey models.StructKey
					for key := range structDefinitions {
						if key.Name == concreteType {
							resolvedKey = key
							found = true
							break
						}
					}

					if !found && len(typeArgs) == 0 {
						// If not a generic instantiation, try to find the base type
						for key := range structDefinitions {
							if key.Name == baseType {
								resolvedKey = key
								found = true
								break
							}
						}
					}

					if found {
						// Print the struct and all referenced structs inline
						printStructDefinitionInline(writer, resolvedKey, structDefinitions, visited)
					} else {
						log.Printf("Warning: Struct '%s' not found for result '%s'", concreteType, result.Name)
					}
				}
			}
		}

		// Errors section
		if len(apiFunc.Errors) > 0 {
			fmt.Fprintf(writer, "### Errors:\n\n")
			fmt.Fprintf(writer, "| Code | Description |\n")
			fmt.Fprintf(writer, "|------|-------------|\n")
			for _, apiError := range apiFunc.Errors {
				fmt.Fprintf(writer, "| %d | %s |\n", apiError.Code, apiError.Description)
			}
			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "---\n\n")
	}

	// Flush the buffer to ensure all content is written
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}

	log.Printf("Documentation successfully generated at %s", outFile)
	return nil
}

// printStructDefinitionInline prints a given struct's definition and all referenced structs inline.
// It uses a visited map to avoid duplicates.
func printStructDefinitionInline(writer *bufio.Writer, key models.StructKey, structDefinitions map[models.StructKey]models.StructDefinition, visited map[models.StructKey]bool) {
	if visited[key] {
		return
	}
	visited[key] = true

	structDef, exists := structDefinitions[key]
	if !exists {
		log.Printf("Warning: Struct '%s.%s' not found in definitions.", key.Package, key.Name)
		return
	}

	fmt.Fprintf(writer, "#### %s.%s\n\n", key.Package, structDef.Name)
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

	// Now, for each field, if it's a struct type, print it inline
	for _, field := range structDef.Fields {
		baseType, typeArgs := utils.ParseGenericType(field.Type)
		if utils.IsBasicType(baseType) {
			continue
		}

		// Resolve the field type
		fieldPkg, fieldTypeName := resolvePackageAndType(baseType, key.Package, map[string]string{}, structDefinitions)
		if fieldTypeName == "" {
			// Cannot resolve type, skip
			continue
		}

		// If this is a generic instantiation, construct the concrete type name
		var concreteType string
		if len(typeArgs) > 0 {
			concreteType = fmt.Sprintf("%s[%s]", fieldTypeName, strings.Join(typeArgs, ", "))
		} else {
			concreteType = fieldTypeName
		}

		var found bool
		var fieldResolvedKey models.StructKey
		for k := range structDefinitions {
			if k.Name == concreteType {
				fieldResolvedKey = k
				found = true
				break
			}
		}

		if !found && len(typeArgs) == 0 {
			// If not found as a generic instantiation, try base type
			for k := range structDefinitions {
				if k.Name == fieldTypeName && (fieldPkg == "" || k.Package == fieldPkg) {
					fieldResolvedKey = k
					found = true
					break
				}
			}
		}

		if found {
			printStructDefinitionInline(writer, fieldResolvedKey, structDefinitions, visited)
		}
	}
}

// resolvePackageAndType resolves the package and type name for a given type.
// If the type is unqualified, it assumes it's in the current package if it exists there.
func resolvePackageAndType(typ string, currentPackage string, importAliases map[string]string, structDefinitions map[models.StructKey]models.StructDefinition) (pkg string, typeName string) {
	if strings.Contains(typ, ".") {
		// Fully qualified type
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

	// Unqualified type
	key := models.StructKey{
		Package: currentPackage,
		Name:    typ,
	}
	if _, exists := structDefinitions[key]; exists {
		return currentPackage, typ
	}

	// Not found in current package
	log.Printf("Type '%s' not found in package '%s'. Ensure it is imported or fully qualified.", typ, currentPackage)
	return "", ""
}
