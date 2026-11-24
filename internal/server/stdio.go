package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/SmilingXinyi/deepseek-search-mcp/internal/logger"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio serves MCP over process stdin/stdout.
func RunStdio() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	s := New()
	logger.Info("MCP Server running on stdio")
	return s.Run(ctx, &mcp.StdioTransport{})
}
