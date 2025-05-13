package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Options contains configuration options for the converter
type Options struct {
	PackageName  string
	TypeMappings map[string]string
}

// DefaultOptions returns the default options for the converter
func DefaultOptions() *Options {
	return &Options{
		PackageName: "schema",
		TypeMappings: map[string]string{
			"string":  "string",
			"integer": "int32",
			"number":  "double",
			"boolean": "bool",
			"array":   "repeated",
			"object":  "message",
		},
	}
}

// ConvertJSONSchemaToProto converts a JSON Schema to Protocol Buffers format
func ConvertJSONSchemaToProto(schemaStr string, opts *Options) (string, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		return "", fmt.Errorf("failed to parse JSON schema: %v", err)
	}

	var proto strings.Builder
	proto.WriteString("syntax = \"proto3\";\n\n")
	proto.WriteString(fmt.Sprintf("package %s;\n\n", opts.PackageName))

	// Collect message definitions
	messages := make(map[string]string)

	// Generate root message fields (if any)
	rootFields := &strings.Builder{}
	fieldNumber := 1
	rootMsgComment := ""
	if desc, ok := schema["description"].(string); ok && desc != "" {
		rootMsgComment = formatDescription(desc)
	}
	if props, ok := schema["properties"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(props))
		for k := range props {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			prop := props[name]
			// Add field description if present
			if propMap, ok := prop.(map[string]interface{}); ok {
				if desc, ok := propMap["description"].(string); ok && desc != "" {
					rootFields.WriteString(formatDescription(desc))
				}
			}
			fieldType, err := processPropertyCollect(name, prop, messages, opts)
			if err != nil {
				return "", err
			}
			if fieldType != "" {
				rootFields.WriteString(fmt.Sprintf("  %s %s = %d;\n", fieldType, SanitizeFieldName(name), fieldNumber))
				fieldNumber++
			}
		}
		messages["Root"] = fmt.Sprintf("%smessage Root {\n%s}\n", rootMsgComment, rootFields.String())
	}

	// Process definitions
	if defs, ok := schema["definitions"].(map[string]interface{}); ok {
		defNames := make([]string, 0, len(defs))
		for defName := range defs {
			defNames = append(defNames, defName)
		}
		sort.Strings(defNames)
		for _, defName := range defNames {
			def := defs[defName]
			if defMap, ok := def.(map[string]interface{}); ok {
				fields := &strings.Builder{}
				fieldNumber := 1
				if props, ok := defMap["properties"].(map[string]interface{}); ok {
					keys := make([]string, 0, len(props))
					for k := range props {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, propName := range keys {
						prop := props[propName]
						// Add field description if present
						if propMap, ok := prop.(map[string]interface{}); ok {
							if desc, ok := propMap["description"].(string); ok && desc != "" {
								fields.WriteString(formatDescription(desc))
							}
						}
						fieldType, err := processPropertyCollect(propName, prop, messages, opts)
						if err != nil {
							return "", err
						}
						if fieldType != "" {
							fields.WriteString(fmt.Sprintf("  %s %s = %d;\n", fieldType, SanitizeFieldName(propName), fieldNumber))
							fieldNumber++
						}
					}
				}
				msgComment := ""
				// Add message description if present
				if desc, ok := defMap["description"].(string); ok && desc != "" {
					msgComment = formatDescription(desc)
				}
				messages[defName] = fmt.Sprintf("%smessage %s {\n%s}\n", msgComment, defName, fields.String())
			}
		}
	}

	// Emit messages in sorted order, Root first if present
	msgNames := make([]string, 0, len(messages))
	for k := range messages {
		msgNames = append(msgNames, k)
	}
	sort.Strings(msgNames)
	// Move 'Root' to the front if present
	if len(msgNames) > 0 {
		for i, n := range msgNames {
			if n == "Root" && i != 0 {
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
func GetProtoType(jsonType string, format string, opts *Options) string {
	if opts == nil {
		opts = DefaultOptions()
	}

	if format == "date-time" {
		return "string" // Could be google.protobuf.Timestamp if needed
	}

	if protoType, ok := opts.TypeMappings[jsonType]; ok {
		return protoType
	}
	return "string" // Default to string for unknown types
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
func processPropertyCollect(name string, prop interface{}, messages map[string]string, opts *Options) (string, error) {
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
		itemType, err := processPropertyCollect(name+"Item", items, messages, opts)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("repeated %s", itemType), nil

	case "object":
		messageName := toProtoMessageName(name)
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
					fieldType, err := processPropertyCollect(nestedName, nestedProp, messages, opts)
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
		return GetProtoType(propType, format, opts), nil
	}
}

// formatDescription formats a description string as a proto comment
func formatDescription(desc string) string {
	lines := strings.Split(desc, "\n")
	var out strings.Builder
	for _, line := range lines {
		out.WriteString("// ")
		out.WriteString(line)
		out.WriteString("\n")
	}
	return out.String()
}

// toProtoMessageName converts a JSON field name to a valid Protocol Buffers message name
func toProtoMessageName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
