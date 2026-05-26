package http

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"proxy/internal/domain"
	"proxy/internal/testutils"
	"proxy/internal/usecases"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitoringHandler_GetMetrics(t *testing.T) {
	collector := usecases.NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})
	collector.CompleteRequest(domain.RequestMetric{
		Method:     "GET",
		Path:       "/proxy",
		StatusCode: 200,
		Latency:    10 * time.Millisecond,
	})
	handler := NewMonitoringHandler(collector)

	req := httptest.NewRequest(stdhttp.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.GetMetrics(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var snapshot domain.MonitoringSnapshot
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&snapshot))
	assert.Equal(t, int64(1), snapshot.Requests.Total)
	assert.Equal(t, int64(1), snapshot.Requests.ByPath["/proxy"])
}

func TestMonitoringHandler_RegisterRoutes(t *testing.T) {
	collector := usecases.NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})
	collector.RecordUpstream(domain.UpstreamMetric{
		Name:       "api",
		Healthy:    true,
		StatusCode: 200,
	})

	mux := stdhttp.NewServeMux()
	NewMonitoringHandler(collector).RegisterRoutes(mux, "/example/metrics")

	req := httptest.NewRequest(stdhttp.MethodGet, "/example/metrics/upstreams", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusOK, rec.Code)

	var upstreams map[string]domain.UpstreamStats
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&upstreams))
	assert.True(t, upstreams["api"].Healthy)
}

func TestMonitoringHandler_MethodNotAllowed(t *testing.T) {
	collector := usecases.NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})
	handler := NewMonitoringHandler(collector)

	req := httptest.NewRequest(stdhttp.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.GetMetrics(rec, req)

	assert.Equal(t, stdhttp.StatusMethodNotAllowed, rec.Code)
}
