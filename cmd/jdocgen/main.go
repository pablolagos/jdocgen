// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/pablolagos/jdocgen/generator"
	"github.com/pablolagos/jdocgen/parser"
)

func main() {
	// Define command-line flags
	outputPath := flag.String("output", "API_Documentation.md", "Path to the output Markdown file")
	dirPath := flag.String("dir", ".", "Directory to parse for Go source files")
	omitRFC := flag.Bool("omit-rfc", false, "Omit JSON-RPC 2.0 specification information from the documentation")

	flag.Parse()

	// Resolve absolute directory path
	absDir, err := filepath.Abs(*dirPath)
	if err != nil {
		log.Fatalf("Error resolving directory path: %v", err)
	}

	// Parse the project to collect API functions and all struct definitions
	apiFunctions, structs, projectInfo, err := parser.ParseProject(absDir)
	if err != nil {
		log.Fatalf("Error parsing project: %v", err)
	}

	// Generate Markdown documentation for API endpoints
	err = generator.GenerateDocumentation(apiFunctions, structs, projectInfo, *outputPath, !*omitRFC)
	if err != nil {
		log.Fatalf("Error generating documentation: %v", err)
	}

	fmt.Printf("Documentation successfully generated at %s\n", *outputPath)
}
