// parser/parser.go
package parser

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pablolagos/jdocgen/models"
)

// Sentinel errors for specific annotation issues
var (
	ErrMissingCommand     = errors.New("missing @Command annotation")
	ErrMultipleResults    = errors.New("multiple @Result annotations found")
	ErrInvalidErrorCode   = errors.New("@Error code must be a numeric literal")
	ErrMissingDescription = errors.New("missing @Description annotation")
	ErrMalformedResult    = errors.New("malformed @Result annotation. Expected format: @Result type description")
)

// ParseProject recursively parses all Go files in the project directory and its subdirectories.
// It returns a slice of APIFunctions, a map of StructDefinitions keyed by StructKey, and ProjectInfo.
func ParseProject(rootDir string) ([]models.APIFunction, map[models.StructKey]models.StructDefinition, models.ProjectInfo, error) {
	var apiFunctions []models.APIFunction
	structDefinitions := make(map[models.StructKey]models.StructDefinition)
	var projectInfo models.ProjectInfo
	projectInfoSet := false

	// Initialize a new FileSet
	fset := token.NewFileSet()

	// To prevent infinite recursion in case of cyclic struct references
	processedStructs := make(map[models.StructKey]bool)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and test files
		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only parse .go files excluding test files
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fileAst, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		currentPackage := fileAst.Name.Name

		// Extract import aliases
		importAliases := extractImportAliases(fileAst)

		// Extract global tags from file-level comments
		if fileAst.Doc != nil && !projectInfoSet {
			globalInfo, err := parseGlobalTags(fileAst.Doc)
			if err == nil {
				projectInfo = globalInfo
				projectInfoSet = true
			}
		}

		// Collect struct definitions
		for _, decl := range fileAst.Decls {
			genDecl, isGen := decl.(*ast.GenDecl)
			if !isGen || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, isType := spec.(*ast.TypeSpec)
				if !isType {
					continue
				}
				structType, isStruct := typeSpec.Type.(*ast.StructType)
				if !isStruct {
					continue
				}

				structDef := models.StructDefinition{
					Name: typeSpec.Name.Name,
				}
				structDef.Description = extractStructDescription(genDecl.Doc)

				for _, field := range structType.Fields.List {
					fieldName := ""
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						// Embedded field
						fieldName = exprToString(field.Type)
					}

					jsonName := fieldName
					if field.Tag != nil {
						tag := field.Tag.Value
						// Extract json tag
						re := regexp.MustCompile(`json:"([^"]+)"`)
						matches := re.FindStringSubmatch(tag)
						if len(matches) > 1 && matches[1] != "-" {
							jsonName = matches[1]
						}
					}

					fieldType := exprToString(field.Type)
					fieldDesc := extractFieldDescription(field.Doc, field.Comment)

					structField := models.StructField{
						Name:        fieldName,
						Type:        fieldType,
						Description: fieldDesc,
						JSONName:    jsonName,
					}
					structDef.Fields = append(structDef.Fields, structField)

					// If the field type is a struct, attempt to parse it recursively
					baseType, pkg := resolveType(fieldType)
					if baseType == "" {
						continue
					}
					// Skip basic types
					if isBasicType(baseType) {
						continue
					}
					// Construct StructKey
					var structKey models.StructKey
					if pkg != "" {
						structKey = models.StructKey{
							Package: pkg,
							Name:    baseType,
						}
					} else {
						structKey = models.StructKey{
							Package: currentPackage,
							Name:    baseType,
						}
					}
					// Avoid re-processing structs
					if _, exists := structDefinitions[structKey]; exists || processedStructs[structKey] {
						continue
					}
					// Attempt to find and parse the nested struct
					nestedStructDef, err := findAndParseStruct(rootDir, structKey, fset, importAliases, structDefinitions, processedStructs)
					if err == nil && nestedStructDef.Name != "" {
						structDefinitions[structKey] = nestedStructDef
						processedStructs[structKey] = true
					}
				}

				key := models.StructKey{
					Package: currentPackage,
					Name:    structDef.Name,
				}
				structDefinitions[key] = structDef
			}
		}

		// Parse functions with annotations
		for _, decl := range fileAst.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Doc == nil {
				continue
			}

			// Extract API functions
			apiFunc, err := parseFunction(fn, currentPackage, importAliases, path, fset, structDefinitions, processedStructs, rootDir)
			if err == nil {
				apiFunctions = append(apiFunctions, apiFunc)
			} else {
				// Check if the error is ErrMissingCommand
				if !errors.Is(err, ErrMissingCommand) {
					// Log other errors with file name and position
					// Extract function name and position
					functionName := fn.Name.Name
					position := fset.Position(fn.Pos())
					log.Printf("Error in file %s at line %d: Function '%s' skipped due to error: %v", position.Filename, position.Line, functionName, err)
				}
				// If ErrMissingCommand, do not log and skip silently
			}

			// Extract global tags from function-level comments if not set
			if !projectInfoSet {
				globalInfo, err := parseGlobalTags(fn.Doc)
				if err == nil {
					projectInfo = globalInfo
					projectInfoSet = true
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, projectInfo, err
	}

	if !projectInfoSet {
		return nil, nil, projectInfo, errors.New("no global tags found in any Go file. Please include global tags in at least one file")
	}

	return apiFunctions, structDefinitions, projectInfo, nil
}

