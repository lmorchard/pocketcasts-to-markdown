package pocketcasts

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError carries an HTTP status from the Pocket Casts API so callers
// can distinguish recoverable conditions (e.g. 401 → relogin and retry)
// from other failures.
type APIError struct {
	Op         string // e.g. "login", "history"
	StatusCode int
	Body       string // best-effort capture of the response body
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("pocketcasts %s: HTTP %d: %s", e.Op, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("pocketcasts %s: HTTP %d", e.Op, e.StatusCode)
}

// IsUnauthorized reports whether err is a *APIError with status 401.
// Useful for the sync command's "try cached token, retry after relogin"
// flow.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// ErrNotAuthenticated is returned by calls that require a token when one
// hasn't been set. Callers must Login or SetToken first.
var ErrNotAuthenticated = errors.New("pocketcasts: not authenticated (call Login or SetToken)")
