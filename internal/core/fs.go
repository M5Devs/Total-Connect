package core

import (
	"context"

	"github.com/M5Devs/Total-Connect/internal/models"
)

// StorageEngine defines the abstraction interface for file system and remote storage operations.
type StorageEngine interface {
	// ListRemotes lists all configured rclone remotes.
	ListRemotes(ctx context.Context) ([]string, error)

	// ListEntries lists files and directories at the given path (e.g. "remote:path" or "/local/path").
	ListEntries(ctx context.Context, remotePath string) ([]models.FileItem, error)

	// Copy copies a file or directory from src path to dst path.
	Copy(ctx context.Context, src, dst string) error

	// Delete removes a file or directory at the given path.
	Delete(ctx context.Context, path string) error

	// Mkdir creates a directory at the given path.
	Mkdir(ctx context.Context, path string) error
}
