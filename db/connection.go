package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"mysql-golang-mcp/config"
)

// Manager handles multiple database connections
type Manager struct {
	config      *config.Config
	connections map[string]*sql.DB
	mu          sync.RWMutex
}

// NewManager creates a new connection manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:      cfg,
		connections: make(map[string]*sql.DB),
	}
}

// GetConnection returns a database connection by name, creating it if necessary
func (m *Manager) GetConnection(name string) (*sql.DB, *config.ConnectionConfig, error) {
	connConfig, exists := m.config.Connections[name]
	if !exists {
		return nil, nil, fmt.Errorf("unknown connection: %s", name)
	}

	m.mu.RLock()
	db, exists := m.connections[name]
	m.mu.RUnlock()

	if exists {
		// Check if connection is still alive
		if err := db.Ping(); err == nil {
			return db, connConfig, nil
		}
		// Connection is dead, close it and reconnect
		db.Close()
	}

	// Create new connection
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if db, exists := m.connections[name]; exists {
		if err := db.Ping(); err == nil {
			return db, connConfig, nil
		}
		db.Close()
	}

	db, err := sql.Open("mysql", connConfig.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open connection '%s': %w", name, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to connect to '%s': %w", name, err)
	}

	m.connections[name] = db
	return db, connConfig, nil
}

// ListConnections returns all configured connection names with their read-only status
func (m *Manager) ListConnections() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(m.config.Connections))
	for name, conn := range m.config.Connections {
		result = append(result, map[string]interface{}{
			"name":      name,
			"read_only": conn.ReadOnly,
		})
	}
	return result
}

// Close closes all open connections
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, db := range m.connections {
		db.Close()
	}
	m.connections = make(map[string]*sql.DB)
}

// QueryResult holds the result of a query
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int                      `json:"count"`
}

// WriteResult holds the result of a write operation
type WriteResult struct {
	RowsAffected int64 `json:"rows_affected"`
	LastInsertID int64 `json:"last_insert_id,omitempty"`
}

// UnsafeResult holds the result of an unsafe operation
type UnsafeResult struct {
	QueryResult  *QueryResult `json:"query_result,omitempty"`
	WriteResult  *WriteResult `json:"write_result,omitempty"`
	Warning      string       `json:"warning"`
	SkippedCheck string       `json:"skipped_check"`
}

