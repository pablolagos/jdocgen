package models

// StructKey uniquely identifies a struct by its package and name.
type StructKey struct {
	Package string
	Name    string
}

// APIFunction represents an API function with its annotations.
type APIFunction struct {
	Command       string
	Description   string
	Parameters    []APIParameter
	Results       []APIReturn
	Errors        []APIError        // Holds errors
	ImportAliases map[string]string // Maps alias to package name
	PackageName   string            // Local package name where the function resides
}

// APIParameter represents a parameter in an API function.
type APIParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIReturn represents a return value from an API function.
type APIReturn struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIError represents an error that an API function can return.
type APIError struct {
	Code        int    // Numeric error code
	Description string // Description of the error
}

// StructDefinition represents a struct with its fields.
type StructDefinition struct {
	Name   string
	Fields []StructField
}

// StructField represents a field within a struct.
type StructField struct {
	Name        string
	Type        string
	Description string
	JSONName    string
}

// ProjectInfo holds global project metadata extracted from annotations.
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
