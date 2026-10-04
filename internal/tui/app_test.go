package tui

import (
	"context"
	"testing"

	"github.com/M5Devs/Total-Connect/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

type mockStorageEngine struct {
	remotes []string
	entries map[string][]models.FileItem
}

func newMockStorageEngine() *mockStorageEngine {
	return &mockStorageEngine{
		remotes: []string{"drive", "s3"},
		entries: map[string][]models.FileItem{
			".": {
				{Name: "folder1", IsDir: true},
				{Name: "file1.txt", IsDir: false, Size: 100},
			},
			"folder1": {
				{Name: "subfile.txt", IsDir: false, Size: 50},
			},
		},
	}
}

func (m *mockStorageEngine) ListRemotes(ctx context.Context) ([]string, error) {
	return m.remotes, nil
}

func (m *mockStorageEngine) CreateRemote(ctx context.Context, name string, remoteType string, params map[string]string) error {
	m.remotes = append(m.remotes, name)
	return nil
}

func (m *mockStorageEngine) DeleteRemote(ctx context.Context, name string) error {
	var filtered []string
	for _, r := range m.remotes {
		if r != name {
			filtered = append(filtered, r)
		}
	}
	m.remotes = filtered
	return nil
}

func (m *mockStorageEngine) ListEntries(ctx context.Context, path string) ([]models.FileItem, error) {
	if items, ok := m.entries[path]; ok {
		return items, nil
	}
	return []models.FileItem{}, nil
}

func (m *mockStorageEngine) Copy(ctx context.Context, src, dst string) error {
	return nil
}

func (m *mockStorageEngine) Move(ctx context.Context, src, dst string) error {
	return nil
}

func (m *mockStorageEngine) Delete(ctx context.Context, path string) error {
	return nil
}

func (m *mockStorageEngine) Mkdir(ctx context.Context, path string) error {
	return nil
}

func TestAppFocusSwitching(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	if app.ActiveID != "left" {
		t.Errorf("expected initial active ID 'left', got %q", app.ActiveID)
	}

	// Press tab
	updatedModel, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updatedModel.(AppModel)

	if app.ActiveID != "right" {
		t.Errorf("expected active ID 'right' after Tab, got %q", app.ActiveID)
	}
	if !app.RightPane.IsActive || app.LeftPane.IsActive {
		t.Errorf("expected RightPane active and LeftPane inactive")
	}

	// Press tab again
	updatedModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = updatedModel.(AppModel)

	if app.ActiveID != "left" {
		t.Errorf("expected active ID 'left' after Tab, got %q", app.ActiveID)
	}
}

func TestAppEntriesLoadedMsg(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	msg := EntriesLoadedMsg{
		PaneID: "left",
		Path:   ".",
		Items: []models.FileItem{
			{Name: "a.txt", IsDir: false},
		},
		Err: nil,
	}

	updatedModel, _ := app.Update(msg)
	app = updatedModel.(AppModel)

	if len(app.LeftPane.Items) == 0 {
		t.Errorf("expected left pane items to be populated")
	}
}

func TestAppRemotePickerModalToggle(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	// Simulate 'r' key press
	updatedModel, cmd := app.Update(tea.KeyMsg{Runes: []rune{'r'}, Type: tea.KeyRunes})
	app = updatedModel.(AppModel)

	if cmd == nil {
		t.Fatal("expected non-nil cmd for remote picker fetch")
	}

	// Execute cmd to get msg
	msg := cmd()
	updatedModel, _ = app.Update(msg)
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalRemotePicker {
		t.Errorf("expected ActiveModal to be ModalRemotePicker, got %v", app.ActiveModal)
	}

	// Cancel with Esc
	updatedModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalNone {
		t.Errorf("expected ActiveModal to be ModalNone after Esc, got %v", app.ActiveModal)
	}
}

func TestAppMkdirModalToggle(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	// Press 'n' for Mkdir modal
	updatedModel, _ := app.Update(tea.KeyMsg{Runes: []rune{'n'}, Type: tea.KeyRunes})
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalMkdir {
		t.Errorf("expected ActiveModal ModalMkdir, got %v", app.ActiveModal)
	}

	// Press Esc to cancel
	updatedModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalNone {
		t.Errorf("expected ActiveModal ModalNone after Esc, got %v", app.ActiveModal)
	}
}

func TestAppHelpModalToggle(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	// Press '?' to trigger Help modal
	updatedModel, _ := app.Update(tea.KeyMsg{Runes: []rune{'?'}, Type: tea.KeyRunes})
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalHelp {
		t.Errorf("expected ActiveModal ModalHelp, got %v", app.ActiveModal)
	}

	// Press Esc to close
	updatedModel, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = updatedModel.(AppModel)

	if app.ActiveModal != ModalNone {
		t.Errorf("expected ActiveModal ModalNone after Esc, got %v", app.ActiveModal)
	}
}

func TestAppProgressMsg(t *testing.T) {
	engine := newMockStorageEngine()
	app := NewAppModel(engine)

	prog := ProgressMsg{
		CurrentFile:      "test.txt",
		BytesTransferred: 500,
		TotalBytes:       1000,
		Percentage:       50.0,
	}

	updatedModel, _ := app.Update(prog)
	app = updatedModel.(AppModel)

	if app.StatusBar.Progress == nil {
		t.Fatal("expected status bar progress to be non-nil")
	}
	if app.StatusBar.Progress.CurrentFile != "test.txt" {
		t.Errorf("expected current file test.txt, got %q", app.StatusBar.Progress.CurrentFile)
	}
}
