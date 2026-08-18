package backend

import (
	"ant-chrome/backend/internal/config"
	"ant-chrome/backend/internal/proxy"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	maxClashSubscriptionBytes = 8 * 1024 * 1024
	clashSubscriptionTimeout  = 10 * time.Second
)

const maxClashSubscriptionUserAgentBytes = 512

type ClashSubscriptionUserAgentOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	UserAgent string `json:"userAgent"`
	Source    string `json:"source"`
}

type ClashSubscriptionFetchOptions struct {
	ProxyID         string `json:"proxyId"`
	UserAgent       string `json:"userAgent"`
	FallbackEnabled bool   `json:"fallbackEnabled"`
}

var clashSubscriptionUserAgentOptions = []ClashSubscriptionUserAgentOption{
	{ID: "ant", Label: "Ant Browser 默认", UserAgent: "clash-verge/2.0 ant-chrome/1.0", Source: "Ant Browser"},
	{ID: "flclash-windows", Label: "FlClash Windows", UserAgent: "FlClash/v0.8.92 clash-verge Platform/windows", Source: "FlClash"},
	{ID: "flclash-clash-verge", Label: "FlClash 兼容 · Clash Verge", UserAgent: "clash-verge/v2.4.2", Source: "FlClash preset"},
	{ID: "flclash-cfw", Label: "FlClash 兼容 · Clash for Windows", UserAgent: "ClashforWindows/0.19.23", Source: "FlClash preset"},
}

var clashSubscriptionUserAgents = func() []string {
	result := make([]string, 0, len(clashSubscriptionUserAgentOptions))
	for _, option := range clashSubscriptionUserAgentOptions {
		result = append(result, option.UserAgent)
	}
	return result
}()

// BrowserProxySubscriptionUserAgents 返回内置的订阅 User-Agent 列表。
func (a *App) BrowserProxySubscriptionUserAgents() []ClashSubscriptionUserAgentOption {
	result := make([]ClashSubscriptionUserAgentOption, len(clashSubscriptionUserAgentOptions))
	copy(result, clashSubscriptionUserAgentOptions)
	return result
}

// BrowserProxyFetchClashByURL 从 Clash 订阅 URL 获取并校验 YAML 配置。
// 为保持旧接口兼容，默认会在请求失败时切换 User-Agent。
func (a *App) BrowserProxyFetchClashByURL(rawURL string) (map[string]interface{}, error) {
	return a.browserProxyFetchClashByURL(rawURL, ClashSubscriptionFetchOptions{FallbackEnabled: true})
}

// BrowserProxyFetchClashByURLWithProxy 通过指定代理获取 Clash 订阅。
// 为保持旧接口兼容，默认会在请求失败时切换 User-Agent。
func (a *App) BrowserProxyFetchClashByURLWithProxy(rawURL string, proxyID string) (map[string]interface{}, error) {
	return a.browserProxyFetchClashByURL(rawURL, ClashSubscriptionFetchOptions{ProxyID: proxyID, FallbackEnabled: true})
}

// BrowserProxyFetchClashByURLWithOptions 支持选择/自定义 User-Agent，并控制失败后是否切换。
func (a *App) BrowserProxyFetchClashByURLWithOptions(rawURL string, options ClashSubscriptionFetchOptions) (map[string]interface{}, error) {
	return a.browserProxyFetchClashByURL(rawURL, options)
}

