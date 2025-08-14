package mcp

import (
	"net/http"
	"sync"

	"go-mcp-sdk/pkg/protocol"

	log "github.com/sirupsen/logrus"
)

// Server holds the state and logic for an MCP server.
type Server struct {
	serverMux    *http.ServeMux
	info         protocol.ImplementationInfo
	capabilities protocol.ServerCapabilities
	sessionLock  sync.RWMutex
	sessions     map[string]*SessionState
	toolLock     sync.RWMutex
	// tools stores the internal representation of registered tools.
	tools map[string]internalRegisteredTool
}

// SessionState holds state for a connected client.
type SessionState struct {
	ClientCapabilities protocol.ClientCapabilities
}

// NewServer creates a new MCP Server.
func NewServer(name, version string, capabilities protocol.ServerCapabilities) *Server {
	s := &Server{
		serverMux:    http.NewServeMux(),
		info:         protocol.ImplementationInfo{Name: name, Version: version},
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