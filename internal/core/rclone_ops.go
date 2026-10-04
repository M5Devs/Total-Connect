package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/rclone/rclone/backend/all" // Register all rclone backends
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/fs/fspath"
	"github.com/rclone/rclone/fs/operations"
	"github.com/rclone/rclone/fs/sync"

	tcconfig "github.com/M5Devs/Total-Connect/internal/config"
	"github.com/M5Devs/Total-Connect/internal/models"
)

// RcloneEngine implements StorageEngine using rclone libraries.
type RcloneEngine struct {
	configPath string
}

// NewRcloneEngine creates and initializes a new RcloneEngine.
func NewRcloneEngine(customConfigPath string) (*RcloneEngine, error) {
	cfgPath := customConfigPath
	if cfgPath == "" {
		cfgPath = tcconfig.GetConfigPath()
	}

	// Install the config file path if it exists or create directory
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	config.SetConfigPath(cfgPath)
	configfile.Install()

	return &RcloneEngine{
		configPath: cfgPath,
	}, nil
}

// ListRemotes returns all configured remote names.
func (r *RcloneEngine) ListRemotes(ctx context.Context) ([]string, error) {
	remotes := config.FileSections()
	return remotes, nil
}

// ListEntries lists files and directories at remotePath (e.g. "remote:path" or "/local/path").
func (r *RcloneEngine) ListEntries(ctx context.Context, remotePath string) ([]models.FileItem, error) {
	if remotePath == "" {
		remotePath = "."
	}

	f, err := fs.NewFs(ctx, remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open fs for %q: %w", remotePath, err)
	}

	entries, err := f.List(ctx, "")
	if err != nil && err != fs.ErrorDirNotFound {
		return nil, fmt.Errorf("failed to list entries for %q: %w", remotePath, err)
	}

	var items []models.FileItem
	for _, entry := range entries {
		item := models.FileItem{
			Name:    entry.Remote(),
			Path:    fspath.JoinRootPath(remotePath, entry.Remote()),
			Size:    entry.Size(),
			ModTime: entry.ModTime(ctx),
		}

		if _, isDir := entry.(fs.Directory); isDir {
			item.IsDir = true
			item.Size = 0
		}

		items = append(items, item)
	}

	return items, nil
}

// Copy copies a file or directory from src to dst.
func (r *RcloneEngine) Copy(ctx context.Context, src, dst string) error {
	srcFs, srcRemote, err := fspath.Split(src)
	if err != nil {
		return fmt.Errorf("invalid src path %q: %w", src, err)
	}
	dstFs, dstRemote, err := fspath.Split(dst)
	if err != nil {
		return fmt.Errorf("invalid dst path %q: %w", dst, err)
	}

	fsrc, err := fs.NewFs(ctx, srcFs)
	if err != nil {
		return fmt.Errorf("failed to open src fs %q: %w", srcFs, err)
	}

	fdst, err := fs.NewFs(ctx, dstFs)
	if err != nil {
		return fmt.Errorf("failed to open dst fs %q: %w", dstFs, err)
	}

	fn, hasProgress := GetProgressHandler(ctx)

	obj, err := fsrc.NewObject(ctx, srcRemote)
	if err == nil {
		// Single file copy
		totalBytes := obj.Size()
		if hasProgress {
			fn(models.Progress{
				CurrentFile:      filepath.Base(src),
				BytesTransferred: 0,
				TotalBytes:       totalBytes,
				Percentage:       0.0,
			})
		}

		err = operations.CopyFile(ctx, fdst, fsrc, dstRemote, srcRemote)
		if err == nil && hasProgress {
			fn(models.Progress{
				CurrentFile:      filepath.Base(src),
				BytesTransferred: totalBytes,
				TotalBytes:       totalBytes,
				Percentage:       100.0,
			})
		}
		return err
	}

	// Directory copy
	fsrcSub, err := fs.NewFs(ctx, src)
	if err != nil {
		return fmt.Errorf("failed to open src dir %q: %w", src, err)
	}
	fdstSub, err := fs.NewFs(ctx, dst)
	if err != nil {
		return fmt.Errorf("failed to open dst dir %q: %w", dst, err)
	}

	if hasProgress {
		fn(models.Progress{
			CurrentFile:      filepath.Base(src),
			BytesTransferred: 0,
			TotalBytes:       0,
			Percentage:       0.0,
		})
	}

	err = sync.CopyDir(ctx, fdstSub, fsrcSub, true)
	if err == nil && hasProgress {
		fn(models.Progress{
			CurrentFile:      filepath.Base(src),
			BytesTransferred: 100,
			TotalBytes:       100,
			Percentage:       100.0,
		})
	}
	return err
}

