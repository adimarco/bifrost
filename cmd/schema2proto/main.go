package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type ProtoField struct {
	Name        string
	Type        string
	Number      int
	IsRepeated  bool
	Description string
}

type ProtoMessage struct {
	Name        string
	Fields      []ProtoField
	Description string
}

type ProtoEnum struct {
	Name        string
	Values      []string
	Description string
}

type ProtoFile struct {
	Package  string
	Messages []ProtoMessage
	Enums    []ProtoEnum
	Imports  []string
}

var typeAliases = map[string]string{
	"Requestid": "string",
	"RequestId": "string",
}

func main() {
	inputFile := flag.String("input", "", "Input JSON Schema file")
	outputFile := flag.String("output", "", "Output .proto file")
	packageName := flag.String("package", "schema", "Package name for the generated proto file")
	flag.Parse()

	if *inputFile == "" || *outputFile == "" {
		fmt.Println("Please provide both input and output file paths")
		flag.Usage()
		os.Exit(1)
	}

	// Read and parse the JSON Schema
	schemaData, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Printf("Error reading schema file: %v\n", err)
		os.Exit(1)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		fmt.Printf("Error parsing schema: %v\n", err)
		os.Exit(1)
	}

	// Create proto file structure
	protoFile := ProtoFile{
		Package: *packageName,
		Imports: []string{"google/protobuf/any.proto"},
	}

	// Process definitions
	if definitions, ok := schema["definitions"].(map[string]interface{}); ok {
		for name, def := range definitions {
			processDefinition(name, def, &protoFile)
		}
	}

	// Generate the proto file
	if err := generateProtoFile(*outputFile, protoFile); err != nil {
		fmt.Printf("Error generating proto file: %v\n", err)
		os.Exit(1)
	}
}

func processDefinition(name string, def interface{}, protoFile *ProtoFile) {
	defMap, ok := def.(map[string]interface{})
	if !ok {
		return
	}

	// Handle enums
	if enumValues, ok := defMap["enum"].([]interface{}); ok {
		enum := ProtoEnum{
			Name:        toProtoName(name),
			Description: formatDescription(getDescription(defMap)),
		}
		for _, v := range enumValues {
			if str, ok := v.(string); ok {
				enum.Values = append(enum.Values, str)
			}
		}
		protoFile.Enums = append(protoFile.Enums, enum)
		return
	}

	// Handle messages
	if properties, ok := defMap["properties"].(map[string]interface{}); ok {
		message := ProtoMessage{
			Name:        toProtoName(name),
			Description: formatDescription(getDescription(defMap)),
		}

		fieldNumber := 1
		for propName, prop := range properties {
			field := processProperty(propName, prop, &fieldNumber)
			if field.Name == "" {
				log.Printf("Warning: Skipping property with empty name in definition %s", name)
				continue
			}
			if field.Type == "" {
				log.Printf("Warning: Property %s in definition %s has empty type, defaulting to google.protobuf.Any", propName, name)
				field.Type = "google.protobuf.Any"
			}
			message.Fields = append(message.Fields, field)
		}

		protoFile.Messages = append(protoFile.Messages, message)
	}
}

