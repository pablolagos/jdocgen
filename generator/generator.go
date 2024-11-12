// generator/generator.go
package generator

import (
	"fmt"
	"strings"

	"github.com/pablolagos/jdocgen/models"
)

// GenerateMarkdown generates Markdown documentation from API functions and struct definitions.
// It places struct definitions adjacent to their usage in Parameters or Return Values and includes global project info.
// Additionally, it appends a note about the documentation generator at the end.
func GenerateMarkdown(functions []models.APIFunction, structs map[string]models.StructDefinition, projectInfo models.ProjectInfo) string {
	var sb strings.Builder

	// Global Project Information
	sb.WriteString(fmt.Sprintf("# %s\n\n", projectInfo.Title))
	sb.WriteString(fmt.Sprintf("**Version:** %s\n\n", projectInfo.Version))
	sb.WriteString(fmt.Sprintf("**Description:**\n%s\n\n", projectInfo.Description))
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

	// Document API Functions
	for _, fn := range functions {
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
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", param.Name, param.Type, param.Description, requiredStatus))
			}
			sb.WriteString("\n")

			// Include struct definitions for parameters if applicable
			for _, param := range fn.Parameters {
				// Handle pointer types
				baseType := param.Type
				if strings.HasPrefix(baseType, "*") {
					baseType = strings.TrimPrefix(baseType, "*")
				}
				if structDef, exists := structs[baseType]; exists {
					sb.WriteString(fmt.Sprintf("#### `%s` Structure\n\n", baseType))
					sb.WriteString("| Field | Type | Description |\n")
					sb.WriteString("|-------|------|-------------|\n")
					for _, field := range structDef.Fields {
						sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", field.JSONName, field.Type, field.Description))
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
				sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", ret.Name, ret.Type, ret.Description))
			}
			sb.WriteString("\n")

			// Include struct definitions for return values if applicable
			for _, ret := range fn.Results {
				// Handle pointer types
				baseType := ret.Type
				if strings.HasPrefix(baseType, "*") {
					baseType = strings.TrimPrefix(baseType, "*")
				}
				if structDef, exists := structs[baseType]; exists {
					sb.WriteString(fmt.Sprintf("#### %s Structure\n\n", baseType))
					sb.WriteString("| Field | Type | Description |\n")
					sb.WriteString("|-------|------|-------------|\n")
					for _, field := range structDef.Fields {
						sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", field.JSONName, field.Type, field.Description))
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
