package admin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseClashProxyYAML 解析 Clash 订阅（YAML `proxies:` 列表）。
// 返回 ok=false 表示内容不是 Clash 配置（调用方继续尝试其他格式）；
// ok=true 时节点与逐原因跳过计数有效。
//
// 每种受支持的节点类型都被反向构造为对应的分享链接 URI，然后复用 URI
// 解析路径做校验与字段映射，保证 Clash 路径与明文/链接订阅的行为完全
// 一致（cipher/alterId/transport/flow 等限制由同一套校验给出跳过原因）。
func parseClashProxyYAML(text string) ([]parsedProxySubscription, map[string]int, bool) {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil || doc.Proxies == nil {
		return nil, nil, false
	}
	skipped := make(map[string]int)
	nodes := make([]parsedProxySubscription, 0, len(doc.Proxies))
	for _, entry := range doc.Proxies {
		if len(nodes) >= proxySubscriptionMaxNodes {
			break
		}
		node, err := parseClashProxyEntry(entry)
		if err != nil {
			reason := err.Error()
			if reason == "" {
				reason = "unsupported node"
			}
			skipped[reason]++
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, skipped, true
}

// parseClashProxyEntry 把单个 Clash proxy 映射到通用节点记录。
func parseClashProxyEntry(entry map[string]any) (parsedProxySubscription, error) {
	server := strings.TrimSpace(clashYAMLString(entry, "server"))
	if server == "" {
		return parsedProxySubscription{}, errors.New("node is missing server")
	}
	port := clashYAMLInt(entry, "port")
	if port < 1 || port > 65535 {
		return parsedProxySubscription{}, errors.New("invalid node port")
	}
	name := clashYAMLString(entry, "name")
	switch strings.ToLower(clashYAMLString(entry, "type")) {
	case "ss":
		return parseClashSSNode(entry, server, port, name)
	case "vmess":
		return parseClashVMESSNode(entry, server, port, name)
	case "vless":
		return parseClashVLESSNode(entry, server, port, name)
	case "trojan":
		return parseClashTrojanNode(entry, server, port, name)
	default:
		return parsedProxySubscription{}, fmt.Errorf("unsupported node type %s", strings.TrimSpace(clashYAMLString(entry, "type")))
	}
}

// parseClashSSNode 构造 SIP002 ss:// URI 后复用现有解析（method/密码校验、
// unsupported plugin 的跳过语义均由此给出）。
func parseClashSSNode(entry map[string]any, server string, port int, name string) (parsedProxySubscription, error) {
	if clashYAMLString(entry, "plugin") != "" {
		return parsedProxySubscription{}, errors.New("unsupported shadowsocks plugin")
	}
	method := clashYAMLString(entry, "cipher")
	password := clashYAMLString(entry, "password")
	if method == "" || password == "" {
		return parsedProxySubscription{}, errors.New("shadowsocks node is missing cipher/password")
	}
	uri := &url.URL{
		Scheme:   "ss",
		User:     url.UserPassword(method, password),
		Host:     net.JoinHostPort(server, strconv.Itoa(port)),
		Fragment: name,
	}
	return parseSSSubscriptionURI(uri)
}

// parseClashVMESSNode 构造 base64(JSON) vmess:// URI 后复用现有解析
// （cipher/alterId/transport 校验均由此给出）。
func parseClashVMESSNode(entry map[string]any, server string, port int, name string) (parsedProxySubscription, error) {
	wsOpts, _ := entry["ws-opts"].(map[string]any)
	wsHeaders, _ := wsOpts["headers"].(map[string]any)
	payload := map[string]string{
		"v":    "2",
		"ps":   name,
		"add":  server,
		"port": strconv.Itoa(port),
		"id":   clashYAMLString(entry, "uuid"),
		"aid":  strconv.Itoa(clashYAMLInt(entry, "alterId")),
		"scy":  clashYAMLString(entry, "cipher"),
		"net":  clashYAMLString(entry, "network"),
		"type": "none",
		"host": clashYAMLString(wsHeaders, "Host"),
		"path": clashYAMLString(wsOpts, "path"),
		"sni":  clashYAMLString(entry, "servername"),
	}
	if clashYAMLBool(entry, "tls") {
		payload["tls"] = "tls"
	}
	if clashYAMLBool(entry, "skip-cert-verify") {
		payload["allowInsecure"] = "1"
	}
	if payload["id"] == "" {
		return parsedProxySubscription{}, errors.New("vmess node is missing uuid")
	}
	return parseVMESSSubscriptionURI(serializeClashVMESSURI(payload, name))
}

// serializeClashVMESSURI 生成标准 base64(JSON) vmess:// URI（与
// proxyutil.SerializeVMESSURL 同一编码约定：RawStdEncoding）。节点名走
// URI fragment（与 nodeDisplayName 的取名逻辑衔接）。
func serializeClashVMESSURI(payload map[string]string, name string) string {
	keys := []string{"v", "ps", "add", "port", "id", "aid", "scy", "net", "type", "host", "path", "tls", "sni", "allowInsecure"}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == "" {
			continue
		}
		keyJSON, _ := json.Marshal(key)
		valueJSON, _ := json.Marshal(value)
		parts = append(parts, string(keyJSON)+":"+string(valueJSON))
	}
	encoded := base64.RawStdEncoding.EncodeToString([]byte("{" + strings.Join(parts, ",") + "}"))
	uri := "vmess://" + encoded
	if name != "" {
		uri += (&url.URL{Fragment: name}).String()
	}
	return uri
}

// parseClashVLESSNode 构造 vless:// URI 后复用现有解析（reality/xtls/grpc
// 等不受支持的传输由 normalizeVLESSSubscriptionOptions 给出跳过原因）。
func parseClashVLESSNode(entry map[string]any, server string, port int, name string) (parsedProxySubscription, error) {
	query := url.Values{}
	if clashYAMLBool(entry, "tls") {
		query.Set("security", "tls")
	}
	if _, hasReality := entry["reality-opts"]; hasReality {
		query.Set("security", "reality")
	}
	network := clashYAMLString(entry, "network")
	if network != "" {
		query.Set("type", network)
	}
	if sni := clashYAMLString(entry, "servername"); sni != "" {
		query.Set("sni", sni)
	}
	if clashYAMLBool(entry, "skip-cert-verify") {
		query.Set("insecure", "1")
	}
	if flow := clashYAMLString(entry, "flow"); flow != "" {
		query.Set("flow", flow)
	}
	if wsOpts, ok := entry["ws-opts"].(map[string]any); ok {
		query.Set("path", clashYAMLString(wsOpts, "path"))
		if headers, ok := wsOpts["headers"].(map[string]any); ok {
			query.Set("host", clashYAMLString(headers, "Host"))
		}
	}
	uri := &url.URL{
		Scheme:   "vless",
		User:     url.User(clashYAMLString(entry, "uuid")),
		Host:     net.JoinHostPort(server, strconv.Itoa(port)),
		RawQuery: query.Encode(),
		Fragment: name,
	}
	if uri.User.Username() == "" {
		return parsedProxySubscription{}, errors.New("vless node is missing uuid")
	}
	return parseVLESSSubscriptionURI(uri)
}

// parseClashTrojanNode 构造 trojan:// URI 后复用现有解析。
func parseClashTrojanNode(entry map[string]any, server string, port int, name string) (parsedProxySubscription, error) {
	password := clashYAMLString(entry, "password")
	if password == "" {
		return parsedProxySubscription{}, errors.New("trojan node is missing password")
	}
	query := url.Values{}
	if sni := clashYAMLString(entry, "sni"); sni != "" {
		query.Set("sni", sni)
		query.Set("peer", sni)
	}
	if clashYAMLBool(entry, "skip-cert-verify") {
		query.Set("allowInsecure", "1")
	}
	uri := &url.URL{
		Scheme:   "trojan",
		User:     url.User(password),
		Host:     net.JoinHostPort(server, strconv.Itoa(port)),
		RawQuery: query.Encode(),
		Fragment: name,
	}
	return parseTrojanSubscriptionURI(uri)
}

// ===== Clash YAML 字段取值辅助 =====

func clashYAMLString(entry map[string]any, key string) string {
	switch value := entry[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

func clashYAMLInt(entry map[string]any, key string) int {
	switch value := entry[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(value))
		return parsed
	default:
		return 0
	}
}

func clashYAMLBool(entry map[string]any, key string) bool {
	switch value := entry[key].(type) {
	case bool:
		return value
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(value))
		return parsed
	default:
		return false
	}
}
