// models/models.go
package models

// StructKey uniquely identifies a struct by its package and name.
type StructKey struct {
	Package string
	Name    string
}

// StructDefinition represents the definition of a struct, including its fields and description.
type StructDefinition struct {
	Name        string
	Description string
	Fields      []StructField
	TypeParams  []TypeParam
}

// StructField represents a single field within a struct.
type StructField struct {
	Name        string
	Type        string
	Description string
	JSONName    string
}

// TypeParam represents a type parameter for generic structs.
type TypeParam struct {
	Name       string
	Constraint string
}

// APIFunction represents an API function with its annotations.
type APIFunction struct {
	Command           string
	Description       string
	Parameters        []APIParameter
	Results           []APIReturn
	Errors            []APIError
	ImportAliases     map[string]string
	PackageName       string
	AdditionalStructs []string
}

// APIParameter represents a parameter of an API function.
type APIParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIReturn represents the return value of an API function.
type APIReturn struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIError represents an error that an API function can return.
type APIError struct {
	Code        int
	Description string
}

// ProjectInfo holds global tags and metadata for the project.
type ProjectInfo struct {
	Title       string
	Version     string
	Description string
	Author      string
	License     string
	Contact     string
	Terms       string
	Repository  string
	Tags        []string
	Copyright   string
}
