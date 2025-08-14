// in schema_generator.go
package mcp

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/invopop/jsonschema"
)

// generateSchemaForType uses reflection to create a JSON schema for a given Go struct type.
func generateSchemaForType(t reflect.Type) (json.RawMessage, error) {
	// If the type is a pointer, get the element type it points to.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// The schema should describe a struct.
	if t.Kind() != reflect.Struct {
		return json.RawMessage(`{"type": "object", "properties": {}}`), nil
	}

	// Step 1: Generate the base schema without using references.
	// This ensures the schema is fully inlined, which is what the MCP spec expects.
	reflector := &jsonschema.Reflector{
		DoNotReference: true,
	}
	schema := reflector.Reflect(reflect.New(t).Interface())

	// Step 2: Add descriptions from struct tags.
	// The jsonschema library does not handle 'description' tags, so we add them here.
	// We must check if the Properties map is nil, as the library may not initialize it.
	if schema.Properties != nil {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				continue
			}
			propertyName := strings.Split(jsonTag, ",")[0]

			// Find the corresponding property in the generated schema.
			if prop, ok := schema.Properties.Get(propertyName); ok {
				// If the property exists, add the description from the tag.
				if descTag := field.Tag.Get("description"); descTag != "" {
					prop.Description = descTag
				}
			}
		}
	}

	// Step 3: Mark all fields as required for simplicity.
	// A more robust solution might inspect struct tags.
	if schema.Properties != nil {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag != "" && jsonTag != "-" {
				propertyName := strings.Split(jsonTag, ",")[0]
				schema.Required = append(schema.Required, propertyName)
			}
		}
	}

	// Step 4: Marshal the final, modified schema into JSON.
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}

	return json.RawMessage(schemaBytes), nil
}
