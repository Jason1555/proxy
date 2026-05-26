package http

import (
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"proxy/internal/domain"
	"proxy/internal/testutils"
	"proxy/internal/usecases"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	collector := usecases.NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})
	next := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "payload", string(body))

		w.WriteHeader(stdhttp.StatusCreated)
		_, err = w.Write([]byte("created"))
		require.NoError(t, err)
	})
	handler := MetricsMiddleware(collector)(next)

	req := httptest.NewRequest(stdhttp.MethodPost, "/proxy/resource", strings.NewReader("payload"))
	req.RemoteAddr = "203.0.113.10:54321"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	require.Equal(t, stdhttp.StatusCreated, rec.Code)

	snapshot := collector.Snapshot()
	assert.Equal(t, int64(1), snapshot.Requests.Total)
	assert.Equal(t, int64(1), snapshot.Requests.Success)
	assert.Equal(t, int64(1), snapshot.Requests.ByStatusClass["2xx"])
	assert.Equal(t, int64(1), snapshot.Requests.ByMethod[stdhttp.MethodPost])
	assert.Equal(t, int64(1), snapshot.Requests.ByPath["/proxy/resource"])
	assert.Equal(t, int64(len("payload")), snapshot.Traffic.BytesIn)
	assert.Equal(t, int64(len("created")), snapshot.Traffic.BytesOut)
}

func TestMetricsMiddleware_UsesForwardedClientIP(t *testing.T) {
	collector := usecases.NewMonitoringService(domain.MonitoringConfig{
		RequestWindow:     domain.NewDefaultMonitoringConfig().RequestWindow,
		MaxLatencySamples: 1,
		MaxPathStats:      1,
		TrackPathStats:    true,
	}, &testutils.MockLogger{})
	next := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	})
	handler := MetricsMiddleware(collector)(next)

	req := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.3, 10.0.0.1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "198.51.100.3", requestClientIP(req))
	assert.Equal(t, int64(1), collector.Snapshot().Requests.Total)
}
