// cmd/jdocgen/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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

	// Validate directory path
	absDir, err := filepath.Abs(*dirPath)
	if err != nil {
		log.Fatalf("Error resolving directory path: %v", err)
	}

	// Parse the project
	functions, structs, projectInfo, err := parser.ParseProject(absDir)
	if err != nil {
		log.Fatalf("Error parsing project: %v", err)
	}

	// Generate Markdown documentation
	markdown := generator.GenerateMarkdown(functions, structs, projectInfo, !*omitRFC)

	// Write to the output file
	err = os.WriteFile(*outputPath, []byte(markdown), 0644)
	if err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}

	fmt.Printf("Documentation successfully generated at %s\n", *outputPath)
}
