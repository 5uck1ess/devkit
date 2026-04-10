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
func NewServer(repoRoot, dataDir, workflowDir string) (*Server, error) {
	dbPath := filepath.Join(dataDir, "devkit.db")
	db, err := lib.OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	principles, err := LoadPrinciples(workflowDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load principles: %v\n", err)
		principles = map[string][]string{}
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

// Serve starts the MCP server on stdio.
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

	return mcpgo.ServeStdio(srv)
}


