package core

import (
	"context"

	"github.com/M5Devs/Total-Connect/internal/models"
)

type progressKeyStruct struct{}

var progressKey = progressKeyStruct{}

// ProgressFunc defines a callback function to receive progress updates during transfers.
type ProgressFunc func(progress models.Progress)

// WithProgressHandler returns a context containing a progress handler callback.
func WithProgressHandler(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKey, fn)
}

// GetProgressHandler retrieves the progress handler callback from context if set.
func GetProgressHandler(ctx context.Context) (ProgressFunc, bool) {
	fn, ok := ctx.Value(progressKey).(ProgressFunc)
	return fn, ok
}

// StorageEngine defines the abstraction interface for file system and remote storage operations.
type StorageEngine interface {
	// ListRemotes lists all configured rclone remotes.
	ListRemotes(ctx context.Context) ([]string, error)

	// ListEntries lists files and directories at the given path (e.g. "remote:path" or "/local/path").
	ListEntries(ctx context.Context, remotePath string) ([]models.FileItem, error)

	// Copy copies a file or directory from src path to dst path.
	Copy(ctx context.Context, src, dst string) error

	// Move moves a file or directory from src path to dst path natively.
	Move(ctx context.Context, src, dst string) error

	// Delete removes a file or directory at the given path.
	Delete(ctx context.Context, path string) error

	// Mkdir creates a directory at the given path.
	Mkdir(ctx context.Context, path string) error
}
