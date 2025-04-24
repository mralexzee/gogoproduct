package knowledge

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewMemoryStore(t *testing.T) {
	store, err := NewMemoryStore()
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	if store == nil {
		t.Fatal("Store should not be nil")
	}

	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}

	info, err := store.Info()
	if err != nil {
		t.Fatalf("Failed to get store info: %v", err)
	}

	if info["implementation"] != "MemoryStore" {
		t.Errorf("Expected implementation to be MemoryStore, got %s", info["implementation"])
	}
	if info["persistent"] != "false" {
		t.Errorf("Expected persistent to be false, got %s", info["persistent"])
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
}

func TestMemoryStore_AddRecord(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Test all field combinations
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	// Create a record with all fields populated
	record := Entry{
		ID:          "test-record-1",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Test content"),
		Importance:  ImportanceHigh,
		CreatedAt:   past,
		UpdatedAt:   now,
		ExpiresAt:   future,
		SourceID:    "test-source",
		SourceType:  "test",
		OwnerID:     "test-owner",
		OwnerType:   "user",
		SubjectIDs:  []string{"subject1", "subject2"},
		SubjectType: "project",
		Tags:        []string{"test", "knowledge"},
		References: []Reference{
			{ID: "ref1", Type: "fact"},
			{ID: "ref2", Type: "decision"},
		},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	if err := store.AddRecord(record); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Try to add the same record again
	if err := store.AddRecord(record); err == nil {
		t.Error("Should not be able to add the same record twice")
	}

	// Test with empty ID
	invalidRecord := record
	invalidRecord.ID = ""
	if err := store.AddRecord(invalidRecord); err == nil {
		t.Error("Should not be able to add record with empty ID")
	}

	// Test with different categories
	categories := []string{CategoryFact, CategoryMessage, CategoryDecision, CategoryAction}
	for _, category := range categories {
		rec := record
		rec.ID = "category-test-" + category
		rec.Category = category
		if err := store.AddRecord(rec); err != nil {
			t.Errorf("Failed to add record with category %s: %v", category, err)
		}
	}

	// Test with different content types
	contentTypes := []string{
		ContentTypeJSON, ContentTypeText, ContentTypeYAML, ContentTypeXML,
		ContentTypeCSV, ContentTypeBinary, ContentTypeHTML, ContentTypeMarkdown,
	}
	for _, contentType := range contentTypes {
		rec := record
		rec.ID = "content-type-test-" + contentType
		rec.ContentType = contentType
		if err := store.AddRecord(rec); err != nil {
			t.Errorf("Failed to add record with content type %s: %v", contentType, err)
		}
	}

	// Test with different importance levels
	importanceLevels := []int{
		ImportanceNone, ImportanceLow, ImportanceMedium, ImportanceHigh, ImportanceCritical,
	}
	for _, importance := range importanceLevels {
		rec := record
		rec.ID = "importance-test-" + string(rune(importance))
		rec.Importance = importance
		if err := store.AddRecord(rec); err != nil {
			t.Errorf("Failed to add record with importance %d: %v", importance, err)
		}
	}

	// Test timestamps
	timestampTests := []struct {
		name      string
		createdAt time.Time
		updatedAt time.Time
		expiresAt time.Time
	}{
		{"zero-timestamps", time.Time{}, time.Time{}, time.Time{}},
		{"past-timestamps", past, past, past},
		{"future-timestamps", future, future, future},
		{"mixed-timestamps", past, now, future},
	}

	for _, tt := range timestampTests {
		rec := record
		rec.ID = "timestamp-test-" + tt.name
		rec.CreatedAt = tt.createdAt
		rec.UpdatedAt = tt.updatedAt
		rec.ExpiresAt = tt.expiresAt
		if err := store.AddRecord(rec); err != nil {
			t.Errorf("Failed to add record with %s: %v", tt.name, err)
		}

		// Check if timestamps were properly initialized
		retrieved, err := store.GetRecord(rec.ID)
		if err != nil {
			t.Errorf("Failed to get record with %s: %v", tt.name, err)
			continue
		}

		if tt.createdAt.IsZero() && retrieved.CreatedAt.IsZero() {
			t.Errorf("CreatedAt should be automatically set when zero")
		}
		if tt.updatedAt.IsZero() && retrieved.UpdatedAt.IsZero() {
			t.Errorf("UpdatedAt should be automatically set when zero")
		}
	}
}

func TestMemoryStore_GetRecord(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create a test record
	record := Entry{
		ID:          "get-test",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Test content for get"),
		Importance:  ImportanceHigh,
		SourceID:    "test-source",
		SourceType:  "test",
		OwnerID:     "test-owner",
		OwnerType:   "user",
		SubjectIDs:  []string{"subject1", "subject2"},
		SubjectType: "project",
		Tags:        []string{"test", "get", "knowledge"},
		References: []Reference{
			{ID: "ref1", Type: "fact"},
		},
		Metadata: map[string]string{
			"key1": "value1",
		},
	}

	if err := store.AddRecord(record); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Get the record
	retrieved, err := store.GetRecord(record.ID)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}

	// Verify all fields match
	if !reflect.DeepEqual(record.ID, retrieved.ID) {
		t.Errorf("ID mismatch: expected %s, got %s", record.ID, retrieved.ID)
	}
	if !reflect.DeepEqual(record.Category, retrieved.Category) {
		t.Errorf("Category mismatch: expected %s, got %s", record.Category, retrieved.Category)
	}
	if !reflect.DeepEqual(record.ContentType, retrieved.ContentType) {
		t.Errorf("ContentType mismatch: expected %s, got %s", record.ContentType, retrieved.ContentType)
	}
	if !bytes.Equal(record.Content, retrieved.Content) {
		t.Errorf("Content mismatch: expected %s, got %s", record.Content, retrieved.Content)
	}
	if !reflect.DeepEqual(record.Importance, retrieved.Importance) {
		t.Errorf("Importance mismatch: expected %d, got %d", record.Importance, retrieved.Importance)
	}
	if !reflect.DeepEqual(record.SourceID, retrieved.SourceID) {
		t.Errorf("SourceID mismatch: expected %s, got %s", record.SourceID, retrieved.SourceID)
	}
	if !reflect.DeepEqual(record.SourceType, retrieved.SourceType) {
		t.Errorf("SourceType mismatch: expected %s, got %s", record.SourceType, retrieved.SourceType)
	}
	if !reflect.DeepEqual(record.OwnerID, retrieved.OwnerID) {
		t.Errorf("OwnerID mismatch: expected %s, got %s", record.OwnerID, retrieved.OwnerID)
	}
	if !reflect.DeepEqual(record.OwnerType, retrieved.OwnerType) {
		t.Errorf("OwnerType mismatch: expected %s, got %s", record.OwnerType, retrieved.OwnerType)
	}
	if !reflect.DeepEqual(record.SubjectIDs, retrieved.SubjectIDs) {
		t.Errorf("SubjectIDs mismatch: expected %v, got %v", record.SubjectIDs, retrieved.SubjectIDs)
	}
	if !reflect.DeepEqual(record.SubjectType, retrieved.SubjectType) {
		t.Errorf("SubjectType mismatch: expected %s, got %s", record.SubjectType, retrieved.SubjectType)
	}
	if !reflect.DeepEqual(record.Tags, retrieved.Tags) {
		t.Errorf("Tags mismatch: expected %v, got %v", record.Tags, retrieved.Tags)
	}
	if !reflect.DeepEqual(record.References, retrieved.References) {
		t.Errorf("References mismatch: expected %v, got %v", record.References, retrieved.References)
	}
	if !reflect.DeepEqual(record.Metadata, retrieved.Metadata) {
		t.Errorf("Metadata mismatch: expected %v, got %v", record.Metadata, retrieved.Metadata)
	}

	// Try to get a non-existent record
	_, err = store.GetRecord("non-existent")
	if err == nil {
		t.Error("Should not be able to get a non-existent record")
	}
}

