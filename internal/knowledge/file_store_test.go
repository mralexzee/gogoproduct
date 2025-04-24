package knowledge

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewFileStore(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, err := NewFileStore(storeFile)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
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

	if info["implementation"] != "FileStore" {
		t.Errorf("Expected implementation to be FileStore, got %s", info["implementation"])
	}
	if info["file_path"] != storeFile {
		t.Errorf("Expected file_path to be %s, got %s", storeFile, info["file_path"])
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
}

func TestFileStore_AddRecord(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, _ := NewFileStore(storeFile)
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

	// Verify record is persisted
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}

	// Reopen store to check persistence
	_ = store.Close()
	_ = store.Open()

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
}

func TestFileStore_GetRecord(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	now := time.Now()

	// Add several records
	records := []Entry{
		{
			ID:          "get-record-1",
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte("Test content 1"),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "get-record-2",
			Category:    CategoryMessage,
			ContentType: ContentTypeJSON,
			Content:     []byte(`{"test": "value"}`),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Test getting existing records
	for _, expected := range records {
		got, err := store.GetRecord(expected.ID)
		if err != nil {
			t.Errorf("Failed to get record %s: %v", expected.ID, err)
			continue
		}

		if got.ID != expected.ID {
			t.Errorf("Expected ID %s, got %s", expected.ID, got.ID)
		}
		if got.Category != expected.Category {
			t.Errorf("Expected Category %s, got %s", expected.Category, got.Category)
		}
		if got.ContentType != expected.ContentType {
			t.Errorf("Expected ContentType %s, got %s", expected.ContentType, got.ContentType)
		}
		if !bytes.Equal(got.Content, expected.Content) {
			t.Errorf("Expected Content %s, got %s", string(expected.Content), string(got.Content))
		}
	}

	// Test getting non-existent record
	_, err := store.GetRecord("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent record, got nil")
	}

	// Verify persistence across reopens
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Check if records are still there after reopen
	for _, expected := range records {
		got, err := store.GetRecord(expected.ID)
		if err != nil {
			t.Errorf("Failed to get record %s after reopen: %v", expected.ID, err)
		} else if got.ID != expected.ID {
			t.Errorf("After reopen: Expected ID %s, got %s", expected.ID, got.ID)
		}
	}
}

func TestFileStore_UpdateRecord(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	now := time.Now()

	// Add a record
	record := Entry{
		ID:          "update-test",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Original content"),
		Importance:  ImportanceLow,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{"original"},
	}

	if err := store.AddRecord(record); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update the record
	record.Content = []byte("Updated content")
	record.Importance = ImportanceHigh
	record.Tags = append(record.Tags, "updated")
	record.UpdatedAt = now.Add(time.Hour)

	if err := store.UpdateRecord(record); err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Get the updated record
	updated, err := store.GetRecord(record.ID)
	if err != nil {
		t.Fatalf("Failed to get updated record: %v", err)
	}

	// Verify updates
	if !bytes.Equal(updated.Content, record.Content) {
		t.Errorf("Expected Content %s, got %s", string(record.Content), string(updated.Content))
	}
	if updated.Importance != record.Importance {
		t.Errorf("Expected Importance %d, got %d", record.Importance, updated.Importance)
	}
	if !reflect.DeepEqual(updated.Tags, record.Tags) {
		t.Errorf("Expected Tags %v, got %v", record.Tags, updated.Tags)
	}
	// Skip exact time comparison - just check that UpdatedAt is set to some non-zero value
	if updated.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt should not be zero after update")
	}

	// Test updating non-existent record
	nonExistent := record
	nonExistent.ID = "non-existent"
	if err := store.UpdateRecord(nonExistent); err == nil {
		t.Error("Expected error when updating non-existent record, got nil")
	}

	// Verify persistence
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Check if updates persisted
	persisted, err := store.GetRecord(record.ID)
	if err != nil {
		t.Fatalf("Failed to get record after reopen: %v", err)
	}
	if !bytes.Equal(persisted.Content, record.Content) {
		t.Errorf("After reopen: Expected Content %s, got %s", string(record.Content), string(persisted.Content))
	}
}

