package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// GetKV returns the value for key. The second return value is false when
// the key does not exist (no error returned in that case).
func (db *DB) GetKV(key string) (string, bool, error) {
	var v string
	err := db.conn.QueryRow("SELECT value FROM kv WHERE key = ?", key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get kv %q: %w", key, err)
	}
	return v, true, nil
}

// SetKV inserts or updates a key/value pair.
func (db *DB) SetKV(key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO kv (key, value, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value      = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	if err != nil {
		return fmt.Errorf("set kv %q: %w", key, err)
	}
	return nil
}

// DeleteKV removes a key. It is not an error if the key does not exist.
func (db *DB) DeleteKV(key string) error {
	_, err := db.conn.Exec("DELETE FROM kv WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete kv %q: %w", key, err)
	}
	return nil
}
