package http

import (
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"proxy/internal/domain"
)

func MetricsMiddleware(collector domain.MonitoringCollector) func(stdhttp.Handler) stdhttp.Handler {
	return func(next stdhttp.Handler) stdhttp.Handler {
		return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			startedAt := time.Now()
			collector.BeginRequest()

			recorder := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     stdhttp.StatusOK,
			}

			defer func() {
				collector.CompleteRequest(domain.RequestMetric{
					Method:     r.Method,
					Path:       r.URL.Path,
					ClientIP:   requestClientIP(r),
					StatusCode: recorder.statusCode,
					BytesIn:    requestBodySize(r),
					BytesOut:   recorder.bytesWritten,
					Latency:    time.Since(startedAt),
					Timestamp:  time.Now(),
				})
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}

type metricsResponseWriter struct {
	stdhttp.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (w *metricsResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *metricsResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(w.statusCode)
	}

	written, err := w.ResponseWriter.Write(body)
	w.bytesWritten += int64(written)
	return written, err
}

func (w *metricsResponseWriter) Unwrap() stdhttp.ResponseWriter {
	return w.ResponseWriter
}

func requestBodySize(r *stdhttp.Request) int64 {
	if r.ContentLength > 0 {
		return r.ContentLength
	}
	return 0
}

func requestClientIP(r *stdhttp.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
