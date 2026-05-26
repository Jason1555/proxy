package infrastructure

import (
	"net"
	"proxy/internal/domain"
)

type Matcher struct {
	parser domain.IPParser
}

func NewMatcher(parser domain.IPParser) domain.IPMatcher {
	return &Matcher{
		parser: parser,
	}
}

func (m *Matcher) Match(ip net.IP, entries []domain.IPEntry) (bool, *domain.IPEntry) {
	for i := range entries {
		entry := &entries[i]

		if entry.Value == ip.String() {
			return true, entry
		}

		if m.parser.IsValidCIDR(entry.Value) {
			if ipNet, err := m.parser.ParseCIDR(entry.Value); err == nil {
				if m.MatchCIDR(ip, ipNet) {
					return true, entry
				}
			}
		}
	}

	return false, nil
}

func (m *Matcher) MatchCIDR(ip net.IP, cidr *net.IPNet) bool {
	if cidr == nil {
		return false
	}

	return cidr.Contains(ip)
}

func (m *Matcher) MatchRange(ip net.IP, start, end net.IP) bool {
	if start == nil || end == nil {
		return false
	}

	startNum := ipToInt(start)
	endNum := ipToInt(end)
	ipNum := ipToInt(ip)

	return ipNum >= startNum && ipNum <= endNum
}

func ipToInt(ip net.IP) int64 {
	return int64(ip[0])<<24 | int64(ip[1])<<16 | int64(ip[2])<<8 | int64(ip[3])
}