// ExecuteQuery executes a SQL query and returns the results
func (m *Manager) ExecuteQuery(connectionName, query string) (*QueryResult, error) {
	db, connConfig, err := m.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	// Check read-only mode
	if connConfig.ReadOnly && !isReadOnlyQuery(query) {
		return nil, fmt.Errorf("connection '%s' is read-only, write operations are not allowed", connectionName)
	}

	// Check for dangerous operations even in write mode
	if !connConfig.ReadOnly && isDangerousQuery(query) {
		return nil, fmt.Errorf("dangerous operations (DROP, ALTER, TRUNCATE, CREATE, GRANT, REVOKE) are not allowed")
	}

	// Block sensitive metadata queries
	if isSensitiveQuery(query) {
		return nil, fmt.Errorf("access to sensitive MySQL metadata is not allowed")
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := &QueryResult{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
	}

	// Prepare value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	rowCount := 0
	for rows.Next() {
		if rowCount >= connConfig.MaxRows {
			break
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for JSON serialization
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result.Rows = append(result.Rows, row)
		rowCount++
	}

	result.Count = rowCount
	return result, nil
}

// isReadOnlyQuery checks if a query is read-only
func isReadOnlyQuery(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	readOnlyPrefixes := []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"}
	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(q, prefix) {
			return true
		}
	}
	return false
}

// isDangerousQuery checks for dangerous DDL operations
func isDangerousQuery(query string) bool {
	q := strings.TrimSpace(strings.ToUpper(query))
	dangerousPrefixes := []string{"DROP", "ALTER", "TRUNCATE", "CREATE", "GRANT", "REVOKE"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(q, prefix) {
			return true
		}
	}
	return false
}

// isSensitiveQuery checks for queries that could expose credentials or sensitive metadata
func isSensitiveQuery(query string) bool {
	q := strings.ToUpper(query)

	// Block SHOW GRANTS
	if strings.Contains(q, "SHOW GRANTS") {
		return true
	}

	// Block access to mysql.user table
	if strings.Contains(q, "MYSQL.USER") {
		return true
	}

	// Block access to information_schema.user_privileges
	if strings.Contains(q, "USER_PRIVILEGES") {
		return true
	}

	// Block SHOW PROCESSLIST (can show connection details)
	if strings.Contains(q, "SHOW PROCESSLIST") || strings.Contains(q, "SHOW FULL PROCESSLIST") {
		return true
	}

	return false
}

// ExecuteWrite executes a write operation (INSERT, UPDATE, DELETE) and returns affected rows
func (m *Manager) ExecuteWrite(connectionName, query string, allowedTypes ...QueryType) (*WriteResult, error) {
	db, connConfig, err := m.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	// Validate query type
	if len(allowedTypes) > 0 {
		if err := ValidateQueryType(query, allowedTypes...); err != nil {
			return nil, err
		}
	}

	// Check read-only mode
	if connConfig.ReadOnly {
		return nil, fmt.Errorf("connection '%s' is read-only, write operations are not allowed", connectionName)
	}

	// Check for dangerous operations
	queryType := DetectQueryType(query)
	if IsDangerousQueryType(queryType) {
		return nil, fmt.Errorf("dangerous operations (DROP, TRUNCATE, CREATE, GRANT, REVOKE) are not allowed. Use mysql_execute_unsafe if you need to bypass this check")
	}

	// Block sensitive metadata queries
	if isSensitiveQuery(query) {
		return nil, fmt.Errorf("access to sensitive MySQL metadata is not allowed")
	}

	result, err := db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return &WriteResult{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
	}, nil
}

// ExecuteAlter executes an ALTER TABLE statement
func (m *Manager) ExecuteAlter(connectionName, query string) (*WriteResult, error) {
	db, connConfig, err := m.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	// Validate query type
	if err := ValidateQueryType(query, QueryTypeAlter); err != nil {
		return nil, err
	}

	// Check read-only mode
	if connConfig.ReadOnly {
		return nil, fmt.Errorf("connection '%s' is read-only, ALTER operations are not allowed", connectionName)
	}

	// Block truly dangerous operations even for ALTER
	q := strings.ToUpper(strings.TrimSpace(query))
	blockedPatterns := []string{"DROP DATABASE", "DROP SCHEMA", "TRUNCATE", "CREATE DATABASE", "GRANT", "REVOKE"}
	for _, pattern := range blockedPatterns {
		if strings.Contains(q, pattern) {
			return nil, fmt.Errorf("operation '%s' is not allowed even with mysql_alter. Use mysql_execute_unsafe if absolutely necessary", pattern)
		}
	}

	// Block sensitive metadata queries
	if isSensitiveQuery(query) {
		return nil, fmt.Errorf("access to sensitive MySQL metadata is not allowed")
	}

	result, err := db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return &WriteResult{
		RowsAffected: rowsAffected,
	}, nil
}

// ExecuteUnsafe executes any query, bypassing dangerous and sensitive query checks
// WARNING: This method should only be used when absolutely necessary
func (m *Manager) ExecuteUnsafe(connectionName, query string) (*UnsafeResult, error) {
	db, connConfig, err := m.GetConnection(connectionName)
	if err != nil {
		return nil, err
	}

	// Still respect read-only mode - that's a configuration choice
	queryType := DetectQueryType(query)
	if connConfig.ReadOnly && !IsReadOnlyQueryType(queryType) {
		return nil, fmt.Errorf("connection '%s' is read-only, write operations are not allowed (even with unsafe mode)", connectionName)
	}

	// Determine what checks we're skipping
	var skippedChecks []string
	if isDangerousQuery(query) {
		skippedChecks = append(skippedChecks, "dangerous query blocking")
	}
	if isSensitiveQuery(query) {
		skippedChecks = append(skippedChecks, "sensitive query blocking")
	}

	skippedCheckMsg := "none"
	if len(skippedChecks) > 0 {
		skippedCheckMsg = strings.Join(skippedChecks, ", ")
	}

	result := &UnsafeResult{
		Warning:      "UNSAFE EXECUTION: This query bypassed safety checks. Ensure you understand the implications.",
		SkippedCheck: skippedCheckMsg,
	}

	// Determine if this is a read or write query
	if IsReadOnlyQueryType(queryType) {
		// Use Query for SELECT-like operations
		rows, err := db.Query(query)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("failed to get columns: %w", err)
		}

		queryResult := &QueryResult{
			Columns: columns,
			Rows:    make([]map[string]interface{}, 0),
		}

		// Prepare value holders
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		rowCount := 0
		for rows.Next() {
			if rowCount >= connConfig.MaxRows {
				break
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, fmt.Errorf("failed to scan row: %w", err)
			}

			row := make(map[string]interface{})
			for i, col := range columns {
				val := values[i]
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			}
			queryResult.Rows = append(queryResult.Rows, row)
			rowCount++
		}

		queryResult.Count = rowCount
		result.QueryResult = queryResult
	} else {
		// Use Exec for write operations
		execResult, err := db.Exec(query)
		if err != nil {
			return nil, fmt.Errorf("query execution failed: %w", err)
		}

		rowsAffected, _ := execResult.RowsAffected()
		lastInsertID, _ := execResult.LastInsertId()

		result.WriteResult = &WriteResult{
			RowsAffected: rowsAffected,
			LastInsertID: lastInsertID,
		}
	}

	return result, nil
}
