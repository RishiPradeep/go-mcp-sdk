package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"go-mcp-sdk/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

func (s *Server) handleInitialize(w http.ResponseWriter, req *protocol.Request) {
	log.Infof("Received initialize request: ID=%s", req.ID.String())
	var initParams protocol.InitializeRequest
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

	result := protocol.InitializeResult{
		ProtocolVersion: negotiatedVersion,
		ServerInfo:      s.info,
		Capabilities:    s.capabilities,
	}

	w.Header().Set("Mcp-Session-Id", sessionID)
	writeSuccessResponse(w, req.ID, result)
}

// --- Tool Method Handlers ---

func (s *Server) handleListTools(w http.ResponseWriter, req *protocol.Request) {
	log.Infof("Received tools/list request: ID=%s", req.ID.String())
	s.toolLock.RLock()
	defer s.toolLock.RUnlock()
	toolList := make([]protocol.Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		toolList = append(toolList, tool.Definition)
	}
	writeSuccessResponse(w, req.ID, protocol.ListToolsResult{Tools: toolList})
}

func (s *Server) handleCallTool(w http.ResponseWriter, req *protocol.Request) {
	var callParams protocol.CallToolRequest
	if err := json.Unmarshal(req.Params, &callParams); err != nil {
		writeErrorResponse(w, req.ID, -32602, "Invalid params for tools/call", err)
		return
	}

	log.Infof("Received tools/call request for tool '%s': ID=%s", callParams.Name, req.ID.String())

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
		errorResult := &protocol.CallToolResult{
			Content: []protocol.ContentBlock{{Type: "text", Text: resultErr.Error()}},
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

	successResult := &protocol.CallToolResult{
		Content: []protocol.ContentBlock{{Type: "text", Text: resultText}},
	}
	writeSuccessResponse(w, req.ID, successResult)
}