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
	ID   string `json:"id" xml:"id" yaml:"id"`       // Unique identifier of the referenced record
	Type string `json:"type" xml:"type" yaml:"type"` // Type of the referenced record
}

// Entry represents a single knowledge entry in the system
type Entry struct {
	ID          string            `json:"id" xml:"id" yaml:"id"`                            // Unique identifier
	Category    string            `json:"category" xml:"category" yaml:"category"`          // High-level category: "fact", "message", "decision", "action"
	ContentType string            `json:"contentType" xml:"contentType" yaml:"contentType"` // MIME type: "application/json", "text/plain", etc.
	Content     []byte            `json:"content" xml:"content" yaml:"content"`             // The actual content in binary form
	Importance  int               `json:"importance" xml:"importance" yaml:"importance"`    // Importance level: 1 (low) to 3 (high)
	CreatedAt   time.Time         `json:"createdAt" xml:"createdAt" yaml:"createdAt"`       // When this knowledge was created
	UpdatedAt   time.Time         `json:"updatedAt" xml:"updatedAt" yaml:"updatedAt"`       // When this knowledge was last modified
	ExpiresAt   time.Time         `json:"expiresAt" xml:"expiresAt" yaml:"expiresAt"`       // When this knowledge will expire
	SourceID    string            `json:"sourceId" xml:"sourceId" yaml:"sourceId"`          // Where this knowledge came from
	SourceType  string            `json:"sourceType" xml:"sourceType" yaml:"sourceType"`    // Type of source: "chat", "api", "observation", etc.
	OwnerID     string            `json:"ownerId" xml:"ownerId" yaml:"ownerId"`             // Who created/owns this knowledge
	OwnerType   string            `json:"ownerType" xml:"ownerType" yaml:"ownerType"`       // Type of owner: "agent", "human", "company", "product", "tool"
	SubjectIDs  []string          `json:"subjectIds" xml:"subjectIds" yaml:"subjectIds"`    // Who/what this knowledge is about (can be multiple)
	SubjectType string            `json:"subjectType" xml:"subjectType" yaml:"subjectType"` // Type of subject: "human", "project", "company", etc.
	Tags        []string          `json:"tags" xml:"tags" yaml:"tags"`                      // Quick categorization for indexing/retrieval
	References  []Reference       `json:"references" xml:"references" yaml:"references"`    // Other knowledge IDs this knowledge references
	Metadata    map[string]string `json:"metadata" xml:"metadata" yaml:"metadata"`          // Flexible key-value pairs for additional context
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
	Field    string      `json:"field" xml:"field" yaml:"field"`          // Name of the field to filter on
	Operator string      `json:"operator" xml:"operator" yaml:"operator"` // Comparison operator: "=", "!=", ">", "<", "IN", "CONTAINS", etc.
	Value    interface{} `json:"value" xml:"value" yaml:"value"`          // Value to compare against
}

// FilterGroup represents a group of conditions with a logical operator
type FilterGroup struct {
	Operator   FilterOperator `json:"operator" xml:"operator" yaml:"operator"`       // Logical operator for this group
	Conditions []Condition    `json:"conditions" xml:"conditions" yaml:"conditions"` // List of conditions in this group
	Groups     []FilterGroup  `json:"groups" xml:"groups" yaml:"groups"`             // Nested groups for complex queries (parentheses)
}

// Filter defines the query structure for searching knowledge records
type Filter struct {
	RootGroup      FilterGroup `json:"rootGroup" xml:"rootGroup" yaml:"rootGroup"`                // The root filter group
	Limit          int         `json:"limit" xml:"limit" yaml:"limit"`                            // Maximum number of results to return
	Offset         int         `json:"offset" xml:"offset" yaml:"offset"`                         // Number of results to skip
	OrderBy        string      `json:"orderBy" xml:"orderBy" yaml:"orderBy"`                      // Field to order results by
	OrderDir       string      `json:"orderDir" xml:"orderDir" yaml:"orderDir"`                   // Order direction: "ASC" or "DESC"
	IncludeDeleted bool        `json:"includeDeleted" xml:"includeDeleted" yaml:"includeDeleted"` // Whether to include soft-deleted records
	OnlyDeleted    bool        `json:"onlyDeleted" xml:"onlyDeleted" yaml:"onlyDeleted"`          // Whether to show only deleted records
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
	LoadRecords(records ...Entry) error           // Bulk load records, updating existing ones and adding new ones
	Open() error                                  // Open/Load datastore
	Flush() error                                 // Write any pending data to the storage, no-op in some providers such as knowledge
	Close() error                                 // Closes storage (files/db connections)
	Info() (map[string]string, error)             // Provides implementation specific information
}
