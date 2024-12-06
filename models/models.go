// models/struct.go
package models

// StructKey uniquely identifies a struct by its package and name.
type StructKey struct {
	Package string
	Name    string
}

// StructField represents a single field within a struct.
type StructField struct {
	Name        string
	Type        string
	Description string
	JSONName    string
}

// StructDefinition represents the definition of a struct, including its fields.
type StructDefinition struct {
	Name        string
	Description string
	Fields      []StructField
}

// APIFunction represents a JSON-RPC API function, including its parameters and results.
type APIFunction struct {
	Command       string
	Description   string
	Parameters    []APIParameter
	Results       []APIReturn
	Errors        []APIError
	ImportAliases map[string]string
	PackageName   string
}

// APIParameter represents a single parameter for an API function.
type APIParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIReturn represents a single return value for an API function.
type APIReturn struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIError represents a single error that an API function might return.
type APIError struct {
	Code        int
	Description string
}

// ProjectInfo holds global information about the project.
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
