package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"tapasrm.dev/cron-ui/backup"
	"tapasrm.dev/cron-ui/cronmgr"
	"tapasrm.dev/cron-ui/storage"
)

func setupLogger() {
	// Use JSON handler for structured logs (better for production)
	// Or use TextHandler for human-readable logs (better for development)
	useJSON := os.Getenv("LOG_FORMAT") == "json"

	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func main() {
	setupLogger()
	ctx := context.Background()

	account := os.Getenv("AZURE_STORAGE_ACCOUNT")
	key := os.Getenv("AZURE_STORAGE_KEY")
	assetsContainer := os.Getenv("ASSETS_CONTAINER")
	backupContainer := os.Getenv("BACKUP_CONTAINER")
	cdnBase := os.Getenv("CDN_BASE_URL")

	// Check if Azure storage is configured
	hasAzureStorage := account != "" && key != "" && assetsContainer != "" && backupContainer != ""

	var blobServer *storage.BlobServer
	var backupStore storage.Storage

	if hasAzureStorage {
		// Initialize Blob Storages
		assetsStore, err := storage.NewAzureBlobStorage(account, key, assetsContainer, cdnBase)
		if err != nil {
			slog.Error("Failed to initialize assets storage", "error", err)
			os.Exit(1)
		}
		backupStore, err = storage.NewAzureBlobStorage(account, key, backupContainer, "")
		if err != nil {
			slog.Error("Failed to initialize backup storage", "error", err)
			os.Exit(1)
		}

		blobServer = &storage.BlobServer{
			Assets:  assetsStore,
			Backups: backupStore,
		}
		slog.Info("Azure blob storage initialized", "assets_container", assetsContainer, "backup_container", backupContainer)
	} else {
		slog.Info("Azure storage not configured, running with local SQLite only", "hint", "Set AZURE_STORAGE_ACCOUNT, AZURE_STORAGE_KEY, ASSETS_CONTAINER, and BACKUP_CONTAINER to enable blob storage")
	}

	db_path := "cron_jobs.db"
	blobName := "cronos_backups/cron_jobs.db"

	// Only restore from backup if backup storage is available
	if backupStore != nil {
		if err := backup.RestoreSQLite(ctx, db_path, blobName, backupStore); err != nil {
			slog.Info("No existing backup found, starting fresh", "error", err)
		}
	}

	manager := cronmgr.NewCronManager()
	manager.Start()

	// Start background sync - pass nil for backupStore if not configured (backup will be disabled)
	manager.StartBackgroundSync(db_path, 30*time.Second, 1*time.Hour, blobName, backupStore)
	defer manager.Stop()

	router := mux.NewRouter()
	router.HandleFunc("/api/jobs", manager.HandleGetJobs).Methods("GET")
	router.HandleFunc("/api/jobs", manager.HandleCreateJob).Methods("POST")
	router.HandleFunc("/api/jobs/{id}", manager.HandleGetJob).Methods("GET")
	router.HandleFunc("/api/jobs/{id}", manager.HandleUpdateJob).Methods("PUT")
	router.HandleFunc("/api/jobs/{id}", manager.HandleDeleteJob).Methods("DELETE")

	// Only register file endpoints if blob storage is available
	if blobServer != nil {
		router.HandleFunc("/api/files", blobServer.HandleFiles).Methods("GET", "POST")
		router.HandleFunc("/api/files/", blobServer.HandleFileOps).Methods("PUT", "DELETE")
	} else {
		// Return 503 Service Unavailable for file endpoints when storage is not configured
		router.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "File storage not available. Configure Azure blob storage to enable this feature.", http.StatusServiceUnavailable)
		}).Methods("GET", "POST")
		router.HandleFunc("/api/files/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "File storage not available. Configure Azure blob storage to enable this feature.", http.StatusServiceUnavailable)
		}).Methods("PUT", "DELETE")
	}

	router.HandleFunc("/api/describe-cron", manager.HandleDescribeCron).Methods("POST")

	// Heartbeat endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// handler := cronmgr.EnableCORS(router)
	// handler = securityHeadersMiddleware(handler)

	slog.Info("Server starting", "address", ":8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
