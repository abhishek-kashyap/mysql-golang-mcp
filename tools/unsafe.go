package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterUnsafeTool registers the mysql_execute_unsafe tool
func RegisterUnsafeTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_execute_unsafe",
		mcp.WithDescription(`⚠️ DANGEROUS: Execute ANY SQL query, bypassing all safety checks.

This tool bypasses:
- Dangerous query blocking (DROP, TRUNCATE, CREATE, GRANT, REVOKE)
- Sensitive query blocking (SHOW GRANTS, mysql.user access)

This tool does NOT bypass:
- Read-only connection restrictions (that's a configuration choice)

Use cases:
- Emergency fixes requiring schema changes
- Legitimate operations blocked by safety checks
- Administrative tasks requiring elevated permissions

NEVER auto-accept this tool. Always review queries carefully.`),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The SQL query to execute (any type allowed)"),
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

		unsafeResult, err := manager.ExecuteUnsafe(connection, sql)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(unsafeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
