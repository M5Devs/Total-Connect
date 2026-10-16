package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var bookmarkMu sync.Mutex

// Bookmark represents a saved shortcut location.
type Bookmark struct {
	Name   string `json:"name"`
	Remote string `json:"remote"`
	Path   string `json:"path"`
}

// GetBookmarksPath returns the path to bookmarks.json in ~/.config/total-connect/
func GetBookmarksPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "total-connect", "bookmarks.json")
}

// LoadBookmarks reads bookmarks from bookmarks.json file.
func LoadBookmarks() ([]Bookmark, error) {
	bookmarkMu.Lock()
	defer bookmarkMu.Unlock()

	bmPath := GetBookmarksPath()
	if _, err := os.Stat(bmPath); os.IsNotExist(err) {
		return []Bookmark{}, nil
	}

	data, err := os.ReadFile(bmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read bookmarks file: %w", err)
	}

	var bookmarks []Bookmark
	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return nil, fmt.Errorf("failed to parse bookmarks json: %w", err)
	}

	return bookmarks, nil
}

// SaveBookmarks writes a list of bookmarks to bookmarks.json file.
func SaveBookmarks(bookmarks []Bookmark) error {
	bookmarkMu.Lock()
	defer bookmarkMu.Unlock()

	bmPath := GetBookmarksPath()
	if err := os.MkdirAll(filepath.Dir(bmPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory for bookmarks: %w", err)
	}

	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal bookmarks: %w", err)
	}

	if err := os.WriteFile(bmPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write bookmarks file: %w", err)
	}

	return nil
}

// AddBookmark appends or updates a bookmark in the bookmarks file.
func AddBookmark(bm Bookmark) error {
	bookmarks, err := LoadBookmarks()
	if err != nil {
		bookmarks = []Bookmark{}
	}

	// Update existing if name or remote+path match
	updated := false
	for i, existing := range bookmarks {
		if existing.Name == bm.Name || (existing.Remote == bm.Remote && existing.Path == bm.Path) {
			bookmarks[i] = bm
			updated = true
			break
		}
	}

	if !updated {
		bookmarks = append(bookmarks, bm)
	}

	return SaveBookmarks(bookmarks)
}

// DeleteBookmark removes a bookmark by index or name.
func DeleteBookmark(name string) error {
	bookmarks, err := LoadBookmarks()
	if err != nil {
		return err
	}

	var filtered []Bookmark
	for _, bm := range bookmarks {
		if bm.Name != name {
			filtered = append(filtered, bm)
		}
	}

	return SaveBookmarks(filtered)
}
