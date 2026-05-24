// Author: Hakan Gunay
// Date: 2026-04-04
// HTTP API - exposes scraper functionality to minigaraj-admin

package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"minigaraj-scraper/internal/crawler/manager"
	"minigaraj-scraper/internal/models"
	"minigaraj-scraper/internal/storage"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	manager *manager.Manager
	repo    *storage.Repository
	logger  *zap.Logger
	apiKey  string
}

// New creates a new API Handler
func New(m *manager.Manager, repo *storage.Repository, logger *zap.Logger, apiKey string) *Handler {
	return &Handler{manager: m, repo: repo, logger: logger, apiKey: apiKey}
}

// RegisterRoutes registers all HTTP routes on the given mux and returns
// the top-level handler with middleware applied
func (h *Handler) RegisterRoutes(mux *http.ServeMux) http.Handler {
	// Health & Metrics
	mux.HandleFunc("GET /health", h.health)
	mux.Handle("GET /metrics", promhttp.Handler())

	// Jobs
	mux.HandleFunc("GET /api/v1/jobs", h.listJobs)
	mux.HandleFunc("POST /api/v1/jobs", h.createJob)
	mux.HandleFunc("GET /api/v1/jobs/{id}", h.getJob)
	mux.HandleFunc("POST /api/v1/jobs/{id}/cancel", h.cancelJob)

	// Raw models
	mux.HandleFunc("GET /api/v1/models", h.listRawModels)
	mux.HandleFunc("POST /api/v1/models/{id}/approve", h.approveModel)
	mux.HandleFunc("POST /api/v1/models/{id}/reject", h.rejectModel)

	// Seed URLs
	mux.HandleFunc("GET /api/v1/seeds", h.listSeeds)
	mux.HandleFunc("POST /api/v1/seeds", h.createSeed)
	mux.HandleFunc("PUT /api/v1/seeds/{id}", h.toggleSeed)
	mux.HandleFunc("DELETE /api/v1/seeds/{id}", h.deleteSeed)

	// Available brands
	mux.HandleFunc("GET /api/v1/brands", h.listBrands)

	// Apply authentication middleware
	return apiKeyAuth(h.apiKey)(mux)
}

// ============================================================================
// Health
// ============================================================================

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		h.logger.Error("health check failed", zap.Error(err))
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ============================================================================
// Jobs
// ============================================================================

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)

	jobs, err := h.repo.ListJobs(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   jobs,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Brand string `json:"brand"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Brand == "" {
		writeError(w, http.StatusBadRequest, "brand is required")
		return
	}

	jobID, err := h.manager.StartJob(r.Context(), body.Brand)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"job_id":  jobID,
		"brand":   body.Brand,
		"message": "job started",
	})
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	job, err := h.repo.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.manager.CancelJob(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ============================================================================
// Raw Models
// ============================================================================

func (h *Handler) listRawModels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := storage.RawModelFilter{
		Brand:  q.Get("brand"),
		Status: q.Get("status"),
		Year:   queryInt(r, "year", 0),
		Limit:  queryInt(r, "limit", 50),
		Offset: queryInt(r, "offset", 0),
	}
	if jobID := q.Get("job_id"); jobID != "" {
		filter.JobID, _ = strconv.ParseInt(jobID, 10, 64)
	}

	items, total, err := h.repo.ListRawModels(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) approveModel(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.repo.ApproveRawModel(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *Handler) rejectModel(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.repo.RejectRawModel(r.Context(), id, body.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// ============================================================================
// Seed URLs
// ============================================================================

func (h *Handler) listSeeds(w http.ResponseWriter, r *http.Request) {
	brand := r.URL.Query().Get("brand")
	seeds, err := h.repo.ListSeedURLs(r.Context(), brand)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": seeds,
	})
}

func (h *Handler) createSeed(w http.ResponseWriter, r *http.Request) {
	var input models.CreateSeedURLInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Brand == "" || input.URL == "" {
		writeError(w, http.StatusBadRequest, "brand and url are required")
		return
	}

	id, err := h.repo.CreateSeedURL(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":    id,
		"brand": input.Brand,
		"url":   input.URL,
	})
}

func (h *Handler) toggleSeed(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.repo.ToggleSeedURL(r.Context(), id, body.IsActive); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":        id,
		"is_active": body.IsActive,
	})
}

func (h *Handler) deleteSeed(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt64(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.repo.DeleteSeedURL(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ============================================================================
// Brands
// ============================================================================

func (h *Handler) listBrands(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"brands": h.manager.AvailableBrands(),
	})
}

// ============================================================================
// Helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func pathInt64(r *http.Request, key string) (int64, error) {
	v := r.PathValue(key)
	return strconv.ParseInt(v, 10, 64)
}
