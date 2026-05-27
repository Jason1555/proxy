package infrastructure

import (
	"fmt"
	"net"
	"proxy/internal/domain"
	"strings"
)

type Parser struct{}

func NewParser() domain.IPParser {
	return &Parser{}
}

func (p *Parser) Parse(ip string) (net.IP, error) {
	ip = strings.TrimSpace(ip)
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	return parsed, nil
}

func (p *Parser) ParseCIDR(cidr string) (*net.IPNet, error) {
	cidr = strings.TrimSpace(cidr)
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %s, %w", cidr, err)
	}

	return ipNet, nil
}

func (p *Parser) IsValidIP(ip string) bool {
	_, err := p.Parse(ip)
	return err == nil
}

func (p *Parser) IsValidCIDR(cidr string) bool {
	_, err := p.ParseCIDR(cidr)
	return err == nil
}
