package connectors

import (
	"MPET/backend/models"
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// ConnectMemcached 连接 Memcached
func (s *ConnectorService) ConnectMemcached(conn *models.Connection) {
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 尝试连接 Memcached
	s.AddLog(conn, "尝试连接 Memcached 服务")
	
	netConn, err := s.DialWithProxy("tcp", addr)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("✗ 连接失败: %v", err))
		return
	}
	defer netConn.Close()

	// 设置超时
	netConn.SetDeadline(time.Now().Add(10 * time.Second))

	s.AddLog(conn, "✓ TCP 连接成功")
	s.AddLog(conn, "尝试执行 stats 命令")

	// 发送 stats 命令
	_, err = netConn.Write([]byte("stats\r\n"))
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("发送命令失败: %v", err)
		s.AddLog(conn, fmt.Sprintf("✗ 发送命令失败: %v", err))
		return
	}

	// 读取响应
	reader := bufio.NewReader(netConn)
	var statsLines []string
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("读取响应失败: %v", err)
			s.AddLog(conn, fmt.Sprintf("✗ 读取响应失败: %v", err))
			return
		}
		
		line = strings.TrimSpace(line)
		if line == "END" {
			break
		}
		
		if strings.HasPrefix(line, "STAT ") {
			statsLines = append(statsLines, line)
		}
	}

	if len(statsLines) == 0 {
		conn.Status = "failed"
		conn.Message = "未获取到统计信息"
		s.AddLog(conn, "✗ 未获取到统计信息")
		return
	}

	s.AddLog(conn, fmt.Sprintf("✓ 获取到 %d 条统计信息", len(statsLines)))
	s.AddLog(conn, "✓ Memcached 未授权访问成功")

	// 解析统计信息
	result := s.parseMemcachedStats(statsLines)
	
	conn.Status = "success"
	conn.Message = "Memcached 未授权访问成功"
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
}

// parseMemcachedStats 解析 Memcached 统计信息
func (s *ConnectorService) parseMemcachedStats(statsLines []string) string {
	var result strings.Builder
	
	result.WriteString("Memcached 服务信息\n")
	result.WriteString(strings.Repeat("=", 45) + "\n\n")
	
	// 关键信息映射
	keyInfo := map[string]string{
		"version":           "版本",
		"pid":               "进程 ID",
		"uptime":            "运行时间(秒)",
		"time":              "当前时间戳",
		"pointer_size":      "指针大小",
		"curr_connections":  "当前连接数",
		"total_connections": "总连接数",
		"curr_items":        "当前项目数",
		"total_items":       "总项目数",
		"bytes":             "已用内存(字节)",
		"max_connections":   "最大连接数",
		"threads":           "线程数",
		"evictions":         "驱逐次数",
		"get_hits":          "GET 命中次数",
		"get_misses":        "GET 未命中次数",
		"cmd_get":           "GET 命令次数",
		"cmd_set":           "SET 命令次数",
	}

	// 分类显示
	result.WriteString("基本信息:\n")
	result.WriteString(strings.Repeat("-", 45) + "\n")
	for _, line := range statsLines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[1]
			value := parts[2]
			
			if label, ok := keyInfo[key]; ok {
				if key == "version" || key == "pid" || key == "pointer_size" || key == "threads" {
					result.WriteString(fmt.Sprintf("%-20s: %s\n", label, value))
				}
			}
		}
	}
	
	result.WriteString("\n连接信息:\n")
	result.WriteString(strings.Repeat("-", 45) + "\n")
	for _, line := range statsLines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[1]
			value := parts[2]
			
			if label, ok := keyInfo[key]; ok {
				if strings.Contains(key, "connection") {
					result.WriteString(fmt.Sprintf("%-20s: %s\n", label, value))
				}
			}
		}
	}
	
	result.WriteString("\n数据统计:\n")
	result.WriteString(strings.Repeat("-", 45) + "\n")
	for _, line := range statsLines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[1]
			value := parts[2]
			
			if label, ok := keyInfo[key]; ok {
				if strings.Contains(key, "items") || strings.Contains(key, "bytes") || 
				   strings.Contains(key, "evictions") {
					result.WriteString(fmt.Sprintf("%-20s: %s\n", label, value))
				}
			}
		}
	}
	
	result.WriteString("\n命令统计:\n")
	result.WriteString(strings.Repeat("-", 45) + "\n")
	for _, line := range statsLines {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[1]
			value := parts[2]
			
			if label, ok := keyInfo[key]; ok {
				if strings.Contains(key, "cmd_") || strings.Contains(key, "get_") {
					result.WriteString(fmt.Sprintf("%-20s: %s\n", label, value))
				}
			}
		}
	}
	
	// result.WriteString("\n可用命令:\n")
	// result.WriteString(strings.Repeat("=", 45) + "\n")
	// result.WriteString("- stats: 查看服务器统计信息\n")
	// result.WriteString("- stats items: 查看项目统计信息\n")
	// result.WriteString("- stats slabs: 查看 slab 统计信息\n")
	// result.WriteString("- stats sizes: 查看大小统计信息\n")
	// result.WriteString("- version: 查看版本信息\n")
	// result.WriteString("- get <key>: 获取指定键的值\n")
	// result.WriteString("- set <key> <flags> <exptime> <bytes>: 设置键值\n")
	// result.WriteString("- delete <key>: 删除指定键\n")
	// result.WriteString("- flush_all: 清空所有缓存（危险操作）\n")
	
	// result.WriteString("\n安全风险:\n")
	// result.WriteString(strings.Repeat("=", 45) + "\n")
	// result.WriteString("⚠️ Memcached 未授权访问漏洞\n")
	// result.WriteString("- 服务未设置认证，任何人都可以访问\n")
	// result.WriteString("- 攻击者可以读取、修改、删除缓存数据\n")
	// result.WriteString("- 可能导致敏感信息泄露\n")
	// result.WriteString("- 建议启用 SASL 认证\n")
	// result.WriteString("- 建议使用防火墙限制访问 IP\n")
	// result.WriteString("- 建议不要将 Memcached 暴露在公网\n")
	
	return result.String()
}

