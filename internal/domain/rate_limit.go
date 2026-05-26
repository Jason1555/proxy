package domain

import "time"

type RateLimitConfig struct {
	RPS int64
	RPM int64
	RPH int64
	RPD int64

	DownloadBytesPerSecond int64
	UploadBytesPerSecond   int64
	TotalBytesPerDay       int64

	MaxConcurrentConnections int64
	NewConnectionsPerSecond  int64

	SubnetMask string

	Enabled bool
}

type RateLimitState struct {
	Key string

	RPSTokens float64
	RPMTokens float64
	RPHTokens float64
	RPDTokens float64

	DownloadTokens   float64
	UploadTokens     float64
	TotalBytesTokens float64

	NewConnectionsTokens float64

	ActiveConnections int64
	TotalRequests     int64
	LimitedRequests   int64

	LastRefillAt time.Time
	CreatedAt    time.Time
	LastSeenAt   time.Time
}

type RateLimitStatus struct {
	Key    string // IP или subnet
	IP     string
	Subnet string

	RPSRemaining int64
	RPMRemaining int64
	RPHRemaining int64
	RPDRemaining int64

	DownloadRemaining   int64
	UploadRemaining     int64
	TotalBytesRemaining int64

	NewConnectionsRemaining int64

	ActiveConnections int64

	IsLimited bool
	Reason    string

	// когда примерно восстановится лимит
	RetryAfter time.Duration

	LimitResetAt map[string]time.Time
}

type RateLimitViolation struct {
	Key       string // IP или subnet
	Timestamp time.Time
	LimitType string
	Reason    string
}

type RateLimitMetrics struct {
	TotalRequests     int64
	LimitedRequests   int64
	BandwidthLimited  int64
	ConnectionLimited int64

	UniqueClients int64

	TopViolators []RateLimitViolator

	ViolationsLastHour int64

	AverageRequestsPerClient float64
}

type RateLimitViolator struct {
	Key           string // IP или subnet
	Violations    int64
	LastViolation time.Time
}
