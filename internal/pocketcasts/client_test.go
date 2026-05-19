package pocketcasts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient returns a Client pointed at a test server whose handler
// is provided by the caller. The server is closed via t.Cleanup.
func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL))
}

func TestLogin_SuccessSetsToken(t *testing.T) {
	var gotBody loginRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/user/login" {
			t.Fatalf("expected /user/login, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("expected json content-type, got %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"token":"tok-abc"}`)
	})

	tok, err := c.Login(context.Background(), "user@example.com", "pw")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if tok != "tok-abc" || c.Token() != "tok-abc" {
		t.Fatalf("token not set: returned=%q, on-client=%q", tok, c.Token())
	}
	if gotBody.Email != "user@example.com" || gotBody.Password != "pw" || gotBody.Scope != "webplayer" {
		t.Fatalf("login body mismatch: %+v", gotBody)
	}
}

func TestLogin_BadCredentialsReturnsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_credentials"}`, http.StatusUnauthorized)
	})

	_, err := c.Login(context.Background(), "u", "p")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", apiErr.StatusCode)
	}
	if !IsUnauthorized(err) {
		t.Fatal("IsUnauthorized should be true")
	}
	if !strings.Contains(apiErr.Body, "invalid_credentials") {
		t.Fatalf("expected body to contain error string, got %q", apiErr.Body)
	}
}

func TestLogin_MissingTokenInResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})
	if _, err := c.Login(context.Background(), "u", "p"); err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestHistory_RequiresToken(t *testing.T) {
	c := New(WithBaseURL("http://unused.invalid"))
	if _, err := c.History(context.Background()); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestHistory_SendsBearerAndDecodesEpisodes(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-xyz" {
			t.Fatalf("expected Bearer header, got %q", got)
		}
		if r.URL.Path != "/user/history" {
			t.Fatalf("expected /user/history, got %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
			"episodes":[
				{"uuid":"ep-1","title":"Ep One","podcastTitle":"Pod","podcastUuid":"pod-1","url":"https://example.com/1.mp3","published":"2026-05-10T00:00:00Z","playedUpTo":42},
				{"uuid":"ep-2","title":"Ep Two","podcastTitle":"Pod","podcastUuid":"pod-1","published":"2026-05-09T00:00:00Z"}
			]
		}`)
	})
	c.SetToken("tok-xyz")

	eps, err := c.History(context.Background())
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(eps))
	}
	if eps[0].UUID != "ep-1" || eps[0].PlayedUpTo != 42 || eps[0].URL == "" {
		t.Fatalf("episode 0 wrong: %+v", eps[0])
	}
	if eps[1].URL != "" {
		t.Fatalf("episode 1 should have empty URL: %+v", eps[1])
	}
}

func TestHistory_UnauthorizedSurfacedAsAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	c.SetToken("stale-tok")

	_, err := c.History(context.Background())
	if !IsUnauthorized(err) {
		t.Fatalf("expected IsUnauthorized, got %v", err)
	}
}

func TestStarred_UsesStarredPath(t *testing.T) {
	var seen string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		_, _ = io.WriteString(w, `{"episodes":[]}`)
	})
	c.SetToken("tok")
	if _, err := c.Starred(context.Background()); err != nil {
		t.Fatalf("Starred: %v", err)
	}
	if seen != "/user/starred" {
		t.Fatalf("expected /user/starred, got %s", seen)
	}
}

func TestDo_BadJSONReturnsDecodeError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not json`)
	})
	c.SetToken("tok")
	_, err := c.History(context.Background())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}
