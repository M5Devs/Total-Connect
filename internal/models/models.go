package models

import (
	"time"
)

// FileItem represents a file or directory item in the file system/remote.
type FileItem struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

// RemoteItem represents a configured rclone remote backend.
type RemoteItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Progress represents transfer progress information.
type Progress struct {
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`
	Percentage       float64 `json:"percentage"`
	SpeedBytesPerSec int64   `json:"speed_bytes_per_sec"`
	CurrentFile      string  `json:"current_file"`
}
