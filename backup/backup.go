package backup

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"tapasrm.dev/cron-ui/storage"
)

const ChecksumFile = ".last_checksum"

// BackupSQLite uploads the SQLite file if checksum changed.
func BackupSQLite(ctx context.Context, dbPath, blobName string, store storage.Storage) error {
	f, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer f.Close()

	// Compute current file checksum
	curChecksum, err := fileChecksum(f)
	if err != nil {
		return fmt.Errorf("checksum: %w", err)
	}
	f.Seek(0, io.SeekStart)

	// Read last uploaded checksum (from local)
	lastChecksum := readLocalChecksum(filepath.Join(filepath.Dir(dbPath), ChecksumFile))

	if curChecksum == lastChecksum {
		slog.Debug("Backup skipped (no change detected)", "path", dbPath, "checksum", curChecksum)
		return nil
	}

	// Upload to blob storage
	slog.Info("Uploading SQLite backup", "path", dbPath, "blob", blobName)
	_, err = store.UploadFile(ctx, blobName, f)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// Save checksum locally
	writeLocalChecksum(filepath.Join(filepath.Dir(dbPath), ChecksumFile), curChecksum)
	slog.Info("Backup successful", "path", dbPath, "blob", blobName, "checksum", curChecksum)
	return nil
}

// ScheduleBackup runs continuous backups every interval.
func ScheduleBackup(ctx context.Context, interval time.Duration, dbPath, blobName string, store storage.Storage) {
	for {
		err := BackupSQLite(ctx, dbPath, blobName, store)
		if err != nil {
			slog.Error("Backup error", "error", err, "path", dbPath, "blob", blobName)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// RestoreSQLite downloads and replaces the local SQLite file if not present or outdated.
func RestoreSQLite(ctx context.Context, dbPath, blobName string, store storage.Storage) error {
	slog.Info("Restoring SQLite from blob", "path", dbPath, "blob", blobName)

	// Download blob
	rc, err := store.DownloadFile(ctx, blobName)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer rc.Close()

	tempFile := dbPath + ".tmp"
	out, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}

	h := md5.New()
	_, err = io.Copy(io.MultiWriter(out, h), rc)
	out.Close()
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	checksum := hex.EncodeToString(h.Sum(nil))

	// Replace existing DB
	err = os.Rename(tempFile, dbPath)
	if err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	writeLocalChecksum(filepath.Join(filepath.Dir(dbPath), ChecksumFile), checksum)
	slog.Info("Restore completed", "path", dbPath, "blob", blobName, "checksum", checksum)
	return nil
}

func fileChecksum(f *os.File) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readLocalChecksum(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeLocalChecksum(path, checksum string) {
	_ = os.WriteFile(path, []byte(checksum), 0644)
}
