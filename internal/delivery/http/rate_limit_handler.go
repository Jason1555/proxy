package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"proxy/internal/usecases"
)

type RateLimitHandler struct {
	service usecases.RateLimitService
}

func NewRateLimitHandler(service usecases.RateLimitService) *RateLimitHandler {
	return &RateLimitHandler{service: service}
}

// GetStatus godoc
// @Summary Get rate limit status for IP
// @Description Returns current token bucket status for given IP
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param ip query string true "Client IP"
// @Success 200 {object} domain.RateLimitStatus
// @Failure 400 {string} string "missing ip"
// @Failure 404 {string} string "not found"
// @Router /rate-limit/status [get]
func (h *RateLimitHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip", http.StatusBadRequest)
		return
	}

	status, err := h.service.GetStatus(r.Context(), ip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeData(w, status)
}

// GetMetrics godoc
// @Summary Get rate limit metrics
// @Description Returns aggregated rate limit statistics
// @Tags rate-limit
// @Accept json
// @Produce json
// @Success 200 {object} domain.RateLimitMetrics
// @Router /rate-limit/metrics [get]
func (h *RateLimitHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.service.GetMetrics(r.Context())
	writeData(w, metrics)
}

// ResetIP godoc
// @Summary Reset rate limit state for IP
// @Description Deletes rate limit state (clears counters)
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param ip query string true "Client IP"
// @Success 204 "No Content"
// @Failure 400 {string} string "missing ip"
// @Router /rate-limit/reset [delete]
func (h *RateLimitHandler) ResetIP(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip", http.StatusBadRequest)
		return
	}

	if err := h.service.ResetIP(r.Context(), ip); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddConnection godoc
// @Summary Add active connection for IP
// @Description Increases active connection counter with rate limit check
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param ip query string true "Client IP"
// @Success 200 {object} map[string]bool
// @Failure 429 {string} string "connection limit exceeded"
// @Router /rate-limit/connection/add [post]
func (h *RateLimitHandler) AddConnection(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip", http.StatusBadRequest)
		return
	}

	ok, err := h.service.AddConnection(r.Context(), ip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	writeData(w, map[string]bool{"allowed": ok})
}

// RemoveConnection godoc
// @Summary Remove active connection for IP
// @Description Decreases active connection counter
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param ip query string true "Client IP"
// @Success 204 "No Content"
// @Router /rate-limit/connection/remove [post]
func (h *RateLimitHandler) RemoveConnection(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "missing ip", http.StatusBadRequest)
		return
	}

	_ = h.service.RemoveConnection(r.Context(), ip)

	w.WriteHeader(http.StatusNoContent)
}

// GetViolators godoc
// @Summary Get top rate limit violators
// @Description Returns IPs sorted by number of violations
// @Tags rate-limit
// @Accept json
// @Produce json
// @Param limit query int false "max results (default 10)"
// @Success 200 {array} domain.RateLimitViolator
// @Router /rate-limit/violators [get]
func (h *RateLimitHandler) GetViolators(w http.ResponseWriter, r *http.Request) {
	limit := 10

	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}

	data := h.service.GetTopViolators(r.Context(), limit)
	writeData(w, data)
}

func writeData(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