func TestMemoryStore_UpdateRecord(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create a test record
	originalRecord := Entry{
		ID:          "update-test",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Original content"),
		Importance:  ImportanceMedium,
		Tags:        []string{"original", "record"},
		Metadata: map[string]string{
			"original": "metadata",
		},
	}

	if err := store.AddRecord(originalRecord); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update the record
	updatedRecord := originalRecord
	updatedRecord.Category = CategoryDecision
	updatedRecord.ContentType = ContentTypeJSON
	updatedRecord.Content = []byte(`{"updated": true}`)
	updatedRecord.Importance = ImportanceHigh
	updatedRecord.Tags = []string{"updated", "record"}
	updatedRecord.Metadata = map[string]string{
		"updated": "metadata",
	}

	if err := store.UpdateRecord(updatedRecord); err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Get the updated record
	retrieved, err := store.GetRecord(updatedRecord.ID)
	if err != nil {
		t.Fatalf("Failed to get updated record: %v", err)
	}

	// Verify fields were updated
	if retrieved.Category != CategoryDecision {
		t.Errorf("Category not updated: expected %s, got %s", CategoryDecision, retrieved.Category)
	}
	if retrieved.ContentType != ContentTypeJSON {
		t.Errorf("ContentType not updated: expected %s, got %s", ContentTypeJSON, retrieved.ContentType)
	}
	if !bytes.Equal(retrieved.Content, []byte(`{"updated": true}`)) {
		t.Errorf("Content not updated: expected %s, got %s", []byte(`{"updated": true}`), retrieved.Content)
	}
	if retrieved.Importance != ImportanceHigh {
		t.Errorf("Importance not updated: expected %d, got %d", ImportanceHigh, retrieved.Importance)
	}
	if !reflect.DeepEqual(retrieved.Tags, []string{"updated", "record"}) {
		t.Errorf("Tags not updated: expected %v, got %v", []string{"updated", "record"}, retrieved.Tags)
	}
	if !reflect.DeepEqual(retrieved.Metadata, map[string]string{"updated": "metadata"}) {
		t.Errorf("Metadata not updated: expected %v, got %v", map[string]string{"updated": "metadata"}, retrieved.Metadata)
	}

	// Try to update a non-existent record
	nonExistentRecord := originalRecord
	nonExistentRecord.ID = "non-existent"
	if err := store.UpdateRecord(nonExistentRecord); err == nil {
		t.Error("Should not be able to update a non-existent record")
	}
}