// findAndParseStruct attempts to locate and parse a struct definition based on StructKey.
// It searches within the specified root directory and its subdirectories.
// Now accepts structDefinitions and processedStructs as parameters to store results.
func findAndParseStruct(rootDir string, key models.StructKey, fset *token.FileSet, importAliases map[string]string, structDefinitions map[models.StructKey]models.StructDefinition, processedStructs map[models.StructKey]bool) (models.StructDefinition, error) {
	var structDef models.StructDefinition

	// Construct the expected file path based on package and struct name
	var searchPackage string
	if key.Package != "" {
		searchPackage = key.Package
	} else {
		searchPackage = "" // Current package
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only parse .go files excluding test files
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fileAst, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		currentPackage := fileAst.Name.Name

		// Check if the file belongs to the desired package
		if searchPackage != "" && currentPackage != searchPackage {
			return nil
		}

		// Look for the struct
		for _, decl := range fileAst.Decls {
			genDecl, isGen := decl.(*ast.GenDecl)
			if !isGen || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, isType := spec.(*ast.TypeSpec)
				if !isType {
					continue
				}
				if typeSpec.Name.Name != key.Name {
					continue
				}
				structType, isStruct := typeSpec.Type.(*ast.StructType)
				if !isStruct {
					continue
				}

				// Initialize StructDefinition
				structDef = models.StructDefinition{
					Name: typeSpec.Name.Name,
				}
				structDef.Description = extractStructDescription(genDecl.Doc)

				// Mark struct as processed to prevent recursion
				processedStructs[key] = true
				structDefinitions[key] = structDef

				for _, field := range structType.Fields.List {
					fieldName := ""
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						// Embedded field
						fieldName = exprToString(field.Type)
					}

					jsonName := fieldName
					if field.Tag != nil {
						tag := field.Tag.Value
						// Extract json tag
						re := regexp.MustCompile(`json:"([^"]+)"`)
						matches := re.FindStringSubmatch(tag)
						if len(matches) > 1 && matches[1] != "-" {
							jsonName = matches[1]
						}
					}

					fieldType := exprToString(field.Type)
					fieldDesc := extractFieldDescription(field.Doc, field.Comment)

					structField := models.StructField{
						Name:        fieldName,
						Type:        fieldType,
						Description: fieldDesc,
						JSONName:    jsonName,
					}
					structDef.Fields = append(structDef.Fields, structField)

					// If the field type is a struct, attempt to parse it recursively
					baseType, pkg := resolveType(fieldType)
					if baseType == "" {
						continue
					}
					// Skip basic types
					if isBasicType(baseType) {
						continue
					}
					// Construct StructKey
					var structKey models.StructKey
					if pkg != "" {
						structKey = models.StructKey{
							Package: pkg,
							Name:    baseType,
						}
					} else {
						structKey = models.StructKey{
							Package: currentPackage,
							Name:    baseType,
						}
					}
					// Avoid re-processing structs
					if _, exists := structDefinitions[structKey]; exists || processedStructs[structKey] {
						continue
					}
					// Attempt to find and parse the nested struct
					nestedStructDef, err := findAndParseStruct(rootDir, structKey, fset, importAliases, structDefinitions, processedStructs)
					if err == nil && nestedStructDef.Name != "" {
						structDefinitions[structKey] = nestedStructDef
					} else {
						// Log a warning if the struct definition couldn't be found
						log.Printf("Warning: Could not find struct definition for '%s' in package '%s'. Detailed field information will not be available.", baseType, pkg)
					}
				}

				return errors.New("struct found and parsed") // Stop searching
			}
		}

		return nil
	})

	if err != nil && err.Error() == "struct found and parsed" {
		return structDef, nil
	}

	return models.StructDefinition{}, errors.New("struct not found")
}

