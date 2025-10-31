package storage

import (
	"context"
	"io"
)

type FileInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Storage defines a generic file storage interface.
type Storage interface {
	ListFiles(ctx context.Context) ([]FileInfo, error)
	DownloadFile(ctx context.Context, name string) (io.ReadCloser, error)
	UploadFile(ctx context.Context, name string, data io.Reader) (FileInfo, error)
	DeleteFile(ctx context.Context, name string) error
	RenameFile(ctx context.Context, oldName, newName string) error
}
