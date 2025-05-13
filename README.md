# JSON Schema to Protocol Buffers Converter

A tool to convert JSON Schema to Protocol Buffers (protobuf) format, with support for customizing the output through various options.

## Features

- Converts JSON Schema to Protocol Buffers format
- Supports custom package names
- Configurable Go package path
- Customizable type mappings
- Support for custom imports
- Handles nested objects and arrays
- Preserves field descriptions as comments
- Generates valid proto3 syntax

## Installation

You can install the tool directly using `go install`:

```bash
# Install the latest version
go install github.com/adimarco/bifrost/cmd/schema2proto@latest

# Or install a specific version
go install github.com/adimarco/bifrost/cmd/schema2proto@v1.0.0
```

After installation, the `schema2proto` binary will be available in your `$GOPATH/bin` directory. Make sure this directory is in your `$PATH`.

Alternatively, you can clone and build from source:

```bash
git clone https://github.com/adimarco/bifrost.git
cd bifrost
make build
```

The binary will be created in the `target/` directory.

## Usage

```bash
schema2proto -input schema.json -output schema.proto [options]
```

### Options

- `-input`: Input JSON Schema file (required)
- `-output`: Output .proto file (required)
- `-package`: Package name for the generated proto file (default: "schema")
- `-go-package`: Go package path (e.g., "github.com/user/project")
- `-imports`: Comma-separated list of proto imports (default: "google/protobuf/any.proto")
- `-type-aliases`: Comma-separated list of type aliases in format 'type=alias' (e.g., "Requestid=string,RequestId=string")

### Examples

1. Basic usage:
```bash
schema2proto -input schema.json -output schema.proto
```

2. Custom package name and Go package path:
```bash
schema2proto -input schema.json -output schema.proto -package mypackage -go-package github.com/user/project/mypackage
```

3. Custom imports:
```bash
schema2proto -input schema.json -output schema.proto -imports "google/protobuf/any.proto,google/protobuf/timestamp.proto"
```

4. Custom type aliases:
```bash
schema2proto -input schema.json -output schema.proto -type-aliases "Requestid=string,RequestId=string,UserID=int64"
```

## Input JSON Schema Example

```json
{
  "definitions": {
    "User": {
      "type": "object",
      "properties": {
        "id": {
          "type": "string",
          "description": "User's unique identifier"
        },
        "name": {
          "type": "string",
          "description": "User's full name"
        },
        "age": {
          "type": "integer",
          "description": "User's age in years"
        },
        "roles": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "User's roles"
        }
      }
    }
  }
}
```

## Generated Protocol Buffers Output

```protobuf
syntax = "proto3";

package mypackage;

option go_package = "github.com/user/project/mypackage";

import "google/protobuf/any.proto";

// User's unique identifier
message User {
  // User's unique identifier
  string id = 1;
  // User's full name
  string name = 2;
  // User's age in years
  int32 age = 3;
  // User's roles
  repeated string roles = 4;
}
```

## Building from Source

```bash
git clone https://github.com/adimarco/bifrost.git
cd bifrost
make build
```

The binary will be created in the `target/` directory.

## Testing

```bash
make test
```

## License

MIT License 