func (a *App) browserProxyFetchClashByURL(rawURL string, options ClashSubscriptionFetchOptions) (map[string]interface{}, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("订阅 URL 不能为空")
	}
	proxyID := strings.TrimSpace(options.ProxyID)

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		return nil, fmt.Errorf("URL 格式无效")
	}
	scheme := strings.ToLower(strings.TrimSpace(parsedURL.Scheme))
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("仅支持 http/https URL")
	}

	client := &http.Client{Timeout: clashSubscriptionTimeout}
	if proxyID != "" {
		proxyClient, err := a.clashSubscriptionProxyClient(proxyID)
		if err != nil {
			return nil, err
		}
		client = proxyClient
	}
	content, payload, usedUserAgent, attemptedUserAgents, err := fetchClashSubscription(
		client,
		parsedURL.String(),
		options.UserAgent,
		options.FallbackEnabled,
		clashSubscriptionTimeout,
	)
	if err != nil {
		return nil, err
	}

	proxyCount := clashProxyCount(payload)
	if proxyCount <= 0 {
		return nil, fmt.Errorf("订阅内容缺少 proxies 节点")
	}

	dnsYAML := extractClashDNSYAML(payload)
	suggestedGroup := suggestClashGroupName(payload, parsedURL.Hostname())

	return map[string]interface{}{
		"url":                 parsedURL.String(),
		"content":             content,
		"proxyCount":          proxyCount,
		"dnsServers":          dnsYAML,
		"suggestedGroup":      suggestedGroup,
		"usedUserAgent":       usedUserAgent,
		"attemptedUserAgents": attemptedUserAgents,
		"fallbackUsed":        len(attemptedUserAgents) > 1,
	}, nil
}

