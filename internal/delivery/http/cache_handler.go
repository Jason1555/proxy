package http

import (
	"encoding/json"
	"net/http"
	"proxy/internal/domain"
	"proxy/internal/usecases"
)

type CacheHandler struct {
	cache usecases.CacheService
}

func NewCacheHandler(cache usecases.CacheService) *CacheHandler {
	return &CacheHandler{cache: cache}
}

type ErrorResponse struct {
	Error string `json:"error" example:"internal server error"`
}

type SuccessResponse struct {
	Message string `json:"message" example:"operation completed successfully"`
}

type InvalidationResponse struct {
	Message string `json:"message" example:"cache invalidated successfully"`
}

// GetStats godoc
// @Summary Get cache statistics
// @Description Returns cache metrics and usage statistics
// @Tags Cache
// @Produce json
// @Success 200 {object} domain.CacheStats
// @Failure 500 {object} ErrorResponse
// @Router /cache/stats [get]
func (h *CacheHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.cache.GetStats(r.Context())

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
}

// Clear godoc
// @Summary Clear cache
// @Description Clears all cache entries
// @Tags Cache
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /cache [delete]
func (h *CacheHandler) Clear(w http.ResponseWriter, r *http.Request) {
	err := h.cache.Clear(r.Context())

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(SuccessResponse{Message: "cache cleared successfully"})
}

// Invalidate godoc
// @Summary Invalidate cache entries
// @Description Invalidates cache entries by key, prefix, tag, or all
// @Tags Cache
// @Accept json
// @Produce json
// @Param request body domain.InvalidationRequest true "Invalidation request"
// @Success 200 {object} InvalidationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /cache/invalidate [post]
func (h *CacheHandler) Invalidate(w http.ResponseWriter, r *http.Request) {
	var req domain.InvalidationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Type == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "invalidation type is required"})
		return
	}

	err := h.cache.Invalidate(r.Context(), req)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(InvalidationResponse{Message: "cache invalidated successfully"})
}
