package main

import "fmt"

// NotesService exposes the card store to the frontend (mirrors the
// localStorage API in frontend/src/lib/savedItems.ts). The frontend keeps
// working unchanged: same shapes, Go-owned IDs.
type NotesService struct {
	store Store
}

// NewNotesService builds a NotesService on a file store at the default path.
func NewNotesService() (*NotesService, error) {
	path, err := DefaultStorePath()
	if err != nil {
		return nil, err
	}
	store, err := NewFileStore(path)
	if err != nil {
		return nil, err
	}
	return &NotesService{store: store}, nil
}

// List returns all saved cards, oldest first.
func (n *NotesService) List() ([]SavedItem, error) {
	return n.store.List()
}

// Save inserts (empty ID) or updates one card and returns the stored copy.
func (n *NotesService) Save(item SavedItem) (SavedItem, error) {
	stored, err := n.store.Save(item)
	if err != nil {
		return SavedItem{}, fmt.Errorf("notes: %w", err)
	}
	return stored, nil
}

// Delete removes one card. Unknown IDs are not an error.
func (n *NotesService) Delete(id string) error {
	if err := n.store.Delete(id); err != nil {
		return fmt.Errorf("notes: %w", err)
	}
	return nil
}