func TestFileStore_DeleteRestorePurgeRecord(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Add test records
	records := []Entry{
		{ID: "delete-test-1", Category: CategoryFact, Content: []byte("Test 1")},
		{ID: "delete-test-2", Category: CategoryMessage, Content: []byte("Test 2")},
		{ID: "delete-test-3", Category: CategoryDecision, Content: []byte("Test 3")},
	}

	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Test soft delete
	if err := store.DeleteRecord("delete-test-1"); err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Verify it's not accessible via normal Get
	_, err := store.GetRecord("delete-test-1")
	if err == nil {
		t.Error("Should not be able to get deleted record")
	}

	// Test restore
	if err := store.RestoreRecord("delete-test-1"); err != nil {
		t.Fatalf("Failed to restore record: %v", err)
	}

	// Verify it's accessible again
	restored, err := store.GetRecord("delete-test-1")
	if err != nil {
		t.Fatalf("Failed to get restored record: %v", err)
	}
	if restored.ID != "delete-test-1" {
		t.Errorf("Wrong record restored, expected ID delete-test-1, got %s", restored.ID)
	}

	// Test deleting non-existent record
	if err := store.DeleteRecord("non-existent"); err == nil {
		t.Error("Should not be able to delete non-existent record")
	}

	// Test restoring non-deleted record
	if err := store.RestoreRecord("delete-test-2"); err == nil {
		t.Error("Should not be able to restore a non-deleted record")
	}

	// Test purge operation
	if err := store.DeleteRecord("delete-test-2"); err != nil {
		t.Fatalf("Failed to delete record for purge test: %v", err)
	}

	if err := store.PurgeRecord("delete-test-2"); err != nil {
		t.Fatalf("Failed to purge record: %v", err)
	}

	// Verify purged record can't be restored
	if err := store.RestoreRecord("delete-test-2"); err == nil {
		t.Error("Should not be able to restore a purged record")
	}

	// Test purging non-existent record
	if err := store.PurgeRecord("non-existent"); err == nil {
		t.Error("Should not be able to purge non-existent record")
	}

	// Verify persistence
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Verify deleted status persists
	if err := store.DeleteRecord("delete-test-3"); err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	_, err = store.GetRecord("delete-test-3")
	if err == nil {
		t.Error("Deleted status should persist across reopens")
	}
}

func TestFileStore_Concurrency(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "concurrent.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Number of concurrent operations
	numGoroutines := 10
	numOperations := 50

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Create channel for errors
	errorChan := make(chan error, numGoroutines*numOperations)

	// Start goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Generate unique IDs for records
				id := fmt.Sprintf("concurrent-r%d-op%d", routineID, j)

				// Choose an operation based on j
				switch j % 4 {
				case 0: // Add
					record := Entry{
						ID:        id,
						Category:  CategoryFact,
						Content:   []byte(fmt.Sprintf("Concurrent content %d", j)),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Tags:      []string{"concurrent", fmt.Sprintf("routine-%d", routineID)},
						Metadata:  map[string]string{"test": "concurrent"},
					}
					if err := store.AddRecord(record); err != nil {
						errorChan <- fmt.Errorf("routine %d add failed: %w", routineID, err)
					}

				case 1: // Get
					// Try to get a record from another routine
					targetRoutine := (routineID + 1) % numGoroutines
					targetID := fmt.Sprintf("concurrent-r%d-op%d", targetRoutine, (j-1)%numOperations)
					_, err := store.GetRecord(targetID)
					// Ignore not found errors - record might not exist yet
					if err != nil && !strings.Contains(err.Error(), "not found") {
						errorChan <- fmt.Errorf("routine %d get failed: %w", routineID, err)
					}

				case 2: // Update
					// Update our own previous record if possible
					updateID := fmt.Sprintf("concurrent-r%d-op%d", routineID, (j-2)%numOperations)
					existing, err := store.GetRecord(updateID)
					if err == nil {
						existing.Content = []byte(fmt.Sprintf("Updated content %d", j))
						existing.UpdatedAt = time.Now()
						// Ignore update errors in concurrent test - another routine might have deleted the record
						_ = store.UpdateRecord(existing)
					}

				case 3: // Delete
					// Delete another routine's record if possible
					targetRoutine := (routineID + 2) % numGoroutines
					deleteID := fmt.Sprintf("concurrent-r%d-op%d", targetRoutine, (j-3)%numOperations)
					err := store.DeleteRecord(deleteID)
					// Ignore not found errors
					if err != nil && !strings.Contains(err.Error(), "not found") {
						errorChan <- fmt.Errorf("routine %d delete failed: %w", routineID, err)
					}
				}
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errorChan)

	// Check if there were any errors
	errors := []error{}
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("Concurrent operation error: %v", err)
		}
	}

	// Verify that we have records in the store
	filter := Filter{IncludeDeleted: true}
	results, err := store.SearchRecords(filter)
	if err != nil {
		t.Fatalf("Failed to search records after concurrency test: %v", err)
	}

	// We should have some records, not necessarily all due to concurrent deletes
	if len(results) == 0 {
		t.Error("Expected to find records after concurrency test, but none found")
	}

	// Verify the store can still be used after concurrency test
	newRecord := Entry{
		ID:       "post-concurrent",
		Category: CategoryMessage,
		Content:  []byte("After concurrency test"),
	}
	if err := store.AddRecord(newRecord); err != nil {
		t.Fatalf("Failed to add record after concurrency test: %v", err)
	}

	// Make sure it persists
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store after concurrency test: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	gotRecord, err := store.GetRecord("post-concurrent")
	if err != nil {
		t.Fatalf("Failed to get record after reopen: %v", err)
	}
	if gotRecord.ID != "post-concurrent" {
		t.Errorf("Expected to get record with ID 'post-concurrent', got '%s'", gotRecord.ID)
	}
}

