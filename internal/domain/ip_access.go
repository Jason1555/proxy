package domain

import (
	"net"
	"time"
)

type IPListType string

const (
	WhiteList IPListType = "whitelist"
	BlackList IPListType = "blacklist"
	GreyList  IPListType = "greylist"
)

type IPEntry struct {
	ID        string     `json:"id"`
	Type      IPListType `json:"type"`
	IP 	  string     `json:"ip"`
	CIDR 	*net.IPNet     `json:"-"`
	StartIP net.IP     `json:"-"`
	EndIP   net.IP     `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IPAccessPolicy struct {
	DefaultPolicy IPListType
	AllowList	 []IPEntry
	BanList	 []IPEntry
	GreyList	 []IPEntry
	LastUpdated time.Time
}

type IPCheckResult struct {
	Allowed bool
	Reason  string
	ListType IPListType
	CheckedAt time.Time
	CachedResult bool
}

type AccessLog struct {
	Timestamp   time.Time `json:"timestamp"`
	ClientIP    string    `json:"client_ip"`
	URL         string    `json:"url"`
	Method      string    `json:"method"`
	StatusCode  int       `json:"status_code"`
	Decision    string    `json:"decision"`
	Reason      string    `json:"reason"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	Duration    float64   `json:"duration_ms"`
}
