package cronmgr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	crondescriptor "github.com/lnquy/cron"
	rcron "github.com/robfig/cron/v3"
	"tapasrm.dev/cron-ui/backup"
	"tapasrm.dev/cron-ui/storage"
)

// JobType represents different types of jobs
type JobType string

const (
	EmailJob  JobType = "email"
	SyncJob   JobType = "sync"
	BackupJob JobType = "backup"
	CustomJob JobType = "custom"
)

// Job represents a cron job configuration
type Job struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         JobType        `json:"type"`
	Schedule     string         `json:"schedule"`
	ScheduleDesc string         `json:"scheduleDesc,omitempty"`
	Enabled      bool           `json:"enabled"`
	Config       map[string]any `json:"config"`
	LastRun      *time.Time     `json:"lastRun,omitempty"`
	NextRun      *time.Time     `json:"nextRun,omitempty"`
	CronEntryID  *rcron.EntryID `json:"-"`
}

// JobExecutor interface for different job types
type JobExecutor interface {
	Execute(config map[string]any) error
	Validate(config map[string]any) error
}

// EmailJobExecutor handles email sending jobs
type EmailJobExecutor struct{}

func (e *EmailJobExecutor) Execute(config map[string]any) error {
	to := config["to"].(string)
	subject := config["subject"].(string)
	_ = config["body"].(string) // body is available but not logged for privacy

	slog.Info("Sending email", "to", to, "subject", subject)
	// Implement actual email sending logic here
	return nil
}

func (e *EmailJobExecutor) Validate(config map[string]any) error {
	if _, ok := config["to"]; !ok {
		return fmt.Errorf("'to' field is required")
	}
	if _, ok := config["subject"]; !ok {
		return fmt.Errorf("'subject' field is required")
	}
	return nil
}

// SyncJobExecutor handles data synchronization jobs
type SyncJobExecutor struct{}

func (s *SyncJobExecutor) Execute(config map[string]any) error {
	source := config["source"].(string)
	destination := config["destination"].(string)

	slog.Info("Syncing data", "source", source, "destination", destination)
	// Implement actual sync logic here
	return nil
}

func (s *SyncJobExecutor) Validate(config map[string]any) error {
	if _, ok := config["source"]; !ok {
		return fmt.Errorf("'source' field is required")
	}
	if _, ok := config["destination"]; !ok {
		return fmt.Errorf("'destination' field is required")
	}
	return nil
}

// BackupJobExecutor handles backup jobs
type BackupJobExecutor struct{}

func (b *BackupJobExecutor) Execute(config map[string]any) error {
	path := config["path"].(string)
	destination := config["destination"].(string)

	slog.Info("Backing up", "path", path, "destination", destination)
	// Implement actual backup logic here
	return nil
}

func (b *BackupJobExecutor) Validate(config map[string]any) error {
	if _, ok := config["path"]; !ok {
		return fmt.Errorf("'path' field is required")
	}
	if _, ok := config["destination"]; !ok {
		return fmt.Errorf("'destination' field is required")
	}
	return nil
}

// CustomJobExecutor handles custom jobs
type CustomJobExecutor struct{}

func (c *CustomJobExecutor) Execute(config map[string]any) error {
	command := config["command"].(string)

	slog.Info("Executing custom command", "command", command)
	// Implement custom command execution here
	return nil
}

func (c *CustomJobExecutor) Validate(config map[string]any) error {
	if _, ok := config["command"]; !ok {
		return fmt.Errorf("'command' field is required")
	}
	return nil
}

// CronManager manages all cron jobs
type CronManager struct {
	cron           *rcron.Cron
	jobs           map[string]*Job
	executors      map[JobType]JobExecutor
	cronDescriptor crondescriptor.ExpressionDescriptor
	mu             sync.RWMutex
	// background sync management
	syncCancel func()
	syncWg     sync.WaitGroup
}

func NewCronManager() *CronManager {
	descriptor, err := crondescriptor.NewDescriptor()
	if err != nil {
		slog.Error("Failed to create cron descriptor", "error", err)
		os.Exit(1)
	}

	return &CronManager{
		cron: rcron.New(rcron.WithSeconds()),
		jobs: make(map[string]*Job),
		executors: map[JobType]JobExecutor{
			EmailJob:  &EmailJobExecutor{},
			SyncJob:   &SyncJobExecutor{},
			BackupJob: &BackupJobExecutor{},
			CustomJob: &CustomJobExecutor{},
		},
		cronDescriptor: *descriptor,
	}
}

func (cm *CronManager) Start() {
	// Attempt to load jobs from the default DB file if it exists.
	const dbPath = "cron_jobs.db"
	if _, err := os.Stat(dbPath); err == nil {
		if err := cm.LoadJobsFromDB(dbPath); err != nil {
			slog.Warn("Failed to load jobs from database", "error", err, "path", dbPath)
		}
	}

	cm.cron.Start()
}

