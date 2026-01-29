package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterWriteTools registers all write operation tools
func RegisterWriteTools(s *server.MCPServer, manager *db.Manager) {
	registerInsertTool(s, manager)
	registerUpdateTool(s, manager)
	registerDeleteTool(s, manager)
	registerAlterTool(s, manager)
	registerExecuteTool(s, manager)
}

// registerInsertTool registers the mysql_insert tool
func registerInsertTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_insert",
		mcp.WithDescription("Execute an INSERT query against the MySQL database. Only INSERT queries are allowed. Medium risk - consider before auto-accepting."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The INSERT query to execute"),
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

		writeResult, err := manager.ExecuteWrite(connection, sql, db.QueryTypeInsert)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(writeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

// registerUpdateTool registers the mysql_update tool
func registerUpdateTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_update",
		mcp.WithDescription("Execute an UPDATE query against the MySQL database. Only UPDATE queries are allowed. High risk - do not auto-accept."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The UPDATE query to execute"),
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

		writeResult, err := manager.ExecuteWrite(connection, sql, db.QueryTypeUpdate)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(writeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

// registerDeleteTool registers the mysql_delete tool
func registerDeleteTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_delete",
		mcp.WithDescription("Execute a DELETE query against the MySQL database. Only DELETE queries are allowed. High risk - do not auto-accept."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The DELETE query to execute"),
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

		writeResult, err := manager.ExecuteWrite(connection, sql, db.QueryTypeDelete)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(writeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

// registerAlterTool registers the mysql_alter tool
func registerAlterTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_alter",
		mcp.WithDescription("Execute an ALTER TABLE query against the MySQL database. Only ALTER queries are allowed. High risk - do not auto-accept. Still blocks DROP DATABASE, CREATE DATABASE, GRANT, REVOKE."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The ALTER query to execute"),
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

		writeResult, err := manager.ExecuteAlter(connection, sql)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(writeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

// registerExecuteTool registers the mysql_execute tool (combined INSERT/UPDATE/DELETE)
func registerExecuteTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("mysql_execute",
		mcp.WithDescription("Execute an INSERT, UPDATE, or DELETE query against the MySQL database. High risk - do not auto-accept."),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("sql",
			mcp.Required(),
			mcp.Description("The INSERT, UPDATE, or DELETE query to execute"),
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

		writeResult, err := manager.ExecuteWrite(connection, sql, db.QueryTypeInsert, db.QueryTypeUpdate, db.QueryTypeDelete)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := json.MarshalIndent(writeResult, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