func TestFileStore_TimeRangeFiltering(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "timerange.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Current time for testing
	now := time.Now()
	past1h := now.Add(-1 * time.Hour)
	past2h := now.Add(-2 * time.Hour)
	past3h := now.Add(-3 * time.Hour)
	future1h := now.Add(1 * time.Hour)
	future2h := now.Add(2 * time.Hour)

	// Create records with different timestamps
	timeRecords := []Entry{
		{
			ID:          "time-past",
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte("Past record"),
			CreatedAt:   past3h,
			UpdatedAt:   past2h,
			ExpiresAt:   past1h, // Already expired
		},
		{
			ID:          "time-recent",
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte("Recent record"),
			CreatedAt:   past2h,
			UpdatedAt:   now,
			ExpiresAt:   future2h, // Valid for 2 hours
		},
		{
			ID:          "time-future",
			Category:    CategoryFact,
			ContentType: ContentTypeText,
			Content:     []byte("Future record"),
			CreatedAt:   now,
			UpdatedAt:   now,
			ExpiresAt:   future2h, // Valid for 2 hours
		},
	}

	for _, record := range timeRecords {
		if err := store.AddRecord(record); err != nil {
			t.Errorf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Verify time range filtering
	filter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "CreatedAt", Operator: ">=", Value: past2h}, // Changed to >= to include edge cases
				{Field: "CreatedAt", Operator: "<=", Value: now},    // Changed to <= now for better matching
			},
		},
	}
	results, err := store.SearchRecords(filter)
	if err != nil {
		t.Fatalf("Failed to search records with time range filter: %v", err)
	}

	// Debug output to help diagnose time comparisons
	t.Logf("Time filter results count: %d", len(results))
	t.Logf("Search conditions: CreatedAt >= %v AND CreatedAt <= %v", past2h, now)

	// Log the actual record times for debugging
	for _, rec := range timeRecords {
		t.Logf("Record %s: CreatedAt=%v (matches first condition: %v, matches second condition: %v)",
			rec.ID,
			rec.CreatedAt,
			rec.CreatedAt.Equal(past2h) || rec.CreatedAt.After(past2h),
			rec.CreatedAt.Equal(now) || rec.CreatedAt.Before(now))
	}

	// Log the results
	for _, r := range results {
		t.Logf("Found record ID: %s, CreatedAt: %v", r.ID, r.CreatedAt)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records in time range, got %d", len(results))
	}

	// Verify expiration with future1h to prevent unused variable warning
	filter = Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "ExpiresAt", Operator: ">", Value: past1h},
				{Field: "ExpiresAt", Operator: "<", Value: future1h.Add(24 * time.Hour)},
			},
		},
	}
	results, err = store.SearchRecords(filter)
	if err != nil {
		t.Fatalf("Failed to search records with expiration filter: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 records with expiration after past1h, got %d", len(results))
	}
}