func TestMemoryStore_DeleteRestorePurgeRecord(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create test records
	recordIDs := []string{"delete-test-1", "delete-test-2", "delete-test-3"}
	for _, id := range recordIDs {
		record := Entry{
			ID:          id,
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte("Content for " + id),
		}
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", id, err)
		}
	}

	// Test deletion
	if err := store.DeleteRecord(recordIDs[0]); err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify record is deleted
	_, err := store.GetRecord(recordIDs[0])
	if err == nil {
		t.Error("Should not be able to get a deleted record")
	}

	// Try to delete a non-existent record
	if err := store.DeleteRecord("non-existent"); err == nil {
		t.Error("Should not be able to delete a non-existent record")
	}

	// Try to delete an already deleted record
	if err := store.DeleteRecord(recordIDs[0]); err == nil {
		t.Error("Should not be able to delete an already deleted record")
	}

	// Test restoration
	if err := store.RestoreRecord(recordIDs[0]); err != nil {
		t.Fatalf("Failed to restore record: %v", err)
	}

	// Verify record is restored and can be retrieved
	restored, err := store.GetRecord(recordIDs[0])
	if err != nil {
		t.Fatalf("Failed to get restored record: %v", err)
	}
	if restored.ID != recordIDs[0] {
		t.Errorf("Restored record ID mismatch: expected %s, got %s", recordIDs[0], restored.ID)
	}

	// Try to restore a non-existent record
	if err := store.RestoreRecord("non-existent"); err == nil {
		t.Error("Should not be able to restore a non-existent record")
	}

	// Try to restore a non-deleted record
	if err := store.RestoreRecord(recordIDs[1]); err == nil {
		t.Error("Should not be able to restore a non-deleted record")
	}

	// Test permanent deletion (purge)
	// First delete the record
	if err := store.DeleteRecord(recordIDs[1]); err != nil {
		t.Fatalf("Failed to delete record before purge: %v", err)
	}

	// Then purge it
	if err := store.PurgeRecord(recordIDs[1]); err != nil {
		t.Fatalf("Failed to purge record: %v", err)
	}

	// Verify record is purged and cannot be restored
	if err := store.RestoreRecord(recordIDs[1]); err == nil {
		t.Error("Should not be able to restore a purged record")
	}

	// Try to purge a non-existent record
	if err := store.PurgeRecord("non-existent"); err == nil {
		t.Error("Should not be able to purge a non-existent record")
	}

	// Purge a non-deleted record
	if err := store.PurgeRecord(recordIDs[2]); err != nil {
		t.Errorf("Should be able to purge a non-deleted record: %v", err)
	}

	// Verify record is purged
	_, err = store.GetRecord(recordIDs[2])
	if err == nil {
		t.Error("Should not be able to get a purged record")
	}
}

