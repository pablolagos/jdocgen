// parser/parser.go
package parser

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
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

func ParseProject(rootDir string) ([]models.APIFunction, map[models.StructKey]models.StructDefinition, models.ProjectInfo, error) {
	var apiFunctions []models.APIFunction
	structDefinitions := make(map[models.StructKey]models.StructDefinition)
	var projectInfo models.ProjectInfo
	projectInfoSet := false

	fset := token.NewFileSet()
	processedStructs := make(map[models.StructKey]bool)

	// First pass: Collect all struct definitions
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileAst, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
		if err != nil {
			return nil
		}

		currentPackage := fileAst.Name.Name

		// Extract global tags
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

				// Capture type parameters if generic
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

				// Process fields
				for _, field := range structType.Fields.List {
					fieldName := ""
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						fieldName = utils.ExprToString(field.Type)
					}

					jsonName := fieldName
					if field.Tag != nil {
						tag := field.Tag.Value
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

					// Note nested structs for processing if needed
					baseType, pkg := utils.ResolveType(fieldType)
					if baseType == "" {
						continue
					}
					if utils.IsBasicType(baseType) {
						continue
					}

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
					if _, exists := structDefinitions[structKey]; exists || processedStructs[structKey] {
						continue
					}
					processedStructs[structKey] = true
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

	log.Println("Collected structs:")
	for key := range structDefinitions {
		log.Printf(" - Package: %s, Struct: %s", key.Package, key.Name)
	}

	// Second pass: process functions
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileAst, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
		if err != nil {
			return nil
		}

		currentPackage := fileAst.Name.Name
		importAliases := extractImportAliases(fileAst)

		// Extract global tags from file-level comments if not set
		if fileAst.Doc != nil && !projectInfoSet {
			globalInfo, err := parseGlobalTags(fileAst.Doc)
			if err == nil {
				projectInfo = globalInfo
				projectInfoSet = true
			}
		}

		for _, decl := range fileAst.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Doc == nil {
				continue
			}

			apiFunc, err := parseFunction(fn, currentPackage, importAliases, path, fset, structDefinitions)
			if err == nil {
				apiFunctions = append(apiFunctions, apiFunc)
			} else {
				if !errors.Is(err, ErrMissingCommand) {
					position := fset.Position(fn.Pos())
					log.Printf("Error in file %s at line %d: Function '%s' skipped due to error: %v", position.Filename, position.Line, fn.Name.Name, err)
				}
			}

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

	log.Println("Final structDefinitions:")
	for key := range structDefinitions {
		log.Printf(" - Package: %s, Struct: %s", key.Package, key.Name)
	}

	return apiFunctions, structDefinitions, projectInfo, nil
}

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
			paramDesc := strings.Join(parts[3:], " ")
			paramDesc = strings.Trim(paramDesc, "\"")
			param := models.APIParameter{
				Name:        paramName,
				Type:        paramType,
				Description: paramDesc,
				Required:    true,
			}
			if strings.HasPrefix(paramDesc, "optional") {
				param.Required = false
				param.Description = strings.TrimPrefix(param.Description, "optional")
				param.Description = strings.TrimSpace(param.Description)
			}
			apiFunc.Parameters = append(apiFunc.Parameters, param)
		case "@Result":
			resultAnnotations = append(resultAnnotations, &ast.Comment{Text: line})
		case "@Error":
			if len(parts) < 3 {
				return apiFunc, errors.New("invalid @Error annotation. Expected format: @Error code \"description\"")
			}
			errorCodeStr := parts[1]
			errorDesc := strings.Join(parts[2:], " ")
			errorDesc = strings.Trim(errorDesc, "\"")
			errorCode, err := strconv.Atoi(errorCodeStr)
			if err != nil {
				return apiFunc, ErrInvalidErrorCode
			}
			apiError := models.APIError{
				Code:        errorCode,
				Description: errorDesc,
			}
			apiFunc.Errors = append(apiFunc.Errors, apiError)
		case "@Additional":
			if len(parts) < 2 {
				return apiFunc, errors.New("invalid @Additional annotation. Expected format: @Additional [package.]structname")
			}
			additionalType := parts[1]
			apiFunc.AdditionalStructs = append(apiFunc.AdditionalStructs, additionalType)
		}
	}

	if len(resultAnnotations) > 1 {
		return apiFunc, fmt.Errorf("%w. JSON-RPC specification enforces a single @Result annotation per function.", ErrMultipleResults)
	}

	if len(resultAnnotations) == 1 {
		line := strings.TrimSpace(resultAnnotations[0].Text)
		parts := strings.Fields(line)
		if len(parts) < 3 {
			return apiFunc, ErrMalformedResult
		}
		resultType := parts[1]
		resultDescParts := parts[2:]
		resultDesc := strings.Join(resultDescParts, " ")
		resultDesc = strings.Trim(resultDesc, "\"")
		result := models.APIReturn{
			Name:        "result",
			Type:        resultType,
			Description: resultDesc,
			Required:    true,
		}
		apiFunc.Results = append(apiFunc.Results, result)

		baseType, typeArgs := utils.ParseGenericType(resultType)
		// Resolve base type to a package and name
		basePkg, baseName := resolvePackageAndType(baseType, currentPackage, importAliases, structDefinitions)

		if baseName != "" {
			log.Printf("Resolved type '%s' to package '%s' and type '%s'", baseType, basePkg, baseName)
		} else {
			log.Printf("Failed to resolve type '%s'", baseType)
		}

		if len(typeArgs) > 0 {
			// Handle generic instantiation
			genBaseTypePkg, genBaseTypeName := basePkg, baseName
			structKey := models.StructKey{
				Package: genBaseTypePkg,
				Name:    genBaseTypeName,
			}
			genericStructDef, exists := structDefinitions[structKey]
			if !exists {
				log.Printf("Warning: Generic struct '%s' not found for result 'result'.", genBaseTypeName)
			} else {
				processedGenArgs := []string{}
				for _, arg := range typeArgs {
					argBasePkg, argBaseName := resolvePackageAndType(arg, currentPackage, importAliases, structDefinitions)
					if argBaseName == "" {
						argBaseName = arg
					}
					if argBasePkg != "" && argBasePkg != currentPackage {
						processedGenArgs = append(processedGenArgs, fmt.Sprintf("%s.%s", argBasePkg, argBaseName))
					} else if argBasePkg == currentPackage {
						processedGenArgs = append(processedGenArgs, argBaseName)
					} else {
						processedGenArgs = append(processedGenArgs, argBaseName)
					}
				}

				concreteTypeName := fmt.Sprintf("%s[%s]", genBaseTypeName, strings.Join(processedGenArgs, ", "))

				concreteKey := models.StructKey{
					Package: genBaseTypePkg,
					Name:    concreteTypeName,
				}

				if _, exists := structDefinitions[concreteKey]; !exists {
					concreteStructDef := models.StructDefinition{
						Name:        concreteTypeName,
						Description: genericStructDef.Description,
					}

					for _, field := range genericStructDef.Fields {
						concreteField := field
						concreteField.Type = utils.ReplaceTypeParams(field.Type, genericStructDef.TypeParams, processedGenArgs)
						concreteStructDef.Fields = append(concreteStructDef.Fields, concreteField)
					}

					structDefinitions[concreteKey] = concreteStructDef
					log.Printf("Created concrete struct '%s' for generic type instantiation.", concreteTypeName)

					// Update the result type to the concrete type
					apiFunc.Results[len(apiFunc.Results)-1].Type = concreteTypeName
				} else {
					log.Printf("Concrete struct '%s' already exists.", concreteTypeName)
					apiFunc.Results[len(apiFunc.Results)-1].Type = concreteTypeName
				}
			}
		} else {
			// Non-generic struct - we already resolved and nothing special needed
			if baseName != "" && basePkg != "" {
				// Update the result type if needed to a fully qualified name if desired
				// For consistency, we keep the original name. It's optional to transform result type to a qualified name.
			}
		}
	}

	if apiFunc.Command == "" {
		return apiFunc, ErrMissingCommand
	}
	if apiFunc.Description == "" {
		return apiFunc, ErrMissingDescription
	}

	return apiFunc, nil
}

