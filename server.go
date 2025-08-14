package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// --- Structs ---

// Server holds the state and logic for an MCP server.
type Server struct {
	serverMux    *http.ServeMux
	info         ImplementationInfo
	capabilities ServerCapabilities
	sessionLock  sync.RWMutex
	sessions     map[string]*SessionState
	toolLock     sync.RWMutex
	// tools stores the internal representation of registered tools.
	tools map[string]internalRegisteredTool
}

// SessionState holds state for a connected client.
type SessionState struct {
	ClientCapabilities ClientCapabilities
}

// internalRegisteredTool stores the processed, ready-to-use tool information.
// This is not exposed to the user of the SDK.
type internalRegisteredTool struct {
	Definition   Tool
	handlerValue reflect.Value
	inputType    reflect.Type
	takesContext bool
}

// --- Public API ---

// NewServer creates a new MCP Server.
func NewServer(name, version string, capabilities ServerCapabilities) *Server {
	s := &Server{
		serverMux:    http.NewServeMux(),
		info:         ImplementationInfo{Name: name, Version: version},
		capabilities: capabilities,
		sessions:     make(map[string]*SessionState),
		tools:        make(map[string]internalRegisteredTool),
	}
	s.serverMux.HandleFunc("/mcp", s.handleMCPRequest)
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	log.Infof("MCP Server '%s' version '%s' listening on %s", s.info.Name, s.info.Version, addr)
	return http.ListenAndServe(addr, s.serverMux)
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

// --- Internal Registration Logic ---

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
	inputSchema, err := generateSchemaForType(inputType)
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

// --- HTTP and Request Handling Logic ---

func (s *Server) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log.Println("Received GET request for SSE stream (not yet implemented). Returning OK.")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var rawMessage map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawMessage); err != nil {
		writeErrorResponse(w, RequestID{}, -32700, "Parse error: Invalid JSON", err)
		return
	}

	if _, ok := rawMessage["id"]; ok {
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			writeErrorResponse(w, RequestID{}, -32700, "Parse error: Invalid Request structure", err)
			return
		}
		s.handleRequest(w, &req)
	} else {
		var notif Notification
		if err := json.Unmarshal(body, &notif); err != nil {
			log.Printf("Error parsing notification: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.handleNotification(w, &notif)
	}
}

func (s *Server) handleRequest(w http.ResponseWriter, req *Request) {
	log.Infof("Received request: Method=%s, ID=%s", req.Method, req.ID.String())

	switch req.Method {
	case "initialize":
		s.handleInitialize(w, req)
	case "tools/list":
		s.handleListTools(w, req)
	case "tools/call":
		s.handleCallTool(w, req)
	default:
		log.Infof("Unknown method: %s", req.Method)
		writeErrorResponse(w, req.ID, -32601, "Method not found", nil)
	}
}

func (s *Server) handleNotification(w http.ResponseWriter, n *Notification) {
	log.Infof("Received notification: Method=%s", n.Method)
	switch n.Method {
	case "notifications/initialized":
		log.Infof("Client confirmed initialization.")
		w.WriteHeader(http.StatusAccepted)
	default:
		log.Infof("Received unhandled notification: %s", n.Method)
		w.WriteHeader(http.StatusAccepted)
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req *Request) {
	var initParams InitializeRequest
	if err := json.Unmarshal(req.Params, &initParams); err != nil {
		writeErrorResponse(w, req.ID, -32602, "Invalid params for initialize", err)
		return
	}

	log.Infof("Client '%s' version '%s' connecting with protocol version '%s'", initParams.ClientInfo.Name, initParams.ClientInfo.Version, initParams.ProtocolVersion)

	negotiatedVersion := initParams.ProtocolVersion
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	s.sessionLock.Lock()
	s.sessions[sessionID] = &SessionState{ClientCapabilities: initParams.Capabilities}
	s.sessionLock.Unlock()
	log.Infof("Created new session: %s", sessionID)

	result := InitializeResult{
		ProtocolVersion: negotiatedVersion,
		ServerInfo:      s.info,
		Capabilities:    s.capabilities,
	}

	w.Header().Set("Mcp-Session-Id", sessionID)
	writeSuccessResponse(w, req.ID, result)
}

// --- Tool Method Handlers ---

func (s *Server) handleListTools(w http.ResponseWriter, req *Request) {
	s.toolLock.RLock()
	defer s.toolLock.RUnlock()
	toolList := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		toolList = append(toolList, tool.Definition)
	}
	writeSuccessResponse(w, req.ID, ListToolsResult{Tools: toolList})
}

func (s *Server) handleCallTool(w http.ResponseWriter, req *Request) {
	var callParams CallToolRequest
	if err := json.Unmarshal(req.Params, &callParams); err != nil {
		writeErrorResponse(w, req.ID, -32602, "Invalid params for tools/call", err)
		return
	}

	s.toolLock.RLock()
	tool, exists := s.tools[callParams.Name]
	s.toolLock.RUnlock()
	if !exists {
		writeErrorResponse(w, req.ID, -32602, fmt.Sprintf("Tool not found: %s", callParams.Name), nil)
		return
	}

	inputValue := reflect.New(tool.inputType.Elem())
	argsBytes, _ := json.Marshal(callParams.Arguments)
	if err := json.Unmarshal(argsBytes, inputValue.Interface()); err != nil {
		writeErrorResponse(w, req.ID, -32602, fmt.Sprintf("Invalid arguments for tool %s", callParams.Name), err)
		return
	}

	callArgs := []reflect.Value{}
	if tool.takesContext {
		callArgs = append(callArgs, reflect.ValueOf(context.Background()))
	}
	callArgs = append(callArgs, inputValue)

	results := tool.handlerValue.Call(callArgs)

	var resultErr error
	if errVal := results[len(results)-1]; !errVal.IsNil() {
		resultErr = errVal.Interface().(error)
	}

	if resultErr != nil {
		errorResult := &CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: resultErr.Error()}},
			IsError: true,
		}
		writeSuccessResponse(w, req.ID, errorResult)
		return
	}

	var resultText string
	if len(results) > 1 {
		resultText = fmt.Sprintf("%v", results[0].Interface())
	} else {
		resultText = "Operation completed successfully."
	}

	successResult := &CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: resultText}},
	}
	writeSuccessResponse(w, req.ID, successResult)
}

// --- Response Writers and Session Helpers ---

func writeSuccessResponse(w http.ResponseWriter, id RequestID, result interface{}) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		writeErrorResponse(w, id, -32603, "Internal server error: failed to marshal result", err)
		return
	}
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultBytes,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Error writing success response: %v", err)
	}
}

func writeErrorResponse(w http.ResponseWriter, id RequestID, code int, message string, data error) {
	var dataStr string
	if data != nil {
		dataStr = data.Error()
	}
	errorObj := &ErrorObject{Code: code, Message: message}
	if dataStr != "" {
		errorObj.Data = dataStr
	}
	resp := Response{JSONRPC: "2.0", ID: id, Error: errorObj}

	w.Header().Set("Content-Type", "application/json")
	switch code {
	case -32700, -32600, -32602:
		w.WriteHeader(http.StatusBadRequest)
	case -32601:
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("Error writing error response: %v", err)
	}
}
