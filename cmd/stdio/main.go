package main

import (
	"os"

	"github.com/SmilingXinyi/deepseek-search-mcp/internal/logger"
	"github.com/SmilingXinyi/deepseek-search-mcp/internal/server"
)

func main() {
	logger.Init()
	if err := server.RunStdio(); err != nil {
		logger.Error("MCP Server error: " + err.Error())
		os.Exit(1)
	}
}
