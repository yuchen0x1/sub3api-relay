package service

import (
	"encoding/base64"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/pkg/proxyutil"
)

const (
	FallbackModeNone   = "none"
	FallbackModeProxy  = "proxy"
	FallbackModeDirect = "direct"
)

type Proxy struct {
	ID       int64
	Name     string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	// Options contains transport-specific settings. VLESS subscriptions use
	// this map for WebSocket path/SNI/Host and TLS options; other protocols
	// normally leave it empty.
	Options        map[string]string
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
	switch p.Protocol {
	case "vless":
		if p.Username != "" {
			u.User = url.User(p.Username)
		}
		u.RawQuery = encodeProxyOptionsQuery(p.Options)
	case "ss":
		// SIP002: websafe base64 of "method:password" as the userinfo.
		if p.Username != "" || p.Password != "" {
			u.User = url.User(base64.RawURLEncoding.EncodeToString(
				[]byte(p.Username + ":" + p.Password)))
		}
	case "trojan":
		// The password is the userinfo username in trojan URIs.
		if p.Username != "" {
			u.User = url.User(p.Username)
		}
		u.RawQuery = encodeProxyOptionsQuery(p.Options)
	case "vmess":
		aid := p.Password
		if aid == "" {
			aid = "0"
		}
		return proxyutil.SerializeVMESSURL(p.Username, aid, p.Host, strconv.Itoa(p.Port), p.Options)
	default:
		if p.Username != "" && p.Password != "" {
			u.User = url.UserPassword(p.Username, p.Password)
		}
	}
	return u.String()
}

// encodeProxyOptionsQuery serializes transport options into URL query form,
// skipping empty values the way the dialers' parsers do.
func encodeProxyOptionsQuery(options map[string]string) string {
	query := url.Values{}
	for key, value := range options {
		if strings.TrimSpace(key) != "" && value != "" {
			query.Set(key, value)
		}
	}
	return query.Encode()
}

func (p *Proxy) PersistencePassword() string {
	return p.Password
}

func (p *Proxy) PersistenceUsername() string {
	return p.Username
}

func (p *Proxy) ApplyPersistenceAuth(username, password string) {
	p.Username = username
	p.Password = password
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
