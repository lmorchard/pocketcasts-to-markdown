package database

import (
	"path/filepath"
	"testing"
)

// newTestDB opens a fresh DB in a t.TempDir() and registers cleanup.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
