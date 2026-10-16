package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBookmarksManagement(t *testing.T) {
	tmpDir := t.TempDir()
	bmFile := filepath.Join(tmpDir, "bookmarks.json")

	// Set HOME env to temp dir so GetBookmarksPath uses it
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	if got := GetBookmarksPath(); got != filepath.Join(tmpDir, ".config", "total-connect", "bookmarks.json") {
		t.Errorf("unexpected GetBookmarksPath: %s", got)
	}

	// 1. Initial Load when file does not exist
	bms, err := LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks failed on non-existent file: %v", err)
	}
	if len(bms) != 0 {
		t.Errorf("expected 0 bookmarks initially, got %d", len(bms))
	}

	// 2. Add Bookmarks
	bm1 := Bookmark{
		Name:   "SourceForge Web",
		Remote: "SourceForge",
		Path:   "/home/project-web/fooproject/htdocs",
	}
	if err := AddBookmark(bm1); err != nil {
		t.Fatalf("AddBookmark 1 failed: %v", err)
	}

	bm2 := Bookmark{
		Name:   "Local Downloads",
		Remote: "local",
		Path:   "/home/user/Downloads",
	}
	if err := AddBookmark(bm2); err != nil {
		t.Fatalf("AddBookmark 2 failed: %v", err)
	}

	bms, err = LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks failed: %v", err)
	}
	if len(bms) != 2 {
		t.Fatalf("expected 2 bookmarks, got %d", len(bms))
	}

	// 3. Update Existing Bookmark
	bm1Update := Bookmark{
		Name:   "SourceForge Web",
		Remote: "SourceForge",
		Path:   "/home/project-web/fooproject/htdocs/v2",
	}
	if err := AddBookmark(bm1Update); err != nil {
		t.Fatalf("AddBookmark update failed: %v", err)
	}

	bms, err = LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks failed: %v", err)
	}
	if len(bms) != 2 {
		t.Errorf("expected still 2 bookmarks after update, got %d", len(bms))
	}
	if bms[0].Path != "/home/project-web/fooproject/htdocs/v2" {
		t.Errorf("expected updated path, got %s", bms[0].Path)
	}

	// 4. Delete Bookmark
	if err := DeleteBookmark("Local Downloads"); err != nil {
		t.Fatalf("DeleteBookmark failed: %v", err)
	}

	bms, err = LoadBookmarks()
	if err != nil {
		t.Fatalf("LoadBookmarks failed: %v", err)
	}
	if len(bms) != 1 || bms[0].Name != "SourceForge Web" {
		t.Errorf("unexpected bookmarks after delete: %+v", bms)
	}

	_ = bmFile
}
