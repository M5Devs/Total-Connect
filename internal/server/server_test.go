package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/M5Devs/Total-Connect/internal/models"
)

type mockStorageEngine struct {
	mu      sync.Mutex
	remotes []string
	entries map[string][]models.FileItem
}

func newMockEngine() *mockStorageEngine {
	return &mockStorageEngine{
		remotes: []string{"drive:", "s3:"},
		entries: map[string][]models.FileItem{
			".": {
				{Name: "file1.txt", Path: "file1.txt", Size: 100, ModTime: time.Now(), IsDir: false},
				{Name: "docs", Path: "docs", Size: 0, ModTime: time.Now(), IsDir: true},
			},
			"docs": {
				{Name: "readme.md", Path: "docs/readme.md", Size: 50, ModTime: time.Now(), IsDir: false},
			},
			"drive:": {
				{Name: "remote_file.pdf", Path: "drive:remote_file.pdf", Size: 500, ModTime: time.Now(), IsDir: false},
				{Name: "cloud_folder", Path: "drive:cloud_folder", Size: 0, ModTime: time.Now(), IsDir: true},
			},
		},
	}
}

func (m *mockStorageEngine) ListRemotes(ctx context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.remotes, nil
}

func (m *mockStorageEngine) ListEntries(ctx context.Context, remotePath string) ([]models.FileItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	items, exists := m.entries[remotePath]
	if !exists {
		return []models.FileItem{}, nil
	}
	return items, nil
}

func (m *mockStorageEngine) Copy(ctx context.Context, src, dst string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	srcDir, srcName := path.Split(src)
	srcDir = strings.TrimSuffix(srcDir, "/")
	if srcDir == "" {
		srcDir = "."
	}

	dstDir, dstName := path.Split(dst)
	dstDir = strings.TrimSuffix(dstDir, "/")
	if dstDir == "" {
		dstDir = "."
	}

	items := m.entries[srcDir]
	var found *models.FileItem
	for _, it := range items {
		if it.Name == srcName {
			found = &it
			break
		}
	}

	if found == nil {
		return fmt.Errorf("source file %q not found", src)
	}

	newItem := *found
	newItem.Name = dstName
	newItem.Path = dst

	m.entries[dstDir] = append(m.entries[dstDir], newItem)
	return nil
}

func (m *mockStorageEngine) Move(ctx context.Context, src, dst string) error {
	if err := m.Copy(ctx, src, dst); err != nil {
		return err
	}
	return m.Delete(ctx, src)
}

func (m *mockStorageEngine) Delete(ctx context.Context, p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir, name := path.Split(p)
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		dir = "."
	}

	items := m.entries[dir]
	var filtered []models.FileItem
	for _, it := range items {
		if it.Name != name {
			filtered = append(filtered, it)
		}
	}
	m.entries[dir] = filtered
	return nil
}

func (m *mockStorageEngine) Mkdir(ctx context.Context, p string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir, name := path.Split(p)
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		dir = "."
	}

	m.entries[dir] = append(m.entries[dir], models.FileItem{
		Name:    name,
		Path:    p,
		Size:    0,
		ModTime: time.Now(),
		IsDir:   true,
	})
	return nil
}

func TestServer_StaticFiles(t *testing.T) {
	srv := NewServer(newMockEngine())
	ts := httptest.NewServer(srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("failed GET /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", res.StatusCode)
	}
}

func TestServer_ListRemotes(t *testing.T) {
	srv := NewServer(newMockEngine())
	ts := httptest.NewServer(srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/remotes")
	if err != nil {
		t.Fatalf("failed GET /api/remotes: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var data struct {
		Remotes []string `json:"remotes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(data.Remotes) != 2 || data.Remotes[0] != "drive:" || data.Remotes[1] != "s3:" {
		t.Errorf("unexpected remotes output: %v", data.Remotes)
	}
}

func TestServer_ListEntries(t *testing.T) {
	srv := NewServer(newMockEngine())
	ts := httptest.NewServer(srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/entries?remote=local&path=.")
	if err != nil {
		t.Fatalf("failed GET /api/entries: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var items []models.FileItem
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode entries: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Verify folder first sorting
	if !items[0].IsDir || items[0].Name != "docs" {
		t.Errorf("expected docs folder first, got %v", items[0])
	}
	if items[1].IsDir || items[1].Name != "file1.txt" {
		t.Errorf("expected file1.txt second, got %v", items[1])
	}
}

func TestServer_CopyMoveMkdirDelete(t *testing.T) {
	mockEng := newMockEngine()
	srv := NewServer(mockEng)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Mkdir
	mkdirBody, _ := json.Marshal(map[string]string{
		"remote": "local",
		"path":   "new_folder",
	})
	res, err := http.Post(ts.URL+"/api/mkdir", "application/json", bytes.NewBuffer(mkdirBody))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("mkdir failed: %v, status: %d", err, res.StatusCode)
	}

	// 2. Copy
	copyBody, _ := json.Marshal(map[string]string{
		"srcRemote": "local",
		"srcPath":   "file1.txt",
		"dstRemote": "local",
		"dstPath":   "file1_copy.txt",
	})
	res, err = http.Post(ts.URL+"/api/copy", "application/json", bytes.NewBuffer(copyBody))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("copy failed: %v, status: %d", err, res.StatusCode)
	}

	// 3. Move
	moveBody, _ := json.Marshal(map[string]string{
		"srcRemote": "local",
		"srcPath":   "file1_copy.txt",
		"dstRemote": "local",
		"dstPath":   "file1_moved.txt",
	})
	res, err = http.Post(ts.URL+"/api/move", "application/json", bytes.NewBuffer(moveBody))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("move failed: %v, status: %d", err, res.StatusCode)
	}

	// 4. Delete
	delBody, _ := json.Marshal(map[string]string{
		"remote": "local",
		"path":   "file1_moved.txt",
	})
	res, err = http.Post(ts.URL+"/api/delete", "application/json", bytes.NewBuffer(delBody))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("delete failed: %v, status: %d", err, res.StatusCode)
	}
}