// isBasicType checks if a given type is a Go basic type.
func isBasicType(typeName string) bool {
	basicTypes := map[string]bool{
		"string":  true,
		"bool":    true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"byte":    true,
		"rune":    true,
		// Add more basic types as needed
	}
	return basicTypes[typeName]
}

// extractImportAliases extracts a map of alias to package name from the file's import declarations.
func extractImportAliases(fileAst *ast.File) map[string]string {
	importAliases := make(map[string]string)
	for _, imp := range fileAst.Imports {
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// Infer package name from import path
			path := strings.Trim(imp.Path.Value, `"`)
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		// Assume package name is the last element of the import path
		importAliases[alias] = alias
	}
	return importAliases
}

// parseFunction parses a function's comments to extract API annotations, including @Error tags.
// It enforces only one @Result annotation and validates @Error codes.
// If multiple @Result annotations are found, it returns ErrMultipleResults with details.
// If @Command is missing, it returns ErrMissingCommand.
// Other annotation issues return corresponding errors.
func parseFunction(fn *ast.FuncDecl, currentPackage string, importAliases map[string]string, fileName string, fset *token.FileSet, structDefinitions map[models.StructKey]models.StructDefinition, processedStructs map[models.StructKey]bool, rootDir string) (models.APIFunction, error) {
	apiFunc := models.APIFunction{
		ImportAliases: importAliases,
		PackageName:   currentPackage,
	}

	var resultAnnotations []*ast.Comment
	scanner := bufio.NewScanner(strings.NewReader(fn.Doc.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "@") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}
		switch parts[0] {
		case "@Command":
			if len(parts) < 2 {
				return apiFunc, errors.New("missing command name in @Command annotation")
			}
			apiFunc.Command = parts[1]
		case "@Description":
			description := strings.TrimPrefix(line, "@Description")
			apiFunc.Description = strings.TrimSpace(description)
		case "@Parameter":
			if len(parts) < 3 {
				return apiFunc, errors.New("invalid @Parameter annotation")
			}
			paramName := parts[1]
			paramType := parts[2]
			isOptional := false
			paramDescParts := parts[3:]
			// Check for 'optional' keyword
			if len(paramDescParts) > 0 && strings.EqualFold(paramDescParts[0], "optional") {
				isOptional = true
				paramDescParts = paramDescParts[1:]
			}
			paramDesc := strings.Join(paramDescParts, " ")
			param := models.APIParameter{
				Name:        paramName,
				Type:        paramType,
				Description: paramDesc,
				Required:    !isOptional,
			}
			apiFunc.Parameters = append(apiFunc.Parameters, param)
		case "@Result":
			// Collect all @Result annotations to check for multiples
			resultAnnotations = append(resultAnnotations, &ast.Comment{Text: line})
		case "@Error":
			if len(parts) < 3 {
				return apiFunc, errors.New("invalid @Error annotation")
			}
			errorCodeStr := parts[1]
			errorDesc := strings.Join(parts[2:], " ")

			// Validate that errorCodeStr is a numeric literal
			errorCode, err := strconv.Atoi(errorCodeStr)
			if err != nil {
				return apiFunc, ErrInvalidErrorCode
			}

			apiError := models.APIError{
				Code:        errorCode,
				Description: errorDesc,
			}
			apiFunc.Errors = append(apiFunc.Errors, apiError)
		}
	}

	// Enforce only one @Result annotation
	if len(resultAnnotations) > 1 {
		return apiFunc, fmt.Errorf("%w. JSON-RPC specification enforces a single @Result annotation per function.", ErrMultipleResults)
	}

	// Process @Result annotations
	if len(resultAnnotations) == 1 {
		line := strings.TrimSpace(resultAnnotations[0].Text)
		parts := strings.Fields(line)
		if len(parts) < 3 {
			return apiFunc, ErrMalformedResult
		}
		// Enforce that the name is implicitly "result" by ensuring the first part after @Result is the type
		resultType := parts[1]
		resultDescParts := parts[2:]
		resultDesc := strings.Join(resultDescParts, " ")
		result := models.APIReturn{
			Name:        "result", // Name is always "result"
			Type:        resultType,
			Description: resultDesc,
			Required:    true, // All return values are required
		}
		apiFunc.Results = append(apiFunc.Results, result)

		// Check if the result type is a struct that needs to be documented
		baseType, pkg := resolveType(resultType)
		if baseType != "" && !isBasicType(baseType) {
			// Construct StructKey
			var structKey models.StructKey
			if pkg != "" {
				structKey = models.StructKey{
					Package: pkg,
					Name:    baseType,
				}
			} else {
				structKey = models.StructKey{
					Package: currentPackage,
					Name:    baseType,
				}
			}
			// If not already processed, attempt to find and parse the struct
			if _, exists := structDefinitions[structKey]; !exists && !processedStructs[structKey] {
				nestedStructDef, err := findAndParseStruct(rootDir, structKey, fset, importAliases, structDefinitions, processedStructs)
				if err == nil && nestedStructDef.Name != "" {
					structDefinitions[structKey] = nestedStructDef
					processedStructs[structKey] = true
				} else {
					// Log a warning if the struct definition couldn't be found
					log.Printf("Warning: Could not find struct definition for '%s' in package '%s'. Detailed field information will not be available.", baseType, pkg)
				}
			}
		}
	}

	// Validate required annotations
	if apiFunc.Command == "" {
		return apiFunc, ErrMissingCommand
	}
	if apiFunc.Description == "" {
		return apiFunc, ErrMissingDescription
	}

	return apiFunc, nil
}

