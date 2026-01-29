package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"mysql-golang-mcp/config"
	"mysql-golang-mcp/db"
	"mysql-golang-mcp/tools"
)

const (
	serverName    = "mysql-mcp"
	serverVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to config.json file")
	flag.Parse()

	// Get config path
	cfgPath := config.GetConfigPath(*configPath)

	// Load configuration
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create connection manager
	manager := db.NewManager(cfg)
	defer manager.Close()

	// Create MCP server
	s := server.NewMCPServer(serverName, serverVersion)

	// Register tools
	tools.RegisterConnectionsTool(s, manager)
	tools.RegisterQueryTool(s, manager) // Deprecated, kept for backward compatibility
	tools.RegisterSchemaTool(s, manager)
	tools.RegisterIndexesTool(s, manager)

	// Register new segregated tools
	tools.RegisterReadTool(s, manager)   // mysql_select
	tools.RegisterWriteTools(s, manager) // mysql_insert, mysql_update, mysql_delete, mysql_alter, mysql_execute
	tools.RegisterUnsafeTool(s, manager) // mysql_execute_unsafe

	// Run with stdio transport
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