// ExecuteMemcachedCommand 执行 Memcached 命令
func (s *ConnectorService) ExecuteMemcachedCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("Memcached 连接未建立")
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	s.AddLog(conn, fmt.Sprintf("执行命令: %s", cmd))

	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	// 建立连接
	netConn, err := s.DialWithProxy("tcp", addr)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("✗ 连接失败: %v", err))
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer netConn.Close()

	// 设置超时
	netConn.SetDeadline(time.Now().Add(10 * time.Second))

	// 发送命令
	_, err = netConn.Write([]byte(cmd + "\r\n"))
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("✗ 发送命令失败: %v", err))
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 读取响应
	reader := bufio.NewReader(netConn)
	var responseLines []string
	
	// 根据命令类型决定如何读取响应
	cmdLower := strings.ToLower(strings.Fields(cmd)[0])
	
	switch cmdLower {
	case "stats", "version":
		// stats 类命令读取到 END
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				s.AddLog(conn, fmt.Sprintf("✗ 读取响应失败: %v", err))
				return "", fmt.Errorf("读取响应失败: %v", err)
			}
			
			line = strings.TrimSpace(line)
			responseLines = append(responseLines, line)
			
			if line == "END" || strings.HasPrefix(line, "VERSION") {
				break
			}
		}
	case "get":
		// get 命令读取值和 END
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				s.AddLog(conn, fmt.Sprintf("✗ 读取响应失败: %v", err))
				return "", fmt.Errorf("读取响应失败: %v", err)
			}
			
			line = strings.TrimSpace(line)
			responseLines = append(responseLines, line)
			
			if line == "END" {
				break
			}
		}
	case "set", "add", "replace", "append", "prepend":
		// 存储命令需要读取数据
		// 这里简化处理，只读取一行响应
		line, err := reader.ReadString('\n')
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 读取响应失败: %v", err))
			return "", fmt.Errorf("读取响应失败: %v", err)
		}
		responseLines = append(responseLines, strings.TrimSpace(line))
	case "delete", "flush_all":
		// 删除命令读取一行响应
		line, err := reader.ReadString('\n')
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 读取响应失败: %v", err))
			return "", fmt.Errorf("读取响应失败: %v", err)
		}
		responseLines = append(responseLines, strings.TrimSpace(line))
	default:
		// 其他命令，尝试读取多行直到超时或 END
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			
			line = strings.TrimSpace(line)
			responseLines = append(responseLines, line)
			
			if line == "END" || line == "OK" || strings.HasPrefix(line, "ERROR") {
				break
			}
			
			if len(responseLines) > 1000 {
				break
			}
		}
	}

	result := strings.Join(responseLines, "\n")
	s.AddLog(conn, fmt.Sprintf("✓ 命令执行成功，返回 %d 行", len(responseLines)))
	
	return result, nil
}
