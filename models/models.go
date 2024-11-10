package models

// APIFunction represents a JSON-RPC API function.
type APIFunction struct {
	Command     string
	Description string
	Parameters  []APIParameter
	Results     []APIReturn
}

// APIParameter represents a parameter of an API function.
type APIParameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// APIReturn represents a return value of an API function.
type APIReturn struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// StructDefinition represents a Go struct definition.
type StructDefinition struct {
	Name   string
	Fields []StructField
}

// StructField represents a field within a Go struct.
type StructField struct {
	Name        string
	Type        string
	Description string
	JSONName    string
}

// ProjectInfo holds global metadata about the project.
type ProjectInfo struct {
	Title       string
	Version     string
	Description string
	Author      string
	Copyright   string
	License     string
	Contact     string
	Terms       string
	Repository  string
	Tags        []string
	// Add other global fields as needed
}
