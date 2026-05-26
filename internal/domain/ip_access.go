package domain

import (
	"net"
	"time"
)

type IPListType string

const (
	AllowList IPListType = "allow"
	DenyList  IPListType = "deny"
	GreyList  IPListType = "grey"
)

type IPEntry struct {
	ID        string         `json:"id"`
	Type      IPListType     `json:"type"`
	Value     string         `json:"value"` //IP or CIDR
	Comment   string         `json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type IPAccessPolicy struct {
	ID            string    `json:"id"`
	DefaultPolicy string    `json:"default_policy"`
	AllowList     []IPEntry `json:"allow_list"`
	DenyList      []IPEntry `json:"deny_list"`
	GreyList      []IPEntry `json:"grey_list"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Version       int       `json:"version"`
}

type IPCheckResult struct {
	IP          string
	IsAllowed   bool
	Reason      string
	ListType    IPListType
	MatchedRule string
	CheckedAt   time.Time
}

type IPFilterConfig struct {
	Enabled           bool
	DefaultPolicy     IPListType
	CacheTTL          time.Duration
	EnableLogging     bool
	EnableMetrics     bool
	MaxListSize       int
	EnableGreyList    bool
	GreyListThreshold int
}

type IPFilterStats struct {
	TotalChecks      int64
	AllowedRequests  int64
	DeniedRequests   int64
	GreyListRequests int64
	CacheHits        int64
	CacheMisses      int64
	AllowListSize    int
	DenyListSize     int
	GreyListSize     int
	LastUpdated      time.Time
}

type IPRange struct {
	Start net.IP
	End   net.IP
	CIDR  *net.IPNet
}

type IPFilterError struct {
	Code    string
	Message string
	IP      string
	Details map[string]any
}

func (e *IPFilterError) Error() string {
	return e.Message
}
