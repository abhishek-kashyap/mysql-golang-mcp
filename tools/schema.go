package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterSchemaTool registers the schema inspection tools
func RegisterSchemaTool(s *server.MCPServer, manager *db.Manager) {
	registerListDatabases(s, manager)
	registerListTables(s, manager)
	registerDescribeTable(s, manager)
}

func registerListDatabases(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("list_databases",
		mcp.WithDescription("List all accessible databases"),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		connection, ok := request.Params.Arguments["connection"].(string)
		if !ok || connection == "" {
			return mcp.NewToolResultError("connection parameter is required"), nil
		}

		queryResult, err := manager.ExecuteQuery(connection, "SHOW DATABASES")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract database names
		databases := make([]string, 0, len(queryResult.Rows))
		for _, row := range queryResult.Rows {
			for _, v := range row {
				if s, ok := v.(string); ok {
					databases = append(databases, s)
				}
			}
		}

		result, err := json.MarshalIndent(databases, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func registerListTables(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("list_tables",
		mcp.WithDescription("List all tables in a database"),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("database",
			mcp.Description("Database name (uses connection default if not provided)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		connection, ok := request.Params.Arguments["connection"].(string)
		if !ok || connection == "" {
			return mcp.NewToolResultError("connection parameter is required"), nil
		}

		database, _ := request.Params.Arguments["database"].(string)

		var query string
		if database != "" {
			query = fmt.Sprintf("SHOW TABLES FROM `%s`", database)
		} else {
			query = "SHOW TABLES"
		}

		queryResult, err := manager.ExecuteQuery(connection, query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract table names
		tables := make([]string, 0, len(queryResult.Rows))
		for _, row := range queryResult.Rows {
			for _, v := range row {
				if s, ok := v.(string); ok {
					tables = append(tables, s)
				}
			}
		}

		result, err := json.MarshalIndent(tables, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func registerDescribeTable(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("describe_table",
		mcp.WithDescription("Get the schema/structure of a table including columns, types, and keys"),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("table",
			mcp.Required(),
			mcp.Description("Table name to describe"),
		),
		mcp.WithString("database",
			mcp.Description("Database name (uses connection default if not provided)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		connection, ok := request.Params.Arguments["connection"].(string)
		if !ok || connection == "" {
			return mcp.NewToolResultError("connection parameter is required"), nil
		}

		table, ok := request.Params.Arguments["table"].(string)
		if !ok || table == "" {
			return mcp.NewToolResultError("table parameter is required"), nil
		}

		database, _ := request.Params.Arguments["database"].(string)

		var query string
		if database != "" {
			query = fmt.Sprintf("DESCRIBE `%s`.`%s`", database, table)
		} else {
			query = fmt.Sprintf("DESCRIBE `%s`", table)
		}

		queryResult, err := manager.ExecuteQuery(connection, query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(queryResult.Rows, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
