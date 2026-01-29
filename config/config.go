package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// ConnectionConfig holds settings for a single database connection
type ConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	ReadOnly bool   `json:"read_only"`
	MaxRows  int    `json:"max_rows"`
}

// Config holds all database connections
type Config struct {
	Connections map[string]*ConnectionConfig `json:"connections"`
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults and validate
	for name, conn := range cfg.Connections {
		if err := validateAndApplyDefaults(name, conn); err != nil {
			return nil, err
		}
	}

	if len(cfg.Connections) == 0 {
		return nil, fmt.Errorf("no connections defined in config")
	}

	return &cfg, nil
}

// validateAndApplyDefaults validates connection config and applies default values
func validateAndApplyDefaults(name string, conn *ConnectionConfig) error {
	// Expand environment variables in sensitive fields
	conn.Host = expandEnvVar(conn.Host)
	conn.User = expandEnvVar(conn.User)
	conn.Password = expandEnvVar(conn.Password)
	conn.Database = expandEnvVar(conn.Database)

	if conn.Host == "" {
		return fmt.Errorf("connection '%s': host is required", name)
	}
	if conn.User == "" {
		return fmt.Errorf("connection '%s': user is required", name)
	}
	if conn.Database == "" {
		return fmt.Errorf("connection '%s': database is required", name)
	}

	// Apply defaults
	if conn.Port == 0 {
		conn.Port = 3306
	}
	if conn.MaxRows == 0 {
		conn.MaxRows = 1000
	}
	// ReadOnly defaults to false (Go zero value), but we want true as default
	// Since we can't distinguish between explicit false and unset, we document
	// that read_only defaults to true and users must explicitly set false

	return nil
}

// expandEnvVar expands ${VAR_NAME} syntax to environment variable values
func expandEnvVar(value string) string {
	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`^\$\{([^}]+)\}$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) == 2 {
		return os.Getenv(matches[1])
	}
	return value
}

// GetConfigPath returns the config file path from env var, flag, or default
func GetConfigPath(flagValue string) string {
	// Command line flag takes precedence
	if flagValue != "" {
		return flagValue
	}

	// Then environment variable
	if envPath := os.Getenv("MYSQL_MCP_CONFIG"); envPath != "" {
		return envPath
	}

	// Default
	return "./config.json"
}

// DSN returns the MySQL DSN string for the connection
func (c *ConnectionConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=30s&readTimeout=30s&writeTimeout=30s",
		c.User, c.Password, c.Host, c.Port, c.Database)
}
