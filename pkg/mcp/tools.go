package mcp

import (
	"context"
	"fmt"
	"reflect"

	"go-mcp-sdk/internal/jsonschema"
	"go-mcp-sdk/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

// ToolRegistration is a struct to define and register their tools.
type ToolRegistration struct {
	Definition protocol.Tool
	// Handler is the strongly-typed function that implements the tool.
	Handler interface{}
}

// internalRegisteredTool stores the processed, ready-to-use tool information.
// This is not exposed to the user of the SDK.
type internalRegisteredTool struct {
	Definition   protocol.Tool
	handlerValue reflect.Value
	inputType    reflect.Type
	takesContext bool
}

// RegisterTools registers a slice of tools, making them available to clients.
// This is the primary method for adding functionality to the server.
func (s *Server) RegisterTools(registrations []ToolRegistration) error {
	for _, reg := range registrations {
		if err := s.registerSingleTool(reg); err != nil {
			// Return on the first error to ensure atomicity.
			return fmt.Errorf("failed to register tool '%s': %w", reg.Definition.Name, err)
		}
	}
	return nil
}

// registerSingleTool is the internal helper that processes one registration.
func (s *Server) registerSingleTool(reg ToolRegistration) error {
	toolDef := reg.Definition
	handlerFn := reg.Handler

	if toolDef.Name == "" {
		return fmt.Errorf("tool definition must include a name")
	}

	handlerVal := reflect.ValueOf(handlerFn)
	handlerType := handlerVal.Type()
	if handlerType.Kind() != reflect.Func {
		return fmt.Errorf("handler must be a function")
	}

	// Validate handler signature and extract input type
	var inputType reflect.Type
	var takesContext bool

	numIn := handlerType.NumIn()
	if numIn > 0 && handlerType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		takesContext = true
	}

	expectedArgCount := 1
	if takesContext {
		expectedArgCount = 2
	}
	if numIn != expectedArgCount {
		return fmt.Errorf("handler has incorrect number of arguments (expected %d, got %d)", expectedArgCount, numIn)
	}

	// The input type is the last argument.
	inputType = handlerType.In(numIn - 1)
	if inputType.Kind() != reflect.Ptr || inputType.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("handler's parameter type must be a pointer to a struct, but got %s", inputType)
	}

	// Generate schema from the input type
	inputSchema, err := jsonschema.GenerateSchemaForType(inputType)
	if err != nil {
		return fmt.Errorf("could not generate schema for type %s: %w", inputType, err)
	}
	toolDef.InputSchema = inputSchema

	// Store the processed tool
	s.toolLock.Lock()
	defer s.toolLock.Unlock()

	if _, exists := s.tools[toolDef.Name]; exists {
		return fmt.Errorf("tool with name '%s' already registered", toolDef.Name)
	}

	s.tools[toolDef.Name] = internalRegisteredTool{
		Definition:   toolDef,
		handlerValue: handlerVal,
		inputType:    inputType,
		takesContext: takesContext,
	}

	log.Infof("Registered tool: %s", toolDef.Name)
	return nil
}