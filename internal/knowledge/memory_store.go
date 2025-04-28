package knowledge

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// MemoryStore implements Store interface using in-memory storage
type MemoryStore struct {
	records     map[string]Entry
	deletedRecs map[string]Entry
	mu          sync.RWMutex
}

// NewMemoryStore creates a new in-memory knowledge store
func NewMemoryStore() (*MemoryStore, error) {
	store := &MemoryStore{
		records:     make(map[string]Entry),
		deletedRecs: make(map[string]Entry),
	}
	return store, nil
}

// Open initializes the memory store
func (m *MemoryStore) Open() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure maps are initialized
	if m.records == nil {
		m.records = make(map[string]Entry)
	}
	if m.deletedRecs == nil {
		m.deletedRecs = make(map[string]Entry)
	}
	return nil
}

// Close releases resources (no-op for memory store)
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear data
	m.records = nil
	m.deletedRecs = nil
	return nil
}

// Flush persists data (no-op for memory store)
func (m *MemoryStore) Flush() error {
	// No-op for memory store since everything is already in memory
	return nil
}

// AddRecord adds a new knowledge record
func (m *MemoryStore) AddRecord(record Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate record
	if record.ID == "" {
		return errors.New("knowledge record must have an ID")
	}

	// Check if record already exists
	if _, exists := m.records[record.ID]; exists {
		return fmt.Errorf("knowledge record with ID %s already exists", record.ID)
	}

	// Set timestamps if not set
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = now
	}

	// Add to records
	m.records[record.ID] = record
	return nil
}

// GetRecord retrieves a knowledge record by ID
func (m *MemoryStore) GetRecord(id string) (Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if record exists
	if record, exists := m.records[id]; exists {
		return record, nil
	}

	return Entry{}, fmt.Errorf("knowledge record with ID %s not found", id)
}

// UpdateRecord updates an existing knowledge record
func (m *MemoryStore) UpdateRecord(record Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if record exists
	if _, exists := m.records[record.ID]; !exists {
		return fmt.Errorf("knowledge record with ID %s not found", record.ID)
	}

	// Update timestamp
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now()
	}

	// Update record
	m.records[record.ID] = record
	return nil
}

// DeleteRecord marks a record as deleted (soft delete)
func (m *MemoryStore) DeleteRecord(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if record exists
	record, exists := m.records[id]
	if !exists {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	// Move to deleted records
	m.deletedRecs[id] = record
	delete(m.records, id)
	return nil
}

// RestoreRecord restores a deleted record
func (m *MemoryStore) RestoreRecord(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if record exists in deleted records
	record, exists := m.deletedRecs[id]
	if !exists {
		return fmt.Errorf("deleted knowledge record with ID %s not found", id)
	}

	// Move to active records
	m.records[id] = record
	delete(m.deletedRecs, id)
	return nil
}

// PurgeRecord permanently deletes a record
func (m *MemoryStore) PurgeRecord(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if record exists in either active or deleted records
	_, existsActive := m.records[id]
	_, existsDeleted := m.deletedRecs[id]

	if !existsActive && !existsDeleted {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	// Remove from appropriate map
	if existsActive {
		delete(m.records, id)
	} else {
		delete(m.deletedRecs, id)
	}
	return nil
}

// SearchRecords searches for records based on the provided filter
func (m *MemoryStore) SearchRecords(filter Filter) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]Entry, 0)

	// Process active records first (unless we only want deleted records)
	if !filter.OnlyDeleted {
		for _, record := range m.records {
			// Apply filter
			if filter.RootGroup.Operator != "" && !m.matchesFilter(record, filter.RootGroup) {
				continue
			}

			// Add to results
			results = append(results, record)
		}
	}

	// Process deleted records if needed
	if filter.IncludeDeleted || filter.OnlyDeleted {
		for _, record := range m.deletedRecs {
			// Apply filter
			if filter.RootGroup.Operator != "" && !m.matchesFilter(record, filter.RootGroup) {
				continue
			}

			// Add to results
			results = append(results, record)
		}
	}

	// Sort results if order is specified
	if filter.OrderBy != "" {
		m.sortRecords(results, filter.OrderBy, filter.OrderDir)
	}

	// Apply limit and offset
	if filter.Limit > 0 || filter.Offset > 0 {
		// Bounds check
		start := filter.Offset
		if start > len(results) {
			start = len(results)
		}

		end := len(results)
		if filter.Limit > 0 && start+filter.Limit < end {
			end = start + filter.Limit
		}

		results = results[start:end]
	}

	return results, nil
}

