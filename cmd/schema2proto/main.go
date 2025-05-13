package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adimarco/bifrost/pkg/converter"
)

func main() {
	inputFile := flag.String("input", "", "Input JSON Schema file")
	outputFile := flag.String("output", "", "Output .proto file")
	packageName := flag.String("package", "schema", "Package name for the generated proto file")
	goPackage := flag.String("go-package", "", "Go package path (e.g., github.com/user/project)")
	imports := flag.String("imports", "google/protobuf/any.proto", "Comma-separated list of proto imports")
	typeAliases := flag.String("type-aliases", "", "Comma-separated list of type aliases in format 'type=alias' (e.g., 'Requestid=string,RequestId=string')")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		fmt.Println("Please provide both input and output file paths")
		flag.Usage()
		os.Exit(1)
	}

	// Parse imports
	importList := strings.Split(*imports, ",")
	for i, imp := range importList {
		importList[i] = strings.TrimSpace(imp)
	}

	// Parse type aliases
	typeAliasMap := make(map[string]string)
	if *typeAliases != "" {
		aliases := strings.Split(*typeAliases, ",")
		for _, alias := range aliases {
			parts := strings.Split(strings.TrimSpace(alias), "=")
			if len(parts) == 2 {
				typeAliasMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Read and parse the JSON Schema
	schemaData, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading schema file: %v\n", err)
		os.Exit(1)
	}

	// Create converter options
	opts := &converter.Options{
		PackageName:  *packageName,
		TypeMappings: typeAliasMap,
	}

	// Convert schema to proto
	protoContent, err := converter.ConvertJSONSchemaToProto(string(schemaData), opts)
	if err != nil {
		fmt.Printf("Error converting schema: %v\n", err)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(*outputFile), 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Write the proto file
	if err := os.WriteFile(*outputFile, []byte(protoContent), 0644); err != nil {
		fmt.Printf("Error writing proto file: %v\n", err)
		os.Exit(1)
	}

	// If go_package is specified, add it to the proto file
	if *goPackage != "" {
		content, err := os.ReadFile(*outputFile)
		if err != nil {
			fmt.Printf("Error reading proto file: %v\n", err)
			os.Exit(1)
		}

		lines := strings.Split(string(content), "\n")
		var newContent strings.Builder
		for i, line := range lines {
			if i == 2 { // After package declaration
				newContent.WriteString(fmt.Sprintf("option go_package = \"%s\";\n\n", *goPackage))
			}
			newContent.WriteString(line)
			newContent.WriteString("\n")
		}

		if err := os.WriteFile(*outputFile, []byte(newContent.String()), 0644); err != nil {
			fmt.Printf("Error writing updated proto file: %v\n", err)
			os.Exit(1)
		}
	}
}