// Move natively moves a file or directory from src to dst.
func (r *RcloneEngine) Move(ctx context.Context, src, dst string) error {
	srcFs, srcRemote, err := fspath.Split(src)
	if err != nil {
		return fmt.Errorf("invalid src path %q: %w", src, err)
	}
	dstFs, dstRemote, err := fspath.Split(dst)
	if err != nil {
		return fmt.Errorf("invalid dst path %q: %w", dst, err)
	}

	fsrc, err := fs.NewFs(ctx, srcFs)
	if err != nil {
		return fmt.Errorf("failed to open src fs %q: %w", srcFs, err)
	}

	fdst, err := fs.NewFs(ctx, dstFs)
	if err != nil {
		return fmt.Errorf("failed to open dst fs %q: %w", dstFs, err)
	}

	fn, hasProgress := GetProgressHandler(ctx)

	obj, err := fsrc.NewObject(ctx, srcRemote)
	if err == nil {
		// Single file move
		totalBytes := obj.Size()
		if hasProgress {
			fn(models.Progress{
				CurrentFile:      filepath.Base(src),
				BytesTransferred: 0,
				TotalBytes:       totalBytes,
				Percentage:       0.0,
			})
		}

		err = operations.MoveFile(ctx, fdst, fsrc, dstRemote, srcRemote)
		if err == nil && hasProgress {
			fn(models.Progress{
				CurrentFile:      filepath.Base(src),
				BytesTransferred: totalBytes,
				TotalBytes:       totalBytes,
				Percentage:       100.0,
			})
		}
		return err
	}

	// Directory move
	fsrcSub, err := fs.NewFs(ctx, src)
	if err != nil {
		return fmt.Errorf("failed to open src dir %q: %w", src, err)
	}
	fdstSub, err := fs.NewFs(ctx, dst)
	if err != nil {
		return fmt.Errorf("failed to open dst dir %q: %w", dst, err)
	}

	if hasProgress {
		fn(models.Progress{
			CurrentFile:      filepath.Base(src),
			BytesTransferred: 0,
			TotalBytes:       0,
			Percentage:       0.0,
		})
	}

	err = sync.MoveDir(ctx, fdstSub, fsrcSub, true, true)
	if err == nil && hasProgress {
		fn(models.Progress{
			CurrentFile:      filepath.Base(src),
			BytesTransferred: 100,
			TotalBytes:       100,
			Percentage:       100.0,
		})
	}
	return err
}

// Delete removes a file or directory at the given path.
func (r *RcloneEngine) Delete(ctx context.Context, path string) error {
	fsPath, remote, err := fspath.Split(path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", path, err)
	}

	f, err := fs.NewFs(ctx, fsPath)
	if err != nil {
		return fmt.Errorf("failed to open fs %q: %w", fsPath, err)
	}

	obj, err := f.NewObject(ctx, remote)
	if err == nil {
		return operations.DeleteFile(ctx, obj)
	}

	return operations.Purge(ctx, f, remote)
}

// Mkdir creates a directory at the given path.
func (r *RcloneEngine) Mkdir(ctx context.Context, path string) error {
	fsPath, remote, err := fspath.Split(path)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", path, err)
	}

	f, err := fs.NewFs(ctx, fsPath)
	if err != nil {
		return fmt.Errorf("failed to open fs %q: %w", fsPath, err)
	}

	return operations.Mkdir(ctx, f, remote)
}
