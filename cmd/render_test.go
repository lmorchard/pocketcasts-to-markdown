package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestParseSinceFlag(t *testing.T) {
	now := time.Date(2026, 5, 18, 20, 0, 0, 0, time.UTC)

	cases := []struct {
		in      string
		wantNil bool
		check   func(t *testing.T, got *time.Time)
	}{
		{in: "", wantNil: true},
		{in: "168h", check: func(t *testing.T, got *time.Time) {
			want := now.Add(-168 * time.Hour)
			if !got.Equal(want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}},
		{in: "2026-04-01", check: func(t *testing.T, got *time.Time) {
			if got.Year() != 2026 || got.Month() != time.April || got.Day() != 1 {
				t.Errorf("got %v", got)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := parseSinceFlag(c.in, now)
			if err != nil {
				t.Fatalf("parseSinceFlag(%q): %v", c.in, err)
			}
			if c.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected non-nil time")
			}
			c.check(t, got)
		})
	}
}

func TestParseSinceFlag_Invalid(t *testing.T) {
	if _, err := parseSinceFlag("not a duration or date", time.Now()); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseUntilFlag(t *testing.T) {
	got, err := parseUntilFlag("2026-04-01")
	if err != nil {
		t.Fatalf("parseUntilFlag: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Hour() != 23 || got.Minute() != 59 {
		t.Errorf("expected end-of-day, got %v", got)
	}

	if got, err := parseUntilFlag(""); got != nil || err != nil {
		t.Errorf("empty input should return nil/nil, got %v/%v", got, err)
	}

	if _, err := parseUntilFlag("garbage"); err == nil {
		t.Error("expected error for garbage input")
	}
}

func TestParseIncludeFlag(t *testing.T) {
	cases := []struct {
		in              string
		wantHist, wantS bool
		wantErr         bool
	}{
		{"history,starred", true, true, false},
		{"history", true, false, false},
		{"starred", false, true, false},
		{" history , starred ", true, true, false},
		{"history,", true, false, false}, // trailing comma OK
		{"", false, false, true},
		{"nope", false, false, true},
		{"history,nope", false, false, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			h, s, err := parseIncludeFlag(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", c.in)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if h != c.wantHist || s != c.wantS {
				t.Errorf("got (history=%v, starred=%v), want (history=%v, starred=%v)", h, s, c.wantHist, c.wantS)
			}
		})
	}
}

func TestParseIncludeFlag_ErrorMessages(t *testing.T) {
	_, _, err := parseIncludeFlag("foobar")
	if err == nil || !strings.Contains(err.Error(), "foobar") {
		t.Errorf("expected error to mention 'foobar', got %v", err)
	}
}
