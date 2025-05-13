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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertJSONSchemaToProto(tt.schema)
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
			got := GetProtoType(tt.jsonType, tt.format)
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