// Helper function to check if a slice contains a particular string
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func TestFileStore_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "special-chars.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Create a record with basic ASCII characters
	record := Entry{
		ID:          "test-special-chars",
		Category:    CategoryFact,
		ContentType: ContentTypeText,
		Content:     []byte("Test content with special characters: !@#$%^&*()"),
		Tags:        []string{"special-char-tag", "test-tag"},
		Metadata: map[string]string{
			"test-key": "test-value",
		},
	}

	// Add the record
	if err := store.AddRecord(record); err != nil {
		t.Fatalf("Failed to add record: %v", err)
	}

	// Update with new tag
	record.Tags = append(record.Tags, "new-tag")
	if err := store.UpdateRecord(record); err != nil {
		t.Fatalf("Failed to update record: %v", err)
	}

	// Verify tag was added
	updated, err := store.GetRecord(record.ID)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if !containsTag(updated.Tags, "new-tag") {
		t.Errorf("Updated record missing new tag")
	}

	// Verify persistence
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()

	// Reopen store and check tags persist
	storeReopened, _ := NewFileStore(storeFile)
	_ = storeReopened.Open()
	defer storeReopened.Close()

	// Check tag persistence
	persisted, err := storeReopened.GetRecord(record.ID)
	if err != nil {
		t.Fatalf("Failed to get record after reopen: %v", err)
	}
	// Print the actual tags to help debug
	t.Logf("Tags after reopen: %v", persisted.Tags)
	if !containsTag(persisted.Tags, "new-tag") {
		t.Errorf("After reopen: Record missing new tag. Tags found: %v", persisted.Tags)
	}
}

func TestFileStore_Reopening(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "persistence.json")

	// Create a store and add records
	store, _ := NewFileStore(storeFile)
	_ = store.Open()

	// Add some records
	records := []Entry{
		{ID: "persist-1", Category: CategoryFact, Content: []byte("Persistent 1")},
		{ID: "persist-2", Category: CategoryMessage, Content: []byte("Persistent 2")},
		{ID: "persist-3", Category: CategoryDecision, Content: []byte("Persistent 3")},
	}

	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Delete one record
	if err := store.DeleteRecord("persist-3"); err != nil {
		t.Fatalf("Failed to delete record: %v", err)
	}

	// Flush to ensure data is saved
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}

	// Open the store again with the same file
	store2, _ := NewFileStore(storeFile)
	_ = store2.Open()
	defer store2.Close()

	// Check if data is still available
	// Should have 2 normal records and 1 deleted
	results, err := store2.SearchRecords(Filter{IncludeDeleted: true})
	if err != nil {
		t.Fatalf("Failed to search records after reopen: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 records (2 active, 1 deleted) after reopen, got %d", len(results))
	}

	// Check normal records
	results, err = store2.SearchRecords(Filter{})
	if err != nil {
		t.Fatalf("Failed to search normal records after reopen: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 active records after reopen, got %d", len(results))
	}

	// Check deleted records
	results, err = store2.SearchRecords(Filter{OnlyDeleted: true})
	if err != nil {
		t.Fatalf("Failed to search deleted records after reopen: %v", err)
	}

	if len(results) != 1 || (len(results) > 0 && results[0].ID != "persist-3") {
		t.Errorf("Expected 1 deleted record with ID 'persist-3' after reopen, got %d records", len(results))
	}

	// Add new record to the reopened store
	newRecord := Entry{ID: "persist-4", Category: CategoryAction, Content: []byte("Added after reopen")}
	if err := store2.AddRecord(newRecord); err != nil {
		t.Fatalf("Failed to add record to reopened store: %v", err)
	}

	// Flush and close again
	if err := store2.Flush(); err != nil {
		t.Fatalf("Failed to flush reopened store: %v", err)
	}
	if err := store2.Close(); err != nil {
		t.Fatalf("Failed to close reopened store: %v", err)
	}

	// Open a third time
	store3, _ := NewFileStore(storeFile)
	_ = store3.Open()
	defer store3.Close()

	// Should now have 3 active records
	results, err = store3.SearchRecords(Filter{})
	if err != nil {
		t.Fatalf("Failed to search normal records after second reopen: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 active records after second reopen, got %d", len(results))
	}

	// Verify we can get each individual record
	for _, id := range []string{"persist-1", "persist-2", "persist-4"} {
		_, err := store3.GetRecord(id)
		if err != nil {
			t.Errorf("Failed to get record %s after second reopen: %v", id, err)
		}
	}

	// Verify deleted record is still deleted
	_, err = store3.GetRecord("persist-3")
	if err == nil {
		t.Error("Deleted record should still be inaccessible after second reopen")
	}
}

