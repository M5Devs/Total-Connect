package models

import (
	"testing"
	"time"
)

func TestFileItem(t *testing.T) {
	now := time.Now()
	item := FileItem{
		Name:    "test.txt",
		Path:    "/tmp/test.txt",
		Size:    1024,
		ModTime: now,
		IsDir:   false,
	}

	if item.Name != "test.txt" {
		t.Errorf("expected Name 'test.txt', got '%s'", item.Name)
	}
	if item.Path != "/tmp/test.txt" {
		t.Errorf("expected Path '/tmp/test.txt', got '%s'", item.Path)
	}
	if item.Size != 1024 {
		t.Errorf("expected Size 1024, got %d", item.Size)
	}
	if !item.ModTime.Equal(now) {
		t.Errorf("expected ModTime %v, got %v", now, item.ModTime)
	}
	if item.IsDir {
		t.Errorf("expected IsDir false, got true")
	}
}

func TestRemoteItem(t *testing.T) {
	remote := RemoteItem{
		Name: "myremote",
		Type: "s3",
	}

	if remote.Name != "myremote" || remote.Type != "s3" {
		t.Errorf("unexpected RemoteItem fields: %+v", remote)
	}
}

func TestProgress(t *testing.T) {
	p := Progress{
		BytesTransferred: 50,
		TotalBytes:       100,
		Percentage:       50.0,
		SpeedBytesPerSec: 10,
		CurrentFile:      "file.bin",
	}

	if p.Percentage != 50.0 {
		t.Errorf("expected Percentage 50.0, got %f", p.Percentage)
	}
}