// StartBackgroundSync starts a goroutine that periodically syncs in-memory jobs
// to the provided DB path and backs up to blob storage if configured.
func (cm *CronManager) StartBackgroundSync(dbPath string, syncInterval time.Duration, backupInterval time.Duration, blobName string, backupStore storage.Storage) {
	// cancel existing if running
	if cm.syncCancel != nil {
		cm.syncCancel()
		cm.syncWg.Wait()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cm.syncCancel = cancel
	cm.syncWg.Add(1)

	go func() {
		defer cm.syncWg.Done()
		syncTicker := time.NewTicker(syncInterval)
		defer syncTicker.Stop()

		// Only create backup ticker if backup store is provided
		var backupTicker *time.Ticker
		var backupChan <-chan time.Time
		if backupStore != nil && blobName != "" {
			backupTicker = time.NewTicker(backupInterval)
			defer backupTicker.Stop()
			backupChan = backupTicker.C
		}
		// If backup is not enabled, backupChan remains nil, which is safe in select

		for {
			select {
			case <-ctx.Done():
				return
			case <-syncTicker.C:
				if err := cm.SaveAllJobsToDB(dbPath); err != nil {
					slog.Warn("Background sync failed", "error", err, "path", dbPath)
				}
			case <-backupChan:
				if err := backup.BackupSQLite(ctx, dbPath, blobName, backupStore); err != nil {
					slog.Warn("Backup failed", "error", err, "path", dbPath, "blob", blobName)
				}
			}
		}
	}()
}

func (cm *CronManager) Stop() {
	// Try to persist current jobs to disk before stopping the scheduler.
	const dbPath = "cron_jobs.db"
	if err := cm.SaveAllJobsToDB(dbPath); err != nil {
		slog.Warn("Failed to save jobs to database", "error", err, "path", dbPath)
	}

	cm.cron.Stop()

	// stop background sync if running
	if cm.syncCancel != nil {
		cm.syncCancel()
		cm.syncWg.Wait()
		cm.syncCancel = nil
	}
}

func (cm *CronManager) AddJob(job *Job) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	executor, ok := cm.executors[job.Type]
	if !ok {
		return fmt.Errorf("unknown job type: %s", job.Type)
	}

	if err := executor.Validate(job.Config); err != nil {
		return fmt.Errorf("job configuration validation failed: %w", err)
	}

	// Generate human-readable description of the cron schedule
	description, err := cm.cronDescriptor.ToDescription(job.Schedule, crondescriptor.Locale_en)
	if err != nil {
		slog.Warn("Could not generate schedule description", "schedule", job.Schedule, "error", err)
		job.ScheduleDesc = job.Schedule
	} else {
		job.ScheduleDesc = description
	}

	if job.Enabled {
		entryID, err := cm.cron.AddFunc(job.Schedule, func() {
			cm.executeJob(job.ID)
		})
		if err != nil {
			return fmt.Errorf("failed to schedule job: %w", err)
		}
		job.CronEntryID = &entryID

		// Get next run time
		entry := cm.cron.Entry(entryID)
		nextRun := entry.Next
		job.NextRun = &nextRun
	}

	cm.jobs[job.ID] = job
	return nil
}

func (cm *CronManager) executeJob(jobID string) {
	cm.mu.RLock()
	job, exists := cm.jobs[jobID]
	if !exists {
		cm.mu.RUnlock()
		return
	}
	// Get job details while holding read lock
	jobName := job.Name
	jobType := job.Type
	jobEnabled := job.Enabled
	config := job.Config
	executor := cm.executors[jobType]
	cm.mu.RUnlock()

	if !jobEnabled {
		return
	}

	slog.Info("Executing job", "job", jobName, "type", jobType, "id", jobID)

	// Execute job outside of lock to avoid blocking other operations
	if err := executor.Execute(config); err != nil {
		slog.Error("Job execution failed", "job", jobName, "id", jobID, "error", err)
	} else {
		slog.Info("Job executed successfully", "job", jobName, "id", jobID)
	}

	// Update job state with write lock
	cm.mu.Lock()
	job, exists = cm.jobs[jobID]
	if !exists {
		cm.mu.Unlock()
		return
	}

	now := time.Now()
	job.LastRun = &now

	// Update next run time if scheduled
	if job.CronEntryID != nil {
		entry := cm.cron.Entry(*job.CronEntryID)
		nextRun := entry.Next
		job.NextRun = &nextRun
	}
	cm.mu.Unlock()
}

func (cm *CronManager) RemoveJob(jobID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	job, exists := cm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.CronEntryID != nil {
		cm.cron.Remove(*job.CronEntryID)
	}

	delete(cm.jobs, jobID)
	return nil
}

func (cm *CronManager) UpdateJob(jobID string, updatedJob *Job) error {
	cm.mu.RLock()
	// Validate the new job before removing the old one
	executor, ok := cm.executors[updatedJob.Type]
	if !ok {
		cm.mu.RUnlock()
		return fmt.Errorf("unknown job type: %s", updatedJob.Type)
	}
	cm.mu.RUnlock()

	// Validate configuration before removing existing job
	if err := executor.Validate(updatedJob.Config); err != nil {
		return fmt.Errorf("job configuration validation failed: %w", err)
	}

	// Validate schedule format (basic check - cron library will validate fully)
	if updatedJob.Schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	// Now safe to remove and add
	if err := cm.RemoveJob(jobID); err != nil {
		return err
	}

	// Ensure the ID matches
	updatedJob.ID = jobID
	return cm.AddJob(updatedJob)
}

func (cm *CronManager) GetJob(jobID string) (*Job, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	job, exists := cm.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

func (cm *CronManager) GetAllJobs() []*Job {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	jobs := make([]*Job, 0, len(cm.jobs))

	// Order jobs by last run time (most recent first)

	for _, job := range cm.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		a := jobs[i].LastRun
		b := jobs[j].LastRun
		// Treat nil LastRun as older than any non-nil LastRun
		if a == nil && b == nil {
			return jobs[i].Name < jobs[j].Name
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})
	return jobs
}

// generateUniqueJobID generates a UUID and checks it against existing jobs to avoid conflicts
func (cm *CronManager) generateUniqueJobID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	maxAttempts := 100 // Prevent infinite loop in the extremely unlikely case of collisions
	for range maxAttempts {
		id := uuid.New().String()
		if _, exists := cm.jobs[id]; !exists {
			return id
		}
	}

	// Fallback: if somehow we hit 100 collisions, use timestamp-based ID
	// This should never happen with UUIDs, but provides a safety net
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
