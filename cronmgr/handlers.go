package cronmgr

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	crondescriptor "github.com/lnquy/cron"
)

// corsResponseWriter wraps http.ResponseWriter to ensure CORS headers are always set
type corsResponseWriter struct {
	http.ResponseWriter
	origin     string
	headersSet bool
}

func (w *corsResponseWriter) WriteHeader(code int) {
	if !w.headersSet {
		w.setCORSHeaders()
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *corsResponseWriter) Write(b []byte) (int, error) {
	if !w.headersSet {
		w.setCORSHeaders()
	}
	return w.ResponseWriter.Write(b)
}

func (w *corsResponseWriter) setCORSHeaders() {
	if w.origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", w.origin)
		w.Header().Add("Vary", "Origin")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.headersSet = true
}

// enableCORS is provided so main can wrap the router
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			corsW := &corsResponseWriter{ResponseWriter: w, origin: origin}
			corsW.setCORSHeaders()
			corsW.WriteHeader(http.StatusOK)
			return
		}

		// Wrap response writer to ensure CORS headers are set on all responses
		corsW := &corsResponseWriter{ResponseWriter: w, origin: origin}
		next.ServeHTTP(corsW, r)
	})
}

func (cm *CronManager) HandleGetJobs(w http.ResponseWriter, r *http.Request) {
	jobs := cm.GetAllJobs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (cm *CronManager) HandleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := cm.GetJob(jobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (cm *CronManager) HandleCreateJob(w http.ResponseWriter, r *http.Request) {
	var job Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if job.ID == "" {
		job.ID = cm.generateUniqueJobID()
	}

	if err := cm.AddJob(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (cm *CronManager) HandleUpdateJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	var job Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job.ID = jobID
	if err := cm.UpdateJob(jobID, &job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (cm *CronManager) HandleDeleteJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if err := cm.RemoveJob(jobID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleDescribeCron returns a human-readable description of a cron expression
func (cm *CronManager) HandleDescribeCron(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schedule string `json:"schedule"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	description, err := cm.cronDescriptor.ToDescription(req.Schedule, crondescriptor.Locale_en)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid cron expression: %v", err), http.StatusBadRequest)
		return
	}

	response := map[string]string{
		"schedule":    req.Schedule,
		"description": description,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
