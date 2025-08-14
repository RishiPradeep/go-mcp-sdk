package main

import (
	"context"
	"fmt"
	"log"

	"go-mcp-sdk/pkg/mcp"
	"go-mcp-sdk/pkg/protocol"
)

// --- Tool Parameter Structs ---
// These structs define the expected inputs for our tools in a type-safe way.
// The SDK will automatically generate JSON schemas from these.

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

func main() {
	// 1. Initialize the server with its name, version, and capabilities.
	// We are enabling the "tools" capability.
	server := mcp.NewServer("GoCalculatorServer", "1.0.0", protocol.ServerCapabilities{
		Tools: &protocol.ServerToolCapabilities{},
	})

	// 2. Define all the tools we want to register in a slice.
	// This makes it easy to see all of the server's functionality in one place.
	toolsToRegister := []mcp.ToolRegistration{
		{
			Definition: protocol.Tool{
				Name:        "calculator/add",
				Title:       "Add Numbers",
				Description: "Calculates the sum of two numbers, a and b.",
			},
			// The handler is a strongly-typed function. The SDK validates that
			// its signature matches the expected pattern and uses *AddParams.
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
			// This handler has the same signature but uses *SubtractParams.
			Handler: func(ctx context.Context, params *SubtractParams) (string, error) {
				result := params.A - params.B
				return fmt.Sprintf("The difference of %f minus %f is %f.", params.A, params.B, result), nil
			},
		},
	}

	// 3. Register all tools with a single, clean API call.
	if err := server.RegisterTools(toolsToRegister); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// 4. Start the server and listen for connections.
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}