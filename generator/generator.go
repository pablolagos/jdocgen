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

	if projectInfo.Author != "" {
		fmt.Fprintf(writer, "**Author:** %s\n\n", projectInfo.Author)
	}
	if projectInfo.License != "" {
		fmt.Fprintf(writer, "**License:** %s\n\n", projectInfo.License)
	}
	if len(projectInfo.Tags) > 0 {
		fmt.Fprintf(writer, "**Tags:** %s\n\n", strings.Join(projectInfo.Tags, ", "))
	}

	if includeRFC {
		fmt.Fprintf(writer, "## JSON-RPC 2.0 Specification\n\n")
		fmt.Fprintf(writer, "This API adheres to the [JSON-RPC 2.0 specification](https://www.jsonrpc.org/specification).\n\n")
		fmt.Fprintf(writer, "**Requests:**\n\n")
		fmt.Fprintf(writer, "Clients must send a JSON object containing the following fields:\n")
		fmt.Fprintf(writer, "- `jsonrpc`: Must be the string \"2.0\".\n")
		fmt.Fprintf(writer, "- `method`: The name of the method to invoke.\n")
		fmt.Fprintf(writer, "- `params`: (Optional) A structured value containing method parameters.\n")
		fmt.Fprintf(writer, "- `id`: An identifier to correlate the request with the response.\n\n")

		fmt.Fprintf(writer, "**Responses:**\n\n")
		fmt.Fprintf(writer, "The server responds with a JSON object containing one of these fields:\n")
		fmt.Fprintf(writer, "- `result`: The data returned by the method if successful.\n")
		fmt.Fprintf(writer, "- `error`: An error object with code, message, and optional data.\n")
		fmt.Fprintf(writer, "- `id`: Matches the request identifier.\n\n")

		fmt.Fprintf(writer, "**Example Request:**\n\n")
		fmt.Fprintf(writer, "```json\n")
		fmt.Fprintf(writer, "{\n")
		fmt.Fprintf(writer, "  \"jsonrpc\": \"2.0\",\n")
		fmt.Fprintf(writer, "  \"method\": \"stats.GetAllMetrics\",\n")
		fmt.Fprintf(writer, "  \"params\": { \"tz\": \"UTC\" },\n")
		fmt.Fprintf(writer, "  \"id\": 1\n")
		fmt.Fprintf(writer, "}\n")
		fmt.Fprintf(writer, "```\n\n")

		fmt.Fprintf(writer, "**Example Response:**\n\n")
		fmt.Fprintf(writer, "```json\n")
		fmt.Fprintf(writer, "{\n")
		fmt.Fprintf(writer, "  \"jsonrpc\": \"2.0\",\n")
		fmt.Fprintf(writer, "  \"result\": {\n")
		fmt.Fprintf(writer, "    \"TotalScannedFiles\": [100, 200],\n")
		fmt.Fprintf(writer, "    \"TotalInfectedFiles\": [5, 10]\n")
		fmt.Fprintf(writer, "  },\n")
		fmt.Fprintf(writer, "  \"id\": 1\n")
		fmt.Fprintf(writer, "}\n")
		fmt.Fprintf(writer, "```\n\n")
	}

	// Write Project Info at the top
	fmt.Fprintf(writer, "# %s\n\n", projectInfo.Title)
	fmt.Fprintf(writer, "Version: %s\n\n", projectInfo.Version)
	if projectInfo.Description != "" {
		fmt.Fprintf(writer, "%s\n\n", projectInfo.Description)
	}

	if projectInfo.Author != "" {
		fmt.Fprintf(writer, "**Author:** %s\n\n", projectInfo.Author)
	}
	if projectInfo.License != "" {
		fmt.Fprintf(writer, "**License:** %s\n\n", projectInfo.License)
	}
	if len(projectInfo.Tags) > 0 {
		fmt.Fprintf(writer, "**Tags:** %s\n\n", strings.Join(projectInfo.Tags, ", "))
	}

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

			// Inline struct documentation for each endpoint
			visited := make(map[models.StructKey]bool) // Reset visited map for every endpoint
			for _, result := range apiFunc.Results {
				baseType, typeArgs := utils.ParseGenericType(result.Type)
				if !utils.IsBasicType(baseType) {
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

		// Add Additional Structs section
		if len(apiFunc.AdditionalStructs) > 0 {
			fmt.Fprintf(writer, "### Additional Structs:\n\n")
			visited := make(map[models.StructKey]bool) // Reset visited map for every endpoint
			for _, additional := range apiFunc.AdditionalStructs {
				baseType, typeArgs := utils.ParseGenericType(additional)
				if utils.IsBasicType(baseType) {
					continue
				}
				// Resolve to package and name
				pkg, baseName := resolvePackageAndType(baseType, apiFunc.PackageName, apiFunc.ImportAliases, structDefinitions)
				if baseName == "" {
					log.Printf("Warning: Struct '%s' not found for @Additional annotation.", additional)
					continue
				}

				var concreteType string
				if len(typeArgs) > 0 {
					// Construct generic name
					// For each arg, also resolve package and name if needed
					resolvedArgs := []string{}
					for _, arg := range typeArgs {
						argPkg, argName := resolvePackageAndType(arg, apiFunc.PackageName, apiFunc.ImportAliases, structDefinitions)
						if argName == "" {
							argName = arg
						}
						if argPkg != "" && argPkg != apiFunc.PackageName {
							resolvedArgs = append(resolvedArgs, fmt.Sprintf("%s.%s", argPkg, argName))
						} else {
							resolvedArgs = append(resolvedArgs, argName)
						}
					}
					concreteType = fmt.Sprintf("%s[%s]", baseName, strings.Join(resolvedArgs, ", "))
				} else {
					concreteType = baseName
				}

				// Find struct definition
				var found bool
				var resolvedKey models.StructKey
				// For generics or normal
				// Generic or not, package is from base
				// If generic, we just store in same package as base type
				if len(typeArgs) > 0 {
					resolvedKey = models.StructKey{
						Package: pkg,
						Name:    concreteType,
					}
					if _, exists := structDefinitions[resolvedKey]; !exists {
						// Create concrete struct if needed (similar to parser logic)
						// If it's generic and not created yet, you must mimic the parser logic or skip
						// For simplicity, assume it's already created. If needed, replicate parser logic here.
						// If not found, warn
						log.Printf("Warning: Concrete struct '%s.%s' not found for @Additional", pkg, concreteType)
						continue
					}
					found = true
				} else {
					// Non-generic
					resolvedKey = models.StructKey{
						Package: pkg,
						Name:    concreteType,
					}
					if _, exists := structDefinitions[resolvedKey]; exists {
						found = true
					}
				}

				if found {
					printStructDefinitionInline(writer, resolvedKey, structDefinitions, visited)
				} else {
					log.Printf("Warning: Struct '%s' not found for @Additional annotation.", additional)
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

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write to output file: %v", err)
	}

	log.Printf("Documentation successfully generated at %s", outFile)
	return nil
}

// printStructDefinitionInline prints a given struct's definition and all referenced structs inline.
// It uses a visited map to avoid duplicates.
func printStructDefinitionInline(writer *bufio.Writer, key models.StructKey, structDefinitions map[models.StructKey]models.StructDefinition, visited map[models.StructKey]bool) {
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
