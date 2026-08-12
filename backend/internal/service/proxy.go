package service

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	FallbackModeNone             = "none"
	FallbackModeProxy            = "proxy"
	FallbackModeDirect           = "direct"
	hysteria2TLSInsecureUsername = "__sub3api_hy2_insecure__"
)

type Proxy struct {
	ID             int64
	Name           string
	Protocol       string
	Host           string
	Port           int
	Username       string
	Password       string
	TLSInsecure    bool
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ExpiresAt      *time.Time
	FallbackMode   string
	BackupProxyID  *int64
	ExpiryWarnDays int
}

func (p *Proxy) IsActive() bool {
	return p.Status == StatusActive
}

// IsExpired 报告代理是否已过期（基于 expires_at，与 status 无关）。
func (p *Proxy) IsExpired(now time.Time) bool {
	return p.ExpiresAt != nil && !p.ExpiresAt.After(now)
}

func (p *Proxy) URL() string {
	u := &url.URL{
		Scheme: p.Protocol,
		Host:   net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
	}
	if p.Protocol == "hysteria2" {
		if p.Password != "" {
			parts := strings.SplitN(p.Password, ":", 2)
			if len(parts) == 2 {
				u.User = url.UserPassword(parts[0], parts[1])
			} else {
				u.User = url.User(p.Password)
			}
		}
		if p.TLSInsecure {
			query := u.Query()
			query.Set("insecure", "1")
			u.RawQuery = query.Encode()
		}
	} else if p.Username != "" && p.Password != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u.String()
}

func (p *Proxy) PersistencePassword() string {
	return p.Password
}

func (p *Proxy) PersistenceUsername() string {
	if p.Protocol == "hysteria2" {
		if p.TLSInsecure {
			return hysteria2TLSInsecureUsername
		}
		return ""
	}
	return p.Username
}

func (p *Proxy) ApplyPersistenceAuth(username, password string) {
	if p.Protocol != "hysteria2" {
		p.Username = username
		p.Password = password
		return
	}
	p.Username = ""
	p.Password = password
	p.TLSInsecure = username == hysteria2TLSInsecureUsername
}

type ProxyWithAccountCount struct {
	Proxy
	AccountCount   int64
	LatencyMs      *int64
	LatencyStatus  string
	LatencyMessage string
	IPAddress      string
	Country        string
	CountryCode    string
	Region         string
	City           string
	QualityStatus  string
	QualityScore   *int
	QualityGrade   string
	QualitySummary string
	QualityChecked *int64
}

type ProxyAccountSummary struct {
	ID       int64
	Name     string
	Platform string
	Type     string
	Status   string
	Notes    *string
}
