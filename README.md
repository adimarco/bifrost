# JSON Schema to Protocol Buffers Converter

This tool converts JSON Schema definitions to Protocol Buffers format, making it easy to use JSON Schema definitions in gRPC services.

## Features

- Converts JSON Schema to Protocol Buffers format
- Preserves descriptions as comments
- Handles nested types and references
- Generates proper field numbers
- Supports enums and repeated fields
- Maintains type mappings (string, number, boolean, etc.)

## Installation

```bash
go install github.com/adimarco/bifrost/cmd/schema2proto@latest
```

## Usage

```bash
schema2proto -input schema.json -output schema.proto -package mypackage
```

### Arguments

- `-input`: Path to the input JSON Schema file (required)
- `-output`: Path to the output .proto file (required)
- `-package`: Package name for the generated proto file (default: "schema")

## Example

Given a JSON Schema like:

```json
{
  "definitions": {
    "Person": {
      "type": "object",
      "properties": {
        "name": {
          "type": "string",
          "description": "The person's name"
        },
        "age": {
          "type": "integer",
          "description": "The person's age"
        }
      }
    }
  }
}
```

The tool will generate a .proto file like:

```protobuf
syntax = "proto3";

package mypackage;

message Person {
  // The person's name
  string name = 1;
  // The person's age
  int64 age = 2;
}
```

## Notes

- The tool automatically handles references to other definitions
- Field numbers are generated sequentially starting from 1
- Enums are converted to Protocol Buffers enums
- Arrays are converted to repeated fields
- Object types without specific properties are converted to `google.protobuf.Any` 