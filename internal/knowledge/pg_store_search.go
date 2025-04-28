package knowledge

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

// SearchRecords searches for records based on the provided filter
func (p *PgStore) SearchRecords(filter Filter) ([]Entry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("store not initialized")
	}

	// Build dynamic query based on filter
	query, args, err := p.buildSearchQuery(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	// Execute query
	rows, err := p.crudDB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	// Process results
	var results []Entry
	for rows.Next() {
		var record Entry
		var refsJSON, metadataJSON []byte
		var expiresAtNull, createdAt, updatedAt sql.NullTime
		var sourceIDNull, sourceTypeNull, ownerIDNull, ownerTypeNull, subjectTypeNull sql.NullString
		var subjectIDs, tags []string
		var isDeleted bool

		// Scan row into variables
		err := rows.Scan(
			&record.ID,
			&record.Category,
			&record.ContentType,
			&record.Content,
			&record.Importance,
			&createdAt,
			&updatedAt,
			&expiresAtNull,
			&sourceIDNull,
			&sourceTypeNull,
			&ownerIDNull,
			&ownerTypeNull,
			pq.Array(&subjectIDs),
			&subjectTypeNull,
			pq.Array(&tags),
			&refsJSON,
			&metadataJSON,
			&isDeleted,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		// Deserialize References from JSONB
		if len(refsJSON) > 0 {
			if err := json.Unmarshal(refsJSON, &record.References); err != nil {
				return nil, fmt.Errorf("failed to deserialize references: %w", err)
			}
		}

		// Deserialize Metadata from JSONB
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &record.Metadata); err != nil {
				return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
			}
		}

		// Set timestamp fields
		if createdAt.Valid {
			record.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			record.UpdatedAt = updatedAt.Time
		}

		// Set nullable fields
		if expiresAtNull.Valid {
			record.ExpiresAt = expiresAtNull.Time
		}

		if sourceIDNull.Valid {
			record.SourceID = sourceIDNull.String
		}

		if sourceTypeNull.Valid {
			record.SourceType = sourceTypeNull.String
		}

		if ownerIDNull.Valid {
			record.OwnerID = ownerIDNull.String
		}

		if ownerTypeNull.Valid {
			record.OwnerType = ownerTypeNull.String
		}

		record.SubjectIDs = subjectIDs

		if subjectTypeNull.Valid {
			record.SubjectType = subjectTypeNull.String
		}

		record.Tags = tags

		results = append(results, record)
	}

	// Check for errors after iterating through rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	// If sorting is requested, sort the results (in-memory sorting)
	if filter.OrderBy != "" {
		p.sortRecords(results, filter.OrderBy, filter.OrderDir)
	}

	return results, nil
}

// buildSearchQuery constructs a SQL query from a Filter
func (p *PgStore) buildSearchQuery(filter Filter) (string, []interface{}, error) {
	var queryBuilder strings.Builder
	var args []interface{}
	var whereClause string
	var err error

	// Base SELECT query
	queryBuilder.WriteString(`
		SELECT 
			id, category, content_type, content, importance,
			created_at, updated_at, expires_at, source_id, source_type,
			owner_id, owner_type, subject_ids, subject_type, tags,
			references, metadata, is_deleted
		FROM knowledge_entry
		WHERE account_id = $1
	`)

	// Add account_id as first argument
	args = append(args, p.accountID)

	// Handle deleted records filter
	if filter.OnlyDeleted {
		queryBuilder.WriteString(" AND is_deleted = true")
	} else if !filter.IncludeDeleted {
		queryBuilder.WriteString(" AND is_deleted = false")
	}

	// Build filter conditions
	if !isEmptyFilterGroup(filter.RootGroup) {
		whereClause, err = p.buildWhereClause(filter.RootGroup, len(args)+1)
		if err != nil {
			return "", nil, err
		}

		queryBuilder.WriteString(" AND (")
		queryBuilder.WriteString(whereClause)
		queryBuilder.WriteString(")")
	}

	// Add ORDER BY clause if specified
	if filter.OrderBy != "" {
		// Map Go struct field names to database column names
		columnName := fieldToColumnName(filter.OrderBy)
		if columnName != "" {
			direction := "ASC"
			if strings.ToUpper(filter.OrderDir) == "DESC" {
				direction = "DESC"
			}
			queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", columnName, direction))
		}
	}

	// Add LIMIT and OFFSET if specified
	if filter.Limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d", filter.Limit))
	}

	if filter.Offset > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" OFFSET %d", filter.Offset))
	}

	return queryBuilder.String(), args, nil
}

