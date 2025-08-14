package protocol

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// RequestID can be a string or number according to JSON-RPC 2.0 spec
type RequestID struct {
	value interface{}
}

// NewRequestID creates a new RequestID from a string
func NewRequestID(id string) RequestID {
	return RequestID{value: id}
}

// NewNumericRequestID creates a new RequestID from a number
func NewNumericRequestID(id float64) RequestID {
	return RequestID{value: id}
}

// String returns the string representation of the ID
func (id RequestID) String() string {
	if id.value == nil {
		return ""
	}
	switch v := id.value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Value returns the underlying value
func (id RequestID) Value() interface{} {
	return id.value
}

// UnmarshalJSON implements custom JSON unmarshaling
func (id *RequestID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		id.value = str
		return nil
	}

	// Try to unmarshal as number
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		id.value = num
		return nil
	}

	// Check if it's null
	if string(data) == "null" {
		id.value = nil
		return nil
	}

	return fmt.Errorf("invalid request ID: must be string, number, or null")
}

// MarshalJSON implements custom JSON marshaling
func (id RequestID) MarshalJSON() ([]byte, error) {
	if id.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(id.value)
}

// Request is a generic JSON-RPC 2.0 request object.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a generic JSON-RPC 2.0 response object.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error.
type ErrorObject struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification is a generic JSON-RPC 2.0 notification object.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// InitializeRequest represents the parameters for the "initialize" method.
// This is sent from the client to the server.
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      ImplementationInfo `json:"clientInfo"`
	Capabilities    ClientCapabilities `json:"capabilities"`
}

// InitializeResult represents the successful result of an "initialize" request.
// This is sent from the server to the client.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ImplementationInfo `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	Instructions    string             `json:"instructions,omitempty"`
}

// ImplementationInfo describes the client or server software.
type ImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Title   string `json:"title,omitempty"`
}

// ClientCapabilities lists the features supported by the client.
type ClientCapabilities struct {
	// For now, we'll keep these as empty structs as placeholders.
	// We'll add fields if/when we implement these features.
	Roots       *struct{} `json:"roots,omitempty"`
	Sampling    *struct{} `json:"sampling,omitempty"`
	Elicitation *struct{} `json:"elicitation,omitempty"`
}

// ServerCapabilities lists the features supported by the server.
type ServerCapabilities struct {
	Tools     *ServerToolCapabilities     `json:"tools,omitempty"`
	Resources *ServerResourceCapabilities `json:"resources,omitempty"`
	Prompts   *ServerPromptCapabilities   `json:"prompts,omitempty"`
	Logging   *struct{}                   `json:"logging,omitempty"`
}

// ServerToolCapabilities specifies tool-related capabilities of the server.
type ServerToolCapabilities struct {
	// If true, the server can send "notifications/tools/list_changed".
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerResourceCapabilities specifies resource-related capabilities.
type ServerResourceCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"`
}

// ServerPromptCapabilities specifies prompt-related capabilities.
type ServerPromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializedNotification represents the parameters for the "notifications/initialized" notification.
// It has no parameters, but we define the struct for clarity.
type InitializedNotification struct{}

// Tool defines the structure for a tool that a client can call.
type Tool struct {
	Name        string          `json:"name"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// ListToolsResult is the response for a "tools/list" request.
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolRequest represents the parameters for a "tools/call" request.
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult is the response from a successful tool call.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a piece of content in a tool's result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}