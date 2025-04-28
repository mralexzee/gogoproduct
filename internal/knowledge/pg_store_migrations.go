package knowledge

import (
	"fmt"
	"log"
)

// Migration represents a database schema migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// List of migrations in order
var migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema creation",
		SQL: `
CREATE TABLE IF NOT EXISTS knowledge_schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS knowledge_entry (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL,
    category TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content BYTEA,
    importance INTEGER,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    source_id TEXT,
    source_type TEXT,
    owner_id TEXT,
    owner_type TEXT,
    subject_ids TEXT[],
    subject_type TEXT,
    tags TEXT[],
    references JSONB,
    metadata JSONB,
    is_deleted BOOLEAN DEFAULT FALSE,
    
    CONSTRAINT knowledge_entry_account_id_id_unique UNIQUE (account_id, id)
);

CREATE INDEX IF NOT EXISTS knowledge_entry_account_id_idx ON knowledge_entry(account_id);
CREATE INDEX IF NOT EXISTS knowledge_entry_account_category_idx ON knowledge_entry(account_id, category);
CREATE INDEX IF NOT EXISTS knowledge_entry_is_deleted_idx ON knowledge_entry(account_id, is_deleted);
CREATE INDEX IF NOT EXISTS knowledge_entry_tags_idx ON knowledge_entry USING GIN(tags);
CREATE INDEX IF NOT EXISTS knowledge_entry_metadata_idx ON knowledge_entry USING GIN(metadata);
`,
	},
	// Add future migrations here...
}

// runMigrations checks the current schema version and applies necessary migrations
func (p *PgStore) runMigrations() error {
	// This operation requires the DDL-capable connection
	db := p.ddlDB

	// Check if schema version table exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_tables 
			WHERE schemaname = 'public' 
			AND tablename = 'knowledge_schema_version'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if schema version table exists: %w", err)
	}

	// Get current version
	currentVersion := 0
	if exists {
		err = db.QueryRow(`
			SELECT COALESCE(MAX(version), 0) 
			FROM knowledge_schema_version
		`).Scan(&currentVersion)

		if err != nil {
			return fmt.Errorf("failed to get current schema version: %w", err)
		}
	}

	// Apply needed migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			// Skip already applied migrations
			continue
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Description)

		// Start a transaction for the migration
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %d: %w", migration.Version, err)
		}

		// Execute the migration
		_, err = tx.Exec(migration.SQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		// Record the migration in the version table
		_, err = tx.Exec(`
			INSERT INTO knowledge_schema_version (version, description)
			VALUES ($1, $2)
		`, migration.Version, migration.Description)

		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("Successfully applied migration %d", migration.Version)
	}

	return nil
}

// validateSchema checks if the database schema is compatible with the current version
func (p *PgStore) validateSchema() error {
	var version int
	err := p.ddlDB.QueryRow(`
		SELECT COALESCE(MAX(version), 0)
		FROM knowledge_schema_version
	`).Scan(&version)

	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	if version < CurrentSchemaVersion {
		return fmt.Errorf("database schema version %d is older than required version %d",
			version, CurrentSchemaVersion)
	}

	// Check if knowledge_entry table exists and has required columns
	rows, err := p.ddlDB.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = 'knowledge_entry'
	`)

	if err != nil {
		return fmt.Errorf("failed to query table schema: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return fmt.Errorf("failed to scan column name: %w", err)
		}
		columns[column] = true
	}

	// Validate essential columns
	requiredColumns := []string{
		"id", "account_id", "category", "content_type", "content",
		"created_at", "updated_at", "is_deleted",
	}

	for _, col := range requiredColumns {
		if !columns[col] {
			return fmt.Errorf("required column '%s' missing from knowledge_entry table", col)
		}
	}

	return nil
}