func (a *App) clashSubscriptionProxyClient(proxyID string) (*http.Client, error) {
	if a == nil || a.config == nil {
		return nil, fmt.Errorf("代理拉取需要应用配置已初始化")
	}
	proxies := a.getLatestProxies()
	found := false
	for _, item := range proxies {
		if strings.EqualFold(strings.TrimSpace(item.ProxyId), proxyID) && strings.TrimSpace(item.ProxyConfig) != "" {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("拉取代理不存在或配置为空")
	}
	connectorType := config.NormalizeBrowserConnectorType(a.config.Browser.DefaultConnectorType)
	return proxy.BuildProxyHTTPClient("", proxyID, proxies, a.xrayMgr, a.singboxMgr, a.clashMgr, connectorType, clashSubscriptionTimeout)
}

func fetchClashSubscriptionWithFallback(client *http.Client, targetURL string, timeout time.Duration) (string, interface{}, error) {
	content, payload, _, _, err := fetchClashSubscription(client, targetURL, "", true, timeout)
	return content, payload, err
}

func fetchClashSubscription(client *http.Client, targetURL string, preferredUserAgent string, fallbackEnabled bool, timeout time.Duration) (string, interface{}, string, []string, error) {
	userAgents, err := buildClashSubscriptionUserAgentOrder(preferredUserAgent, fallbackEnabled)
	if err != nil {
		return "", nil, "", nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var lastErr error
	attempted := make([]string, 0, len(userAgents))
	for _, userAgent := range userAgents {
		if ctx.Err() != nil {
			return "", nil, "", attempted, clashSubscriptionTimeoutError(timeout)
		}
		attempted = append(attempted, userAgent)
		content, payload, fetchErr := fetchClashSubscriptionWithUserAgent(ctx, client, targetURL, userAgent)
		if fetchErr == nil {
			return content, payload, userAgent, attempted, nil
		}
		lastErr = fetchErr
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", nil, "", attempted, clashSubscriptionTimeoutError(timeout)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("没有可用的 User-Agent")
	}
	return "", nil, "", attempted, fmt.Errorf("订阅获取失败: %w", lastErr)
}

func buildClashSubscriptionUserAgentOrder(preferredUserAgent string, fallbackEnabled bool) ([]string, error) {
	preferredUserAgent = strings.TrimSpace(preferredUserAgent)
	if err := validateClashSubscriptionUserAgent(preferredUserAgent); err != nil {
		return nil, err
	}

	candidates := make([]string, 0, len(clashSubscriptionUserAgents)+1)
	if preferredUserAgent != "" {
		candidates = append(candidates, preferredUserAgent)
	}
	if fallbackEnabled || preferredUserAgent == "" {
		candidates = append(candidates, clashSubscriptionUserAgents...)
	}

	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		key := strings.ToLower(candidate)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("没有可用的 User-Agent")
	}
	return result, nil
}

func validateClashSubscriptionUserAgent(userAgent string) error {
	if userAgent == "" {
		return nil
	}
	if len([]byte(userAgent)) > maxClashSubscriptionUserAgentBytes {
		return fmt.Errorf("User-Agent 不能超过 512 字节")
	}
	if strings.ContainsAny(userAgent, "\r\n") {
		return fmt.Errorf("User-Agent 不能包含换行符")
	}
	return nil
}

func clashSubscriptionTimeoutError(timeout time.Duration) error {
	seconds := int(timeout / time.Second)
	if seconds <= 0 {
		return fmt.Errorf("拉取订阅超时")
	}
	return fmt.Errorf("拉取订阅超时（%d秒）", seconds)
}

func fetchClashSubscriptionWithUserAgent(ctx context.Context, client *http.Client, targetURL string, userAgent string) (string, interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("创建请求失败")
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/yaml,text/yaml,text/plain,*/*")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("网络请求失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxClashSubscriptionBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("读取订阅内容失败")
	}
	if len(body) > maxClashSubscriptionBytes {
		return "", nil, fmt.Errorf("订阅内容过大（超过 8MB）")
	}

	content, payload, err := normalizeClashSubscriptionContent(body)
	if err != nil {
		return "", nil, err
	}
	return content, payload, nil
}

func normalizeClashSubscriptionContent(body []byte) (string, interface{}, error) {
	baseText := strings.TrimSpace(strings.ReplaceAll(string(body), "\r\n", "\n"))
	if baseText == "" {
		return "", nil, fmt.Errorf("订阅内容为空")
	}

	tryTexts := make([]string, 0, 4)
	tryTexts = append(tryTexts, baseText)

	if unescaped, err := url.QueryUnescape(baseText); err == nil {
		unescaped = strings.TrimSpace(strings.ReplaceAll(unescaped, "\r\n", "\n"))
		if unescaped != "" && unescaped != baseText {
			tryTexts = append(tryTexts, unescaped)
		}
	}

	if decoded, ok := decodeBase64Text(baseText); ok {
		tryTexts = append(tryTexts, decoded)
	}

	for _, text := range tryTexts {
		payload, ok := parseClashPayload(text)
		if !ok {
			continue
		}
		if clashProxyCount(payload) > 0 {
			return text, payload, nil
		}
	}

	for _, text := range tryTexts {
		converted, ok := convertProxyURIListToClashYAML(text)
		if !ok {
			continue
		}
		payload, parsed := parseClashPayload(converted)
		if parsed && clashProxyCount(payload) > 0 {
			return converted, payload, nil
		}
	}

	return "", nil, fmt.Errorf("URL 内容不是有效 Clash YAML 或 URI 订阅（需包含 proxies 或支持的代理 URI）")
}

func convertProxyURIListToClashYAML(text string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	proxies := make([]map[string]interface{}, 0)
	for index, line := range lines {
		raw := strings.TrimSpace(line)
		if raw == "" {
			continue
		}
		node, ok := proxyURIToClashNode(raw, index)
		if ok {
			proxies = append(proxies, node)
		}
	}
	if len(proxies) == 0 {
		return "", false
	}
	payload := map[string]interface{}{"proxies": proxies}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func proxyURIToClashNode(raw string, index int) (map[string]interface{}, bool) {
	u, err := url.Parse(raw)
	if err != nil || strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Hostname()) == "" {
		return nil, false
	}
	scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
	port := 0
	if portText := strings.TrimSpace(u.Port()); portText != "" {
		if parsedPort, err := strconv.Atoi(portText); err == nil {
			port = parsedPort
		}
	}
	if port == 0 {
		return nil, false
	}
	name := proxyURIName(u, index)
	switch scheme {
	case "trojan":
		password := u.User.Username()
		if password == "" {
			return nil, false
		}
		node := map[string]interface{}{
			"name":     name,
			"type":     "trojan",
			"server":   u.Hostname(),
			"port":     port,
			"password": password,
		}
		q := u.Query()
		if sni := firstNonEmptyQueryParam(q, "sni", "peer", "servername"); sni != "" {
			node["sni"] = sni
		}
		if network := firstNonEmptyQueryParam(q, "type", "network"); network != "" {
			node["network"] = network
		}
		if uriBoolParam(q, "insecure", "allowInsecure", "skip-cert-verify") {
			node["skip-cert-verify"] = true
		}
		return node, true
	default:
		return nil, false
	}
}

func proxyURIName(u *url.URL, index int) string {
	if u.Fragment != "" {
		if name, err := url.QueryUnescape(u.Fragment); err == nil && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
		if strings.TrimSpace(u.Fragment) != "" {
			return strings.TrimSpace(u.Fragment)
		}
	}
	return fmt.Sprintf("导入代理 %d", index+1)
}

func firstNonEmptyQueryParam(q url.Values, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(q.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func uriBoolParam(q url.Values, keys ...string) bool {
	for _, key := range keys {
		value := strings.ToLower(strings.TrimSpace(q.Get(key)))
		if value == "1" || value == "true" || value == "yes" {
			return true
		}
	}
	return false
}

func decodeBase64Text(raw string) (string, bool) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return "", false
	}
	// 一些订阅会返回 URL-safe base64 或缺少 padding，这里都尝试一遍。
	padded := candidate
	if mod := len(padded) % 4; mod != 0 {
		padded += strings.Repeat("=", 4-mod)
	}

	encoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encoders {
		if data, err := enc.DecodeString(candidate); err == nil {
			decoded := strings.TrimSpace(strings.ReplaceAll(string(data), "\r\n", "\n"))
			if decoded != "" {
				return decoded, true
			}
		}
		if data, err := enc.DecodeString(padded); err == nil {
			decoded := strings.TrimSpace(strings.ReplaceAll(string(data), "\r\n", "\n"))
			if decoded != "" {
				return decoded, true
			}
		}
	}
	return "", false
}

func parseClashPayload(text string) (interface{}, bool) {
	var payload interface{}
	if err := yaml.Unmarshal([]byte(text), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func clashProxyCount(payload interface{}) int {
	if m := toStringMap(payload); m != nil {
		if arr, ok := m["proxies"].([]interface{}); ok {
			return len(arr)
		}
		if arr, ok := m["proxy"].([]interface{}); ok {
			return len(arr)
		}
		if arr, ok := m["Proxy"].([]interface{}); ok {
			return len(arr)
		}
	}
	if arr, ok := payload.([]interface{}); ok {
		return len(arr)
	}
	return 0
}

func extractClashDNSYAML(payload interface{}) string {
	m := toStringMap(payload)
	if m == nil {
		return ""
	}
	dnsRaw, exists := m["dns"]
	if !exists || dnsRaw == nil {
		return ""
	}
	data, err := yaml.Marshal(map[string]interface{}{
		"dns": dnsRaw,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func suggestClashGroupName(payload interface{}, fallbackHost string) string {
	fallbackHost = strings.TrimSpace(fallbackHost)
	m := toStringMap(payload)
	if m != nil {
		if groups, ok := m["proxy-groups"].([]interface{}); ok {
			for _, item := range groups {
				if groupMap := toStringMap(item); groupMap != nil {
					if name := strings.TrimSpace(getMapString(groupMap, "name")); name != "" {
						return name
					}
				}
			}
		}
	}
	if strings.HasPrefix(strings.ToLower(fallbackHost), "www.") {
		fallbackHost = fallbackHost[4:]
	}
	return fallbackHost
}

func toStringMap(value interface{}) map[string]interface{} {
	switch m := value.(type) {
	case map[string]interface{}:
		return m
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, v := range m {
			key := fmt.Sprint(k)
			out[key] = v
		}
		return out
	default:
		return nil
	}
}

func getMapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}