// buildWhereClause recursively builds the WHERE clause for a filter group
func (p *PgStore) buildWhereClause(group FilterGroup, startParamIndex int) (string, error) {
	var (
		clauses    []string
		paramIndex = startParamIndex
		operator   string
	)

	// Set the operator
	switch group.Operator {
	case OpAnd:
		operator = "AND"
	case OpOr:
		operator = "OR"
	case OpNot:
		operator = "AND NOT"
	default:
		operator = "AND" // Default to AND if not specified
	}

	// Process conditions
	for _, condition := range group.Conditions {
		clause, nextParamIndex, err := p.buildConditionClause(condition, paramIndex)
		if err != nil {
			return "", err
		}
		if clause != "" {
			clauses = append(clauses, clause)
		}
		paramIndex = nextParamIndex
	}

	// Process nested groups
	for _, nestedGroup := range group.Groups {
		nestedClause, err := p.buildWhereClause(nestedGroup, paramIndex)
		if err != nil {
			return "", err
		}
		if nestedClause != "" {
			clauses = append(clauses, "("+nestedClause+")")
		}
	}

	// Combine clauses with the appropriate operator
	if len(clauses) > 0 {
		return strings.Join(clauses, " "+operator+" "), nil
	}

	return "", nil
}

// buildConditionClause builds a SQL condition from a Filter condition
func (p *PgStore) buildConditionClause(condition Condition, paramIndex int) (string, int, error) {
	// Map Go struct field names to database column names
	columnName := fieldToColumnName(condition.Field)
	if columnName == "" {
		return "", paramIndex, fmt.Errorf("unknown field: %s", condition.Field)
	}

	// Special handling for different field types and operators
	switch condition.Field {
	case "Metadata":
		// For Metadata field, we need to use JSONB operators
		switch condition.Operator {
		case "=", "!=":
			// For metadata equality, we check if a specific key has a specific value
			if mapValue, ok := condition.Value.(map[string]interface{}); ok {
				var clauses []string
				for key, value := range mapValue {
					jsonbPath := fmt.Sprintf("metadata->>'%s'", key)
					// Convert value to string for the query
					fmt.Sprintf("%v", value) // Evaluated but unused, just to create correct SQL

					if condition.Operator == "=" {
						clauses = append(clauses, fmt.Sprintf("%s = $%d", jsonbPath, paramIndex))
					} else {
						clauses = append(clauses, fmt.Sprintf("%s != $%d", jsonbPath, paramIndex))
					}
					paramIndex++
				}
				if len(clauses) > 0 {
					op := " AND "
					if condition.Operator == "!=" {
						op = " OR "
					}
					return "(" + strings.Join(clauses, op) + ")", paramIndex, nil
				}
			}
			return "", paramIndex, fmt.Errorf("invalid metadata value format")

		case "CONTAINS":
			// Check if the metadata contains a specific key
			if _, ok := condition.Value.(string); ok {
				return fmt.Sprintf("metadata ? $%d", paramIndex), paramIndex + 1, nil
			}
			return "", paramIndex, fmt.Errorf("CONTAINS operator for Metadata requires string value")
		}

	case "Tags", "SubjectIDs":
		// Array fields
		switch condition.Operator {
		case "=":
			// Single value in array
			return fmt.Sprintf("$%d = ANY(%s)", paramIndex, columnName), paramIndex + 1, nil
		case "CONTAINS":
			// Array contains specific value
			return fmt.Sprintf("$%d = ANY(%s)", paramIndex, columnName), paramIndex + 1, nil
		case "@>":
			// Array contains all values from another array
			return fmt.Sprintf("%s @> $%d", columnName, paramIndex), paramIndex + 1, nil
		}

	case "Content":
		// Binary content field
		switch condition.Operator {
		case "=":
			return fmt.Sprintf("%s = $%d", columnName, paramIndex), paramIndex + 1, nil
		case "!=":
			return fmt.Sprintf("%s != $%d", columnName, paramIndex), paramIndex + 1, nil
		case "CONTAINS":
			// For bytea CONTAINS, we convert to string and use LIKE
			return fmt.Sprintf("encode(%s, 'escape') LIKE '%%' || encode($%d, 'escape') || '%%'",
				columnName, paramIndex), paramIndex + 1, nil
		}

	case "CreatedAt", "UpdatedAt", "ExpiresAt":
		// Time fields
		switch condition.Operator {
		case "=":
			return fmt.Sprintf("%s::date = $%d::date", columnName, paramIndex), paramIndex + 1, nil
		case ">":
			return fmt.Sprintf("%s > $%d", columnName, paramIndex), paramIndex + 1, nil
		case "<":
			return fmt.Sprintf("%s < $%d", columnName, paramIndex), paramIndex + 1, nil
		case ">=":
			return fmt.Sprintf("%s >= $%d", columnName, paramIndex), paramIndex + 1, nil
		case "<=":
			return fmt.Sprintf("%s <= $%d", columnName, paramIndex), paramIndex + 1, nil
		case "BETWEEN":
			// Handle time range with array of two values
			if timeRange, ok := condition.Value.([]interface{}); ok && len(timeRange) == 2 {
				return fmt.Sprintf("%s BETWEEN $%d AND $%d",
					columnName, paramIndex, paramIndex+1), paramIndex + 2, nil
			}
			return "", paramIndex, fmt.Errorf("BETWEEN operator requires array of two values")
		}

	case "References":
		// Handle references as JSONB
		switch condition.Operator {
		case "CONTAINS":
			// Check if references contains an object with specific ID
			if _, ok := condition.Value.(string); ok {
				return fmt.Sprintf("references @> $%d", paramIndex), paramIndex + 1, nil
			}
			return "", paramIndex, fmt.Errorf("CONTAINS operator for References requires string value")
		}

	default:
		// Standard fields
		switch condition.Operator {
		case "=":
			return fmt.Sprintf("%s = $%d", columnName, paramIndex), paramIndex + 1, nil
		case "!=":
			return fmt.Sprintf("%s != $%d", columnName, paramIndex), paramIndex + 1, nil
		case ">":
			return fmt.Sprintf("%s > $%d", columnName, paramIndex), paramIndex + 1, nil
		case "<":
			return fmt.Sprintf("%s < $%d", columnName, paramIndex), paramIndex + 1, nil
		case ">=":
			return fmt.Sprintf("%s >= $%d", columnName, paramIndex), paramIndex + 1, nil
		case "<=":
			return fmt.Sprintf("%s <= $%d", columnName, paramIndex), paramIndex + 1, nil
		case "LIKE", "CONTAINS":
			return fmt.Sprintf("%s LIKE '%%' || $%d || '%%'", columnName, paramIndex), paramIndex + 1, nil
		case "STARTSWITH":
			return fmt.Sprintf("%s LIKE $%d || '%%'", columnName, paramIndex), paramIndex + 1, nil
		case "ENDSWITH":
			return fmt.Sprintf("%s LIKE '%%' || $%d", columnName, paramIndex), paramIndex + 1, nil
		case "IN":
			// IN operator with array of values
			return fmt.Sprintf("%s IN (select unnest($%d::text[]))", columnName, paramIndex), paramIndex + 1, nil
		case "IS NULL":
			return fmt.Sprintf("%s IS NULL", columnName), paramIndex, nil
		case "IS NOT NULL":
			return fmt.Sprintf("%s IS NOT NULL", columnName), paramIndex, nil
		}
	}

	return "", paramIndex, fmt.Errorf("unsupported operator %s for field %s", condition.Operator, condition.Field)
}

