package memory

import (
	"fmt"
	"path/filepath"
)

// Info provides implementation-specific information about the file memory store
// This method is required by the MemoryStore interface
func (f *FileMemoryStore) Info() (map[string]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	info := make(map[string]string)

	// Add basic implementation info
	info["implementation"] = "FileMemoryStore"
	info["file_path"] = f.filename
	info["file_name"] = filepath.Base(f.filename)
	info["record_count"] = fmt.Sprintf("%d", len(f.records))
	info["deleted_count"] = fmt.Sprintf("%d", len(f.deletedRecs))
	info["is_dirty"] = fmt.Sprintf("%t", f.isDirty)

	return info, nil
}
