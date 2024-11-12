package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pablolagos/jdocgen/generator"
	"github.com/pablolagos/jdocgen/parser"
)

func main() {
	outputPath := flag.String("output", "API_Documentation.md", "Path to the output Markdown file")
	dirPath := flag.String("dir", ".", "Directory of the Go project to parse")
	flag.Parse()

	apiFunctions, structs, projectInfo, err := parser.ParseProject(*dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing project: %v\n", err)
		os.Exit(1)
	}

	// Adjust StructKey.Package as needed; if your structs are all in the same package, set accordingly
	// For simplicity, assuming all structs are in the root package (Package: "")
	// Modify this logic based on your project's package structure
	for key := range structs {
		if key.Package == "" {
			key.Package = "handlers" // Replace with your actual package name if different
			structs[key] = structs[key]
		}
	}

	markdown := generator.GenerateMarkdown(apiFunctions, structs, projectInfo)

	err = os.WriteFile(*outputPath, []byte(markdown), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Documentation generated successfully at %s\n", *outputPath)
}