func processProperty(name string, prop interface{}, fieldNumber *int) ProtoField {
	propMap, ok := prop.(map[string]interface{})
	if !ok {
		return ProtoField{}
	}

	field := ProtoField{
		Name:        toProtoName(name),
		Number:      *fieldNumber,
		Description: formatDescription(getDescription(propMap)),
	}

	// Handle $ref
	if ref, ok := propMap["$ref"].(string); ok {
		field.Type = toProtoName(strings.TrimPrefix(ref, "#/definitions/"))
		*fieldNumber++
		return field
	}

	// Handle type
	if typeStr, ok := propMap["type"].(string); ok {
		if typeStr == "array" {
			field.IsRepeated = true
			if items, ok := propMap["items"].(map[string]interface{}); ok {
				// Array of references
				if ref, ok := items["$ref"].(string); ok {
					field.Type = toProtoName(strings.TrimPrefix(ref, "#/definitions/"))
				} else if itemType, ok := items["type"].(string); ok {
					field.Type = protoType(itemType)
				} else {
					log.Printf("Warning: Array property %s has ambiguous item type, defaulting to google.protobuf.Any", name)
					field.Type = "google.protobuf.Any"
				}
			} else {
				log.Printf("Warning: Array property %s has no items, defaulting to google.protobuf.Any", name)
				field.Type = "google.protobuf.Any"
			}
		} else {
			field.Type = protoType(typeStr)
		}
	} else if anyOf, ok := propMap["anyOf"].([]interface{}); ok && len(anyOf) > 0 {
		log.Printf("Warning: Property %s uses anyOf, defaulting to google.protobuf.Any", name)
		field.Type = "google.protobuf.Any"
	} else if oneOf, ok := propMap["oneOf"].([]interface{}); ok && len(oneOf) > 0 {
		log.Printf("Warning: Property %s uses oneOf, defaulting to google.protobuf.Any", name)
		field.Type = "google.protobuf.Any"
	} else {
		log.Printf("Warning: Property %s has no type or $ref, defaulting to google.protobuf.Any", name)
		field.Type = "google.protobuf.Any"
	}

	if alias, ok := typeAliases[field.Type]; ok {
		field.Type = alias
	}

	*fieldNumber++
	return field
}

func protoType(jsonType string) string {
	switch jsonType {
	case "string":
		return "string"
	case "number":
		return "double"
	case "integer":
		return "int64"
	case "boolean":
		return "bool"
	case "object":
		return "google.protobuf.Any"
	default:
		return "google.protobuf.Any"
	}
}

func toProtoName(name string) string {
	// Convert to PascalCase and remove special characters
	if name == "" {
		return ""
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.Title(strings.ToLower(part))
	}
	return strings.Join(parts, "")
}

func getDescription(def map[string]interface{}) string {
	if desc, ok := def["description"].(string); ok {
		return desc
	}
	return ""
}

func formatDescription(desc string) string {
	// Replace newlines with spaces and clean up multiple spaces
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.Join(strings.Fields(desc), " ")
	// Remove any non-ASCII characters
	var result strings.Builder
	for _, r := range desc {
		if r < 128 {
			result.WriteRune(r)
		} else {
			result.WriteRune(' ')
		}
	}
	return result.String()
}

func generateProtoFile(outputPath string, protoFile ProtoFile) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "syntax = \"proto3\";\n\n")
	fmt.Fprintf(file, "package %s;\n\n", protoFile.Package)
	fmt.Fprintf(file, "option go_package = \"github.com/adimarco/bifrost/mcp\";\n\n")

	// Write imports
	for _, imp := range protoFile.Imports {
		fmt.Fprintf(file, "import \"%s\";\n", imp)
	}
	fmt.Fprintf(file, "\n")

	// Write enums
	for _, enum := range protoFile.Enums {
		if enum.Description != "" {
			fmt.Fprintf(file, "// %s\n", enum.Description)
		}
		fmt.Fprintf(file, "enum %s {\n", enum.Name)
		for i, value := range enum.Values {
			fmt.Fprintf(file, "  %s = %d;\n", strings.ToUpper(value), i)
		}
		fmt.Fprintf(file, "}\n\n")
	}

	// Write messages
	for _, message := range protoFile.Messages {
		if message.Description != "" {
			fmt.Fprintf(file, "// %s\n", message.Description)
		}
		fmt.Fprintf(file, "message %s {\n", message.Name)
		for _, field := range message.Fields {
			if field.Description != "" {
				fmt.Fprintf(file, "  // %s\n", field.Description)
			}
			repeated := ""
			if field.IsRepeated {
				repeated = "repeated "
			}
			typeName := field.Type
			if alias, ok := typeAliases[typeName]; ok {
				typeName = alias
			}
			fmt.Fprintf(file, "  %s%s %s = %d;\n", repeated, typeName, field.Name, field.Number)
		}
		fmt.Fprintf(file, "}\n\n")
	}

	return nil
}
