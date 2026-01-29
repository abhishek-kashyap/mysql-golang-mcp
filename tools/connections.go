package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterConnectionsTool registers the list_connections tool
func RegisterConnectionsTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("list_connections",
		mcp.WithDescription("List all configured database connections with their read-only status"),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		connections := manager.ListConnections()

		result, err := json.MarshalIndent(connections, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
