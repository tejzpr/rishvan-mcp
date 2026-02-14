package handler

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tejzpr/rishvan-mcp/internal/browser"
	"github.com/tejzpr/rishvan-mcp/internal/config"
	"github.com/tejzpr/rishvan-mcp/internal/db"
	"github.com/tejzpr/rishvan-mcp/internal/manager"
	"github.com/tejzpr/rishvan-mcp/internal/webserver"
)

var (
	browserOpened bool
	browserMu     sync.Mutex
)

func AskRishvan(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	question, err := request.RequireString("question")
	if err != nil {
		return mcp.NewToolResultError("question is required"), nil
	}
	appName, err := request.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}

	// Ensure DB is initialized
	if _, err := db.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Start web server
	if err := webserver.Start(); err != nil {
		return nil, fmt.Errorf("failed to start web server: %w", err)
	}

	// Open browser on first invocation
	browserMu.Lock()
	if !browserOpened {
		browserOpened = true
		browserMu.Unlock()
		_ = browser.Open("http://localhost:56234")
	} else {
		browserMu.Unlock()
	}

	// Create request and get response channel
	reqID, ch, err := manager.Instance.CreateRequest(config.IDEName, appName, question)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Notify frontend via SSE
	manager.Broker.Publish(reqID, config.IDEName, appName, question)

	// Block until human responds or context is cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case response, ok := <-ch:
		if !ok {
			return mcp.NewToolResultError("request channel closed unexpectedly"), nil
		}
		return mcp.NewToolResultText(response), nil
	}
}