// fieldToColumnName maps Go struct field names to database column names
func fieldToColumnName(field string) string {
	// Map of Go struct field names to database column names
	fieldMap := map[string]string{
		"ID":          "id",
		"Category":    "category",
		"ContentType": "content_type",
		"Content":     "content",
		"Importance":  "importance",
		"CreatedAt":   "created_at",
		"UpdatedAt":   "updated_at",
		"ExpiresAt":   "expires_at",
		"SourceID":    "source_id",
		"SourceType":  "source_type",
		"OwnerID":     "owner_id",
		"OwnerType":   "owner_type",
		"SubjectIDs":  "subject_ids",
		"SubjectType": "subject_type",
		"Tags":        "tags",
		"References":  "references",
		"Metadata":    "metadata",
	}

	columnName, exists := fieldMap[field]
	if !exists {
		return ""
	}
	return columnName
}

// isEmptyFilterGroup checks if a filter group is effectively empty
func isEmptyFilterGroup(group FilterGroup) bool {
	return len(group.Conditions) == 0 && len(group.Groups) == 0
}

// sortRecords sorts records by the specified field and direction
// This is a fallback for complex sorts that can't be handled in SQL
func (p *PgStore) sortRecords(records []Entry, orderBy, orderDir string) {
	// Implementation reused from memory_store.go
	sort.Slice(records, func(i, j int) bool {
		// Get field values using reflection
		iValue := reflect.ValueOf(records[i]).FieldByName(orderBy)
		jValue := reflect.ValueOf(records[j]).FieldByName(orderBy)

		// Check if field exists
		if !iValue.IsValid() || !jValue.IsValid() {
			return false
		}

		// Compare based on type
		ascending := orderDir != "DESC"

		switch iValue.Kind() {
		case reflect.String:
			if ascending {
				return iValue.String() < jValue.String()
			}
			return iValue.String() > jValue.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if ascending {
				return iValue.Int() < jValue.Int()
			}
			return iValue.Int() > jValue.Int()
		case reflect.Float32, reflect.Float64:
			if ascending {
				return iValue.Float() < jValue.Float()
			}
			return iValue.Float() > jValue.Float()
		case reflect.Struct:
			// Special handling for time.Time
			if iTime, ok := iValue.Interface().(time.Time); ok {
				if jTime, ok := jValue.Interface().(time.Time); ok {
					if ascending {
						return iTime.Before(jTime)
					}
					return jTime.Before(iTime)
				}
			}
			fallthrough
		default:
			// Default string comparison
			iStr := fmt.Sprintf("%v", iValue.Interface())
			jStr := fmt.Sprintf("%v", jValue.Interface())
			if ascending {
				return iStr < jStr
			}
			return iStr > jStr
		}
	})
}
