package mcp

import (
	"encoding/json"
	"io"
	"net/http"

	"go-mcp-sdk/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

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
		writeErrorResponse(w, protocol.RequestID{}, -32700, "Parse error: Invalid JSON", err)
		return
	}

	if _, ok := rawMessage["id"]; ok {
		var req protocol.Request
		if err := json.Unmarshal(body, &req); err != nil {
			writeErrorResponse(w, protocol.RequestID{}, -32700, "Parse error: Invalid Request structure", err)
			return
		}
		s.handleRequest(w, &req)
	} else {
		var notif protocol.Notification
		if err := json.Unmarshal(body, &notif); err != nil {
			log.Printf("Error parsing notification: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.handleNotification(w, &notif)
	}
}

func (s *Server) handleRequest(w http.ResponseWriter, req *protocol.Request) {
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

func (s *Server) handleNotification(w http.ResponseWriter, n *protocol.Notification) {
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

func writeSuccessResponse(w http.ResponseWriter, id protocol.RequestID, result interface{}) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		writeErrorResponse(w, id, -32603, "Internal server error: failed to marshal result", err)
		return
	}
	resp := protocol.Response{
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

func writeErrorResponse(w http.ResponseWriter, id protocol.RequestID, code int, message string, data error) {
	var dataStr string
	if data != nil {
		dataStr = data.Error()
	}
	errorObj := &protocol.ErrorObject{Code: code, Message: message}
	if dataStr != "" {
		errorObj.Data = dataStr
	}
	resp := protocol.Response{JSONRPC: "2.0", ID: id, Error: errorObj}

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