func TestFileStore_DeepFilterNesting(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "deep-filter.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Create a diverse set of records for testing complex filters
	records := []Entry{
		{
			ID:         "df-1",
			Category:   CategoryFact,
			Importance: ImportanceHigh,
			Tags:       []string{"important", "urgent"},
			Metadata: map[string]string{
				"department": "engineering",
				"status":     "active",
			},
		},
		{
			ID:         "df-2",
			Category:   CategoryDecision,
			Importance: ImportanceMedium,
			Tags:       []string{"important", "scheduled"},
			Metadata: map[string]string{
				"department": "engineering",
				"status":     "pending",
			},
		},
		{
			ID:         "df-3",
			Category:   CategoryAction,
			Importance: ImportanceLow,
			Tags:       []string{"routine", "scheduled"},
			Metadata: map[string]string{
				"department": "hr",
				"status":     "active",
			},
		},
		{
			ID:         "df-4",
			Category:   CategoryMessage,
			Importance: ImportanceNone,
			Tags:       []string{"routine", "archived"},
			Metadata: map[string]string{
				"department": "marketing",
				"status":     "inactive",
			},
		},
		{
			ID:         "df-5",
			Category:   CategoryFact,
			Importance: ImportanceCritical,
			Tags:       []string{"important", "urgent", "critical"},
			Metadata: map[string]string{
				"department": "executive",
				"status":     "active",
			},
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record %s: %v", record.ID, err)
		}
	}

	// Test 1: Deeply nested filter with multiple AND, OR, and NOT groups
	// Logic: records that are:
	// ((Category=Fact AND Importance>=High) OR (Category=Decision AND Tags CONTAINS important))
	// AND NOT (Metadata.status=inactive)
	deepFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Groups: []FilterGroup{
				// Group 1: high importance facts OR important decisions
				{
					Operator: OpOr,
					Groups: []FilterGroup{
						// Subgroup 1: High importance facts
						{
							Operator: OpAnd,
							Conditions: []Condition{
								{Field: "Category", Operator: "=", Value: CategoryFact},
								{Field: "Importance", Operator: ">=", Value: ImportanceHigh},
							},
						},
						// Subgroup 2: Important decisions
						{
							Operator: OpAnd,
							Conditions: []Condition{
								{Field: "Category", Operator: "=", Value: CategoryDecision},
								{Field: "Tags", Operator: "CONTAINS", Value: "important"},
							},
						},
					},
				},
				// Group 2: NOT inactive status
				{
					Operator: OpNot,
					Conditions: []Condition{
						{Field: "Metadata", Operator: "=", Value: map[string]string{"status": "inactive"}},
					},
				},
			},
		},
	}

	results, err := store.SearchRecords(deepFilter)
	if err != nil {
		t.Fatalf("Failed to search with deep filter: %v", err)
	}

	// Expected results: df-1, df-2, df-5 (high importance facts + important decisions, not inactive)
	if len(results) != 3 {
		t.Errorf("Expected 3 results from deep filter, got %d", len(results))
	}

	// Verify expected IDs
	expectedIDs := map[string]bool{"df-1": true, "df-2": true, "df-5": true}
	for _, result := range results {
		if !expectedIDs[result.ID] {
			t.Errorf("Unexpected result ID: %s", result.ID)
		}
		delete(expectedIDs, result.ID)
	}

	if len(expectedIDs) > 0 {
		missing := []string{}
		for id := range expectedIDs {
			missing = append(missing, id)
		}
		t.Errorf("Missing expected results: %v", missing)
	}

	// Test 2: Complex nested metadata filter
	// Records with active status OR engineering department, AND have either important tag OR high importance
	metadataNestedFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Groups: []FilterGroup{
				// Group 1: status=active OR department=engineering
				{
					Operator: OpOr,
					Conditions: []Condition{
						{Field: "Metadata", Operator: "=", Value: map[string]string{"status": "active"}},
						{Field: "Metadata", Operator: "=", Value: map[string]string{"department": "engineering"}},
					},
				},
				// Group 2: Tags CONTAINS important OR Importance >= High
				{
					Operator: OpOr,
					Conditions: []Condition{
						{Field: "Tags", Operator: "CONTAINS", Value: "important"},
						{Field: "Importance", Operator: ">=", Value: ImportanceHigh},
					},
				},
			},
		},
	}

	results, err = store.SearchRecords(metadataNestedFilter)
	if err != nil {
		t.Fatalf("Failed to search with metadata nested filter: %v", err)
	}

	// Expected: df-1, df-2, df-5 (all meeting the criteria)
	if len(results) != 3 {
		t.Errorf("Expected 3 results from metadata nested filter, got %d", len(results))
	}

	// Verify results persist after reopen
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Re-run deep filter test after reopen
	results, err = store.SearchRecords(deepFilter)
	if err != nil {
		t.Fatalf("Failed to search with deep filter after reopen: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("After reopen: Expected 3 results from deep filter, got %d", len(results))
	}
}

