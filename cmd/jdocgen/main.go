package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablolagos/jdocgen/generator"
	"github.com/pablolagos/jdocgen/parser"
)

func main() {
	outputPath := flag.String("output", "API_Documentation.md", "Path to the output Markdown file")
	dirPath := flag.String("dir", ".", "Directory of the Go project to parse")
	flag.Parse()

	// Resolve rootDir if it's a relative path
	if !filepath.IsAbs(*dirPath) {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
			os.Exit(1)
		}
		*dirPath = filepath.Join(wd, *dirPath)
	}

	apiFunctions, structs, projectInfo, err := parser.ParseProject(*dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing project: %v\n", err)
		os.Exit(1)
	}

	markdown := generator.GenerateMarkdown(apiFunctions, structs, projectInfo)

	err = os.WriteFile(*outputPath, []byte(markdown), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing documentation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Documentation generated successfully at %s\n", *outputPath)
}
