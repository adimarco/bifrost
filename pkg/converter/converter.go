package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ConvertJSONSchemaToProto converts a JSON Schema to Protocol Buffers format
func ConvertJSONSchemaToProto(schemaStr string) (string, error) {
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		return "", fmt.Errorf("failed to parse JSON schema: %v", err)
	}

	var proto strings.Builder
	proto.WriteString("syntax = \"proto3\";\n\n")
	proto.WriteString("package schema;\n\n")

	// Collect message definitions
	messages := make(map[string]string)

	// Generate root message fields
	rootFields := &strings.Builder{}
	fieldNumber := 1
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			prop := props[name]
			fieldType, err := processPropertyCollect(name, prop, messages)
			if err != nil {
				return "", err
			}
			if fieldType != "" {
				rootFields.WriteString(fmt.Sprintf("  %s %s = %d;\n", fieldType, SanitizeFieldName(name), fieldNumber))
				fieldNumber++
			}
		}
	}
	messages["Root"] = fmt.Sprintf("message Root {\n%s}\n", rootFields.String())

	// Emit messages in sorted order, Root first
	msgNames := make([]string, 0, len(messages))
	for k := range messages {
		msgNames = append(msgNames, k)
	}
	sort.Strings(msgNames)
	if msgNames[0] != "Root" {
		for i, n := range msgNames {
			if n == "Root" {
				msgNames[0], msgNames[i] = msgNames[i], msgNames[0]
				break
			}
		}
	}
	for _, name := range msgNames {
		proto.WriteString(messages[name])
		if !strings.HasSuffix(messages[name], "\n") {
			proto.WriteString("\n")
		}
	}
	return proto.String(), nil
}

// GetProtoType returns the Protocol Buffers type for a given JSON Schema type
func GetProtoType(jsonType string, format string) string {
	switch jsonType {
	case "string":
		if format == "date-time" {
			return "string" // Could be google.protobuf.Timestamp if needed
		}
		return "string"
	case "integer":
		return "int32"
	case "number":
		return "double"
	case "boolean":
		return "bool"
	case "array":
		return "repeated"
	case "object":
		return "message"
	default:
		return "string" // Default to string for unknown types
	}
}

// SanitizeFieldName converts a JSON field name to a valid Protocol Buffers field name
func SanitizeFieldName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-z0-9]`)
	name = reg.ReplaceAllString(name, "_")

	// If the name starts with a number, move the number to the end
	if len(name) > 0 && regexp.MustCompile(`^[0-9]`).MatchString(name) {
		numbers := regexp.MustCompile(`^[0-9]+`).FindString(name)
		rest := regexp.MustCompile(`^[0-9]+`).ReplaceAllString(name, "")
		if rest == "" {
			rest = "field"
		}
		name = rest + numbers
	}

	return name
}

// processPropertyCollect returns the proto type for a property, and collects message definitions in messages map
func processPropertyCollect(name string, prop interface{}, messages map[string]string) (string, error) {
	propMap, ok := prop.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid property format for %s", name)
	}

	propType, _ := propMap["type"].(string)
	format, _ := propMap["format"].(string)

	switch propType {
	case "array":
		items, ok := propMap["items"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid array items format for %s", name)
		}
		itemType, err := processPropertyCollect(name+"Item", items, messages)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("repeated %s", itemType), nil

	case "object":
		messageName := strings.Title(SanitizeFieldName(name))
		if _, exists := messages[messageName]; !exists {
			fields := &strings.Builder{}
			if props, ok := propMap["properties"].(map[string]interface{}); ok {
				keys := make([]string, 0, len(props))
				for k := range props {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				fieldNumber := 1
				for _, nestedName := range keys {
					nestedProp := props[nestedName]
					fieldType, err := processPropertyCollect(nestedName, nestedProp, messages)
					if err != nil {
						return "", err
					}
					if fieldType != "" {
						fields.WriteString(fmt.Sprintf("  %s %s = %d;\n", fieldType, SanitizeFieldName(nestedName), fieldNumber))
						fieldNumber++
					}
				}
			}
			messages[messageName] = fmt.Sprintf("message %s {\n%s}\n", messageName, fields.String())
		}
		return messageName, nil

	default:
		return GetProtoType(propType, format), nil
	}
}
