package knowledge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileStore data structure
type fileData struct {
	Records     map[string]Entry `json:"records"`
	DeletedRecs map[string]Entry `json:"deleted_records"`
}

// FileStore implements Store interface using a JSON file for storage
type FileStore struct {
	filename    string
	records     map[string]Entry
	deletedRecs map[string]Entry
	isDirty     bool
	mu          sync.RWMutex
}

// NewFileStore creates new file-based knowledge store
func NewFileStore(filename string) (*FileStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for knowledge file: %w", err)
		}
	}

	store := &FileStore{
		filename:    filename,
		records:     make(map[string]Entry),
		deletedRecs: make(map[string]Entry),
		isDirty:     false,
	}

	return store, nil
}

// Open loads the knowledge store from file
func (f *FileStore) Open() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(f.filename); os.IsNotExist(err) {
		// File doesn't exist yet, initialize empty store
		f.records = make(map[string]Entry)
		f.deletedRecs = make(map[string]Entry)
		return nil
	}

	// Read file
	data, err := os.ReadFile(f.filename)
	if err != nil {
		return fmt.Errorf("failed to read knowledge file: %w", err)
	}

	// Unmarshal data
	var fileData fileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to parse knowledge file: %w", err)
	}

	// Copy data to store
	f.records = fileData.Records
	f.deletedRecs = fileData.DeletedRecs
	f.isDirty = false

	return nil
}

// Close flushes data to disk and releases resources
func (f *FileStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// If there are changes, flush to disk
	if f.isDirty {
		if err := f.flush(); err != nil {
			return err
		}
	}

	// Clear in-knowledge data
	f.records = nil
	f.deletedRecs = nil

	return nil
}

// Flush writes current data to disk if needed
func (f *FileStore) Flush() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isDirty {
		return f.flush()
	}
	return nil
}

// internal flush method (must be called with lock held)
func (f *FileStore) flush() error {
	// Create storage structure
	fileData := fileData{
		Records:     f.records,
		DeletedRecs: f.deletedRecs,
	}

	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge data: %w", err)
	}

	// Write to temp file
	tempFile := f.filename + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write knowledge data to temp file: %w", err)
	}

	// Rename temp file to actual file (atomic operation)
	if err := os.Rename(tempFile, f.filename); err != nil {
		return fmt.Errorf("failed to save knowledge file: %w", err)
	}

	f.isDirty = false
	return nil
}

