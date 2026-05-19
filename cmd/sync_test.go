package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lmorchard/pocketcasts-to-markdown/internal/pocketcasts"
)

// nopLogger discards every method call. Avoids depending on logrus
// during unit tests.
type nopLogger struct{}

func (nopLogger) Debug(args ...any)                {}
func (nopLogger) Infof(format string, args ...any) {}
func (nopLogger) Warnf(format string, args ...any) {}

// fakeAPI is an httptest server tracking login + history call counts.
// /user/login returns the configured token. /user/history returns 401
// while currentToken == staleToken; once a fresh login bumps it, /user/history
// returns the configured episode payload.
type fakeAPI struct {
	loginCalls   int
	historyCalls int
	staleToken   string
	freshToken   string
	episodes     []pocketcasts.Episode
}

func (f *fakeAPI) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/login":
			f.loginCalls++
			_, _ = io.WriteString(w, `{"token":"`+f.freshToken+`"}`)
		case "/user/history":
			f.historyCalls++
			auth := r.Header.Get("Authorization")
			if auth == "Bearer "+f.staleToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"episodes": f.episodes}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}
}

func TestFetchWithRelogin_NoCachedToken_LogsInOnce(t *testing.T) {
	api := &fakeAPI{
		freshToken: "fresh",
		episodes:   []pocketcasts.Episode{{UUID: "ep-1"}},
	}
	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)

	client := pocketcasts.New(pocketcasts.WithBaseURL(srv.URL))
	saved := ""
	save := func(t string) error { saved = t; return nil }

	eps, err := fetchWithRelogin(context.Background(), nopLogger{}, client, "u", "p", save, client.History)
	if err != nil {
		t.Fatalf("fetchWithRelogin: %v", err)
	}
	if len(eps) != 1 || eps[0].UUID != "ep-1" {
		t.Fatalf("unexpected episodes: %+v", eps)
	}
	if api.loginCalls != 1 || api.historyCalls != 1 {
		t.Fatalf("expected 1 login + 1 history, got %d/%d", api.loginCalls, api.historyCalls)
	}
	if saved != "fresh" {
		t.Fatalf("expected saved token=fresh, got %q", saved)
	}
}

func TestFetchWithRelogin_ValidCachedTokenSkipsLogin(t *testing.T) {
	api := &fakeAPI{
		freshToken: "ignored",
		episodes:   []pocketcasts.Episode{{UUID: "ep-1"}},
	}
	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)

	client := pocketcasts.New(pocketcasts.WithBaseURL(srv.URL))
	client.SetToken("known-good") // not the stale one, so /user/history accepts it
	save := func(string) error { t.Fatal("saveToken should not be called"); return nil }

	if _, err := fetchWithRelogin(context.Background(), nopLogger{}, client, "u", "p", save, client.History); err != nil {
		t.Fatalf("fetchWithRelogin: %v", err)
	}
	if api.loginCalls != 0 {
		t.Fatalf("expected 0 logins, got %d", api.loginCalls)
	}
	if api.historyCalls != 1 {
		t.Fatalf("expected 1 history call, got %d", api.historyCalls)
	}
}

func TestFetchWithRelogin_StaleTokenTriggersReloginAndRetry(t *testing.T) {
	api := &fakeAPI{
		staleToken: "stale",
		freshToken: "fresh",
		episodes:   []pocketcasts.Episode{{UUID: "ep-1"}},
	}
	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)

	client := pocketcasts.New(pocketcasts.WithBaseURL(srv.URL))
	client.SetToken("stale")
	saved := ""
	save := func(t string) error { saved = t; return nil }

	eps, err := fetchWithRelogin(context.Background(), nopLogger{}, client, "u", "p", save, client.History)
	if err != nil {
		t.Fatalf("fetchWithRelogin: %v", err)
	}
	if len(eps) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(eps))
	}
	if api.loginCalls != 1 {
		t.Fatalf("expected 1 login, got %d", api.loginCalls)
	}
	if api.historyCalls != 2 {
		t.Fatalf("expected 2 history calls (stale + retry), got %d", api.historyCalls)
	}
	if saved != "fresh" {
		t.Fatalf("expected saved=fresh, got %q", saved)
	}
}

func TestFetchWithRelogin_StaleTokenNoCredsReturnsActionableError(t *testing.T) {
	api := &fakeAPI{
		staleToken: "stale",
		freshToken: "fresh",
	}
	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)

	client := pocketcasts.New(pocketcasts.WithBaseURL(srv.URL))
	client.SetToken("stale")
	save := func(string) error { t.Fatal("saveToken should not be called"); return nil }

	_, err := fetchWithRelogin(context.Background(), nopLogger{}, client, "", "", save, client.History)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if api.loginCalls != 0 {
		t.Fatalf("expected zero login attempts, got %d", api.loginCalls)
	}
	if !strings.Contains(err.Error(), "login") {
		t.Fatalf("expected error to mention `login`, got %v", err)
	}
}

func TestFetchWithRelogin_LoginFailurePropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	client := pocketcasts.New(pocketcasts.WithBaseURL(srv.URL))
	save := func(string) error { return nil }

	_, err := fetchWithRelogin(context.Background(), nopLogger{}, client, "u", "wrong", save, client.History)
	if err == nil {
		t.Fatal("expected error from failed login")
	}
	var apiErr *pocketcasts.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 401 {
		t.Fatalf("expected 401 APIError under wrapping, got %v", err)
	}
}
