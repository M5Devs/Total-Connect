package core

import (
	"context"
	"testing"
	"time"

	"github.com/M5Devs/Total-Connect/internal/models"
)

type mockStorageEngine struct {
	remotes []string
	items   map[string][]models.FileItem
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

func (m *mockStorageEngine) ListEntries(ctx context.Context, remotePath string) ([]models.FileItem, error) {
	return m.items[remotePath], nil
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

func TestSessionManager(t *testing.T) {
	mockEngine := &mockStorageEngine{
		remotes: []string{"remote1", "remote2"},
		items: map[string][]models.FileItem{
			"/local": {
				{Name: "file1.txt", Path: "/local/file1.txt", Size: 100, ModTime: time.Now(), IsDir: false},
			},
		},
	}

	sm := NewSessionManager(mockEngine)
	if sm.GetEngine() != mockEngine {
		t.Error("expected mockEngine to be set")
	}

	sm.SetPanePath("left", "/local")

	pane, ok := sm.GetPane("left")
	if !ok || pane.CurrentPath != "/local" {
		t.Fatalf("expected left pane path /local, got %+v", pane)
	}

	items, err := sm.RefreshPane(context.Background(), "left")
	if err != nil {
		t.Fatalf("RefreshPane failed: %v", err)
	}

	if len(items) != 1 || items[0].Name != "file1.txt" {
		t.Errorf("unexpected items: %+v", items)
	}
}
