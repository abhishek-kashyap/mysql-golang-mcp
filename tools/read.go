package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterReadTool registers the mysql_select tool for read operations
func RegisterReadTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_select",
		mcp.WithDescription("Execute a SELECT query against the MySQL database. Only SELECT queries are allowed. Safe for auto-accept in MCP clients."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The SELECT query to execute"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		connection, ok := request.Params.Arguments["connection"].(string)
		if !ok || connection == "" {
			return mcp.NewToolResultError("connection parameter is required"), nil
		}

		sql, ok := request.Params.Arguments["sql"].(string)
		if !ok || sql == "" {
			return mcp.NewToolResultError("sql parameter is required"), nil
		}

		// Validate that this is a SELECT query
		if err := db.ValidateQueryType(sql, db.QueryTypeSelect); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		queryResult, err := manager.ExecuteQuery(connection, sql)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(queryResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
