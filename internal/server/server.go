package server

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/SmilingXinyi/deepseek-search-mcp/internal/deepseek"
)

// New composes an MCP server with DeepSeek search and fetch tools.
// Transports bind in stdio.go and http.go.
func New() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "deepseek-search-mcp", Version: "0.1.0"}, nil)
	openWorld := true
	annotations := &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  &openWorld,
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "deepseek_search",
		Title:       "DeepSeek web search",
		Description: "Search the web with the DeepSeek Search API. Returns ranked results with title, URL, snippet, summary, site, publication time, authority (0-4, -1 unknown), and rerank_score (0-3). An empty result list is a successful search, not an error. Query operators such as site:, -site:, lang:, after:, and before: are forwarded as-is.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deepseek.SearchInput) (*mcp.CallToolResult, deepseek.SearchOutput, error) {
		client, err := deepseek.NewFromEnv()
		if err != nil {
			return nil, deepseek.SearchOutput{}, err
		}
		out, text, err := client.SearchTool(ctx, in)
		if err != nil {
			return nil, deepseek.SearchOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "deepseek_fetch",
		Title:       "DeepSeek page fetch",
		Description: "Fetch one web page with the DeepSeek Fetch API and return its title and Markdown content. The URL must be absolute http or https, must include a host, and must not include a username or password.",
		Annotations: annotations,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deepseek.FetchInput) (*mcp.CallToolResult, deepseek.FetchOutput, error) {
		client, err := deepseek.NewFromEnv()
		if err != nil {
			return nil, deepseek.FetchOutput{}, err
		}
		out, text, err := client.FetchTool(ctx, in)
		if err != nil {
			return nil, deepseek.FetchOutput{}, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, out, nil
	})

	return s
}