// matchesFilter checks if a record matches the filter group
func (m *MemoryStore) matchesFilter(record Entry, group FilterGroup) bool {
	// Empty group matches everything
	if len(group.Conditions) == 0 && len(group.Groups) == 0 {
		return true
	}

	// Logic depends on operator
	switch group.Operator {
	case OpAnd:
		// Everything must match
		for _, condition := range group.Conditions {
			if !m.matchesCondition(record, condition) {
				return false
			}
		}

		for _, subgroup := range group.Groups {
			if !m.matchesFilter(record, subgroup) {
				return false
			}
		}

		return true

	case OpOr:
		// At least one must match
		if len(group.Conditions) > 0 {
			for _, condition := range group.Conditions {
				if m.matchesCondition(record, condition) {
					return true
				}
			}
		}

		if len(group.Groups) > 0 {
			for _, subgroup := range group.Groups {
				if m.matchesFilter(record, subgroup) {
					return true
				}
			}
		}

		return false

	case OpNot:
		// Inverse match
		result := true

		if len(group.Conditions) > 0 {
			for _, condition := range group.Conditions {
				if m.matchesCondition(record, condition) {
					result = false
					break
				}
			}
		}

		if result && len(group.Groups) > 0 {
			for _, subgroup := range group.Groups {
				if m.matchesFilter(record, subgroup) {
					result = false
					break
				}
			}
		}

		return result

	default:
		// Unknown operator
		return false
	}
}

// matchesCondition checks if a record matches a specific condition
func (m *MemoryStore) matchesCondition(record Entry, condition Condition) bool {
	// Special handling for metadata
	if condition.Field == "Metadata" {
		return m.matchesMetadata(record.Metadata, condition)
	}

	// Special handling for time fields - direct comparison without reflection
	switch condition.Field {
	case "CreatedAt":
		if condTime, ok := condition.Value.(time.Time); ok {
			return m.compareValues(record.CreatedAt, condition.Operator, condTime)
		}
	case "UpdatedAt":
		if condTime, ok := condition.Value.(time.Time); ok {
			return m.compareValues(record.UpdatedAt, condition.Operator, condTime)
		}
	case "ExpiresAt":
		if condTime, ok := condition.Value.(time.Time); ok {
			return m.compareValues(record.ExpiresAt, condition.Operator, condTime)
		}
	case "Content":
		// Special handling for Content ([]byte)
		switch v := condition.Value.(type) {
		case string:
			if condition.Operator == "=" {
				return string(record.Content) == v
			} else if condition.Operator == "CONTAINS" {
				return strings.Contains(string(record.Content), v)
			}
		case []byte:
			if condition.Operator == "=" {
				return bytes.Equal(record.Content, v)
			} else if condition.Operator == "CONTAINS" {
				return bytes.Contains(record.Content, v)
			}
		}
		// For other value types or operators, fallback to standard handling
	}

	// Get field value using reflection
	value := reflect.ValueOf(record).FieldByName(condition.Field)
	if !value.IsValid() {
		return false
	}

	// Special handling for slice types
	if value.Kind() == reflect.Slice {
		return m.matchesSlice(value, condition)
	}

	// Compare field value with condition value
	fieldValue := value.Interface()
	return m.compareValues(fieldValue, condition.Operator, condition.Value)
}

// matchesMetadata checks if metadata matches a condition
func (m *MemoryStore) matchesMetadata(metadata map[string]string, condition Condition) bool {
	// Condition value should be a map for metadata comparison
	metaCondition, ok := condition.Value.(map[string]interface{})
	if !ok {
		// Try string key as direct lookup
		if key, ok := condition.Value.(string); ok {
			// Check if key exists
			_, exists := metadata[key]
			if condition.Operator == "EXISTS" {
				return exists
			} else if condition.Operator == "NOT EXISTS" {
				return !exists
			} else {
				// Direct value comparison not possible without a key-value pair
				return false
			}
		}
		return false
	}

	// Handle different operators
	for key, value := range metaCondition {
		metaValue, exists := metadata[key]
		if !exists {
			return false
		}

		// Compare values
		if !m.compareValues(metaValue, condition.Operator, value) {
			return false
		}
	}

	return true
}

