package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pablolagosm/jdocgen/generator"
	"github.com/pablolagosm/jdocgen/parser"
)

func main() {
	// Define command-line flags
	output := flag.String("output", "API_Documentation.md", "Output file for the documentation")
	dir := flag.String("dir", ".", "Root directory to parse for Go files")
	flag.Parse()

	// Verify that the directory exists
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Error determining absolute path: %v", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", absDir)
	}

	// Parse the project directory recursively
	functions, structs, projectInfo, err := parser.ParseProject(absDir)
	if err != nil {
		log.Fatalf("Error parsing project: %v", err)
	}

	if len(functions) == 0 {
		log.Println("No API functions found with the specified annotations.")
	}

	// Generate Markdown
	markdown := generator.GenerateMarkdown(functions, structs, projectInfo)

	// Write to the output file
	err = ioutil.WriteFile(*output, []byte(markdown), 0644)
	if err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}

	fpath, _ := filepath.Abs(*output)
	fmt.Printf("Documentation successfully generated at %s\n", fpath)
}
