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

func (m *mockStorageEngine) CreateRemote(ctx context.Context, name string, remoteType string, params map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := strings.TrimSuffix(name, ":")
	m.remotes = append(m.remotes, clean+":")
	return nil
}

func (m *mockStorageEngine) GetRemoteConfig(ctx context.Context, name string) (string, map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := strings.TrimSuffix(name, ":")
	for _, r := range m.remotes {
		if strings.TrimSuffix(r, ":") == clean {
			return "mock", map[string]string{"host": "example.com"}, nil
		}
	}
	return "", nil, fmt.Errorf("remote not found")
}

func (m *mockStorageEngine) DeleteRemote(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := strings.TrimSuffix(name, ":")
	var filtered []string
	for _, r := range m.remotes {
		if strings.TrimSuffix(r, ":") != clean {
			filtered = append(filtered, r)
		}
	}
	m.remotes = filtered
	return nil
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

func TestServer_CreateAndDeleteRemote(t *testing.T) {
	mockEng := newMockEngine()
	srv := NewServer(mockEng)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Create Remote
	createReq := map[string]interface{}{
		"name": "MySourceForge",
		"type": "ftp",
		"parameters": map[string]string{
			"host": "frs.sourceforge.net",
			"user": "myuser",
			"pass": "mypassword",
			"port": "21",
			"tls":  "false",
		},
	}
	body, _ := json.Marshal(createReq)
	res, err := http.Post(ts.URL+"/api/remotes/create", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed POST /api/remotes/create: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", res.StatusCode)
	}

	remotes, _ := mockEng.ListRemotes(context.Background())
	found := false
	for _, r := range remotes {
		if strings.HasPrefix(r, "MySourceForge") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected MySourceForge remote in list, got %v", remotes)
	}

	// 2. Delete Remote
	delReq := map[string]string{
		"name": "MySourceForge",
	}
	bodyDel, _ := json.Marshal(delReq)
	resDel, err := http.Post(ts.URL+"/api/remotes/delete", "application/json", bytes.NewBuffer(bodyDel))
	if err != nil {
		t.Fatalf("failed POST /api/remotes/delete: %v", err)
	}
	defer resDel.Body.Close()

	if resDel.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resDel.StatusCode)
	}

	remotes, _ = mockEng.ListRemotes(context.Background())
	for _, r := range remotes {
		if strings.HasPrefix(r, "MySourceForge") {
			t.Fatalf("expected MySourceForge remote to be deleted, still found in %v", remotes)
		}
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

func TestServer_GetRemoteConfig(t *testing.T) {
	mockEng := newMockEngine()
	srv := NewServer(mockEng)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/remotes/config?name=drive:")
	if err != nil {
		t.Fatalf("failed GET /api/remotes/config: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}

	var data struct {
		Name       string            `json:"name"`
		Type       string            `json:"type"`
		Parameters map[string]string `json:"parameters"`
	}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data.Name != "drive:" || data.Type != "mock" {
		t.Errorf("unexpected response: %+v", data)
	}
}

func TestServer_BasicAuth(t *testing.T) {
	srv := NewServer(newMockEngine(), WithAuth("admin:secret"))
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// 1. Request without auth header should fail with 401
	res, err := http.Get(ts.URL + "/api/remotes")
	if err != nil {
		t.Fatalf("failed GET /api/remotes: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", res.StatusCode)
	}

	// 2. Request with invalid credentials should fail with 401
	req, _ := http.NewRequest("GET", ts.URL+"/api/remotes", nil)
	req.SetBasicAuth("admin", "wrong")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed GET with bad auth: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", res.StatusCode)
	}

	// 3. Request with valid credentials should succeed
	req, _ = http.NewRequest("GET", ts.URL+"/api/remotes", nil)
	req.SetBasicAuth("admin", "secret")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed GET with good auth: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", res.StatusCode)
	}
}

func TestServer_HiddenFiles(t *testing.T) {
	mockEng := newMockEngine()
	mockEng.entries["."] = append(mockEng.entries["."], models.FileItem{
		Name:    ".env",
		Path:    ".env",
		Size:    12,
		ModTime: time.Now(),
		IsDir:   false,
	}, models.FileItem{
		Name:    ".git",
		Path:    ".git",
		Size:    0,
		ModTime: time.Now(),
		IsDir:   true,
	})

	srv := NewServer(mockEng)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Default (hidden not set or hidden=false): dotfiles excluded
	res, err := http.Get(ts.URL + "/api/entries?remote=local&path=.")
	if err != nil {
		t.Fatalf("failed GET /api/entries: %v", err)
	}
	var items []models.FileItem
	json.NewDecoder(res.Body).Decode(&items)
	res.Body.Close()

	for _, item := range items {
		if strings.HasPrefix(item.Name, ".") {
			t.Errorf("expected no dotfiles, got %s", item.Name)
		}
	}

	// With hidden=true: dotfiles included
	res, err = http.Get(ts.URL + "/api/entries?remote=local&path=.&hidden=true")
	if err != nil {
		t.Fatalf("failed GET /api/entries: %v", err)
	}
	var itemsWithHidden []models.FileItem
	json.NewDecoder(res.Body).Decode(&itemsWithHidden)
	res.Body.Close()

	foundDotEnv := false
	for _, item := range itemsWithHidden {
		if item.Name == ".env" {
			foundDotEnv = true
			break
		}
	}
	if !foundDotEnv {
		t.Errorf("expected .env file in response when hidden=true")
	}
}
