package converter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertJSONSchemaToProto(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		expected string
		wantErr  bool
	}{
		{
			name: "simple object schema",
			schema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

message Root {
  int32 age = 1;
  string name = 2;
}
`,
			wantErr: false,
		},
		{
			name: "nested object schema",
			schema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string"
							},
							"address": {
								"type": "object",
								"properties": {
									"street": {
										"type": "string"
									},
									"city": {
										"type": "string"
									}
								}
							}
						}
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

message Root {
  User user = 1;
}

message Address {
  string city = 1;
  string street = 2;
}

message User {
  Address address = 1;
  string name = 2;
}
`,
			wantErr: false,
		},
		{
			name: "array schema",
			schema: `{
				"type": "object",
				"properties": {
					"tags": {
						"type": "array",
						"items": {
							"type": "string"
						}
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

message Root {
  repeated string tags = 1;
}`,
			wantErr: false,
		},
		{
			name:     "invalid json",
			schema:   `{invalid json}`,
			expected: "",
			wantErr:  true,
		},
		{
			name: "schema with descriptions",
			schema: `{
				"type": "object",
				"description": "A test object with descriptions",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the object"
					},
					"age": {
						"type": "integer",
						"description": "The age of the object"
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

// A test object with descriptions
message Root {
// The age of the object
  int32 age = 1;
// The name of the object
  string name = 2;
}
`,
			wantErr: false,
		},
		{
			name: "schema with definitions",
			schema: `{
				"definitions": {
					"Person": {
						"type": "object",
						"properties": {
							"name": {"type": "string"},
							"age": {"type": "integer"}
						}
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

message Person {
  int32 age = 1;
  string name = 2;
}
`,
			wantErr: false,
		},
		{
			name: "definition with descriptions",
			schema: `{
				"definitions": {
					"Pet": {
						"type": "object",
						"description": "A pet object",
						"properties": {
							"species": {"type": "string", "description": "The species of the pet"},
							"age": {"type": "integer", "description": "The age of the pet"}
						}
					}
				}
			}`,
			expected: `syntax = "proto3";

package schema;

// A pet object
message Pet {
// The age of the pet
  int32 age = 1;
// The species of the pet
  string species = 2;
}
`,
			wantErr: false,
		},
		{
			name:   "empty properties",
			schema: `{"type": "object", "properties": {}}`,
			expected: `syntax = "proto3";

package schema;

message Root {
}
`,
			wantErr: false,
		},
		{
			name:   "empty definitions",
			schema: `{"definitions": {}}`,
			expected: `syntax = "proto3";

package schema;
`,
			wantErr: false,
		},
		{
			name:   "custom package name",
			schema: `{"type": "object", "properties": {"foo": {"type": "string"}}}`,
			expected: `syntax = "proto3";

package custompkg;

message Root {
  string foo = 1;
}
`,
			wantErr: false,
		},
		{
			name:   "custom type mapping",
			schema: `{"type": "object", "properties": {"flag": {"type": "boolean"}}}`,
			expected: `syntax = "proto3";

package schema;

message Root {
  BOOL flag = 1;
}
`,
			wantErr: false,
		},
		{
			name:   "array of objects",
			schema: `{"type": "object", "properties": {"items": {"type": "array", "items": {"type": "object", "properties": {"id": {"type": "string"}}}}}}`,
			expected: `syntax = "proto3";

package schema;

message Root {
  repeated ItemsItem items = 1;
}

message ItemsItem {
  string id = 1;
}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts *Options
			if tt.name == "custom package name" {
				opts = &Options{PackageName: "custompkg", TypeMappings: DefaultOptions().TypeMappings}
			} else if tt.name == "custom type mapping" {
				opts = &Options{PackageName: "schema", TypeMappings: map[string]string{"boolean": "BOOL"}}
			} else {
				opts = DefaultOptions()
			}
			got, err := ConvertJSONSchemaToProto(tt.schema, opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			// Normalize whitespace and newlines for comparison
			norm := func(s string) string {
				s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
				s = strings.ReplaceAll(s, "\n\n", "\n")
				for strings.Contains(s, "\n\n") {
					s = strings.ReplaceAll(s, "\n\n", "\n")
				}
				return s
			}
			assert.Equal(t, norm(tt.expected), norm(got))
		})
	}
}

func TestGetProtoType(t *testing.T) {
	tests := []struct {
		name     string
		jsonType string
		format   string
		want     string
	}{
		{"string type", "string", "", "string"},
		{"integer type", "integer", "", "int32"},
		{"number type", "number", "", "double"},
		{"boolean type", "boolean", "", "bool"},
		{"date-time format", "string", "date-time", "string"},
		{"unknown type", "unknown", "", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetProtoType(tt.jsonType, tt.format, DefaultOptions())
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanitizeFieldName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "name", "name"},
		{"with spaces", "user name", "user_name"},
		{"with special chars", "user-name", "user_name"},
		{"with numbers", "user123", "user123"},
		{"starts with number", "123user", "user123"},
		{"all caps", "USERNAME", "username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFieldName(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
