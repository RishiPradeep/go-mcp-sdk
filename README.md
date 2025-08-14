# Go MCP SDK

This is a Go SDK for building servers that comply with the Multi-Capability Protocol (MCP). It provides the basic building blocks for creating an MCP server, registering tools, and handling requests.

## Disclaimer

This SDK is currently under active development and should be considered a work in progress. The API may change, and some features may not be fully implemented. Please use with caution and feel free to contribute to its development.

## Features

*   **MCP Server**: A server that can handle MCP requests.
*   **Tool Registration**: An easy way to register tools and their handlers.
*   **JSON Schema Generation**: Automatic generation of JSON schemas for tool inputs.
*   **Structured Logging**: Colored, structured logging for easy debugging.

## Getting Started

This tutorial will guide you through the process of creating a simple calculator server using the Go MCP SDK.

### Prerequisites

*   Go 1.21 or later installed on your system.

### Using the SDK

Since this SDK has not been published to a public repository, you will need to use it as a local module. To do this, you can use a `replace` directive in your `go.mod` file to point to your local copy of the `go-mcp-sdk` repository.

For example:
```
replace go-mcp-sdk => /path/to/your/local/go-mcp-sdk
```

### Creating a Server

Here are the steps to create a new MCP server:

**1. Initialize the Server**

First, create a new server instance with its name, version, and capabilities.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go-mcp-sdk/pkg/mcp"
	"go-mcp-sdk/pkg/protocol"
)

func main() {
	server := mcp.NewServer("GoCalculatorServer", "1.0.0", protocol.ServerCapabilities{
		Tools: &protocol.ServerToolCapabilities{},
	})

	// ...
}
```

**2. Define Tool Parameter Structs**

For each tool, define a struct that represents its input parameters. The SDK will automatically generate a JSON schema from these structs.

```go
// AddParams defines the input for our "add" tool.
// The `description` tag is used to generate the schema description for each property.
type AddParams struct {
	A float64 `json:"a" description:"The first number to add."`
	B float64 `json:"b" description:"The second number to add."`
}

// SubtractParams defines the input for our "subtract" tool.
type SubtractParams struct {
	A float64 `json:"a" description:"The number to subtract from (minuend)."`
	B float64 `json:"b" description:"The number to subtract (subtrahend)."`
}
```

**3. Define and Register Tools**

Next, define your tools and their handlers. The handler is a strongly-typed function that takes a context and a pointer to your parameter struct.

```go
	toolsToRegister := []mcp.ToolRegistration{
		{
			Definition: protocol.Tool{
				Name:        "calculator/add",
				Title:       "Add Numbers",
				Description: "Calculates the sum of two numbers, a and b.",
			},
			Handler: func(ctx context.Context, params *AddParams) (string, error) {
				result := params.A + params.B
				return fmt.Sprintf("The sum of %f and %f is %f.", params.A, params.B, result), nil
			},
		},
		{
			Definition: protocol.Tool{
				Name:        "calculator/subtract",
				Title:       "Subtract Numbers",
				Description: "Calculates the difference between two numbers, a - b.",
			},
			Handler: func(ctx context.Context, params *SubtractParams) (string, error) {
				result := params.A - params.B
				return fmt.Sprintf("The difference of %f minus %f is %f.", params.A, params.B, result), nil
			},
		},
	}

	if err := server.RegisterTools(toolsToRegister); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}
```

**4. Start the Server**

Finally, start the server and listen for connections.

```go
	log.Println("Starting calculator server on :8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

### Full Example: `calculator-server`

Here is the full code for the example server located in `examples/calculator-server/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"go-mcp-sdk/pkg/mcp"
	"go-mcp-sdk/pkg/protocol"
)

// AddParams defines the input for our "add" tool.
type AddParams struct {
	A float64 `json:"a" description:"The first number to add."`
	B float64 `json:"b" description:"The second number to add."`
}

// SubtractParams defines the input for our "subtract" tool.
type SubtractParams struct {
	A float64 `json:"a" description:"The number to subtract from (minuend)."`
	B float64 `json:"b" description:"The number to subtract (subtrahend)."`
}

func main() {
	server := mcp.NewServer("GoCalculatorServer", "1.0.0", protocol.ServerCapabilities{
		Tools: &protocol.ServerToolCapabilities{},
	})

	toolsToRegister := []mcp.ToolRegistration{
		{
			Definition: protocol.Tool{
				Name:        "calculator/add",
				Title:       "Add Numbers",
				Description: "Calculates the sum of two numbers, a and b.",
			},
			Handler: func(ctx context.Context, params *AddParams) (string, error) {
				result := params.A + params.B
				return fmt.Sprintf("The sum of %f and %f is %f.", params.A, params.B, result), nil
			},
		},
		{
			Definition: protocol.Tool{
				Name:        "calculator/subtract",
				Title:       "Subtract Numbers",
				Description: "Calculates the difference between two numbers, a - b.",
			},
			Handler: func(ctx context.Context, params *SubtractParams) (string, error) {
				result := params.A - params.B
				return fmt.Sprintf("The difference of %f minus %f is %f.", params.A, params.B, result), nil
			},
		},
	}

	if err := server.RegisterTools(toolsToRegister); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	log.Println("Starting calculator server on :8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a pull request.
