package proxyutil

import (
	"strconv"
)

// Exported node-shaped views of the protocol URI parsers, for the proxy
// subscription importer: it stores generic host/port/credential records and
// needs per-protocol validation plus field extraction.

// SSNode is a validated Shadowsocks SIP002 node.
type SSNode struct {
	Method   string
	Password string
	Host     string
	Port     int
}

// ParseSSNode validates an ss:// URI and extracts the node fields.
func ParseSSNode(raw string) (SSNode, error) {
	cfg, err := parseSSURI(raw)
	if err != nil {
		return SSNode{}, err
	}
	port, _ := strconv.Atoi(cfg.Port)
	return SSNode{Method: cfg.Method, Password: cfg.Password, Host: cfg.Host, Port: port}, nil
}

// TrojanNode is a validated trojan:// node.
type TrojanNode struct {
	Password string
	Host     string
	Port     int
	SNI      string
	Insecure bool
}

// ParseTrojanNode validates a trojan:// URI and extracts the node fields.
func ParseTrojanNode(raw string) (TrojanNode, error) {
	cfg, err := parseTrojanURI(raw)
	if err != nil {
		return TrojanNode{}, err
	}
	port, _ := strconv.Atoi(cfg.Port)
	return TrojanNode{Password: cfg.Password, Host: cfg.Host, Port: port, SNI: cfg.SNI, Insecure: cfg.Insecure}, nil
}

// VMESSNode is a validated vmess:// node.
type VMESSNode struct {
	UUID       string
	Cipher     string
	Network    string
	TLS        bool
	SNI        string
	Host       string // WebSocket Host header override
	Path       string
	Insecure   bool
	ServerHost string
	ServerPort int
}

// ParseVMESSNode validates a vmess:// URI and extracts the node fields.
func ParseVMESSNode(raw string) (VMESSNode, error) {
	cfg, err := ParseVMESSConfig(raw)
	if err != nil {
		return VMESSNode{}, err
	}
	port, _ := strconv.Atoi(cfg.ServerPort)
	return VMESSNode{
		UUID:       cfg.UUID.String(),
		Cipher:     cfg.Cipher,
		Network:    cfg.Network,
		TLS:        cfg.TLS,
		SNI:        cfg.SNI,
		Host:       cfg.Host,
		Path:       cfg.Path,
		Insecure:   cfg.Insecure,
		ServerHost: cfg.ServerHost,
		ServerPort: port,
	}, nil
}
