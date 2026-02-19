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

	// Ensure DB is initialized (needed for primary mode)
	if _, err := db.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Start web server (or detect existing one)
	if err := webserver.Start(); err != nil {
		return nil, fmt.Errorf("failed to start web server: %w", err)
	}

	// Open browser on first invocation
	browserMu.Lock()
	if !browserOpened {
		browserOpened = true
		browserMu.Unlock()
		_ = browser.Open(webserver.BaseURL)
	} else {
		browserMu.Unlock()
	}

	if webserver.IsPrimary {
		return askLocal(ctx, appName, question)
	}
	return askRemote(ctx, appName, question)
}

// askLocal handles the request in-process (primary server mode).
func askLocal(ctx context.Context, appName, question string) (*mcp.CallToolResult, error) {
	reqID, ch, err := manager.Instance.CreateRequest(config.SourceName, appName, question)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Notify frontend via SSE
	manager.Broker.Publish(reqID, config.SourceName, appName, question)

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

// askRemote delegates to the primary rishvan-mcp server via HTTP.
func askRemote(ctx context.Context, appName, question string) (*mcp.CallToolResult, error) {
	reqID, err := webserver.RemoteCreateRequest(config.SourceName, appName, question)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote request: %w", err)
	}

	response, err := webserver.RemotePollResponse(ctx, reqID)
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(response), nil
}
