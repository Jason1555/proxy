package domain

import (
	"context"
	"net"
)

type IPParser interface {
	Parse(ip string) (net.IP, error)
	ParseCIDR(cidr string) (*net.IPNet, error)
	IsValidIP(ip string) bool
	IsValidCIDR(cidr string) bool
}

type IPMatcher interface {
	Match(ip net.IP, entries []IPEntry) (bool, *IPEntry)
	MatchCIDR(ip net.IP, cidr *net.IPNet) bool
	MatchRange(ip net.IP, start, end net.IP) bool
}

type IPFilter interface {
	CheckIP(ctx context.Context, ip string) (*IPCheckResult, error)
	CheckIPBatch(ctx context.Context, ips []string) ([]IPCheckResult, error)
	AddToAllowList(ctx context.Context, pattern, comment string) (*IPEntry, error)
	AddToDenyList(ctx context.Context, pattern, comment string) (*IPEntry, error)
	AddToGreyList(ctx context.Context, pattern, comment string) (*IPEntry, error)
	RemoveEntry(ctx context.Context, id string) error
	GetPolicy(ctx context.Context) (*IPAccessPolicy, error)
	SetPolicy(ctx context.Context, policy *IPAccessPolicy) error
	ReloadPolicy(ctx context.Context) error
	GetStats(ctx context.Context) (*IPFilterStats, error)
	ClearCache(ctx context.Context) error
	Validate(ctx context.Context) error
}

type IPFilterCache interface {
	Get(ip string) (*IPCheckResult, bool)
	Set(ip string, result *IPCheckResult)
	Clear()
	GetStats() map[string]any
}
