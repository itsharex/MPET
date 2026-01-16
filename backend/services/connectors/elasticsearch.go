package connectors

import (
	"MPET/backend/models"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ConnectElasticsearch 连接 Elasticsearch
func (s *ConnectorService) ConnectElasticsearch(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "9200"
	}

	hostInput := strings.TrimSpace(conn.IP)
	if hostInput == "" {
		conn.Status = "failed"
		conn.Message = "连接失败: 未指定目标地址"
		s.AddLog(conn, "未指定目标地址")
		return
	}

	defaultPath := "/_nodes"
	requestPath := ""
	scheme := "http"

	lowerHost := strings.ToLower(hostInput)
	if strings.HasPrefix(lowerHost, "http://") || strings.HasPrefix(lowerHost, "https://") {
		if parsed, err := url.Parse(hostInput); err == nil {
			scheme = parsed.Scheme
			hostInput = parsed.Host
			requestPath = parsed.RequestURI()
		}
	} else if strings.Contains(hostInput, "/") {
		parts := strings.SplitN(hostInput, "/", 2)
		hostInput = parts[0]
		requestPath = "/" + parts[1]
	}

	if requestPath == "" || requestPath == "/" {
		requestPath = defaultPath
	}

	normalizedHost := hostInput
	if _, _, err := net.SplitHostPort(normalizedHost); err != nil {
		normalizedHost = net.JoinHostPort(normalizedHost, port)
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, normalizedHost)
	targetURL := baseURL + requestPath

	s.AddLog(conn, fmt.Sprintf("目标地址: %s", targetURL))
	s.AddLog(conn, fmt.Sprintf("请求路径: %s", requestPath))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		DisableKeepAlives:     true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return s.DialContextWithProxy(ctx, network, address)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("创建请求失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	req.Header.Set("Accept", "application/json, text/plain;q=0.9, */*;q=0.8")
	req.Header.Set("User-Agent", "AttackLogin-Elasticsearch-Scanner/1.0")

	if conn.User != "" || conn.Pass != "" {
		s.AddLog(conn, "使用 Basic Auth 进行认证")
		req.SetBasicAuth(conn.User, conn.Pass)
	}

	s.AddLog(conn, "发送 HTTP 请求...")
	resp, err := client.Do(req)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("请求失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("读取响应失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	body := strings.TrimSpace(string(bodyBytes))
	if body == "" {
		body = "(响应为空)"
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.AddLog(conn, fmt.Sprintf("✓ HTTP %d 请求成功", resp.StatusCode))
		conn.Status = "success"
		conn.Message = fmt.Sprintf("连接成功（HTTP %d）", resp.StatusCode)
		s.SetConnectionResult(conn, body)
		conn.ConnectedAt = time.Now()
	} else {
		s.AddLog(conn, fmt.Sprintf("✗ 请求失败，状态码 %d", resp.StatusCode))
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败（HTTP %d）", resp.StatusCode)
		s.SetConnectionResult(conn, body)
	}
}

// ExecuteElasticsearchCommand 执行 Elasticsearch API 命令
func (s *ConnectorService) ExecuteElasticsearchCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("Elasticsearch 连接未建立")
	}

	// 支持的 API 端点映射
	validCommands := map[string]string{
		"health":       "/_cluster/health",
		"stats":        "/_cluster/stats",
		"nodes":        "/_nodes",
		"indices":      "/_cat/indices?v",
		"shards":       "/_cat/shards?v",
		"allocation":   "/_cat/allocation?v",
		"count":        "/_count",
		"settings":     "/_cluster/settings",
		"version":      "/",
		"tasks":        "/_tasks",
		"pending":      "/_cluster/pending_tasks",
	}

	cmd := strings.TrimSpace(strings.ToLower(command))
	apiPath, exists := validCommands[cmd]
	if !exists {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: health, stats, nodes, indices, shards, allocation, count, settings, version, tasks, pending", command)
	}

	port := conn.Port
	if port == "" {
		port = "9200"
	}

	// 构建请求 URL
	scheme := "http"
	hostInput := strings.TrimSpace(conn.IP)
	
	// 检查是否包含协议
	lowerHost := strings.ToLower(hostInput)
	if strings.HasPrefix(lowerHost, "http://") || strings.HasPrefix(lowerHost, "https://") {
		if parsed, err := url.Parse(hostInput); err == nil {
			scheme = parsed.Scheme
			hostInput = parsed.Host
		}
	}

	// 标准化主机地址
	normalizedHost := hostInput
	if _, _, err := net.SplitHostPort(normalizedHost); err != nil {
		normalizedHost = net.JoinHostPort(normalizedHost, port)
	}

	targetURL := fmt.Sprintf("%s://%s%s", scheme, normalizedHost, apiPath)

	// 创建 HTTP 客户端
	transport := &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		DisableKeepAlives:     true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return s.DialContextWithProxy(ctx, network, address)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Accept", "application/json, text/plain;q=0.9, */*;q=0.8")
	req.Header.Set("User-Agent", "MPET-Elasticsearch-Client/1.0")

	// 如果有认证信息，添加 Basic Auth
	if conn.User != "" || conn.Pass != "" {
		req.SetBasicAuth(conn.User, conn.Pass)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	body := strings.TrimSpace(string(bodyBytes))
	if body == "" {
		body = "(响应为空)"
	}

	// 检查状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// 尝试格式化 JSON
		formattedBody := s.formatJSON(body)
		return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, formattedBody), nil
	} else {
		return "", fmt.Errorf("请求失败 (HTTP %d)\n\n%s", resp.StatusCode, body)
	}
}

// formatJSON 格式化 JSON 字符串
func (s *ConnectorService) formatJSON(jsonStr string) string {
	// 简单的 JSON 格式化
	var result strings.Builder
	indent := 0
	inString := false
	escape := false

	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		// 处理转义字符
		if escape {
			result.WriteByte(char)
			escape = false
			continue
		}

		if char == '\\' {
			result.WriteByte(char)
			escape = true
			continue
		}

		// 处理字符串
		if char == '"' {
			inString = !inString
			result.WriteByte(char)
			continue
		}

		if inString {
			result.WriteByte(char)
			continue
		}

		// 格式化非字符串内容
		switch char {
		case '{', '[':
			result.WriteByte(char)
			indent++
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
		case '}', ']':
			indent--
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
			result.WriteByte(char)
		case ',':
			result.WriteByte(char)
			result.WriteByte('\n')
			result.WriteString(strings.Repeat("  ", indent))
		case ':':
			result.WriteString(": ")
		case ' ', '\t', '\n', '\r':
			// 跳过空白字符
		default:
			result.WriteByte(char)
		}
	}

	return result.String()
}
