package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"encoding/json/jsontext"
	"encoding/json/v2"
	"uuid"
)

// Store persists completed cards. FileStore is the current implementation;
// a future database backend only needs to satisfy this interface.
type Store interface {
	// List returns all items oldest-first (UUIDv7 IDs sort chronologically).
	List() ([]SavedItem, error)
	// Save inserts (empty ID) or updates (existing ID) one item and returns
	// the stored copy with ID and CreatedAt assigned.
	Save(item SavedItem) (SavedItem, error)
	// Delete removes one item by ID. Unknown IDs are not an error.
	Delete(id string) error
}

// validIntent reports whether s names a card intent (anything but "none").
func validIntent(s string) bool {
	switch IntentKey(s) {
	case IntentEvent, IntentReminder, IntentTodo, IntentTimer, IntentHabit,
		IntentColor, IntentSplit, IntentExpense, IntentConvert, IntentCalc,
		IntentTravel, IntentPoll, IntentContact, IntentLink, IntentCountdown,
		IntentTimezone, IntentRandom, IntentGoal, IntentNote:
		return true
	default:
		return false
	}
}

// FileStore is a Store backed by a single JSON array file.
type FileStore struct {
	mu   sync.Mutex
	path string
}

// NewFileStore creates a FileStore at path, creating parent directories.
func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("notes store: mkdir: %w", err)
	}
	return &FileStore{path: path}, nil
}

// DefaultStorePath is ~/.local/share/gonotes/notes.json (XDG data dir).
func DefaultStorePath() (string, error) {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("notes store: data dir: %w", err)
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "gonotes", "notes.json"), nil
}

// read assumes mu is held. Missing file means empty store; corrupt file is
// backed up aside and treated as empty rather than fatal.
func (s *FileStore) read() ([]SavedItem, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []SavedItem{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("notes store: read: %w", err)
	}
	var items []SavedItem
	if err := json.Unmarshal(raw, &items); err != nil {
		backup := s.path + ".broken." + time.Now().Format("20060102-150405")
		_ = os.Rename(s.path, backup)
		return []SavedItem{}, nil
	}
	if items == nil {
		items = []SavedItem{}
	}
	return items, nil
}

// write assumes mu is held. Atomic via temp file + rename.
func (s *FileStore) write(items []SavedItem) error {
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf, jsontext.WithIndent("  "))
	if err := json.MarshalEncode(enc, items); err != nil {
		return fmt.Errorf("notes store: encode: %w", err)
	}
	raw := append(buf.Bytes(), '\n')
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "notes-*.json")
	if err != nil {
		return fmt.Errorf("notes store: temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("notes store: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("notes store: close: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("notes store: commit: %w", err)
	}
	return nil
}

// List returns all items oldest-first.
func (s *FileStore) List() ([]SavedItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

// Save inserts or updates one item.
func (s *FileStore) Save(item SavedItem) (SavedItem, error) {
	if !validIntent(string(item.Intent)) {
		return SavedItem{}, fmt.Errorf("notes store: unknown intent %q", item.Intent)
	}
	if strings.TrimSpace(item.Text) == "" {
		return SavedItem{}, fmt.Errorf("notes store: empty text")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return SavedItem{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewV7().String()
		item.CreatedAt = time.Now().UnixMilli()
		items = append(items, item)
	} else {
		uid, err := uuid.Parse(item.ID)
		if err != nil {
			return SavedItem{}, fmt.Errorf("notes store: bad id %q", item.ID)
		}
		item.ID = uid.String() // canonical form
		updated := false
		for i := range items {
			if items[i].ID == item.ID {
				item.CreatedAt = items[i].CreatedAt
				items[i] = item
				updated = true
				break
			}
		}
		if !updated {
			return SavedItem{}, fmt.Errorf("notes store: unknown id %q", item.ID)
		}
	}
	if err := s.write(items); err != nil {
		return SavedItem{}, err
	}
	return item, nil
}

// Delete removes one item. Unknown or malformed IDs are not an error.
func (s *FileStore) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil
	}
	id = uid.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.read()
	if err != nil {
		return err
	}
	kept := items[:0]
	for _, it := range items {
		if it.ID != id {
			kept = append(kept, it)
		}
	}
	if len(kept) == len(items) {
		return nil
	}
	return s.write(kept)
}
