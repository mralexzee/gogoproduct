package knowledge

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Environment variable constants
const (
	// Connection strings
	EnvPgCrudConnStr = "GGP_KB_PGSQL_CRUD_CN"
	EnvPgDdlConnStr  = "GGP_KB_PGSQL_DDL_CN"

	// Connection pool settings
	EnvPgMaxConns     = "GGP_KB_PGSQL_MAX_CONNS"
	EnvPgIdleConns    = "GGP_KB_PGSQL_IDLE_CONNS"
	EnvPgConnLifetime = "GGP_KB_PGSQL_CONN_LIFETIME"

	// Default values
	DefaultMaxConns     = 10
	DefaultIdleConns    = 5
	DefaultConnLifetime = 3600
)

// SQL statements for knowledge entry table
const (
	// Table name
	TableKnowledgeEntry = "knowledge_entry"
	TableSchemaVersion  = "knowledge_schema_version"

	// Schema version
	CurrentSchemaVersion = 1
)

// PgConfig holds PostgreSQL configuration options
type PgConfig struct {
	CrudConnStr  string
	DdlConnStr   string
	MaxConns     int
	IdleConns    int
	ConnLifetime time.Duration
	AccountID    string
}

// PgStore implements Store interface using PostgreSQL
type PgStore struct {
	config      PgConfig
	crudDB      *sql.DB
	ddlDB       *sql.DB
	accountID   string
	initialized bool
	mu          sync.RWMutex
}

// NewPgStore creates a new PostgreSQL knowledge store
func NewPgStore(config PgConfig) (*PgStore, error) {
	// Validate account ID
	if config.AccountID == "" {
		return nil, errors.New("account ID cannot be empty")
	}

	accID, err := uuid.Parse(config.AccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID format: %w", err)
	}

	if accID == uuid.Nil {
		return nil, errors.New("account ID cannot be nil UUID")
	}

	if config.MaxConns <= 0 {
		config.MaxConns = DefaultMaxConns
	}

	if config.IdleConns <= 0 {
		config.IdleConns = DefaultIdleConns
	}

	if config.ConnLifetime <= 0 {
		config.ConnLifetime = time.Duration(time.Second * DefaultConnLifetime)
	}

	if config.CrudConnStr == "" {
		return nil, errors.New("CRUD connection string cannot be empty")
	}

	if config.DdlConnStr == "" {
		return nil, errors.New("DDL connection string cannot be empty")
	}

	store := &PgStore{
		config:      config,
		accountID:   accID.String(),
		initialized: false,
	}

	return store, nil
}

// Open initializes connections to PostgreSQL and runs migrations if needed
func (p *PgStore) Open() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize CRUD DB connection
	crudDB, err := sql.Open("postgres", p.config.CrudConnStr)
	if err != nil {
		return fmt.Errorf("failed to open CRUD database connection: %w", err)
	}

	// Set connection pool parameters
	crudDB.SetMaxOpenConns(p.config.MaxConns)
	crudDB.SetMaxIdleConns(p.config.IdleConns)
	crudDB.SetConnMaxLifetime(p.config.ConnLifetime)

	// Verify connection
	if err := crudDB.Ping(); err != nil {
		crudDB.Close()
		return fmt.Errorf("failed to ping CRUD database: %w", err)
	}

	// Initialize DDL DB connection
	ddlDB, err := sql.Open("postgres", p.config.DdlConnStr)
	if err != nil {
		crudDB.Close()
		return fmt.Errorf("failed to open DDL database connection: %w", err)
	}

	// Set connection pool parameters for DDL connection
	ddlDB.SetMaxOpenConns(p.config.MaxConns)
	ddlDB.SetMaxIdleConns(p.config.IdleConns)
	ddlDB.SetConnMaxLifetime(p.config.ConnLifetime)

	// Verify connection
	if err := ddlDB.Ping(); err != nil {
		crudDB.Close()
		ddlDB.Close()
		return fmt.Errorf("failed to ping DDL database: %w", err)
	}

	p.crudDB = crudDB
	p.ddlDB = ddlDB

	// Run migrations
	if err := p.runMigrations(); err != nil {
		crudDB.Close()
		ddlDB.Close()
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	p.initialized = true
	return nil
}

// Close closes database connections
func (p *PgStore) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return nil
	}

	// Close CRUD connection
	if p.crudDB != nil {
		if err := p.crudDB.Close(); err != nil {
			return fmt.Errorf("error closing CRUD database connection: %w", err)
		}
		p.crudDB = nil
	}

	// Close DDL connection
	if p.ddlDB != nil {
		if err := p.ddlDB.Close(); err != nil {
			return fmt.Errorf("error closing DDL database connection: %w", err)
		}
		p.ddlDB = nil
	}

	p.initialized = false
	return nil
}

// Flush is a no-op for PostgreSQL store since all operations are immediate
func (p *PgStore) Flush() error {
	// No-op for PostgreSQL storage
	return nil
}

// Info provides implementation-specific information about the PostgreSQL knowledge store
func (p *PgStore) Info() (map[string]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, errors.New("store not initialized")
	}

	info := make(map[string]string)

	// Add basic implementation info
	info["implementation"] = "PostgreSQLStore"
	info["account_id"] = p.accountID
	info["persistent"] = "true"

	// Get record count
	var recordCount int
	err := p.crudDB.QueryRow("SELECT COUNT(*) FROM "+TableKnowledgeEntry+" WHERE account_id = $1 AND is_deleted = false",
		p.accountID).Scan(&recordCount)

	if err != nil {
		info["record_count"] = "error"
	} else {
		info["record_count"] = fmt.Sprintf("%d", recordCount)
	}

	// Get deleted record count
	var deletedCount int
	err = p.crudDB.QueryRow("SELECT COUNT(*) FROM "+TableKnowledgeEntry+" WHERE account_id = $1 AND is_deleted = true",
		p.accountID).Scan(&deletedCount)

	if err != nil {
		info["deleted_count"] = "error"
	} else {
		info["deleted_count"] = fmt.Sprintf("%d", deletedCount)
	}

	return info, nil
}