func parseGlobalTags(cg *ast.CommentGroup) (models.ProjectInfo, error) {
	projectInfo := models.ProjectInfo{}
	scanner := bufio.NewScanner(strings.NewReader(cg.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

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

func extractImportAliases(fileAst *ast.File) map[string]string {
	importAliases := make(map[string]string)
	for _, imp := range fileAst.Imports {
		var alias string
		var pkgName string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			path := strings.Trim(imp.Path.Value, `"`)
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		path := strings.Trim(imp.Path.Value, `"`)
		parts := strings.Split(path, "/")
		pkgName = parts[len(parts)-1]
		importAliases[alias] = pkgName
	}
	return importAliases
}

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

// resolvePackageAndType returns a package and name for any type.
// If it's fully qualified (package.struct), it splits it.
// If not, it tries to find it in the current package or import aliases.
// For generics, we do not attempt to resolve package per argument here; it's done later.
func resolvePackageAndType(typ string, currentPackage string, importAliases map[string]string, structDefinitions map[models.StructKey]models.StructDefinition) (pkg string, typeName string) {
	if strings.Contains(typ, ".") {
		// Possibly fully qualified or alias
		p, n := utils.SplitQualifiedName(typ)
		if p != "" && n != "" {
			// Check if p is an alias
			if actualPkg, exists := importAliases[p]; exists {
				return actualPkg, n
			}
			// p is actually the package name
			return p, n
		}
		// If we can't split properly, just return empty
		return "", typ
	}

	// Unqualified name: assume current package if it exists
	key := models.StructKey{
		Package: currentPackage,
		Name:    typ,
	}
	if _, exists := structDefinitions[key]; exists {
		return currentPackage, typ
	}

	// Not found
	log.Printf("Type '%s' not found in package '%s'. Ensure it is imported or fully qualified.", typ, currentPackage)
	return "", ""
}
