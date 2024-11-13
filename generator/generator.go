package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pablolagos/jdocgen/models"
)

// GenerateMarkdown generates Markdown documentation from API functions and struct definitions.
// It conditionally includes JSON-RPC 2.0 information based on the includeRFC flag.
// Additionally, it appends a note about the documentation generator at the end.
func GenerateMarkdown(functions []models.APIFunction, structs map[models.StructKey]models.StructDefinition, projectInfo models.ProjectInfo, includeRFC bool) string {
	var sb strings.Builder

	// Global Project Information
	sb.WriteString(fmt.Sprintf("# %s\n\n", projectInfo.Title))
	sb.WriteString(fmt.Sprintf("**Version:** %s\n\n", projectInfo.Version))
	sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", projectInfo.Description))
	if projectInfo.Author != "" {
		sb.WriteString(fmt.Sprintf("**Author:** %s\n\n", projectInfo.Author))
	}
	if projectInfo.License != "" {
		sb.WriteString(fmt.Sprintf("**License:** %s\n\n", projectInfo.License))
	}
	if projectInfo.Contact != "" {
		sb.WriteString(fmt.Sprintf("**Contact:** %s\n\n", projectInfo.Contact))
	}
	if projectInfo.Terms != "" {
		sb.WriteString(fmt.Sprintf("**Terms of Service:** %s\n\n", projectInfo.Terms))
	}
	if projectInfo.Repository != "" {
		sb.WriteString(fmt.Sprintf("**Repository:** [%s](%s)\n\n", projectInfo.Repository, projectInfo.Repository))
	}
	if len(projectInfo.Tags) > 0 {
		sb.WriteString("**Tags:** ")
		var tags []string
		for _, tag := range projectInfo.Tags {
			tags = append(tags, strings.TrimSpace(tag))
		}
		sb.WriteString(strings.Join(tags, ", "))
		sb.WriteString("\n\n")
	}
	if projectInfo.Copyright != "" {
		sb.WriteString(fmt.Sprintf("**Copyright:** %s\n\n", projectInfo.Copyright))
	}

	sb.WriteString("---\n\n")

	// Introduction Section
	if includeRFC {
		sb.WriteString("## Introduction\n\n")
		sb.WriteString("This API adheres to the [JSON-RPC 2.0](https://www.jsonrpc.org/specification) specification, a lightweight remote procedure call (RPC) protocol encoded in JSON. JSON-RPC 2.0 allows for invoking methods on a server by sending JSON-encoded requests and receiving JSON-encoded responses.\n\n")
		sb.WriteString("**Key Features of JSON-RPC 2.0:**\n\n")
		sb.WriteString("- **Simple and Lightweight:** Minimalist design without unnecessary features.\n")
		sb.WriteString("- **Transport Agnostic:** Can be used over various transport protocols such as HTTP, WebSocket, etc.\n")
		sb.WriteString("- **Batch Requests:** Supports sending multiple requests in a single call.\n")
		sb.WriteString("- **Notifications:** Allows sending requests that do not require responses.\n\n")

		// JSON-RPC 2.0 Request and Response Structures Section
		sb.WriteString("## JSON-RPC 2.0 Request and Response Structures\n\n")
		sb.WriteString("The API follows the [JSON-RPC 2.0](https://www.jsonrpc.org/specification) specification, which defines a simple and lightweight protocol for remote procedure calls using JSON.\n\n")

		sb.WriteString("### Request Structure\n\n")
		sb.WriteString("A JSON-RPC request object must contain the following members:\n\n")
		sb.WriteString("- **jsonrpc**: A string specifying the version of the JSON-RPC protocol. Must be exactly `\"2.0\"`.\n")
		sb.WriteString("- **method**: A string containing the name of the method to be invoked.\n")
		sb.WriteString("- **params** (optional): A structured value that holds the parameter values to be used during the invocation of the method.\n")
		sb.WriteString("- **id**: An identifier established by the client that must be unique for each request. It is used to match responses with requests.\n\n")

		sb.WriteString("**Example:**\n\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"jsonrpc\": \"2.0\",\n")
		sb.WriteString("  \"method\": \"getLicense\",\n")
		sb.WriteString("  \"params\": { \"userId\": 12345 },\n")
		sb.WriteString("  \"id\": 1\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")

		sb.WriteString("### Response Structure\n\n")
		sb.WriteString("A JSON-RPC response object must contain the following members:\n\n")
		sb.WriteString("- **jsonrpc**: A string specifying the version of the JSON-RPC protocol. Must be exactly `\"2.0\"`.\n")
		sb.WriteString("- **result**: This member is required on success. It holds the value returned by the invoked method.\n")
		sb.WriteString("- **error**: This member is required on error. It contains an error object with information about the error.\n")
		sb.WriteString("- **id**: The same identifier as the request it is responding to.\n\n")

		sb.WriteString("**Example (Success):**\n\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"jsonrpc\": \"2.0\",\n")
		sb.WriteString("  \"result\": { \"licenseCode\": \"ABC123\", \"isValid\": true },\n")
		sb.WriteString("  \"id\": 1\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")

		sb.WriteString("**Example (Error):**\n\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"jsonrpc\": \"2.0\",\n")
		sb.WriteString("  \"error\": {\n")
		sb.WriteString("    \"code\": -32601,\n")
		sb.WriteString("    \"message\": \"Method not found\"\n")
		sb.WriteString("  },\n")
		sb.WriteString("  \"id\": 1\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n\n")
	}

	// API Overview
	sb.WriteString("## API Overview\n\n")
	sb.WriteString("This document describes the functions available through the JSON-RPC API.\n\n")

	// Sort API functions alphabetically by Command
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Command < functions[j].Command
	})

	// Document API Functions
	for _, fn := range functions {
		// Remove backticks from function titles
		sb.WriteString(fmt.Sprintf("## %s\n\n", fn.Command))
		sb.WriteString(fmt.Sprintf("%s\n\n", fn.Description))

		// Parameters
		if len(fn.Parameters) > 0 {
			sb.WriteString("### Parameters\n\n")
			sb.WriteString("| Name | Type | Description | Required |\n")
			sb.WriteString("|------|------|-------------|----------|\n")
			for _, param := range fn.Parameters {
				requiredStatus := "Yes"
				if !param.Required {
					requiredStatus = "*No*"
				}
				// Remove backticks from table cells and use bold for field names
				sb.WriteString(fmt.Sprintf("| **%s** | %s | %s | %s |\n", param.Name, param.Type, param.Description, requiredStatus))
			}
			sb.WriteString("\n")

			// Include struct definitions for parameters if applicable
			for _, param := range fn.Parameters {
				baseType, pkg := resolveType(param.Type)
				if baseType == "" {
					continue
				}
				structDef, exists := findStruct(structs, baseType, pkg, fn.PackageName)
				if exists {
					sb.WriteString(fmt.Sprintf("#### %s Structure\n\n", baseType))
					sb.WriteString("| Field | Type | Description |\n")
					sb.WriteString("|-------|------|-------------|\n")
					for _, field := range structDef.Fields {
						// Use bold for field names
						sb.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", field.JSONName, field.Type, field.Description))
					}
					sb.WriteString("\n")
				}
			}
		}

		// Return Values
		if len(fn.Results) > 0 {
			sb.WriteString("### Return Values\n\n")
			sb.WriteString("| Name | Type | Description |\n")
			sb.WriteString("|------|------|-------------|\n")
			for _, ret := range fn.Results {
				// Remove backticks from table cells and use bold for field names
				sb.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", ret.Name, ret.Type, ret.Description))
			}
			sb.WriteString("\n")

			// Include struct definitions for return values if applicable
			for _, ret := range fn.Results {
				baseType, pkg := resolveType(ret.Type)
				if baseType == "" {
					continue
				}
				structDef, exists := findStruct(structs, baseType, pkg, fn.PackageName)
				if exists {
					sb.WriteString(fmt.Sprintf("#### %s Structure\n\n", baseType))
					sb.WriteString("| Field | Type | Description |\n")
					sb.WriteString("|-------|------|-------------|\n")
					for _, field := range structDef.Fields {
						// Use bold for field names
						sb.WriteString(fmt.Sprintf("| **%s** | %s | %s |\n", field.JSONName, field.Type, field.Description))
					}
					sb.WriteString("\n")
				}
			}
		}

		sb.WriteString("---\n\n")
	}

	// Append Generator Note
	sb.WriteString("## Documentation Generator\n\n")
	sb.WriteString("This documentation was automatically generated using [jdocgen](https://github.com/pablolagos/jdocgen), a CLI tool for generating Markdown documentation from annotated Go source files.\n\n")

	return sb.String()
}

// resolveType parses the type string to extract the base type and its package if present.
// For example:
// - "License" returns ("License", "")
// - "jrpc.License" returns ("License", "jrpc")
func resolveType(typeStr string) (string, string) {
	if strings.Contains(typeStr, ".") {
		parts := strings.Split(typeStr, ".")
		if len(parts) == 2 {
			return parts[1], parts[0]
		}
	}
	return typeStr, ""
}

// findStruct searches for a struct by its name and package.
// If packageName is empty, it searches for structs with the given name regardless of package.
// It prioritizes local package structs over imported ones when packageName is empty.
func findStruct(structs map[models.StructKey]models.StructDefinition, name string, packageName string, localPackage string) (models.StructDefinition, bool) {
	if packageName != "" {
		key := models.StructKey{
			Package: packageName,
			Name:    name,
		}
		structDef, exists := structs[key]
		return structDef, exists
	}

	// First, search in the local package
	key := models.StructKey{
		Package: localPackage,
		Name:    name,
	}
	structDef, exists := structs[key]
	if exists {
		return structDef, true
	}

	// If not found locally, search globally (first match)
	for _, structDef := range structs {
		if structDef.Name == name {
			return structDef, true
		}
	}
	return models.StructDefinition{}, false
}