func TestMemoryStore_SearchRecords(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create a set of test records
	records := []Entry{
		{
			ID:         "search-1",
			Category:   CategoryFact,
			Importance: ImportanceHigh,
			Tags:       []string{"search", "high"},
			Metadata: map[string]string{
				"searchable": "true",
				"type":       "fact",
			},
		},
		{
			ID:         "search-2",
			Category:   CategoryDecision,
			Importance: ImportanceMedium,
			Tags:       []string{"search", "medium"},
			Metadata: map[string]string{
				"searchable": "true",
				"type":       "decision",
			},
		},
		{
			ID:         "search-3",
			Category:   CategoryAction,
			Importance: ImportanceLow,
			Tags:       []string{"search", "low"},
			Metadata: map[string]string{
				"searchable": "false",
				"type":       "action",
			},
		},
		{
			ID:         "nosearch-1",
			Category:   CategoryMessage,
			Importance: ImportanceNone,
			Tags:       []string{"nosearch"},
			Metadata: map[string]string{
				"searchable": "false",
				"type":       "message",
			},
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Test 1: Simple search with no filters
	results, err := store.SearchRecords(Filter{})
	if err != nil {
		t.Fatalf("Failed to search records: %v", err)
	}
	if len(results) != len(records) {
		t.Errorf("Expected %d results, got %d", len(records), len(results))
	}

	// Test 2: Filter by Category with AND operator
	categoryFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Category", Operator: "=", Value: CategoryFact},
			},
		},
	}
	results, err = store.SearchRecords(categoryFilter)
	if err != nil {
		t.Fatalf("Failed to search records by category: %v", err)
	}
	if len(results) != 1 || results[0].ID != "search-1" {
		t.Errorf("Expected 1 result with ID 'search-1', got %d results", len(results))
	}

	// Test 3: Filter by Importance with greater than operator
	importanceFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Importance", Operator: ">", Value: ImportanceLow},
			},
		},
	}
	results, err = store.SearchRecords(importanceFilter)
	if err != nil {
		t.Fatalf("Failed to search records by importance: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Importance <= ImportanceLow {
			t.Errorf("Result %s has importance %d, which is not > %d", result.ID, result.Importance, ImportanceLow)
		}
	}

	// Test 4: Filter by Tags with CONTAINS operator
	tagsFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Tags", Operator: "CONTAINS", Value: "medium"},
			},
		},
	}
	results, err = store.SearchRecords(tagsFilter)
	if err != nil {
		t.Fatalf("Failed to search records by tags: %v", err)
	}
	if len(results) != 1 || results[0].ID != "search-2" {
		t.Errorf("Expected 1 result with ID 'search-2', got %d results", len(results))
	}

	// Test 5: Filter by Metadata with complex condition
	metadataFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Metadata", Operator: "=", Value: map[string]interface{}{"searchable": "true"}},
			},
		},
	}
	results, err = store.SearchRecords(metadataFilter)
	if err != nil {
		t.Fatalf("Failed to search records by metadata: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if result.Metadata["searchable"] != "true" {
			t.Errorf("Result %s has metadata['searchable'] = %s, expected 'true'", result.ID, result.Metadata["searchable"])
		}
	}

	// Test 6: Complex filter with nested groups and OR+AND operators
	complexFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Tags", Operator: "CONTAINS", Value: "search"},
			},
			Groups: []FilterGroup{
				{
					Operator: OpOr,
					Conditions: []Condition{
						{Field: "Category", Operator: "=", Value: CategoryFact},
						{Field: "Importance", Operator: "=", Value: ImportanceLow},
					},
				},
			},
		},
	}
	results, err = store.SearchRecords(complexFilter)
	if err != nil {
		t.Fatalf("Failed to search records with complex filter: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	resultIDs := make(map[string]bool)
	for _, result := range results {
		resultIDs[result.ID] = true
	}
	if !resultIDs["search-1"] || !resultIDs["search-3"] {
		t.Errorf("Expected results to include 'search-1' and 'search-3', got %v", resultIDs)
	}

	// Test 7: Sorting, limit, and offset
	sortedFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Tags", Operator: "CONTAINS", Value: "search"},
			},
		},
		OrderBy:  "Importance",
		OrderDir: "DESC", // High to low
		Limit:    2,
		Offset:   0,
	}
	results, err = store.SearchRecords(sortedFilter)
	if err != nil {
		t.Fatalf("Failed to search sorted records: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(results))
	}
	if results[0].Importance < results[1].Importance {
		t.Errorf("Expected results sorted by Importance DESC, got %d before %d",
			results[0].Importance, results[1].Importance)
	}

	// Test 8: Test with offset
	offsetFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Tags", Operator: "CONTAINS", Value: "search"},
			},
		},
		OrderBy:  "Importance",
		OrderDir: "DESC",
		Limit:    2,
		Offset:   1, // Skip the highest importance
	}
	results, err = store.SearchRecords(offsetFilter)
	if err != nil {
		t.Fatalf("Failed to search records with offset: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results with offset, got %d", len(results))
	}
	if results[0].ID == "search-1" { // search-1 has highest importance and should be skipped
		t.Errorf("Expected highest importance record to be skipped with offset, but got it")
	}

	// Test 9: Filter with NOT operator
	notFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpNot,
			Conditions: []Condition{
				{Field: "Category", Operator: "=", Value: CategoryFact},
			},
		},
	}
	results, err = store.SearchRecords(notFilter)
	if err != nil {
		t.Fatalf("Failed to search records with NOT operator: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("Expected 3 results with NOT operator, got %d", len(results))
	}
	for _, result := range results {
		if result.Category == CategoryFact {
			t.Errorf("Result %s has Category %s which should have been excluded", result.ID, result.Category)
		}
	}

	// Test 10: Deleted records handling
	// Delete a record first
	if err := store.DeleteRecord("search-1"); err != nil {
		t.Fatalf("Failed to delete record for test: %v", err)
	}

	// Default filter should exclude deleted
	defaultFilter := Filter{}
	results, err = store.SearchRecords(defaultFilter)
	if err != nil {
		t.Fatalf("Failed to search with default filter: %v", err)
	}
	if len(results) != len(records)-1 {
		t.Errorf("Expected %d results (excluding deleted), got %d", len(records)-1, len(results))
	}
	for _, result := range results {
		if result.ID == "search-1" {
			t.Errorf("Result includes deleted record 'search-1'")
		}
	}

	// Include deleted records
	includeDeletedFilter := Filter{
		IncludeDeleted: true,
	}
	results, err = store.SearchRecords(includeDeletedFilter)
	if err != nil {
		t.Fatalf("Failed to search with includeDeleted: %v", err)
	}
	if len(results) != len(records) {
		t.Errorf("Expected %d results (including deleted), got %d", len(records), len(results))
	}
	foundDeleted := false
	for _, result := range results {
		if result.ID == "search-1" {
			foundDeleted = true
			break
		}
	}
	if !foundDeleted {
		t.Errorf("Results should include deleted record 'search-1' but did not")
	}

	// Only deleted records
	onlyDeletedFilter := Filter{
		OnlyDeleted: true,
	}
	results, err = store.SearchRecords(onlyDeletedFilter)
	if err != nil {
		t.Fatalf("Failed to search with onlyDeleted: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result (only deleted), got %d", len(results))
	}
	if len(results) > 0 && results[0].ID != "search-1" {
		t.Errorf("Expected only deleted record 'search-1', got '%s'", results[0].ID)
	}
}

func TestMemoryStore_Flush(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Flush should be a no-op for memory store
	if err := store.Flush(); err != nil {
		t.Errorf("Flush should be a no-op but returned error: %v", err)
	}
}

