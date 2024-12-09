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
	"strconv"
	"strings"

	"github.com/pablolagos/jdocgen/models"
	"github.com/pablolagos/jdocgen/utils"
)

// Sentinel errors for specific annotation issues
var (
	ErrMissingCommand     = errors.New("missing @Command annotation")
	ErrMultipleResults    = errors.New("multiple @Result annotations found")
	ErrInvalidErrorCode   = errors.New("@Error code must be a numeric literal")
	ErrMissingDescription = errors.New("missing @Description annotation")
	ErrMalformedResult    = errors.New("malformed @Result annotation. Expected format: @Result type \"description\"")
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

	// First Pass: Collect all struct definitions
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

				// Capture type parameters if the struct is generic
				if typeSpec.TypeParams != nil {
					for _, field := range typeSpec.TypeParams.List {
						for _, name := range field.Names {
							param := models.TypeParam{
								Name: name.Name,
							}
							if field.Type != nil {
								param.Constraint = utils.ExprToString(field.Type)
							}
							structDef.TypeParams = append(structDef.TypeParams, param)
						}
					}
				}

				for _, field := range structType.Fields.List {
					fieldName := ""
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						// Embedded field
						fieldName = utils.ExprToString(field.Type)
					}

					jsonName := fieldName
					if field.Tag != nil {
						tag := field.Tag.Value
						// Extract json tag
						jsonName = utils.ExtractJSONTag(tag, fieldName)
					}

					fieldType := utils.ExprToString(field.Type)
					fieldDesc := extractFieldDescription(field.Doc, field.Comment)

					structField := models.StructField{
						Name:        fieldName,
						Type:        fieldType,
						Description: fieldDesc,
						JSONName:    jsonName,
					}
					structDef.Fields = append(structDef.Fields, structField)

					// If the field type is a struct, note it for potential future processing
					baseType, pkg := utils.ResolveType(fieldType)
					if baseType == "" {
						continue
					}
					// Skip basic types
					if utils.IsBasicType(baseType) {
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
					// Mark as processed to prevent infinite loops
					processedStructs[structKey] = true
					// Note: We're not parsing nested structs in the first pass to avoid complexity
				}

				key := models.StructKey{
					Package: currentPackage,
					Name:    structDef.Name,
				}
				structDefinitions[key] = structDef

				log.Printf("Collected struct: Package='%s', Name='%s'", key.Package, key.Name)
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, projectInfo, err
	}

	// Optional: Log collected structs for verification
	log.Println("Collected structs:")
	for key := range structDefinitions {
		log.Printf(" - Package: %s, Struct: %s", key.Package, key.Name)
	}

	// Second Pass: Process functions with annotations
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
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

		// Extract global tags from file-level comments if not set
		if fileAst.Doc != nil && !projectInfoSet {
			globalInfo, err := parseGlobalTags(fileAst.Doc)
			if err == nil {
				projectInfo = globalInfo
				projectInfoSet = true
			}
		}

		// Parse functions with annotations
		for _, decl := range fileAst.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Doc == nil {
				continue
			}

			// Extract API functions
			apiFunc, err := parseFunction(fn, currentPackage, importAliases, path, fset, structDefinitions)
			if err == nil {
				apiFunctions = append(apiFunctions, apiFunc)
			} else {
				// Check if the error is ErrMissingCommand
				if !errors.Is(err, ErrMissingCommand) {
					// Log other errors with file name and position
					// Extract function name and position
					position := fset.Position(fn.Pos())
					log.Printf("Error in file %s at line %d: Function '%s' skipped due to error: %v", position.Filename, position.Line, fn.Name.Name, err)
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

	// Log final structDefinitions for verification
	log.Println("Final structDefinitions:")
	for key := range structDefinitions {
		log.Printf(" - Package: %s, Struct: %s", key.Package, key.Name)
	}

	return apiFunctions, structDefinitions, projectInfo, nil
}

// parseFunction parses a function's comments to extract API annotations, including @Error tags.
// It handles generic type instantiations by creating concrete StructDefinitions.
func parseFunction(fn *ast.FuncDecl, currentPackage string, importAliases map[string]string, fileName string, fset *token.FileSet, structDefinitions map[models.StructKey]models.StructDefinition) (models.APIFunction, error) {
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
			if len(parts) < 4 {
				return apiFunc, errors.New("invalid @Parameter annotation. Expected format: @Parameter name type \"description\"")
			}
			paramName := parts[1]
			paramType := parts[2]
			// Assume the description is enclosed in quotes
			paramDesc := strings.Join(parts[3:], " ")
			paramDesc = strings.Trim(paramDesc, "\"")
			param := models.APIParameter{
				Name:        paramName,
				Type:        paramType,
				Description: paramDesc,
				Required:    true, // Default to required
			}
			// Check if the parameter is optional
			if strings.HasPrefix(paramDesc, "optional") {
				param.Required = false
				param.Description = strings.TrimPrefix(param.Description, "optional")
				param.Description = strings.TrimSpace(param.Description)
			}
			apiFunc.Parameters = append(apiFunc.Parameters, param)
		case "@Result":
			// Collect all @Result annotations to check for multiples
			resultAnnotations = append(resultAnnotations, &ast.Comment{Text: line})
		case "@Error":
			if len(parts) < 3 {
				return apiFunc, errors.New("invalid @Error annotation. Expected format: @Error code \"description\"")
			}
			errorCodeStr := parts[1]
			errorDesc := strings.Join(parts[2:], " ")
			errorDesc = strings.Trim(errorDesc, "\"")

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
		resultDesc = strings.Trim(resultDesc, "\"")
		result := models.APIReturn{
			Name:        "result", // Name is always "result"
			Type:        resultType,
			Description: resultDesc,
			Required:    true, // All return values are required
		}
		apiFunc.Results = append(apiFunc.Results, result)

		// Check if the result type is a generic struct
		baseType, typeArgs := utils.ParseGenericType(resultType)
		if len(typeArgs) > 0 {
			// Handle generic type instantiation
			genBaseType := baseType
			genArgs := typeArgs

			// Resolve package for generic base type
			genBaseTypePkg, genBaseTypeName := resolvePackageAndType(genBaseType, currentPackage, importAliases, structDefinitions)

			if genBaseTypeName != "" {
				log.Printf("Resolved generic type '%s' to package '%s' and type '%s'", genBaseType, genBaseTypePkg, genBaseTypeName)
			} else {
				log.Printf("Failed to resolve generic type '%s'", genBaseType)
			}

			// Construct StructKey for generic base type
			structKey := models.StructKey{
				Package: genBaseTypePkg,
				Name:    genBaseTypeName,
			}
			genericStructDef, exists := structDefinitions[structKey]
			if !exists {
				log.Printf("Warning: Generic struct '%s' not found for result 'result'.", genBaseTypeName)
			} else {
				// Create a concrete StructDefinition for Pagination[ReportItem]
				// Fully qualify type arguments if they belong to other packages
				processedGenArgs := []string{}
				for _, arg := range genArgs {
					argPkg, argTypeName := resolvePackageAndType(arg, currentPackage, importAliases, structDefinitions)
					if argPkg != "" && argPkg != currentPackage {
						// Use package alias to qualify the type
						processedGenArgs = append(processedGenArgs, fmt.Sprintf("%s.%s", argPkg, argTypeName))
					} else if argPkg == currentPackage {
						// Type belongs to the current package; use unqualified name
						processedGenArgs = append(processedGenArgs, argTypeName)
					} else {
						// Type package not determined; log a warning and use unqualified name
						log.Printf("Warning: Unable to determine package for type '%s'. Using unqualified name.", arg)
						processedGenArgs = append(processedGenArgs, argTypeName)
					}
				}

				concreteTypeName := fmt.Sprintf("%s[%s]", genBaseTypeName, strings.Join(processedGenArgs, ", "))

				concreteKey := models.StructKey{
					Package: genBaseTypePkg,
					Name:    concreteTypeName,
				}

				// Avoid duplicating concrete struct definitions
				if _, exists := structDefinitions[concreteKey]; !exists {
					concreteStructDef := models.StructDefinition{
						Name:        concreteTypeName,
						Description: genericStructDef.Description,
					}

					// Replace type parameters with concrete types in fields
					for _, field := range genericStructDef.Fields {
						concreteField := field
						concreteField.Type = utils.ReplaceTypeParams(field.Type, genericStructDef.TypeParams, processedGenArgs)
						concreteStructDef.Fields = append(concreteStructDef.Fields, concreteField)
					}

					// Add the concrete struct to structDefinitions
					structDefinitions[concreteKey] = concreteStructDef

					// Log the creation of the concrete struct
					log.Printf("Created concrete struct '%s' for generic type instantiation.", concreteTypeName)
				} else {
					log.Printf("Concrete struct '%s' already exists.", concreteTypeName)
				}
			}
		} else {
			// Non-generic struct
			// Resolve package for base type
			baseTypePkg, baseTypeName := resolvePackageAndType(baseType, currentPackage, importAliases, structDefinitions)

			if baseTypeName != "" {
				log.Printf("Resolved type '%s' to package '%s' and type '%s'", baseType, baseTypePkg, baseTypeName)
			} else {
				log.Printf("Failed to resolve type '%s'", baseType)
			}

			// Construct StructKey
			structKey := models.StructKey{
				Package: baseTypePkg,
				Name:    baseTypeName,
			}

			// Lookup the struct in structDefinitions
			if _, exists := structDefinitions[structKey]; exists {
				log.Printf("Found struct '%s' in package '%s' for result 'result'.", baseTypeName, baseTypePkg)
			} else {
				log.Printf("Warning: Struct '%s' not found for result 'result'.", baseTypeName)
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

// extractImportAliases extracts a map of alias to package name from the file's import declarations.
func extractImportAliases(fileAst *ast.File) map[string]string {
	importAliases := make(map[string]string)
	for _, imp := range fileAst.Imports {
		var alias string
		var pkgName string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// Infer package name from import path
			path := strings.Trim(imp.Path.Value, `"`)
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		// Assume package name is the last element of the import path
		path := strings.Trim(imp.Path.Value, `"`)
		parts := strings.Split(path, "/")
		pkgName = parts[len(parts)-1]
		importAliases[alias] = pkgName
	}
	return importAliases
}

// extractFieldDescription extracts the description from a field's comment groups (both Doc and Comment).
func extractFieldDescription(doc *ast.CommentGroup, comment *ast.CommentGroup) string {
	comments := []string{}

	if doc != nil {
		for _, c := range doc.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "/*"))
			line = strings.TrimSpace(strings.TrimSuffix(line, "*/"))
			if line != "" {
				comments = append(comments, line)
			}
		}
	}

	if comment != nil {
		for _, c := range comment.List {
			line := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			line = strings.TrimSpace(strings.TrimPrefix(line, "/*"))
			line = strings.TrimSpace(strings.TrimSuffix(line, "*/"))
			if line != "" {
				comments = append(comments, line)
			}
		}
	}

	return strings.Join(comments, " ")
}