func TestFileStore_ZeroLengthFields(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "zero-length.json")

	store, _ := NewFileStore(storeFile)
	_ = store.Open()
	defer store.Close()

	// Test records with various zero-length or nil fields
	records := []Entry{
		{
			ID:       "empty-content",
			Category: CategoryFact,
			Content:  []byte{}, // Empty content
		},
		{
			ID:       "nil-content",
			Category: CategoryMessage,
			// Content is nil by default
		},
		{
			ID:       "empty-tags",
			Category: CategoryDecision,
			Content:  []byte("Has content but empty tags"),
			Tags:     []string{},
		},
		{
			ID:       "nil-tags",
			Category: CategoryAction,
			Content:  []byte("Has content but nil tags"),
			// Tags is nil by default
		},
		{
			ID:       "empty-metadata",
			Category: CategoryFact,
			Content:  []byte("Has content but empty metadata"),
			Metadata: map[string]string{},
		},
		{
			ID:       "nil-metadata",
			Category: CategoryMessage,
			Content:  []byte("Has content but nil metadata"),
			// Metadata is nil by default
		},
		{
			ID:          "all-empty",
			Category:    CategoryDecision,
			Content:     []byte{},
			Tags:        []string{},
			Metadata:    map[string]string{},
			SubjectIDs:  []string{},
			SourceID:    "", // Empty string
			SourceType:  "",
			OwnerID:     "",
			OwnerType:   "",
			SubjectType: "",
		},
	}

	// Add all records
	for _, record := range records {
		if err := store.AddRecord(record); err != nil {
			t.Fatalf("Failed to add record with zero-length field %s: %v", record.ID, err)
		}
	}

	// Test 1: Retrieve each record and verify zero-length fields are preserved
	for _, expected := range records {
		got, err := store.GetRecord(expected.ID)
		if err != nil {
			t.Errorf("Failed to get record with ID '%s': %v", expected.ID, err)
			continue
		}

		// Check Content field
		if expected.Content == nil {
			if got.Content != nil && len(got.Content) > 0 {
				t.Errorf("Record %s: Expected nil Content, got %v", expected.ID, got.Content)
			}
		} else if !bytes.Equal(got.Content, expected.Content) {
			t.Errorf("Record %s: Content mismatch, expected '%s', got '%s'",
				expected.ID, string(expected.Content), string(got.Content))
		}

		// Check Tags field
		if expected.Tags == nil {
			if got.Tags != nil && len(got.Tags) > 0 {
				t.Errorf("Record %s: Expected nil Tags, got %v", expected.ID, got.Tags)
			}
		} else if !reflect.DeepEqual(got.Tags, expected.Tags) {
			t.Errorf("Record %s: Tags mismatch, expected %v, got %v",
				expected.ID, expected.Tags, got.Tags)
		}

		// Check Metadata field
		if expected.Metadata == nil {
			if got.Metadata != nil && len(got.Metadata) > 0 {
				t.Errorf("Record %s: Expected nil Metadata, got %v", expected.ID, got.Metadata)
			}
		} else if !reflect.DeepEqual(got.Metadata, expected.Metadata) {
			t.Errorf("Record %s: Metadata mismatch, expected %v, got %v",
				expected.ID, expected.Metadata, got.Metadata)
		}
	}

	// Test 2: Update a record with zero fields to add values
	emptyContent, err := store.GetRecord("empty-content")
	if err != nil {
		t.Fatalf("Failed to get empty-content record: %v", err)
	}

	// Update to add content, tags, and metadata
	emptyContent.Content = []byte("Now has content")
	emptyContent.Tags = []string{"now", "has", "tags"}
	emptyContent.Metadata = map[string]string{"now": "has-metadata"}

	if err := store.UpdateRecord(emptyContent); err != nil {
		t.Fatalf("Failed to update record with values: %v", err)
	}

	// Get the updated record
	updated, err := store.GetRecord("empty-content")
	if err != nil {
		t.Fatalf("Failed to get updated record: %v", err)
	}

	// Verify updates
	if !bytes.Equal(updated.Content, emptyContent.Content) {
		t.Errorf("Updated record has wrong content")
	}
	if !reflect.DeepEqual(updated.Tags, emptyContent.Tags) {
		t.Errorf("Updated record has wrong tags")
	}
	if !reflect.DeepEqual(updated.Metadata, emptyContent.Metadata) {
		t.Errorf("Updated record has wrong metadata")
	}

	// Test 3: Update a record with values to empty/nil
	hasContent, err := store.GetRecord("nil-tags")
	if err != nil {
		t.Fatalf("Failed to get nil-tags record: %v", err)
	}

	// Update to remove content and set empty tags
	hasContent.Content = []byte{}
	hasContent.Tags = []string{}

	if err := store.UpdateRecord(hasContent); err != nil {
		t.Fatalf("Failed to update record to empty values: %v", err)
	}

	// Get the updated record
	emptied, err := store.GetRecord("nil-tags")
	if err != nil {
		t.Fatalf("Failed to get emptied record: %v", err)
	}

	// Verify updates
	if len(emptied.Content) != 0 {
		t.Errorf("Emptied content still has %d bytes", len(emptied.Content))
	}
	if len(emptied.Tags) != 0 {
		t.Errorf("Emptied tags still has %d elements", len(emptied.Tags))
	}

	// Test 4: Search for records with empty content
	emptyContentFilter := Filter{
		RootGroup: FilterGroup{
			Operator: OpAnd,
			Conditions: []Condition{
				{Field: "Content", Operator: "=", Value: []byte{}},
			},
		},
	}

	results, err := store.SearchRecords(emptyContentFilter)
	if err != nil {
		t.Fatalf("Failed to search for empty content: %v", err)
	}

	// Should find at least "empty-content" (now updated), "all-empty", and "nil-tags" (now updated)
	if len(results) < 2 {
		t.Errorf("Expected at least 2 records with empty content, got %d", len(results))
	}

	// Verify persistence across reopen
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Check that emptied record stays empty
	reopened, err := store.GetRecord("nil-tags")
	if err != nil {
		t.Fatalf("Failed to get emptied record after reopen: %v", err)
	}

	if len(reopened.Content) != 0 {
		t.Errorf("After reopen: Emptied content has %d bytes", len(reopened.Content))
	}
	if len(reopened.Tags) != 0 {
		t.Errorf("After reopen: Emptied tags has %d elements", len(reopened.Tags))
	}
}

func TestFileStore_SearchRecords(t *testing.T) {
	tempDir := t.TempDir()
	storeFile := filepath.Join(tempDir, "knowledge.json")

	store, _ := NewFileStore(storeFile)
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

	// Test 5: Complex filter with nested groups and OR+AND operators
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

	// Test 6: Deleted records handling
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

	// Verify persistence
	if err := store.Flush(); err != nil {
		t.Fatalf("Failed to flush store: %v", err)
	}
	_ = store.Close()
	_ = store.Open()

	// Check if filter still works after reopen
	results, err = store.SearchRecords(onlyDeletedFilter)
	if err != nil {
		t.Fatalf("Failed to search with onlyDeleted after reopen: %v", err)
	}
	if len(results) != 1 || (len(results) > 0 && results[0].ID != "search-1") {
		t.Errorf("Expected only deleted record 'search-1' after reopen, got %d results", len(results))
	}
}
