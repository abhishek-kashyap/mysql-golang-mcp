package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterQueryTool registers the mysql_query tool
// Deprecated: Use mysql_select, mysql_insert, mysql_update, mysql_delete, mysql_alter, or mysql_execute instead.
func RegisterQueryTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_query",
		mcp.WithDescription(`[DEPRECATED] Execute a SQL query against the MySQL database.

This tool is deprecated. Please use the specific tools instead:
- mysql_select: For SELECT queries (safe for auto-accept)
- mysql_insert: For INSERT queries
- mysql_update: For UPDATE queries
- mysql_delete: For DELETE queries
- mysql_alter: For ALTER TABLE queries
- mysql_execute: For INSERT/UPDATE/DELETE combined
- mysql_execute_unsafe: For queries blocked by safety checks

For read-only connections, only SELECT/SHOW/DESCRIBE/EXPLAIN queries are allowed.`),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The SQL query to execute"),
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
