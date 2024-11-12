package parser

import (
	"bufio"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pablolagos/jdocgen/models"
)

// ParseProject recursively parses all Go files in the project directory and its subdirectories.
// It returns a slice of APIFunctions, a map of StructDefinitions, and ProjectInfo.
func ParseProject(rootDir string) ([]models.APIFunction, map[string]models.StructDefinition, models.ProjectInfo, error) {
	var apiFunctions []models.APIFunction
	structDefinitions := make(map[string]models.StructDefinition)
	var projectInfo models.ProjectInfo
	projectInfoSet := false

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor, hidden directories, and test files
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
		fset := token.NewFileSet()
		fileAst, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

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
				}
				structDefinitions[structDef.Name] = structDef
			}
		}

		// Parse functions with annotations
		for _, decl := range fileAst.Decls {
			fn, isFn := decl.(*ast.FuncDecl)
			if !isFn || fn.Doc == nil {
				continue
			}

			// Extract API functions
			apiFunc, err := parseFunction(fn)
			if err == nil {
				apiFunctions = append(apiFunctions, apiFunc)
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

// parseFunction parses a function's comments to extract API annotations.
func parseFunction(fn *ast.FuncDecl) (models.APIFunction, error) {
	apiFunc := models.APIFunction{}
	scanner := bufio.NewScanner(strings.NewReader(fn.Doc.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "@") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
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
			if len(parts) < 3 {
				return apiFunc, errors.New("invalid @Result annotation")
			}
			resultName := parts[1]
			resultType := parts[2]
			// Check for 'optional' keyword in @Result
			if len(parts) > 3 && strings.EqualFold(parts[3], "optional") {
				return apiFunc, errors.New("@Result annotations should not be marked as optional")
			}
			resultDescParts := parts[3:]
			resultDesc := strings.Join(resultDescParts, " ")
			result := models.APIReturn{
				Name:        resultName,
				Type:        resultType,
				Description: resultDesc,
				Required:    true, // All return values are required
			}
			apiFunc.Results = append(apiFunc.Results, result)
		}
	}

	// Validate required annotations
	if apiFunc.Command == "" {
		return apiFunc, errors.New("missing @Command annotation")
	}
	if apiFunc.Description == "" {
		return apiFunc, errors.New("missing @Description annotation")
	}

	return apiFunc, nil
}

// parseGlobalTags parses global tags from a CommentGroup (file-level or function-level).
func parseGlobalTags(cg *ast.CommentGroup) (models.ProjectInfo, error) {
	projectInfo := models.ProjectInfo{}
	scanner := bufio.NewScanner(strings.NewReader(cg.Text()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "@") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
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
