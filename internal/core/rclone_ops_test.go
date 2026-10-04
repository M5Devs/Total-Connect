package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRcloneEngineLocalOps(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "rclone.conf")

	engine, err := NewRcloneEngine(cfgPath)
	if err != nil {
		t.Fatalf("failed to create RcloneEngine: %v", err)
	}

	ctx := context.Background()

	// Test ListRemotes (may be empty)
	remotes, err := engine.ListRemotes(ctx)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v", err)
	}
	_ = remotes

	// Create test directory
	testSubDir := filepath.Join(tmpDir, "testdir")
	err = engine.Mkdir(ctx, testSubDir)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(testSubDir, "sample.txt")
	err = os.WriteFile(testFile, []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// List entries
	items, err := engine.ListEntries(ctx, testSubDir)
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "sample.txt" {
		t.Errorf("expected sample.txt in entries, got %+v", items)
	}

	// Copy file
	dstFile := filepath.Join(testSubDir, "sample_copy.txt")
	err = engine.Copy(ctx, testFile, dstFile)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Errorf("expected copied file to exist at %s", dstFile)
	}

	// Delete file
	err = engine.Delete(ctx, dstFile)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := os.Stat(dstFile); !os.IsNotExist(err) {
		t.Errorf("expected deleted file to no longer exist at %s", dstFile)
	}
}
