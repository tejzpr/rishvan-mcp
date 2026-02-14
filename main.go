package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/tejzpr/rishvan-mcp/internal/config"
	"github.com/tejzpr/rishvan-mcp/internal/handler"
	"github.com/tejzpr/rishvan-mcp/internal/webserver"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

func main() {
	// Parse --ide argument
	ideName := ""
	for i, arg := range os.Args[1:] {
		if arg == "--ide" && i+1 < len(os.Args[1:]) {
			ideName = os.Args[i+2]
			break
		}
	}
	if ideName == "" {
		fmt.Fprintf(os.Stderr, "error: --ide <name> is required\nusage: rishvan-mcp --ide <ide-name>\n")
		os.Exit(1)
	}
	config.IDEName = ideName

	// Set up embedded frontend filesystem
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		fmt.Printf("failed to load embedded frontend: %v\n", err)
		return
	}
	webserver.EmbeddedFS = distFS

	// Create MCP server
	s := server.NewMCPServer(
		"rishvan-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register ask_rishvan tool
	tool := mcp.NewTool("ask_rishvan",
		mcp.WithDescription("Ask a human for input, recommendation, or guidance. Opens a web UI for the human to respond."),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question, recommendation request, or 'what to do next' prompt for the human"),
		),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("The name of the application or project context"),
		),
	)
	s.AddTool(tool, handler.AskRishvan)

	// Start stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}