// matchesSlice checks if a slice field matches a condition
func (m *MemoryStore) matchesSlice(field reflect.Value, condition Condition) bool {
	switch condition.Operator {
	case "CONTAINS":
		value := condition.Value
		for i := 0; i < field.Len(); i++ {
			item := field.Index(i).Interface()
			// Simple equality check
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", value) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// compareValues compares two values based on the operator
func (m *MemoryStore) compareValues(fieldValue interface{}, operator string, conditionValue interface{}) bool {
	// Special handling for time comparisons
	fieldTime, fieldIsTime := fieldValue.(time.Time)
	valueTime, valueIsTime := conditionValue.(time.Time)

	if fieldIsTime && valueIsTime {
		switch operator {
		case "=":
			return fieldTime.Equal(valueTime)
		case "!=":
			return !fieldTime.Equal(valueTime)
		case ">":
			return fieldTime.After(valueTime)
		case "<":
			return fieldTime.Before(valueTime)
		case ">=":
			return fieldTime.After(valueTime) || fieldTime.Equal(valueTime)
		case "<=":
			return fieldTime.Before(valueTime) || fieldTime.Equal(valueTime)
		case "CONTAINS":
			return false // Time cannot contain another time
		default:
			return false
		}
	}

	// Special handling for []byte content
	fieldBytes, fieldIsBytes := fieldValue.([]byte)
	valueBytes, valueIsBytes := conditionValue.([]byte)
	valueString, valueIsString := conditionValue.(string)

	if fieldIsBytes {
		// Compare byte content with string or bytes
		fieldString := string(fieldBytes)
		var valueStr string
		if valueIsBytes {
			valueStr = string(valueBytes)
		} else if valueIsString {
			valueStr = valueString
		} else {
			valueStr = fmt.Sprintf("%v", conditionValue)
		}

		switch operator {
		case "=":
			return fieldString == valueStr
		case "!=":
			return fieldString != valueStr
		case "CONTAINS":
			return strings.Contains(fieldString, valueStr)
		default:
			// For other operators, continue with normal comparison
		}
	}

	// Convert to comparable strings for simple comparison
	fieldStr := fmt.Sprintf("%v", fieldValue)
	valueStr := fmt.Sprintf("%v", conditionValue)

	switch operator {
	case "=":
		return fieldStr == valueStr
	case "!=":
		return fieldStr != valueStr
	case "CONTAINS":
		return strings.Contains(fieldStr, valueStr)
	case ">":
		// Try numeric comparison
		var fieldNum, valueNum float64
		_, err1 := fmt.Sscanf(fieldStr, "%f", &fieldNum)
		_, err2 := fmt.Sscanf(valueStr, "%f", &valueNum)
		if err1 == nil && err2 == nil {
			return fieldNum > valueNum
		}
		// Fall back to string comparison
		return fieldStr > valueStr
	case "<":
		// Try numeric comparison
		var fieldNum, valueNum float64
		_, err1 := fmt.Sscanf(fieldStr, "%f", &fieldNum)
		_, err2 := fmt.Sscanf(valueStr, "%f", &valueNum)
		if err1 == nil && err2 == nil {
			return fieldNum < valueNum
		}
		// Fall back to string comparison
		return fieldStr < valueStr
	case ">=":
		// Try numeric comparison
		var fieldNum, valueNum float64
		_, err1 := fmt.Sscanf(fieldStr, "%f", &fieldNum)
		_, err2 := fmt.Sscanf(valueStr, "%f", &valueNum)
		if err1 == nil && err2 == nil {
			return fieldNum >= valueNum
		}
		// Fall back to string comparison
		return fieldStr >= valueStr
	case "<=":
		// Try numeric comparison
		var fieldNum, valueNum float64
		_, err1 := fmt.Sscanf(fieldStr, "%f", &fieldNum)
		_, err2 := fmt.Sscanf(valueStr, "%f", &valueNum)
		if err1 == nil && err2 == nil {
			return fieldNum <= valueNum
		}
		// Fall back to string comparison
		return fieldStr <= valueStr
	default:
		return false
	}
}

// sortRecords sorts records by the specified field and direction
func (m *MemoryStore) sortRecords(records []Entry, orderBy, orderDir string) {
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

// LoadRecords loads multiple records into the store
// If a record with the same ID exists, it's updated; otherwise, it's added
func (m *MemoryStore) LoadRecords(records ...Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for empty input
	if len(records) == 0 {
		return nil
	}

	// First validate that all records have IDs and there are no duplicates in the input
	seenIDs := make(map[string]bool)
	for i, record := range records {
		// Validate record has an ID
		if record.ID == "" {
			return fmt.Errorf("record at index %d must have an ID", i)
		}

		// Check for duplicate IDs in the input
		if seenIDs[record.ID] {
			return fmt.Errorf("duplicate record ID found in input: %s", record.ID)
		}
		seenIDs[record.ID] = true
	}

	// Process all records (add new ones, update existing ones)
	now := time.Now()
	for _, record := range records {
		// Check if record exists (update) or not (add)
		_, exists := m.records[record.ID]

		// Set timestamps appropriately
		if !exists {
			// New record: set createdAt if not set
			if record.CreatedAt.IsZero() {
				record.CreatedAt = now
			}
		}

		// Always set updatedAt timestamp if not explicitly set
		if record.UpdatedAt.IsZero() {
			record.UpdatedAt = now
		}

		// Store the record (add or update)
		m.records[record.ID] = record
	}

	return nil
}

// Info provides implementation-specific information about the memory store
func (m *MemoryStore) Info() (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make(map[string]string)

	// Add basic implementation info
	info["implementation"] = "MemoryStore"
	info["record_count"] = fmt.Sprintf("%d", len(m.records))
	info["deleted_count"] = fmt.Sprintf("%d", len(m.deletedRecs))
	info["persistent"] = "false"

	return info, nil
}
