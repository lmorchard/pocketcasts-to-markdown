package database

import "testing"

func TestKV_GetMissingReturnsFoundFalse(t *testing.T) {
	db := newTestDB(t)

	v, found, err := db.GetKV("nope")
	if err != nil {
		t.Fatalf("GetKV: %v", err)
	}
	if found || v != "" {
		t.Fatalf("expected empty/false, got %q/%v", v, found)
	}
}

func TestKV_SetGetUpdate(t *testing.T) {
	db := newTestDB(t)

	if err := db.SetKV("auth.token", "abc"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, found, err := db.GetKV("auth.token")
	if err != nil || !found || v != "abc" {
		t.Fatalf("got (%q, %v, %v); want (abc, true, nil)", v, found, err)
	}

	if err := db.SetKV("auth.token", "xyz"); err != nil {
		t.Fatalf("update: %v", err)
	}
	v, _, err = db.GetKV("auth.token")
	if err != nil || v != "xyz" {
		t.Fatalf("got %q after update; want xyz", v)
	}
}

func TestKV_Delete(t *testing.T) {
	db := newTestDB(t)

	if err := db.SetKV("k", "v"); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteKV("k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, found, _ := db.GetKV("k"); found {
		t.Fatal("expected deleted key to not be found")
	}

	// Deleting a missing key is a no-op.
	if err := db.DeleteKV("missing"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}
