package knowledge

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// SQL queries for CRUD operations
const (
	// Insert query for AddRecord
	sqlInsertRecord = `
		INSERT INTO knowledge_entry (
			id, account_id, category, content_type, content, importance,
			created_at, updated_at, expires_at, source_id, source_type,
			owner_id, owner_type, subject_ids, subject_type, tags,
			references, metadata, is_deleted
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, false
		)
	`

	// Select query for GetRecord
	sqlGetRecord = `
		SELECT 
			id, category, content_type, content, importance,
			created_at, updated_at, expires_at, source_id, source_type,
			owner_id, owner_type, subject_ids, subject_type, tags,
			references, metadata, is_deleted
		FROM knowledge_entry
		WHERE id = $1 AND account_id = $2
	`

	// Update query for UpdateRecord
	sqlUpdateRecord = `
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

	// Delete query for DeleteRecord (soft delete)
	sqlDeleteRecord = `
		UPDATE knowledge_entry
		SET is_deleted = true, updated_at = $3
		WHERE id = $1 AND account_id = $2 AND is_deleted = false
	`

	// Restore query for RestoreRecord
	sqlRestoreRecord = `
		UPDATE knowledge_entry
		SET is_deleted = false, updated_at = $3
		WHERE id = $1 AND account_id = $2 AND is_deleted = true
	`

	// Purge query for PurgeRecord
	sqlPurgeRecord = `
		DELETE FROM knowledge_entry
		WHERE id = $1 AND account_id = $2
	`
)

// AddRecord adds a new knowledge record
func (p *PgStore) AddRecord(record Entry) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

	// Validate record
	if record.ID == "" {
		return errors.New("knowledge record must have an ID")
	}

	// Parse ID as UUID
	id, err := uuid.Parse(record.ID)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Check if record already exists
	var exists bool
	err = p.crudDB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM knowledge_entry 
			WHERE id = $1 AND account_id = $2
		)
	`, id, p.accountID).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	if exists {
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

	// Serialize References to JSONB
	refsJSON, err := json.Marshal(record.References)
	if err != nil {
		return fmt.Errorf("failed to serialize references: %w", err)
	}

	// Serialize Metadata to JSONB
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Execute insert
	_, err = p.crudDB.Exec(
		sqlInsertRecord,
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

	if err != nil {
		return fmt.Errorf("failed to insert record: %w", err)
	}

	return nil
}

// GetRecord retrieves a knowledge record by ID
func (p *PgStore) GetRecord(id string) (Entry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return Entry{}, errors.New("store not initialized")
	}

	// Parse ID as UUID
	recordID, err := uuid.Parse(id)
	if err != nil {
		return Entry{}, fmt.Errorf("invalid record ID format: %w", err)
	}

	var record Entry
	var refsJSON, metadataJSON []byte
	var expiresAtNull sql.NullTime
	var sourceIDNull, sourceTypeNull, ownerIDNull, ownerTypeNull, subjectTypeNull sql.NullString
	var subjectIDs, tags []string
	var isDeleted bool

	// Execute query
	err = p.crudDB.QueryRow(
		sqlGetRecord,
		recordID,
		p.accountID,
	).Scan(
		&id, // We'll overwrite the ID with the correct format
		&record.Category,
		&record.ContentType,
		&record.Content,
		&record.Importance,
		&record.CreatedAt,
		&record.UpdatedAt,
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
		if err == sql.ErrNoRows {
			return Entry{}, fmt.Errorf("knowledge record with ID %s not found", id)
		}
		return Entry{}, fmt.Errorf("failed to retrieve record: %w", err)
	}

	// Set ID in the correct format (UUID with dashes)
	record.ID = id

	// Deserialize References from JSONB
	if len(refsJSON) > 0 {
		if err := json.Unmarshal(refsJSON, &record.References); err != nil {
			return Entry{}, fmt.Errorf("failed to deserialize references: %w", err)
		}
	}

	// Deserialize Metadata from JSONB
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &record.Metadata); err != nil {
			return Entry{}, fmt.Errorf("failed to deserialize metadata: %w", err)
		}
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

	// If the record is deleted and we didn't specify to include deleted records,
	// return an error indicating the record is deleted
	if isDeleted {
		return Entry{}, fmt.Errorf("knowledge record with ID %s is deleted", id)
	}

	return record, nil
}

// UpdateRecord updates an existing knowledge record
func (p *PgStore) UpdateRecord(record Entry) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

	// Validate record
	if record.ID == "" {
		return errors.New("knowledge record must have an ID")
	}

	// Parse ID as UUID
	id, err := uuid.Parse(record.ID)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Check if record exists
	var exists bool
	err = p.crudDB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM knowledge_entry 
			WHERE id = $1 AND account_id = $2 AND is_deleted = false
		)
	`, id, p.accountID).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("knowledge record with ID %s not found", record.ID)
	}

	// Update timestamp
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now()
	}

	// Serialize References to JSONB
	refsJSON, err := json.Marshal(record.References)
	if err != nil {
		return fmt.Errorf("failed to serialize references: %w", err)
	}

	// Serialize Metadata to JSONB
	metadataJSON, err := json.Marshal(record.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Execute update
	result, err := p.crudDB.Exec(
		sqlUpdateRecord,
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

	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	// Check number of rows affected
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking update result: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("knowledge record with ID %s not found", record.ID)
	}

	return nil
}

// DeleteRecord marks a record as deleted (soft delete)
func (p *PgStore) DeleteRecord(id string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

	// Parse ID as UUID
	recordID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Execute soft delete
	result, err := p.crudDB.Exec(
		sqlDeleteRecord,
		recordID,
		p.accountID,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	// Check number of rows affected
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking delete result: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	return nil
}

// RestoreRecord restores a deleted record
func (p *PgStore) RestoreRecord(id string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

	// Parse ID as UUID
	recordID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Execute restore
	result, err := p.crudDB.Exec(
		sqlRestoreRecord,
		recordID,
		p.accountID,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to restore record: %w", err)
	}

	// Check number of rows affected
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking restore result: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("deleted knowledge record with ID %s not found", id)
	}

	return nil
}

// PurgeRecord permanently deletes a record
func (p *PgStore) PurgeRecord(id string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return errors.New("store not initialized")
	}

	// Parse ID as UUID
	recordID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid record ID format: %w", err)
	}

	// Execute purge
	result, err := p.crudDB.Exec(
		sqlPurgeRecord,
		recordID,
		p.accountID,
	)

	if err != nil {
		return fmt.Errorf("failed to purge record: %w", err)
	}

	// Check number of rows affected
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking purge result: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("knowledge record with ID %s not found", id)
	}

	return nil
}

// Helper functions for handling null values

// nullStringValue returns a sql.NullString for a string that might be empty
func nullStringValue(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullTimeValue returns a sql.NullTime for a time that might be zero
func nullTimeValue(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
