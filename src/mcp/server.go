package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/5uck1ess/devkit/lib"
	mcpgo "github.com/mark3labs/mcp-go/server"
)

// Server wraps the devkit engine as an MCP server.
type Server struct {
	dataDir     string
	workflowDir string
	repoRoot    string
	db          *lib.DB
	git         *lib.Git
	principles  map[string][]string
}

// NewServer creates a devkit MCP server.
//
// Only dataDir is strictly required (the server needs somewhere to persist
// workflow state). repoRoot and workflowDir may be empty — the server still
// boots and answers the MCP initialize handshake, so the client doesn't see
// an opaque -32000. Individual tool calls that need git state or workflow
// definitions are responsible for returning a structured error when the
// required input is missing. See issue #105.
func NewServer(repoRoot, dataDir, workflowDir string) (*Server, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("dataDir must be non-empty")
	}

	dbPath := filepath.Join(dataDir, "devkit.db")
	db, err := lib.OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	principles := map[string][]string{}
	if workflowDir != "" {
		loaded, err := LoadPrinciples(workflowDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load principles: %v\n", err)
		} else {
			principles = loaded
		}
	}

	return &Server{
		dataDir:     dataDir,
		workflowDir: workflowDir,
		repoRoot:    repoRoot,
		db:          db,
		git:         &lib.Git{Dir: repoRoot},
		principles:  principles,
	}, nil
}

// Close releases server resources (database connection).
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Serve starts the MCP server on stdio, respecting ctx for graceful shutdown.
func (s *Server) Serve(ctx context.Context) error {
	srv := mcpgo.NewMCPServer("devkit-engine", "1.0.0")

	tool, handler := s.startTool()
	srv.AddTool(tool, handler)

	tool, handler = s.advanceTool()
	srv.AddTool(tool, handler)

	tool, handler = s.statusTool()
	srv.AddTool(tool, handler)

	tool, handler = s.listTool()
	srv.AddTool(tool, handler)

	tool, handler = s.askTool()
	srv.AddTool(tool, handler)

	stdio := mcpgo.NewStdioServer(srv)
	return stdio.Listen(ctx, os.Stdin, os.Stdout)
}
