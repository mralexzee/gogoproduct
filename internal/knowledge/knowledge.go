package knowledge

import "time"

// Category constants
const (
	CategoryFact     = "fact"     // Hard rules: tech stack choices, company policies, established truths
	CategoryMessage  = "message"  // Individual messages in conversations between agents or humans
	CategoryDecision = "decision" // Decisions made with context, reasoning, and authority
	CategoryAction   = "action"   // Records of actions taken: "created project", "deployed service"
)

// ContentType constants
const (
	ContentTypeJSON     = "application/json"         // JSON content type
	ContentTypeText     = "text/plain"               // Text content type
	ContentTypeYAML     = "application/yaml"         // YAML content type
	ContentTypeXML      = "application/xml"          // XML content type
	ContentTypeCSV      = "text/csv"                 // CSV content type
	ContentTypeBinary   = "application/octet-stream" // Binary content type
	ContentTypeHTML     = "text/html"                // HTML content type
	ContentTypeMarkdown = "text/markdown"            // Markdown content type
)

// Importance levels for knowledge records
const (
	ImportanceNone     = 0   // No special importance, routine information
	ImportanceLow      = 25  // Somewhat important, but not critical
	ImportanceMedium   = 50  // Moderately important information
	ImportanceHigh     = 75  // Highly important information
	ImportanceCritical = 100 // Critical, must-not-forget information
)

// Reference represents a reference to another knowledge record
type Reference struct {
	ID   string
	Type string
}

// Entry represents a single knowledge entry in the system
type Entry struct {
	ID          string            // Unique identifier
	Category    string            // High-level category: "fact", "message", "decision", "action"
	ContentType string            // MIME type: "application/json", "text/plain", etc.
	Content     []byte            // The actual content in binary form
	Importance  int               // Importance level: 1 (low) to 3 (high)
	CreatedAt   time.Time         // When this knowledge was created
	UpdatedAt   time.Time         // When this knowledge was last modified
	ExpiresAt   time.Time         // When this knowledge will expire
	SourceID    string            // Where this knowledge came from
	SourceType  string            // Type of source: "chat", "api", "observation", etc.
	OwnerID     string            // Who created/owns this knowledge
	OwnerType   string            // Type of owner: "agent", "human", "company", "product", "tool"
	SubjectIDs  []string          // Who/what this knowledge is about (can be multiple)
	SubjectType string            // Type of subject: "human", "project", "company", etc.
	Tags        []string          // Quick categorization for indexing/retrieval
	References  []Reference       // Other knowledge IDs this knowledge references
	Metadata    map[string]string // Flexible key-value pairs for additional context
}

// FilterOperator defines the type of logical operation to perform
type FilterOperator string

// FilterOperator constants
const (
	OpAnd FilterOperator = "AND" // Logical AND
	OpOr  FilterOperator = "OR"  // Logical OR
	OpNot FilterOperator = "NOT" // Logical NOT
)

// Condition represents a single filter condition
type Condition struct {
	Field    string      // Name of the field to filter on
	Operator string      // Comparison operator: "=", "!=", ">", "<", "IN", "CONTAINS", etc.
	Value    interface{} // Value to compare against
}

// FilterGroup represents a group of conditions with a logical operator
type FilterGroup struct {
	Operator   FilterOperator // Logical operator for this group
	Conditions []Condition    // List of conditions in this group
	Groups     []FilterGroup  // Nested groups for complex queries (parentheses)
}

// Filter defines the query structure for searching knowledge records
type Filter struct {
	RootGroup      FilterGroup // The root filter group
	Limit          int         // Maximum number of results to return
	Offset         int         // Number of results to skip
	OrderBy        string      // Field to order results by
	OrderDir       string      // Order direction: "ASC" or "DESC"
	IncludeDeleted bool        // Whether to include soft-deleted records
	OnlyDeleted    bool        // Whether to show only deleted records
}

// Store interface for knowledge storage
type Store interface {
	AddRecord(record Entry) error                 // Add a record ot the storage
	GetRecord(id string) (Entry, error)           // Retrieve record by ID
	UpdateRecord(record Entry) error              // Update record
	DeleteRecord(id string) error                 // Delete a record, this is soft delete
	RestoreRecord(id string) error                // Un-delete a record
	PurgeRecord(id string) error                  // Permanent deletion
	SearchRecords(filter Filter) ([]Entry, error) // Generic, full search
	Open() error                                  // Open/Load datastore
	Flush() error                                 // Write any pending data to the storage, no-op in some providers such as knowledge
	Close() error                                 // Closes storage (files/db connections)
	Info() (map[string]string, error)             // Provides implementation specific information
}
