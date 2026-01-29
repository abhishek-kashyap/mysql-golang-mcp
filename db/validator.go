package db

import (
	"fmt"
	"strings"
)

// QueryType represents the type of SQL query
type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeSelect
	QueryTypeInsert
	QueryTypeUpdate
	QueryTypeDelete
	QueryTypeAlter
	QueryTypeShow
	QueryTypeDescribe
	QueryTypeExplain
	QueryTypeDrop
	QueryTypeTruncate
	QueryTypeCreate
	QueryTypeGrant
	QueryTypeRevoke
	QueryTypeSet
	QueryTypeUse
)

// DetectQueryType analyzes a SQL query and returns its type
func DetectQueryType(query string) QueryType {
	q := strings.TrimSpace(strings.ToUpper(query))

	// Map of prefixes to query types (order matters for some overlapping cases)
	prefixMap := []struct {
		prefix    string
		queryType QueryType
	}{
		{"SELECT", QueryTypeSelect},
		{"INSERT", QueryTypeInsert},
		{"UPDATE", QueryTypeUpdate},
		{"DELETE", QueryTypeDelete},
		{"ALTER", QueryTypeAlter},
		{"SHOW", QueryTypeShow},
		{"DESCRIBE", QueryTypeDescribe},
		{"DESC ", QueryTypeDescribe}, // Note: space to avoid matching "DELETE"
		{"EXPLAIN", QueryTypeExplain},
		{"DROP", QueryTypeDrop},
		{"TRUNCATE", QueryTypeTruncate},
		{"CREATE", QueryTypeCreate},
		{"GRANT", QueryTypeGrant},
		{"REVOKE", QueryTypeRevoke},
		{"SET", QueryTypeSet},
		{"USE", QueryTypeUse},
	}

	for _, pm := range prefixMap {
		if strings.HasPrefix(q, pm.prefix) {
			return pm.queryType
		}
	}

	return QueryTypeUnknown
}

// ValidateQueryType checks if the query matches one of the allowed types
func ValidateQueryType(query string, allowed ...QueryType) error {
	detected := DetectQueryType(query)

	for _, qt := range allowed {
		if detected == qt {
			return nil
		}
	}

	// Build error message with expected types
	expectedLabels := make([]string, len(allowed))
	for i, qt := range allowed {
		expectedLabels[i] = GetQueryTypeLabel(qt)
	}

	expected := strings.Join(expectedLabels, "/")
	got := GetQueryTypeLabel(detected)

	return fmt.Errorf("query type mismatch: expected %s, got %s. Use the appropriate tool for this query type", expected, got)
}

// GetQueryTypeLabel returns a human-readable label for a query type
func GetQueryTypeLabel(qt QueryType) string {
	labels := map[QueryType]string{
		QueryTypeUnknown:  "UNKNOWN",
		QueryTypeSelect:   "SELECT",
		QueryTypeInsert:   "INSERT",
		QueryTypeUpdate:   "UPDATE",
		QueryTypeDelete:   "DELETE",
		QueryTypeAlter:    "ALTER",
		QueryTypeShow:     "SHOW",
		QueryTypeDescribe: "DESCRIBE",
		QueryTypeExplain:  "EXPLAIN",
		QueryTypeDrop:     "DROP",
		QueryTypeTruncate: "TRUNCATE",
		QueryTypeCreate:   "CREATE",
		QueryTypeGrant:    "GRANT",
		QueryTypeRevoke:   "REVOKE",
		QueryTypeSet:      "SET",
		QueryTypeUse:      "USE",
	}

	if label, ok := labels[qt]; ok {
		return label
	}
	return "UNKNOWN"
}

// IsReadOnlyQueryType returns true if the query type is read-only
func IsReadOnlyQueryType(qt QueryType) bool {
	switch qt {
	case QueryTypeSelect, QueryTypeShow, QueryTypeDescribe, QueryTypeExplain:
		return true
	default:
		return false
	}
}

// IsDangerousQueryType returns true if the query type is dangerous (DDL)
func IsDangerousQueryType(qt QueryType) bool {
	switch qt {
	case QueryTypeDrop, QueryTypeTruncate, QueryTypeCreate, QueryTypeGrant, QueryTypeRevoke:
		return true
	default:
		return false
	}
}
