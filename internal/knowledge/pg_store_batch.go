package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SQL queries for batch operations
const (
	// Check record existence
	sqlCheckRecordExists = `
		SELECT EXISTS(
			SELECT 1 FROM knowledge_entry 
			WHERE id = $1 AND account_id = $2
		)
	`

	// Insert query for LoadRecords
	sqlBatchInsertRecord = `
		INSERT INTO knowledge_entry (
			id, account_id, category, content_type, content, importance,
			created_at, updated_at, expires_at, source_id, source_type,
			owner_id, owner_type, subject_ids, subject_type, tags,
			references, metadata, is_deleted
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, false
		)
	`

	// Update query for LoadRecords
	sqlBatchUpdateRecord = `
		UPDATE knowledge_entry
		SET 
			category = $3, 
			content_type = $4, 
			content = $5, 
			importance = $6,
			updated_at = $7, 
			expires_at = $8, 
			source_id = $9, 
			source_type = $10,
			owner_id = $11, 
			owner_type = $12, 
			subject_ids = $13, 
			subject_type = $14, 
			tags = $15,
			references = $16, 
			metadata = $17
		WHERE id = $1 AND account_id = $2
	`
)

// LoadRecords loads multiple records into the store
// If a record with the same ID exists, it's updated; otherwise, it's added
func (p *PgStore) LoadRecords(records ...Entry) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

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

		// Validate ID is a valid UUID
		_, err := uuid.Parse(record.ID)
		if err != nil {
			return fmt.Errorf("record at index %d has invalid UUID format: %w", i, err)
		}

		// Check for duplicate IDs in the input
		if seenIDs[record.ID] {
			return fmt.Errorf("duplicate record ID found in input: %s", record.ID)
		}
		seenIDs[record.ID] = true
	}

	// Start a transaction for the batch operation
	tx, err := p.crudDB.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Process all records (add new ones, update existing ones)
	now := time.Now()
	for _, record := range records {
		// Parse ID
		id, _ := uuid.Parse(record.ID) // Already validated in previous loop

		// Check if record exists
		var exists bool
		err = tx.QueryRow(sqlCheckRecordExists, id, p.accountID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check record %s existence: %w", record.ID, err)
		}

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

		// Serialize References to JSONB
		refsJSON, err := json.Marshal(record.References)
		if err != nil {
			return fmt.Errorf("failed to serialize references for record %s: %w", record.ID, err)
		}

		// Serialize Metadata to JSONB
		metadataJSON, err := json.Marshal(record.Metadata)
		if err != nil {
			return fmt.Errorf("failed to serialize metadata for record %s: %w", record.ID, err)
		}

		if exists {
			// Update existing record
			_, err = tx.Exec(
				sqlBatchUpdateRecord,
				id,                                  // $1
				p.accountID,                         // $2
				record.Category,                     // $3
				record.ContentType,                  // $4
				record.Content,                      // $5
				record.Importance,                   // $6
				record.UpdatedAt,                    // $7
				nullTimeValue(record.ExpiresAt),     // $8
				nullStringValue(record.SourceID),    // $9
				nullStringValue(record.SourceType),  // $10
				nullStringValue(record.OwnerID),     // $11
				nullStringValue(record.OwnerType),   // $12
				pq.Array(record.SubjectIDs),         // $13
				nullStringValue(record.SubjectType), // $14
				pq.Array(record.Tags),               // $15
				refsJSON,                            // $16
				metadataJSON,                        // $17
			)
		} else {
			// Insert new record
			_, err = tx.Exec(
				sqlBatchInsertRecord,
				id,                                  // $1
				p.accountID,                         // $2
				record.Category,                     // $3
				record.ContentType,                  // $4
				record.Content,                      // $5
				record.Importance,                   // $6
				record.CreatedAt,                    // $7
				record.UpdatedAt,                    // $8
				nullTimeValue(record.ExpiresAt),     // $9
				nullStringValue(record.SourceID),    // $10
				nullStringValue(record.SourceType),  // $11
				nullStringValue(record.OwnerID),     // $12
				nullStringValue(record.OwnerType),   // $13
				pq.Array(record.SubjectIDs),         // $14
				nullStringValue(record.SubjectType), // $15
				pq.Array(record.Tags),               // $16
				refsJSON,                            // $17
				metadataJSON,                        // $18
			)
		}

		if err != nil {
			return fmt.Errorf("failed to %s record %s: %w",
				map[bool]string{true: "update", false: "insert"}[exists],
				record.ID, err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
