package usecases

import (
	"proxy/internal/domain"
	"proxy/internal/testutils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonitoringService_RecordRequest(t *testing.T) {
	service := NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})

	service.BeginRequest()
	service.CompleteRequest(domain.RequestMetric{
		Method:     "GET",
		Path:       "/api/users",
		ClientIP:   "192.168.1.10",
		StatusCode: 200,
		BytesIn:    15,
		BytesOut:   42,
		Latency:    125 * time.Millisecond,
	})

	snapshot := service.Snapshot()

	assert.Equal(t, int64(1), snapshot.Requests.Total)
	assert.Equal(t, int64(0), snapshot.Requests.InFlight)
	assert.Equal(t, int64(1), snapshot.Requests.Success)
	assert.Equal(t, int64(0), snapshot.Requests.Errors)
	assert.Equal(t, int64(1), snapshot.Requests.ByStatusClass["2xx"])
	assert.Equal(t, int64(1), snapshot.Requests.ByMethod["GET"])
	assert.Equal(t, int64(1), snapshot.Requests.ByPath["/api/users"])
	assert.Equal(t, int64(15), snapshot.Traffic.BytesIn)
	assert.Equal(t, int64(42), snapshot.Traffic.BytesOut)
	assert.InDelta(t, 125, snapshot.Requests.Latency.AvgMS, 0.1)
	assert.Greater(t, snapshot.Requests.RPS, 0.0)
}

func TestMonitoringService_RecordErrorsAndPercentiles(t *testing.T) {
	service := NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})

	service.CompleteRequest(domain.RequestMetric{Method: "GET", Path: "/a", StatusCode: 200, Latency: 10 * time.Millisecond})
	service.CompleteRequest(domain.RequestMetric{Method: "GET", Path: "/b", StatusCode: 503, Latency: 30 * time.Millisecond})
	service.CompleteRequest(domain.RequestMetric{Method: "POST", Path: "/b", StatusCode: 404, Latency: 20 * time.Millisecond})

	snapshot := service.Snapshot()

	assert.Equal(t, int64(3), snapshot.Requests.Total)
	assert.Equal(t, int64(2), snapshot.Requests.Success)
	assert.Equal(t, int64(1), snapshot.Requests.Errors)
	assert.Equal(t, int64(1), snapshot.Requests.ByStatusClass["5xx"])
	assert.Equal(t, int64(1), snapshot.Requests.ByStatusClass["4xx"])
	assert.InDelta(t, 10, snapshot.Requests.Latency.MinMS, 0.1)
	assert.InDelta(t, 30, snapshot.Requests.Latency.MaxMS, 0.1)
	assert.InDelta(t, 20, snapshot.Requests.Latency.AvgMS, 0.1)
	assert.GreaterOrEqual(t, snapshot.Requests.Latency.P95MS, snapshot.Requests.Latency.P50MS)
}

func TestMonitoringService_RecordDomainMetrics(t *testing.T) {
	service := NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})

	service.RecordIPAccess(domain.IPAccessMetric{
		IP:       "10.0.0.1",
		Allowed:  false,
		ListType: domain.DenyList,
		Reason:   "Matched deny list",
	})
	service.RecordIPAccess(domain.IPAccessMetric{
		IP:       "10.0.0.2",
		Allowed:  false,
		ListType: domain.GreyList,
		Reason:   "Matched grey list",
	})
	service.RecordRateLimitViolation(domain.RateLimitMetric{
		Key:    "10.0.0.0/24",
		Reason: "Exceeded requests per second",
	})
	service.RecordUpstream(domain.UpstreamMetric{
		Name:       "api",
		Healthy:    true,
		StatusCode: 200,
		Latency:    15 * time.Millisecond,
	})

	snapshot := service.Snapshot()

	assert.Equal(t, int64(2), snapshot.IPAccess.Total)
	assert.Equal(t, int64(2), snapshot.IPAccess.Denied)
	assert.Equal(t, int64(1), snapshot.IPAccess.Grey)
	assert.Equal(t, int64(1), snapshot.IPAccess.ByReason["Matched deny list"])
	assert.Equal(t, int64(1), snapshot.RateLimit.LimitedRequests)
	assert.Equal(t, int64(1), snapshot.RateLimit.ByKey["10.0.0.0/24"])

	upstream := snapshot.Upstreams["api"]
	assert.True(t, upstream.Healthy)
	assert.Equal(t, int64(1), upstream.TotalRequests)
	assert.InDelta(t, 15, upstream.AvgLatencyMS, 0.1)
}

func TestMonitoringService_Reset(t *testing.T) {
	service := NewMonitoringService(domain.NewDefaultMonitoringConfig(), &testutils.MockLogger{})

	service.CompleteRequest(domain.RequestMetric{Method: "GET", Path: "/", StatusCode: 200})
	service.RecordRateLimitViolation(domain.RateLimitMetric{Key: "client"})
	service.Reset()

	snapshot := service.Snapshot()

	assert.Equal(t, int64(0), snapshot.Requests.Total)
	assert.Equal(t, int64(0), snapshot.RateLimit.LimitedRequests)
	assert.Empty(t, snapshot.Upstreams)
}