// TestMemoryStoreConcurrency tests concurrent access to the memory store
func TestMemoryStoreConcurrency(t *testing.T) {
	store, _ := NewMemoryStore()
	if err := store.Open(); err != nil {
		t.Fatalf("Failed to open memory store: %v", err)
	}
	defer store.Close()

	// Number of concurrent operations
	const numOps = 100
	// Number of goroutines
	const numGoroutines = 10

	wg := sync.WaitGroup{}
	// Channel to collect errors
	errorCh := make(chan error, numOps*numGoroutines)

	// Launch multiple goroutines to perform concurrent operations
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < numOps; i++ {
				// Create unique ID for each operation
				id := fmt.Sprintf("concurrent-%d-%d", goroutineID, i)

				// Create a record
				record := Entry{
					ID:          id,
					Category:    CategoryFact,
					ContentType: ContentTypeText,
					Content:     []byte(fmt.Sprintf("Concurrent test content %d-%d", goroutineID, i)),
					Importance:  ImportanceMedium,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					OwnerID:     "test-owner",
					OwnerType:   "human",
				}

				// Randomly select an operation: add, get, update, or delete
				op := i % 4

				switch op {
				case 0: // Add
					if err := store.AddRecord(record); err != nil {
						errorCh <- fmt.Errorf("failed to add record %s: %v", id, err)
					}
				case 1: // Get
					// Add first then get
					if err := store.AddRecord(record); err != nil {
						errorCh <- fmt.Errorf("failed to add record before get %s: %v", id, err)
						continue
					}
					_, err := store.GetRecord(id)
					if err != nil {
						errorCh <- fmt.Errorf("failed to get record %s: %v", id, err)
					}
				case 2: // Update
					// Add first then update
					if err := store.AddRecord(record); err != nil {
						errorCh <- fmt.Errorf("failed to add record before update %s: %v", id, err)
						continue
					}
					record.Content = []byte(fmt.Sprintf("Updated concurrent content %d-%d", goroutineID, i))
					if err := store.UpdateRecord(record); err != nil {
						errorCh <- fmt.Errorf("failed to update record %s: %v", id, err)
					}
				case 3: // Delete
					// Add first then delete
					if err := store.AddRecord(record); err != nil {
						errorCh <- fmt.Errorf("failed to add record before delete %s: %v", id, err)
						continue
					}
					if err := store.DeleteRecord(id); err != nil {
						errorCh <- fmt.Errorf("failed to delete record %s: %v", id, err)
					}
				}
			}
		}(g)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorCh)

	// Collect any errors
	errors := []error{}
	for err := range errorCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("Concurrent error: %v", err)
		}
		t.Fatalf("Encountered %d errors during concurrent operations", len(errors))
	}

	// Final verification - make sure store is still operational
	filter := Filter{
		RootGroup: FilterGroup{
			Operator:   OpAnd,
			Conditions: []Condition{},
		},
	}

	_, err := store.SearchRecords(filter)
	if err != nil {
		t.Fatalf("Store is not operational after concurrent testing: %v", err)
	}
}

// TestMemoryStore_TimeRangeFiltering tests filtering records by time ranges
func TestMemoryStore_TimeRangeFiltering(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create baseline time points
	now := time.Now()
	pastHour := now.Add(-1 * time.Hour)
	past24Hours := now.Add(-24 * time.Hour)
	past48Hours := now.Add(-48 * time.Hour)
	// Not using futureHour in this test
	future24Hours := now.Add(24 * time.Hour)
	future48Hours := now.Add(48 * time.Hour)

	// Create test records with different timestamps
	records := []Entry{
		{ // Very old record, expired
			ID:         "time-past-expired",
			Category:   CategoryFact,
			Content:    []byte("Old content that expired"),
			CreatedAt:  past48Hours,
			UpdatedAt:  past48Hours,
			ExpiresAt:  past24Hours, // Already expired
			Importance: ImportanceLow,
		},
		{ // Old record, not expired
			ID:         "time-past-valid",
			Category:   CategoryFact,
			Content:    []byte("Old content still valid"),
			CreatedAt:  past48Hours,
			UpdatedAt:  pastHour,      // Updated recently
			ExpiresAt:  future24Hours, // Not expired yet
			Importance: ImportanceMedium,
		},
		{ // New record
			ID:         "time-recent",
			Category:   CategoryFact,
			Content:    []byte("Recent content"),
			CreatedAt:  pastHour,
			UpdatedAt:  now,
			ExpiresAt:  future48Hours,
			Importance: ImportanceHigh,
		},
		{ // Future record (e.g., scheduled)
			ID:         "time-future",
			Category:   CategoryAction,
			Content:    []byte("Future scheduled action"),
			CreatedAt:  now,
			UpdatedAt:  now,
			ExpiresAt:  future48Hours,
			Importance: ImportanceHigh,
		},
	}

	// Add records to the store
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Test 1: Filter by CreatedAt > past24Hours
	createdAfterFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "CreatedAt", Operator: ">", Value: past24Hours},
			},
		},
	}

	results, err := store.SearchRecords(createdAfterFilter)
	if err != nil {
		t.Fatalf("Failed to search with CreatedAt filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records created after %v, got %d", past24Hours, len(results))
	}

	// Verify results contain only the expected records
	found := make(map[string]bool)
	for _, r := range results {
		found[r.ID] = true
		if r.CreatedAt.Before(past24Hours) || r.CreatedAt.Equal(past24Hours) {
			t.Errorf("Record %s has CreatedAt %v which is not after %v", r.ID, r.CreatedAt, past24Hours)
		}
	}

	expectedIDs := []string{"time-recent", "time-future"}
	for _, id := range expectedIDs {
		if !found[id] {
			t.Errorf("Expected to find record %s but it was missing from results", id)
		}
	}

	// Test 2: Filter by UpdatedAt between pastHour and now
	updatedRangeFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "UpdatedAt", Operator: ">", Value: pastHour},
				{Field: "UpdatedAt", Operator: "<=", Value: now.Add(time.Second)}, // Adding a second for test reliability
			},
		},
	}

	results, err = store.SearchRecords(updatedRangeFilter)
	if err != nil {
		t.Fatalf("Failed to search with UpdatedAt range filter: %v", err)
	}

	// Expecting time-recent and time-future which were updated between pastHour and now
	if len(results) != 2 {
		t.Errorf("Expected 2 records updated between %v and %v, got %d", pastHour, now, len(results))
	}

	// Test 3: Filter by not yet expired records (ExpiresAt > now)
	notExpiredFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "ExpiresAt", Operator: ">", Value: now},
			},
		},
	}

	results, err = store.SearchRecords(notExpiredFilter)
	if err != nil {
		t.Fatalf("Failed to search for non-expired records: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 non-expired records, got %d", len(results))
	}

	// Should not contain the expired record
	for _, r := range results {
		if r.ID == "time-past-expired" {
			t.Errorf("Results should not include expired record 'time-past-expired'")
		}
	}

	// Test 4: Complex time filter - recently updated old records (created before 24h, updated after 1h)
	complexTimeFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "CreatedAt", Operator: "<", Value: past24Hours},
				{Field: "UpdatedAt", Operator: ">=", Value: pastHour},
			},
		},
	}

	// Log the filter and data for debugging
	t.Logf("Complex filter looking for: CreatedAt < %v AND UpdatedAt > %v", past24Hours, pastHour)

	// Log all record times for debugging
	for _, rec := range records {
		t.Logf("Record %s: CreatedAt=%v, UpdatedAt=%v", rec.ID, rec.CreatedAt, rec.UpdatedAt)
	}

	results, err = store.SearchRecords(complexTimeFilter)
	if err != nil {
		t.Fatalf("Failed to search with complex time filter: %v", err)
	}

	// Should only return the "time-past-valid" record
	if len(results) != 1 || (len(results) > 0 && results[0].ID != "time-past-valid") {
		if len(results) > 0 {
			t.Errorf("Expected only 'time-past-valid' for complex time filter, got %d records with IDs: %s",
				len(results), results[0].ID)
		} else {
			t.Errorf("Expected only 'time-past-valid' for complex time filter, got %d records", len(results))
		}
	}
}

