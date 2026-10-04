package tui

import (
	"testing"
	"time"

	"github.com/M5Devs/Total-Connect/internal/models"
)

func TestPaneCursorMovement(t *testing.T) {
	pane := NewPaneModel("test", "/tmp")
	items := []models.FileItem{
		{Name: "file1.txt", IsDir: false},
		{Name: "file2.txt", IsDir: false},
		{Name: "dir1", IsDir: true},
	}
	pane.SetItems(items)

	// Since path is "/tmp", ".." item is prepended at index 0
	if len(pane.Items) != 4 {
		t.Fatalf("expected 4 items (including ..), got %d", len(pane.Items))
	}

	if pane.Cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", pane.Cursor)
	}

	pane.MoveCursorDown()
	if pane.Cursor != 1 {
		t.Errorf("expected cursor 1 after MoveCursorDown, got %d", pane.Cursor)
	}

	pane.MoveCursorUp()
	if pane.Cursor != 0 {
		t.Errorf("expected cursor 0 after MoveCursorUp, got %d", pane.Cursor)
	}

	// Test boundary up
	pane.MoveCursorUp()
	if pane.Cursor != 0 {
		t.Errorf("expected cursor to remain 0, got %d", pane.Cursor)
	}

	// Move to bottom
	for i := 0; i < 10; i++ {
		pane.MoveCursorDown()
	}
	if pane.Cursor != len(pane.Items)-1 {
		t.Errorf("expected cursor at bottom %d, got %d", len(pane.Items)-1, pane.Cursor)
	}
}

func TestTotalCommanderDirectorySorting(t *testing.T) {
	pane := NewPaneModel("test", "/some/dir")

	rawItems := []models.FileItem{
		{Name: "zebra.txt", IsDir: false},
		{Name: "BetaDir", IsDir: true},
		{Name: "apple.txt", IsDir: false},
		{Name: "AlphaDir", IsDir: true},
		{Name: "Banana.txt", IsDir: false},
	}

	pane.SetItems(rawItems)

	if len(pane.Items) != 6 {
		t.Fatalf("expected 6 items, got %d", len(pane.Items))
	}

	expectedNames := []string{"..", "AlphaDir", "BetaDir", "apple.txt", "Banana.txt", "zebra.txt"}
	for i, name := range expectedNames {
		if pane.Items[i].Name != name {
			t.Errorf("at index %d: expected %q, got %q", i, name, pane.Items[i].Name)
		}
	}
}

func TestGetParentPathAndJoinPath(t *testing.T) {
	tests := []struct {
		current  string
		parent   string
		child    string
		joined   string
	}{
		{"/a/b/c", "/a/b", "sub", "/a/b/c/sub"},
		{"drive:folder/sub", "drive:folder", "file.txt", "drive:folder/sub/file.txt"},
		{"drive:", "drive:", "rootdir", "drive:rootdir"},
		{".", ".", "test", "test"},
	}

	for _, tt := range tests {
		gotParent := GetParentPath(tt.current)
		if gotParent != tt.parent {
			t.Errorf("GetParentPath(%q) = %q, want %q", tt.current, gotParent, tt.parent)
		}

		gotJoined := JoinPath(tt.current, tt.child)
		if gotJoined != tt.joined {
			t.Errorf("JoinPath(%q, %q) = %q, want %q", tt.current, tt.child, gotJoined, tt.joined)
		}
	}
}

func TestFormatSize(t *testing.T) {
	if s := formatSize(500); s != "500 B" {
		t.Errorf("formatSize(500) = %q, want '500 B'", s)
	}
	if s := formatSize(2048); s != "2.0 KB" {
		t.Errorf("formatSize(2048) = %q, want '2.0 KB'", s)
	}
}

func TestPaneSelectedItem(t *testing.T) {
	pane := NewPaneModel("test", ".")
	items := []models.FileItem{
		{Name: "doc.pdf", Size: 1024, ModTime: time.Now()},
	}
	pane.SetItems(items)

	sel := pane.SelectedItem()
	if sel == nil {
		t.Fatal("expected non-nil selected item")
	}
	if sel.Name != "doc.pdf" {
		t.Errorf("expected selected item 'doc.pdf', got %q", sel.Name)
	}
}