// extractStructDescription extracts the description of a struct from its comment group.
func extractStructDescription(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var desc []string
	scanner := bufio.NewScanner(strings.NewReader(cg.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)
		if line != "" {
			desc = append(desc, line)
		}
	}
	return strings.Join(desc, " ")
}

// parseGlobalTags parses global tags from a CommentGroup (file-level or function-level).
func parseGlobalTags(cg *ast.CommentGroup) (models.ProjectInfo, error) {
	projectInfo := models.ProjectInfo{}
	scanner := bufio.NewScanner(strings.NewReader(cg.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Remove comment prefixes like "//", "/*", and "*/"
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, "@") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		// Convert annotation to lowercase for case-insensitive matching
		annotation := strings.ToLower(parts[0])
		switch annotation {
		case "@title":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @title annotation")
			}
			projectInfo.Title = strings.Join(parts[1:], " ")
		case "@version":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @version annotation")
			}
			projectInfo.Version = strings.Join(parts[1:], " ")
		case "@description":
			description := strings.TrimPrefix(line, "@description")
			projectInfo.Description = strings.TrimSpace(description)
		case "@author":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @author annotation")
			}
			projectInfo.Author = strings.Join(parts[1:], " ")
		case "@license":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @license annotation")
			}
			projectInfo.License = strings.Join(parts[1:], " ")
		case "@contact":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @contact annotation")
			}
			projectInfo.Contact = strings.Join(parts[1:], " ")
		case "@terms":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @terms annotation")
			}
			projectInfo.Terms = strings.Join(parts[1:], " ")
		case "@repository":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @repository annotation")
			}
			projectInfo.Repository = strings.Join(parts[1:], " ")
		case "@tags":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @tags annotation")
			}
			tags := strings.Join(parts[1:], " ")
			projectInfo.Tags = strings.Split(tags, ",")
		case "@copyright":
			if len(parts) < 2 {
				return projectInfo, errors.New("missing value in @copyright annotation")
			}
			projectInfo.Copyright = strings.Join(parts[1:], " ")
		}
	}

	// Validate mandatory global tags
	if projectInfo.Title == "" {
		return projectInfo, errors.New("missing @title annotation")
	}
	if projectInfo.Version == "" {
		return projectInfo, errors.New("missing @version annotation")
	}
	if projectInfo.Description == "" {
		return projectInfo, errors.New("missing @description annotation")
	}

	return projectInfo, nil
}

// extractFieldDescription extracts the description from a field's comment groups (both Doc and Comment).
func extractFieldDescription(doc *ast.CommentGroup, comment *ast.CommentGroup) string {
	comments := []string{}

	if doc != nil {
		for _, c := range doc.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "/*"))
			line = strings.TrimSpace(strings.TrimSuffix(line, "*/"))
			comments = append(comments, line)
		}
	}

	if comment != nil {
		for _, c := range comment.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "/*"))
			line = strings.TrimSpace(strings.TrimSuffix(line, "*/"))
			comments = append(comments, line)
		}
	}

	return strings.Join(comments, " ")
}

// exprToString converts an ast.Expr to its string representation.
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return ""
	}
}

// resolveType parses the type string to extract the base type and its package if present.
// It now handles composite types like slices and pointers by stripping their prefixes.
func resolveType(typeStr string) (string, string) {
	// Regular expression to match pointers and slices/arrays
	re := regexp.MustCompile(`^[\*\[\]]+`)
	typeStr = re.ReplaceAllString(typeStr, "")

	if strings.Contains(typeStr, ".") {
		parts := strings.Split(typeStr, ".")
		if len(parts) == 2 {
			return parts[1], parts[0]
		}
	}
	return typeStr, ""
}
