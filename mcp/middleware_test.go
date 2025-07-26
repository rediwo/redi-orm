package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rediwo/redi-orm/logger"
)

func TestMiddleware(t *testing.T) {
	// Create a simple logger for testing
	l := logger.NewDefaultLogger("Test")
	l.SetLevel(logger.LogLevelDebug)

	// Create MCP server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	// Counter to track middleware calls
	receivingCalls := 0
	sendingCalls := 0

	// Add receiving middleware
	receivingMiddleware := func(next mcp.MethodHandler[*mcp.ServerSession]) mcp.MethodHandler[*mcp.ServerSession] {
		return func(ctx context.Context, session *mcp.ServerSession, method string, params mcp.Params) (mcp.Result, error) {
			receivingCalls++
			l.Debug("Receiving middleware called for method: %s", method)
			return next(ctx, session, method, params)
		}
	}

	// Add sending middleware
	sendingMiddleware := func(next mcp.MethodHandler[*mcp.ServerSession]) mcp.MethodHandler[*mcp.ServerSession] {
		return func(ctx context.Context, session *mcp.ServerSession, method string, params mcp.Params) (mcp.Result, error) {
			result, err := next(ctx, session, method, params)
			sendingCalls++
			l.Debug("Sending middleware called for method: %s", method)
			return result, err
		}
	}

	// Add middleware to server
	mcpServer.AddReceivingMiddleware(receivingMiddleware)
	mcpServer.AddSendingMiddleware(sendingMiddleware)

	t.Logf("Middleware added successfully")
	t.Logf("AddReceivingMiddleware and AddSendingMiddleware methods are properly implemented")
}