// AddRecord adds a new knowledge record
func (f *FileStore) AddRecord(record Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Validate record
	if record.ID == "" {
		return errors.New("knowledge record must have an ID")
	}

	// Check if record already exists
	if _, exists := f.records[record.ID]; exists {
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
	f.records[record.ID] = record
	f.isDirty = true

	return nil
}

// GetRecord retrieves a knowledge record by ID
func (f *FileStore) GetRecord(id string) (Entry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Check if record exists
	if record, exists := f.records[id]; exists {
		return record, nil
	}

	return Entry{}, fmt.Errorf("knowledge record with ID %s not found", id)
}

// UpdateRecord updates an existing knowledge record
func (f *FileStore) UpdateRecord(record Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if record exists
	if _, exists := f.records[record.ID]; !exists {
		return fmt.Errorf("knowledge record with ID %s not found", record.ID)
	}

	// Update timestamp
	record.UpdatedAt = time.Now()

	// Update record
	f.records[record.ID] = record
	f.isDirty = true

	return nil
}

// DeleteRecord marks a record as deleted (soft delete)
func (f *FileStore) DeleteRecord(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if record exists
	record, exists := f.records[id]
	if !exists {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	// Move record to deleted records
	f.deletedRecs[id] = record
	delete(f.records, id)
	f.isDirty = true

	return nil
}

// RestoreRecord restores a deleted record
func (f *FileStore) RestoreRecord(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if record exists in deleted records
	record, exists := f.deletedRecs[id]
	if !exists {
		return fmt.Errorf("deleted knowledge record with ID %s not found", id)
	}

	// Move record back to active records
	f.records[id] = record
	delete(f.deletedRecs, id)
	f.isDirty = true

	return nil
}

// PurgeRecord permanently deletes a record
func (f *FileStore) PurgeRecord(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if record exists in either active or deleted records
	_, existsActive := f.records[id]
	_, existsDeleted := f.deletedRecs[id]

	if !existsActive && !existsDeleted {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	// Remove from appropriate map
	if existsActive {
		delete(f.records, id)
	} else {
		delete(f.deletedRecs, id)
	}

	f.isDirty = true
	return nil
}

// SearchRecords searches for records based on the provided filter
func (f *FileStore) SearchRecords(filter Filter) ([]Entry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create result slice
	var results []Entry

	// Determine which records to search
	recordsToSearch := make(map[string]Entry)

	// Add active records if not OnlyDeleted
	if !filter.OnlyDeleted {
		for k, v := range f.records {
			recordsToSearch[k] = v
		}
	}

	// Add deleted records if IncludeDeleted or OnlyDeleted
	if filter.IncludeDeleted || filter.OnlyDeleted {
		for k, v := range f.deletedRecs {
			recordsToSearch[k] = v
		}
	}

	// Apply filters
	for _, record := range recordsToSearch {
		if f.matchesFilter(record, filter.RootGroup) {
			results = append(results, record)
		}
	}

	// Sort results if OrderBy is specified
	if filter.OrderBy != "" {
		f.sortRecords(results, filter.OrderBy, filter.OrderDir)
	}

	// Apply pagination
	if len(results) > 0 {
		// Handle offset
		if filter.Offset > 0 {
			if filter.Offset >= len(results) {
				return []Entry{}, nil
			}
			results = results[filter.Offset:]
		}

		// Handle limit
		if filter.Limit > 0 && filter.Limit < len(results) {
			results = results[:filter.Limit]
		}
	}

	return results, nil
}

// matchesFilter checks if a record matches the filter group
func (f *FileStore) matchesFilter(record Entry, group FilterGroup) bool {
	// Default to AND if no operator specified
	operator := group.Operator
	if operator == "" {
		operator = OpAnd
	}

	// If no conditions or groups, return true
	if len(group.Conditions) == 0 && len(group.Groups) == 0 {
		return true
	}

	// Check conditions
	conditionResults := make([]bool, 0, len(group.Conditions))
	for _, condition := range group.Conditions {
		conditionResults = append(conditionResults, f.matchesCondition(record, condition))
	}

	// Check groups
	groupResults := make([]bool, 0, len(group.Groups))
	for _, subgroup := range group.Groups {
		groupResults = append(groupResults, f.matchesFilter(record, subgroup))
	}

	// Combine all results based on operator
	allResults := append(conditionResults, groupResults...)
	switch operator {
	case OpAnd:
		// All must be true
		for _, result := range allResults {
			if !result {
				return false
			}
		}
		return len(allResults) > 0
	case OpOr:
		// At least one must be true
		for _, result := range allResults {
			if result {
				return true
			}
		}
		return false
	case OpNot:
		// Negate the result
		if len(allResults) == 1 {
			return !allResults[0]
		}
		// If multiple conditions, NOT means none should match (equivalent to AND NOT)
		for _, result := range allResults {
			if result {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// matchesCondition checks if a record matches a specific condition
func (f *FileStore) matchesCondition(record Entry, condition Condition) bool {
	// Special handling for metadata
	if condition.Field == "Metadata" {
		return f.matchesMetadata(record.Metadata, condition)
	}

	// Special handling for time fields with direct access (more accurate than reflection)
	switch condition.Field {
	case "CreatedAt":
		if valueTime, ok := condition.Value.(time.Time); ok {
			return f.compareValues(record.CreatedAt, condition.Operator, valueTime)
		}
	case "UpdatedAt":
		if valueTime, ok := condition.Value.(time.Time); ok {
			return f.compareValues(record.UpdatedAt, condition.Operator, valueTime)
		}
	case "ExpiresAt":
		if valueTime, ok := condition.Value.(time.Time); ok {
			return f.compareValues(record.ExpiresAt, condition.Operator, valueTime)
		}
	case "Content":
		// Special handling for Content which is []byte
		if valueBytes, ok := condition.Value.([]byte); ok {
			return f.compareValues(record.Content, condition.Operator, valueBytes)
		} else if valueStr, ok := condition.Value.(string); ok {
			return f.compareValues(record.Content, condition.Operator, []byte(valueStr))
		}
	case "Tags":
		// Direct handling for Tags which is []string
		if condition.Operator == "CONTAINS" {
			if valueStr, ok := condition.Value.(string); ok {
				for _, tag := range record.Tags {
					if tag == valueStr {
						return true
					}
				}
				return false
			}
		}
	}

	// Get field value with reflection for other fields
	rValue := reflect.ValueOf(record)
	field := rValue.FieldByName(condition.Field)

	// Check if field exists
	if !field.IsValid() {
		return false
	}

	// Special handling for slice fields (References, SubjectIDs)
	if field.Kind() == reflect.Slice && condition.Operator == "CONTAINS" {
		return f.matchesSlice(field, condition)
	}

	// Compare field value with condition value
	return f.compareValues(field.Interface(), condition.Operator, condition.Value)
}

// matchesMetadata checks if metadata matches a condition
func (f *FileStore) matchesMetadata(metadata map[string]string, condition Condition) bool {
	// Handle nil or empty metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// Handle various operators
	switch condition.Operator {
	case "=":
		// Check if metadata contains all key-value pairs from condition
		if condMap, ok := condition.Value.(map[string]string); ok {
			// Empty condition map matches any metadata (including empty)
			if len(condMap) == 0 {
				return len(metadata) == 0
			}

			// Check if all keys in condMap exist in metadata with matching values
			for k, v := range condMap {
				metaVal, exists := metadata[k]
				if !exists || metaVal != v {
					return false
				}
			}
			return true
		}
		return false

	case "!=":
		// Check if metadata doesn't match all key-value pairs from condition
		if condMap, ok := condition.Value.(map[string]string); ok {
			// Empty condition map matches any non-empty metadata
			if len(condMap) == 0 {
				return len(metadata) > 0
			}

			// Check if any key in condMap doesn't exist in metadata or has a different value
			for k, v := range condMap {
				metaVal, exists := metadata[k]
				if !exists || metaVal != v {
					return true
				}
			}
			return false
		}
		return true // If condition value is not a map, it can't match metadata

	case "CONTAINS":
		// Check if metadata contains the key-value pair
		if condMap, ok := condition.Value.(map[string]string); ok && len(condMap) > 0 {
			for k, v := range condMap {
				metaVal, exists := metadata[k]
				if exists && metaVal == v {
					return true // Found at least one matching key-value pair
				}
			}
			return false // No matching key-value pairs found
		}
		// If single key provided as string, check if it exists
		if keyStr, ok := condition.Value.(string); ok {
			_, exists := metadata[keyStr]
			return exists
		}
		return false

	default:
		return false // Unsupported operator for metadata
	}
}

// matchesSlice checks if a slice field matches a condition
func (f *FileStore) matchesSlice(field reflect.Value, condition Condition) bool {
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
func (f *FileStore) compareValues(fieldValue interface{}, operator string, conditionValue interface{}) bool {
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
			return false // Time values don't support CONTAINS
		default:
			return false
		}
	}

	// Special handling for byte slice comparisons
	fieldBytes, fieldIsBytes := fieldValue.([]byte)
	valueBytes, valueIsBytes := conditionValue.([]byte)
	if fieldIsBytes && valueIsBytes {
		switch operator {
		case "=":
			return bytes.Equal(fieldBytes, valueBytes)
		case "!=":
			return !bytes.Equal(fieldBytes, valueBytes)
		case "CONTAINS":
			return bytes.Contains(fieldBytes, valueBytes)
		default:
			// For other operators, compare the string representation
			fieldStr := string(fieldBytes)
			valueStr := string(valueBytes)
			return f.compareStringValues(fieldStr, operator, valueStr)
		}
	}

	// If field is []byte but condition is string
	if fieldIsBytes {
		valueStr, valueIsStr := conditionValue.(string)
		if valueIsStr {
			switch operator {
			case "=":
				return string(fieldBytes) == valueStr
			case "!=":
				return string(fieldBytes) != valueStr
			case "CONTAINS":
				return bytes.Contains(fieldBytes, []byte(valueStr))
			default:
				// For other operators, compare the string representation
				fieldStr := string(fieldBytes)
				return f.compareStringValues(fieldStr, operator, valueStr)
			}
		}
	}

	// Handle string comparisons
	fieldStr, fieldIsStr := fieldValue.(string)
	valueStr, valueIsStr := conditionValue.(string)
	if fieldIsStr && valueIsStr {
		return f.compareStringValues(fieldStr, operator, valueStr)
	}

	// Default comparison using string representation for all other types
	fieldStrRepr := fmt.Sprintf("%v", fieldValue)
	valueStrRepr := fmt.Sprintf("%v", conditionValue)
	return f.compareStringValues(fieldStrRepr, operator, valueStrRepr)
}

// compareStringValues compares two string values based on the operator
func (f *FileStore) compareStringValues(fieldStr, operator, valueStr string) bool {
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
func (f *FileStore) sortRecords(records []Entry, orderBy, orderDir string) {
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
func (f *FileStore) LoadRecords(records ...Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

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
		_, exists := f.records[record.ID]

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
		f.records[record.ID] = record
	}

	// Mark the store as dirty since we've modified records
	f.isDirty = true

	return nil
}

// Info provides implementation-specific information about the file knowledge store
// This method is required by the Store interface
func (f *FileStore) Info() (map[string]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	info := make(map[string]string)

	// Add basic implementation info
	info["implementation"] = "FileStore"
	info["file_path"] = f.filename
	info["file_name"] = filepath.Base(f.filename)
	info["record_count"] = fmt.Sprintf("%d", len(f.records))
	info["deleted_count"] = fmt.Sprintf("%d", len(f.deletedRecs))
	info["is_dirty"] = fmt.Sprintf("%t", f.isDirty)

	return info, nil
}