// TestMemoryStore_SpecialCharacters tests handling of special characters in various fields
func TestMemoryStore_SpecialCharacters(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create records with special characters in different fields
	records := []Entry{
		{
			ID:         "special-id-1@#$%^&*()[]{}",
			Category:   CategoryFact,
			Content:    []byte("Content with special chars: !@#$%^&*()"),
			Importance: ImportanceMedium,
			Tags:       []string{"tag with spaces", "tag,with,commas", "tag+with+plus"},
			Metadata: map[string]string{
				"key with spaces":  "value with spaces",
				"key=with=equals":  "value=with=equals",
				"key/with/slashes": "value/with/slashes",
			},
		},
		{
			ID:         "special-id-2<>?:\"'|;",
			Category:   CategoryDecision,
			Content:    []byte(`Content with quotes: "quoted text" and 'single quoted text'`),
			Importance: ImportanceLow,
			OwnerID:    "owner with spaces and (parentheses)",
			SourceID:   "source/with/slashes",
		},
		{
			ID:          "special-id-3😀🙏👍",
			Category:    CategoryMessage,
			Content:     []byte("Content with emoji: 😀🙏👍"),
			Importance:  ImportanceHigh,
			SubjectIDs:  []string{"subject/with/slashes", "subject.with.dots"},
			SubjectType: "type with spaces and symbols: !@#",
		},
		{
			ID:         "special-id-4%20%22%3C%3E",
			Category:   CategoryAction,
			Content:    []byte("Content with URL encoding: %20%22%3C%3E"),
			Importance: ImportanceHigh,
			SourceType: "type-with-unicode-Привет-你好",
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record with special chars %s: %v", record.ID, err)
		}
	}

	// Test 1: Retrieve each record by its special ID
	for _, expected := range records {
		actual, err := store.GetRecord(expected.ID)
		if err != nil {
			t.Errorf("Failed to get record with special ID %s: %v", expected.ID, err)
			continue
		}

		if !bytes.Equal(actual.Content, expected.Content) {
			t.Errorf("Content mismatch for record %s: got %s, want %s",
				expected.ID, string(actual.Content), string(expected.Content))
		}

		// Check metadata for the first record which has special chars in metadata
		if expected.ID == "special-id-1@#$%^&*()[]{}" {
			for k, expectedValue := range expected.Metadata {
				if actualValue, ok := actual.Metadata[k]; !ok || actualValue != expectedValue {
					t.Errorf("Metadata mismatch for key %s: got %s, want %s",
						k, actualValue, expectedValue)
				}
			}
		}
	}

	// Test 2: Filter by content with special characters
	emojisFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Content", Operator: "CONTAINS", Value: "emoji: 😀"},
			},
		},
	}

	results, err := store.SearchRecords(emojisFilter)
	if err != nil {
		t.Fatalf("Failed to search with emoji filter: %v", err)
	}

	if len(results) != 1 || results[0].ID != "special-id-3😀🙏👍" {
		t.Errorf("Expected 1 result with emoji content, got %d", len(results))
	}

	// Test 3: Filter by tag with spaces
	spaceTagFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Tags", Operator: "CONTAINS", Value: "tag with spaces"},
			},
		},
	}

	results, err = store.SearchRecords(spaceTagFilter)
	if err != nil {
		t.Fatalf("Failed to search with spaces in tags: %v", err)
	}

	if len(results) != 1 || results[0].ID != "special-id-1@#$%^&*()[]{}" {
		t.Errorf("Expected 1 result with 'tag with spaces', got %d", len(results))
	}

	// Test 4: Filter by special SourceType (unicode characters)
	unicodeFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "SourceType", Operator: "CONTAINS", Value: "Привет"},
			},
		},
	}

	results, err = store.SearchRecords(unicodeFilter)
	if err != nil {
		t.Fatalf("Failed to search with unicode filter: %v", err)
	}

	if len(results) != 1 || results[0].ID != "special-id-4%20%22%3C%3E" {
		t.Errorf("Expected 1 result with unicode SourceType, got %d", len(results))
	}

	// Test 5: Update record with special characters
	remoteRecord, err := store.GetRecord("special-id-2<>?:\"'|;")
	if err != nil {
		t.Fatalf("Failed to get record for update: %v", err)
	}

	remoteRecord.Content = []byte(`Updated content with more "quotes" and special chars: $%^&*()[]{}|\/'`)
	remoteRecord.Tags = []string{"new tag with spaces", "another,special,tag"}

	if err := store.UpdateRecord(remoteRecord); err != nil {
		t.Fatalf("Failed to update record with special characters: %v", err)
	}

	// Verify the update worked
	updated, err := store.GetRecord("special-id-2<>?:\"'|;")
	if err != nil {
		t.Fatalf("Failed to get updated record: %v", err)
	}

	if !bytes.Equal(updated.Content, remoteRecord.Content) {
		t.Errorf("Content not updated correctly. Got %s, want %s",
			string(updated.Content), string(remoteRecord.Content))
	}

	if len(updated.Tags) != 2 || updated.Tags[0] != "new tag with spaces" {
		t.Errorf("Tags not updated correctly. Got %v, want %v",
			updated.Tags, remoteRecord.Tags)
	}
}

