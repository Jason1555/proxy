package http

import (
	"encoding/json"
	stdhttp "net/http"
	"strings"

	"proxy/internal/domain"
)

type MonitoringHandler struct {
	collector domain.MonitoringCollector
}

func NewMonitoringHandler(collector domain.MonitoringCollector) *MonitoringHandler {
	return &MonitoringHandler{collector: collector}
}

func (h *MonitoringHandler) RegisterRoutes(mux *stdhttp.ServeMux, prefix string) {
	if prefix == "" {
		prefix = "/metrics"
	}
	prefix = strings.TrimRight(prefix, "/")

	mux.HandleFunc(prefix, h.GetMetrics)
	mux.HandleFunc(prefix+"/health", h.GetHealth)
	mux.HandleFunc(prefix+"/upstreams", h.GetUpstreams)
}

func (h *MonitoringHandler) GetMetrics(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		writeJSON(w, stdhttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, stdhttp.StatusOK, h.collector.Snapshot())
}

func (h *MonitoringHandler) GetHealth(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		writeJSON(w, stdhttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	snapshot := h.collector.Snapshot()
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"status":         "ok",
		"started_at":     snapshot.StartedAt,
		"uptime_seconds": snapshot.UptimeSeconds,
	})
}

func (h *MonitoringHandler) GetUpstreams(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodGet {
		writeJSON(w, stdhttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, stdhttp.StatusOK, h.collector.Snapshot().Upstreams)
}

func writeJSON(w stdhttp.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
