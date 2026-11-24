package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/SmilingXinyi/deepseek-search-mcp/internal/logger"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func requireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := os.Getenv("MCP_AUTH_TOKEN")
		if expected == "" {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if token != expected {
			logger.Warn("Unauthorized HTTP request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RunHTTP serves Streamable HTTP on /mcp.
func RunHTTP() error {
	host := getenv("MCP_HOST", "127.0.0.1")
	port := getenv("MCP_PORT", "3000")
	addr := host + ":" + port

	s := New()
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	mux := http.NewServeMux()
	mux.Handle("/mcp", requireBearer(handler))

	logger.Info("MCP Streamable HTTP server listening on http://" + addr + "/mcp")
	return http.ListenAndServe(addr, mux)
}
