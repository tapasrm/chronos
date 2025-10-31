package storage

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// BlobServer manages different types of storage.
type BlobServer struct {
	Assets  Storage
	Backups Storage
}

func (s *BlobServer) HandleFiles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		files, err := s.Assets.ListFiles(ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(files)

	case http.MethodPost:
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "file not found", 400)
			return
		}
		defer file.Close()

		info, err := s.Assets.UploadFile(ctx, header.Filename, file)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(info)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *BlobServer) HandleFileOps(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	name := strings.TrimPrefix(r.URL.Path, "/files/")
	if name == "" {
		http.Error(w, "filename required", 400)
		return
	}

	switch {
	case r.Method == http.MethodDelete:
		if err := s.Assets.DeleteFile(ctx, name); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Deleted\n"))

	case r.Method == http.MethodPut && strings.HasSuffix(name, "/rename"):
		oldName := strings.TrimSuffix(name, "/rename")
		newName := r.URL.Query().Get("to")
		if newName == "" {
			http.Error(w, "missing ?to=<newName>", 400)
			return
		}
		if err := s.Assets.RenameFile(ctx, oldName, newName); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("Renamed\n"))

	default:
		http.Error(w, "unsupported operation", 405)
	}
}
