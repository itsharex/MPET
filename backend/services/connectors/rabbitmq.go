package connectors

import (
	"MPET/backend/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

// ConnectRabbitMQ 连接 RabbitMQ
func (s *ConnectorService) ConnectRabbitMQ(conn *models.Connection) {
	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: RabbitMQ 连接暂不支持代理，将尝试直接连接")
	}

	var username, password string
	var connected bool

	// 如果用户提供了用户名和密码，直接使用，跳过默认用户连接
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", conn.User, conn.Pass, conn.IP, conn.Port)
		client, err := amqp.Dial(amqpURL)
		if err == nil {
			s.AddLog(conn, "✓ 密码认证成功")
			username = conn.User
			password = conn.Pass
			connected = true
			client.Close()
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 用户认证失败: %v", err))
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("连接失败: %v", err)
			s.AddLog(conn, "密码认证失败")
			return
		}
	} else {
		// 尝试未授权访问（默认用户 guest/guest）
		s.AddLog(conn, "尝试默认用户 guest/guest 连接")
		amqpURL := fmt.Sprintf("amqp://guest:guest@%s:%s/", conn.IP, conn.Port)
		client, err := amqp.Dial(amqpURL)
		if err == nil {
			s.AddLog(conn, "✓ 默认用户 guest/guest 连接成功")
			username = "guest"
			password = "guest"
			connected = true
			client.Close()
		} else {
			s.AddLog(conn, fmt.Sprintf("✗ 默认用户连接失败: %v", err))

			// 尝试使用提供的用户名（无密码）
			if conn.User != "" {
				pass := conn.Pass
				if pass == "" {
					s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
				} else {
					s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
				}
				amqpURL = fmt.Sprintf("amqp://%s:%s@%s:%s/", conn.User, pass, conn.IP, conn.Port)
				client, err = amqp.Dial(amqpURL)
				if err == nil {
					if pass == "" {
						s.AddLog(conn, "✓ 无密码连接成功")
					} else {
						s.AddLog(conn, "✓ 密码认证成功")
					}
					username = conn.User
					password = pass
					connected = true
					client.Close()
				} else {
					s.AddLog(conn, fmt.Sprintf("✗ 用户认证失败: %v", err))
				}
			}
		}
	}

	if !connected {
		conn.Status = "failed"
		conn.Message = "连接失败: 所有尝试均失败"
		s.AddLog(conn, "所有连接尝试均失败")
		return
	}

	// 连接成功，执行 list_connections
	s.AddLog(conn, "执行 list_connections")
	result := s.GetRabbitMQConnections(conn.IP, username, password)
	conn.Status = "success"
	if conn.User != "" && conn.Pass != "" {
		conn.Message = "连接成功（使用用户名密码）"
	} else if username == "guest" {
		conn.Message = "连接成功（未授权访问，默认用户 guest/guest）"
	} else {
		conn.Message = "连接成功"
	}
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()
}

// GetRabbitMQConnections 获取 RabbitMQ 连接列表
func (s *ConnectorService) GetRabbitMQConnections(ip, username, password string) string {
	// RabbitMQ Management API 常见端口
	managementPorts := []string{"15672", "15671", "15673"}

	var results []string
	results = append(results, "连接列表:")
	results = append(results, strings.Repeat("-", 80))

	for _, port := range managementPorts {
		url := fmt.Sprintf("http://%s:%s/api/connections", ip, port)
		s.AddLogForConnections(&results, fmt.Sprintf("尝试连接 Management API (端口 %s)", port))

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			s.AddLogForConnections(&results, fmt.Sprintf("创建请求失败: %v", err))
			continue
		}

		// 设置 Basic Auth
		req.SetBasicAuth(username, password)

		// 创建 HTTP 客户端，设置超时
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			s.AddLogForConnections(&results, fmt.Sprintf("请求失败: %v", err))
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			s.AddLogForConnections(&results, fmt.Sprintf("HTTP 状态码: %d", resp.StatusCode))
			continue
		}

		// 解析 JSON 响应
		var connections []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&connections); err != nil {
			s.AddLogForConnections(&results, fmt.Sprintf("解析 JSON 失败: %v", err))
			continue
		}

		s.AddLogForConnections(&results, fmt.Sprintf("✓ 成功获取连接列表 (端口 %s, 共 %d 个连接)", port, len(connections)))
		results = append(results, "")

		if len(connections) == 0 {
			results = append(results, "当前没有活跃连接")
		} else {
			// 格式化输出连接信息
			for i, conn := range connections {
				results = append(results, fmt.Sprintf("连接 #%d:", i+1))
				if name, ok := conn["name"].(string); ok {
					results = append(results, fmt.Sprintf("  名称: %s", name))
				}
				if user, ok := conn["user"].(string); ok {
					results = append(results, fmt.Sprintf("  用户: %s", user))
				}
				if peerHost, ok := conn["peer_host"].(string); ok {
					results = append(results, fmt.Sprintf("  对端地址: %s", peerHost))
				}
				if peerPort, ok := conn["peer_port"].(float64); ok {
					results = append(results, fmt.Sprintf("  对端端口: %.0f", peerPort))
				}
				if state, ok := conn["state"].(string); ok {
					results = append(results, fmt.Sprintf("  状态: %s", state))
				}
				if channels, ok := conn["channels"].(float64); ok {
					results = append(results, fmt.Sprintf("  通道数: %.0f", channels))
				}
				if connectedAt, ok := conn["connected_at"].(float64); ok {
					connectedTime := time.Unix(int64(connectedAt)/1000, 0)
					results = append(results, fmt.Sprintf("  连接时间: %s", connectedTime.Format("2006-01-02 15:04:05")))
				}
				results = append(results, "")
			}
		}

		// 成功获取后不再尝试其他端口
		return strings.Join(results, "\n")
	}

	results = append(results, "无法连接到 Management API（已尝试端口: 15672, 15671, 15673）")
	return strings.Join(results, "\n")
}

// AddLogForConnections 为连接列表添加日志（辅助函数）
func (s *ConnectorService) AddLogForConnections(results *[]string, message string) {
	*results = append(*results, message)
}
