package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/db"
)

// RegisterIndexesTool registers the get_indexes tool
func RegisterIndexesTool(s *server.MCPServer, manager *db.Manager) {
	tool := mcp.NewTool("get_indexes",
		mcp.WithDescription("Get indexes for a table including index name, columns, and uniqueness"),
		mcp.WithString("connection",
			mcp.Required(),
			mcp.Description("The named connection to use (from config)"),
		),
		mcp.WithString("table",
			mcp.Required(),
			mcp.Description("Table name to get indexes for"),
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
			query = fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", database, table)
		} else {
			query = fmt.Sprintf("SHOW INDEX FROM `%s`", table)
		}

		queryResult, err := manager.ExecuteQuery(connection, query)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Format indexes in a more readable structure
		indexMap := make(map[string]map[string]interface{})
		for _, row := range queryResult.Rows {
			keyName, _ := row["Key_name"].(string)
			if keyName == "" {
				continue
			}

			if _, exists := indexMap[keyName]; !exists {
				nonUnique := row["Non_unique"]
				isUnique := false
				switch v := nonUnique.(type) {
				case int64:
					isUnique = v == 0
				case string:
					isUnique = v == "0"
				}

				indexMap[keyName] = map[string]interface{}{
					"name":    keyName,
					"unique":  isUnique,
					"columns": []string{},
				}
			}

			colName, _ := row["Column_name"].(string)
			if colName != "" {
				cols := indexMap[keyName]["columns"].([]string)
				indexMap[keyName]["columns"] = append(cols, colName)
			}
		}

		// Convert map to slice
		indexes := make([]map[string]interface{}, 0, len(indexMap))
		for _, idx := range indexMap {
			indexes = append(indexes, idx)
		}

		result, err := json.MarshalIndent(indexes, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("failed to format result: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
