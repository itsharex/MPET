package connectors

import (
	"MPET/backend/models"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ConnectMQTT 连接 MQTT
func (s *ConnectorService) ConnectMQTT(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "1883" // MQTT 默认端口
	}
	addr := net.JoinHostPort(conn.IP, port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
		s.AddLog(conn, "注意: MQTT 连接暂不支持代理，将尝试直接连接")
	}

	var client mqtt.Client
	var username, password string

	// 设置 MQTT 客户端选项
	opts := mqtt.NewClientOptions()
	opts.AddBroker(addr)
	opts.SetClientID(fmt.Sprintf("batch-connector-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(false)
	opts.SetCleanSession(true)

	// 如果用户提供了用户名和密码，直接使用
	if conn.User != "" && conn.Pass != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 密码认证", conn.User))
		opts.SetUsername(conn.User)
		opts.SetPassword(conn.Pass)
		username = conn.User
		password = conn.Pass
	} else if conn.User != "" {
		s.AddLog(conn, fmt.Sprintf("尝试用户 %s 无密码连接", conn.User))
		opts.SetUsername(conn.User)
		username = conn.User
	} else {
		s.AddLog(conn, "尝试未授权访问（无用户名密码）")
	}

	// 创建客户端
	client = mqtt.NewClient(opts)

	// 尝试连接
	s.AddLog(conn, "正在连接 MQTT Broker...")
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		s.AddLog(conn, fmt.Sprintf("✗ MQTT 连接失败: %v", token.Error()))
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", token.Error())
		return
	}

	// 检查连接状态
	if !client.IsConnected() {
		s.AddLog(conn, "✗ MQTT 连接失败: 连接超时")
		conn.Status = "failed"
		conn.Message = "连接失败: 连接超时"
		client.Disconnect(250)
		return
	}

	s.AddLog(conn, "✓ MQTT 连接成功")
	s.AddLog(conn, "获取 MQTT Broker 基础信息")

	// 获取 MQTT 基础信息
	result := s.GetMQTTInfo(client, addr, username)
	conn.Status = "success"
	if username != "" {
		if password != "" {
			conn.Message = fmt.Sprintf("连接成功（用户: %s）", username)
		} else {
			conn.Message = fmt.Sprintf("连接成功（用户: %s，无密码）", username)
		}
	} else {
		conn.Message = "连接成功（未授权访问）"
	}
	s.SetConnectionResult(conn, result)
	conn.ConnectedAt = time.Now()

	// 断开连接
	client.Disconnect(250)
}

// GetMQTTInfo 获取 MQTT Broker 基础信息
func (s *ConnectorService) GetMQTTInfo(client mqtt.Client, addr, username string) string {
	var results []string
	results = append(results, "MQTT Broker 基础信息:")
	results = append(results, strings.Repeat("-", 50))

	// Broker 地址
	results = append(results, fmt.Sprintf("Broker 地址: %s", addr))

	// 客户端 ID
	if client != nil && client.IsConnected() {
		results = append(results, "连接状态: 已连接")
	} else {
		results = append(results, "连接状态: 未连接")
	}

	// 用户名
	if username != "" {
		results = append(results, fmt.Sprintf("认证用户: %s", username))
	} else {
		results = append(results, "认证用户: 无（未授权访问）")
	}

	// 尝试订阅系统主题获取版本信息（如果支持）
	results = append(results, "")
	results = append(results, "尝试获取 Broker 信息...")

	// 尝试订阅 $SYS 主题（很多 MQTT Broker 支持）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 尝试获取一些系统主题信息
	sysTopics := []string{
		"$SYS/broker/version",
		"$SYS/broker/uptime",
		"$SYS/broker/clients/connected",
	}

	var receivedInfo []string
	messageReceived := make(chan bool, 1)

	for _, topic := range sysTopics {
		token := client.Subscribe(topic, 0, func(c mqtt.Client, msg mqtt.Message) {
			receivedInfo = append(receivedInfo, fmt.Sprintf("  %s: %s", msg.Topic(), string(msg.Payload())))
			messageReceived <- true
		})

		if token.WaitTimeout(1*time.Second) && token.Error() == nil {
			// 等待消息或超时
			select {
			case <-messageReceived:
				// 收到消息，继续
			case <-ctx.Done():
				// 超时，继续下一个
			}
			client.Unsubscribe(topic)
		}
	}

	if len(receivedInfo) > 0 {
		results = append(results, "系统主题信息:")
		results = append(results, receivedInfo...)
	} else {
		results = append(results, "未获取到系统主题信息（Broker 可能不支持 $SYS 主题）")
	}

	// 测试发布和订阅功能
	results = append(results, "")
	results = append(results, "功能测试:")
	testTopic := fmt.Sprintf("test/batch-connector/%d", time.Now().UnixNano())
	testMessage := "test message from batch-connector"

	// 订阅测试主题
	var testReceived bool
	subToken := client.Subscribe(testTopic, 0, func(c mqtt.Client, msg mqtt.Message) {
		testReceived = true
	})

	if subToken.WaitTimeout(1*time.Second) && subToken.Error() == nil {
		results = append(results, fmt.Sprintf("  订阅功能: 正常（主题: %s）", testTopic))

		// 发布测试消息
		pubToken := client.Publish(testTopic, 0, false, testMessage)
		if pubToken.WaitTimeout(1*time.Second) && pubToken.Error() == nil {
			// 等待消息接收
			time.Sleep(500 * time.Millisecond)
			if testReceived {
				results = append(results, "  发布/订阅功能: 正常")
			} else {
				results = append(results, "  发布功能: 正常（但未收到订阅消息）")
			}
		} else {
			results = append(results, fmt.Sprintf("  发布功能: 失败 (%v)", pubToken.Error()))
		}

		// 取消订阅
		client.Unsubscribe(testTopic)
	} else {
		results = append(results, fmt.Sprintf("  订阅功能: 失败 (%v)", subToken.Error()))
	}

	return strings.Join(results, "\n")
}
