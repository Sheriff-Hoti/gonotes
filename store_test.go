package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *FileStore {
	t.Helper()
	s, err := NewFileStore(filepath.Join(t.TempDir(), "notes.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestFileStoreRoundTrip(t *testing.T) {
	s := testStore(t)

	a, err := s.Save(SavedItem{Intent: IntentTodo, Summary: "2 items", Text: "buy milk, eggs"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || a.CreatedAt == 0 {
		t.Fatalf("expected assigned ID/CreatedAt, got %+v", a)
	}
	b, err := s.Save(SavedItem{Intent: IntentCalc, Summary: "result", Text: "18% of 3450"})
	if err != nil {
		t.Fatal(err)
	}
	if !(a.ID < b.ID) {
		t.Fatalf("UUIDv7 IDs should sort chronologically: %q vs %q", a.ID, b.ID)
	}

	items, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != a.ID || items[1].ID != b.ID {
		t.Fatalf("unexpected list order: %+v", items)
	}

	a.Summary = "3 items"
	updated, err := s.Save(a)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Summary != "3 items" || updated.CreatedAt != a.CreatedAt {
		t.Fatalf("update should keep CreatedAt: %+v", updated)
	}

	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	items, _ = s.List()
	if len(items) != 1 || items[0].ID != b.ID {
		t.Fatalf("unexpected list after delete: %+v", items)
	}
	if err := s.Delete("nonexistent"); err != nil {
		t.Fatalf("unknown delete should be nil: %v", err)
	}
}

func TestFileStoreValidation(t *testing.T) {
	s := testStore(t)
	if _, err := s.Save(SavedItem{Intent: "bogus", Text: "x"}); err == nil {
		t.Fatal("expected intent error")
	}
	if _, err := s.Save(SavedItem{Intent: IntentNote, Text: "  "}); err == nil {
		t.Fatal("expected empty-text error")
	}
	if _, err := s.Save(SavedItem{ID: "missing", Intent: IntentNote, Text: "x"}); err == nil {
		t.Fatal("expected unknown-id error")
	}
}

func TestFileStoreCorruptRecovers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.List()
	if err != nil || len(items) != 0 {
		t.Fatalf("corrupt file should yield empty store: %+v %v", items, err)
	}
	matches, _ := filepath.Glob(path + ".broken.*")
	if len(matches) != 1 {
		t.Fatalf("expected one backup, got %v", matches)
	}
}