// TestMemoryStore_Reopening tests closing and reopening the memory store
func TestMemoryStore_Reopening(t *testing.T) {
	// Create and populate a store
	store, _ := NewMemoryStore()
	_ = store.Open()

	// Add a record
	record := Entry{
		ID:          "reopen-test-1",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Test reopening"),
		Importance:  ImportanceMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.AddRecord(record); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close the store: %v", err)
	}

	// Try operations on closed store - should fail
	_, err := store.GetRecord("reopen-test-1")
	if err == nil {
		t.Error("Should not be able to get records from closed store")
	}

	// Reopen the store
	if err := store.Open(); err != nil {
		t.Fatalf("Failed to reopen the store: %v", err)
	}

	// Previous data should be gone as it's an in-memory store
	_, err = store.GetRecord("reopen-test-1")
	if err == nil {
		t.Error("In-memory store should not persist data after close/reopen")
	}

	// Add a new record
	newRecord := Entry{
		ID:          "reopen-test-2",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Test after reopening"),
		Importance:  ImportanceMedium,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := store.AddRecord(newRecord); err != nil {
		t.Fatalf("Failed to add record after reopening: %v", err)
	}

	// Should be able to get the new record
	retrieved, err := store.GetRecord("reopen-test-2")
	if err != nil {
		t.Fatalf("Failed to get record after reopening: %v", err)
	}

	if !bytes.Equal(retrieved.Content, newRecord.Content) {
		t.Errorf("Content mismatch after reopen: got %s, want %s",
			string(retrieved.Content), string(newRecord.Content))
	}

	// Clean up
	_ = store.Close()
}

// TestMemoryStore_DeepFilterNesting tests deeply nested filter conditions
func TestMemoryStore_DeepFilterNesting(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Add test records for deep filter testing
	records := []Entry{
		{
			ID:         "deep-filter-1",
			Category:   CategoryFact,
			SourceType: "system",
			Importance: ImportanceHigh,
			Tags:       []string{"deep", "filter", "test"},
			Metadata: map[string]string{
				"depth": "level1",
				"test":  "true",
			},
		},
		{
			ID:         "deep-filter-2",
			Category:   CategoryAction,
			SourceType: "user",
			Importance: ImportanceMedium,
			Tags:       []string{"deep", "nested", "complex"},
			Metadata: map[string]string{
				"depth": "level2",
				"test":  "true",
			},
		},
		{
			ID:         "deep-filter-3",
			Category:   CategoryMessage,
			SourceType: "system",
			Importance: ImportanceLow,
			Tags:       []string{"simple", "basic"},
			Metadata: map[string]string{
				"depth": "level3",
				"test":  "false",
			},
		},
		{
			ID:         "deep-filter-4",
			Category:   CategoryDecision,
			SourceType: "user",
			Importance: ImportanceHigh,
			Tags:       []string{"complex", "advanced"},
			Metadata: map[string]string{
				"depth": "level2",
				"test":  "false",
			},
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Construct a deeply nested filter structure (5 levels deep)
	// Conceptually: Find records that are:
	// (CategoryFact OR CategoryAction) AND
	// (
	//   (ImportanceHigh AND SourceType="system") OR
	//   (ImportanceMedium AND SourceType="user" AND
	//     (Metadata.depth="level2" AND Tags CONTAINS "complex")
	//   )
	// )
	deepFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Groups: []FilterGroup{
				{ // Group 1: Category is Fact OR Action
					Operator: OpOr,
					Conditions: []Condition{
						{Field: "Category", Operator: "=", Value: CategoryFact},
						{Field: "Category", Operator: "=", Value: CategoryAction},
					},
				},
				{ // Group 2: Complex nested conditions
					Operator: OpOr,
					Groups: []FilterGroup{
						{ // Group 2.1: HighImportance AND SystemSource
							Operator: OpAnd,
							Conditions: []Condition{
								{Field: "Importance", Operator: "=", Value: ImportanceHigh},
								{Field: "SourceType", Operator: "=", Value: "system"},
							},
						},
						{ // Group 2.2: MediumImportance AND UserSource AND more conditions
							Operator: OpAnd,
							Conditions: []Condition{
								{Field: "Importance", Operator: "=", Value: ImportanceMedium},
								{Field: "SourceType", Operator: "=", Value: "user"},
							},
							Groups: []FilterGroup{
								{ // Group 2.2.1: Metadata depth AND Tags
									Operator: OpAnd,
									Conditions: []Condition{
										{Field: "Metadata", Operator: "=", Value: map[string]interface{}{"depth": "level2"}},
										{Field: "Tags", Operator: "CONTAINS", Value: "complex"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Execute the complex filter
	results, err := store.SearchRecords(deepFilter)
	if err != nil {
		t.Fatalf("Failed to execute deep filter: %v", err)
	}

	// We expect exactly 2 records to match:
	// 1. deep-filter-1 (CategoryFact + ImportanceHigh + SourceType=system)
	// 2. deep-filter-2 (CategoryAction + ImportanceMedium + SourceType=user + depth=level2 + Tags=complex)
	if len(results) != 2 {
		t.Errorf("Expected 2 results from deep filter, got %d", len(results))
	}

	expectedIDs := map[string]bool{
		"deep-filter-1": true,
		"deep-filter-2": true,
	}

	for _, r := range results {
		if !expectedIDs[r.ID] {
			t.Errorf("Unexpected record %s found in deep filter results", r.ID)
		}
		delete(expectedIDs, r.ID) // Remove to track what we've found
	}

	// Check if any expected records weren't found
	for id := range expectedIDs {
		t.Errorf("Expected record %s not found in deep filter results", id)
	}
}

// TestMemoryStore_ZeroLengthFields tests handling of empty and zero-length fields
func TestMemoryStore_ZeroLengthFields(t *testing.T) {
	store, _ := NewMemoryStore()
	_ = store.Open()
	defer store.Close()

	// Create records with empty fields
	records := []Entry{
		{
			ID:          "empty-content",
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte(""), // Empty content
			Importance:  ImportanceLow,
			SourceID:    "test",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "empty-tags",
			Category:    CategoryMessage,
			ContentType: ContentTypeText,
			Content:     []byte("Has content but empty tags"),
			Importance:  ImportanceMedium,
			SourceID:    "test",
			Tags:        []string{}, // Empty tags slice
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "nil-fields",
			Category:    CategoryAction,
			ContentType: ContentTypeText,
			Content:     []byte("Has content but nil refs and subjects"),
			Importance:  ImportanceHigh,
			SourceID:    "test",
			References:  nil, // Nil references
			SubjectIDs:  nil, // Nil subjects
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "empty-metadata",
			Category:    CategoryDecision,
			ContentType: ContentTypeText,
			Content:     []byte("Has content but empty metadata"),
			Importance:  ImportanceHigh,
			SourceID:    "test",
			Metadata:    map[string]string{}, // Empty metadata map
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record with empty fields %s: %v", record.ID, err)
		}
	}

	// Test 1: Verify we can retrieve records with empty fields
	for _, expected := range records {
		actual, err := store.GetRecord(expected.ID)
		if err != nil {
			t.Errorf("Failed to get record with empty fields %s: %v", expected.ID, err)
			continue
		}

		// Verify specific empty fields based on record ID
		switch expected.ID {
		case "empty-content":
			if len(actual.Content) != 0 {
				t.Errorf("Expected empty content, got %d bytes", len(actual.Content))
			}
		case "empty-tags":
			if actual.Tags == nil {
				t.Errorf("Expected empty tags slice, got nil")
			} else if len(actual.Tags) != 0 {
				t.Errorf("Expected 0 tags, got %d", len(actual.Tags))
			}
		case "nil-fields":
			if actual.References != nil && len(actual.References) != 0 {
				t.Errorf("Expected nil or empty references, got %v", actual.References)
			}
			if actual.SubjectIDs != nil && len(actual.SubjectIDs) != 0 {
				t.Errorf("Expected nil or empty subjects, got %v", actual.SubjectIDs)
			}
		case "empty-metadata":
			if actual.Metadata == nil {
				t.Errorf("Expected empty metadata map, got nil")
			} else if len(actual.Metadata) != 0 {
				t.Errorf("Expected 0 metadata entries, got %d", len(actual.Metadata))
			}
		}
	}

	// Test 2: Search by empty content
	emptyContentFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Content", Operator: "=", Value: ""},
			},
		},
	}

	results, err := store.SearchRecords(emptyContentFilter)
	if err != nil {
		t.Fatalf("Failed to search for empty content: %v", err)
	}

	if len(results) != 1 || results[0].ID != "empty-content" {
		t.Errorf("Expected 1 result with empty content, got %d", len(results))
	}

	// Test 3: Update empty fields to non-empty and vice versa
	// Get the empty content record and update it with content
	emptyContentRecord, err := store.GetRecord("empty-content")
	if err != nil {
		t.Fatalf("Failed to get empty content record: %v", err)
	}

	// Update with content
	emptyContentRecord.Content = []byte("Now has content")
	if err := store.UpdateRecord(emptyContentRecord); err != nil {
		t.Fatalf("Failed to update previously empty content: %v", err)
	}

	// Get a record with content and update to empty
	recordWithContent, err := store.GetRecord("empty-tags")
	if err != nil {
		t.Fatalf("Failed to get record with content: %v", err)
	}

	// Update to empty content
	recordWithContent.Content = []byte("")
	if err := store.UpdateRecord(recordWithContent); err != nil {
		t.Fatalf("Failed to update to empty content: %v", err)
	}

	// Verify updates
	updatedEmptyContent, err := store.GetRecord("empty-content")
	if err != nil {
		t.Fatalf("Failed to get previously empty content record: %v", err)
	}

	if string(updatedEmptyContent.Content) != "Now has content" {
		t.Errorf("Content not updated correctly. Expected 'Now has content', got '%s'",
			string(updatedEmptyContent.Content))
	}

	updatedToEmpty, err := store.GetRecord("empty-tags")
	if err != nil {
		t.Fatalf("Failed to get updated to empty record: %v", err)
	}

	if len(updatedToEmpty.Content) != 0 {
		t.Errorf("Content not updated to empty. Got '%s'", string(updatedToEmpty.Content))
	}
}
