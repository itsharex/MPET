package connectors

import (
	"MPET/backend/models"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
)

// ConnectZookeeper 连接 ZooKeeper 并执行 ls /
func (s *ConnectorService) ConnectZookeeper(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "2181"
	}

	rawHosts := strings.TrimSpace(conn.IP)
	if rawHosts == "" {
		conn.Status = "failed"
		conn.Message = "连接失败: 未指定目标地址"
		s.AddLog(conn, "未提供 ZooKeeper 地址")
		return
	}

	splitHosts := strings.FieldsFunc(rawHosts, func(r rune) bool {
		switch r {
		case ',', ';', ' ', '\n', '\t':
			return true
		default:
			return false
		}
	})
	if len(splitHosts) == 0 {
		splitHosts = []string{rawHosts}
	}

	var servers []string
	for _, host := range splitHosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(host); err == nil {
			servers = append(servers, host)
		} else {
			servers = append(servers, net.JoinHostPort(host, port))
		}
	}

	if len(servers) == 0 {
		conn.Status = "failed"
		conn.Message = "连接失败: 无效的目标地址"
		s.AddLog(conn, "未能解析任何有效的 ZooKeeper 节点")
		return
	}

	s.AddLog(conn, fmt.Sprintf("目标节点: %s", strings.Join(servers, ", ")))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	dialer := func(network, address string, timeout time.Duration) (net.Conn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return s.DialContextWithProxy(ctx, network, address)
	}

	sessionTimeout := 10 * time.Second
	zkConn, events, err := zk.Connect(
		servers,
		sessionTimeout,
		zk.WithDialer(dialer),
		zk.WithLogger(log.New(io.Discard, "", 0)),
	)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接 ZooKeeper 失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	defer zkConn.Close()

	connected := false
	timeout := time.After(15 * time.Second)
	for !connected {
		select {
		case event := <-events:
			if event.Err != nil {
				conn.Status = "failed"
				conn.Message = fmt.Sprintf("连接事件错误: %v", event.Err)
				s.AddLog(conn, conn.Message)
				return
			}
			s.AddLog(conn, fmt.Sprintf("事件: %s / %s", event.Type.String(), event.State.String()))
			if event.State == zk.StateAuthFailed {
				conn.Status = "failed"
				conn.Message = "认证失败: digest 账号/密码错误"
				s.AddLog(conn, conn.Message)
				return
			}
			if event.State == zk.StateConnected || event.State == zk.StateConnectedReadOnly {
				connected = true
			}
		case <-timeout:
			conn.Status = "failed"
			conn.Message = "连接超时: 未能在 15 秒内建立会话"
			s.AddLog(conn, conn.Message)
			return
		}
	}
	s.AddLog(conn, "✓ 成功建立 ZooKeeper 会话")

	if conn.User != "" {
		if conn.Pass == "" {
			s.AddLog(conn, "提供了用户名但未提供密码，将尝试无密码 digest 认证")
		}
		authPayload := fmt.Sprintf("%s:%s", conn.User, conn.Pass)
		if err := zkConn.AddAuth("digest", []byte(authPayload)); err != nil {
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("添加 digest 认证失败: %v", err)
			s.AddLog(conn, conn.Message)
			return
		}
		s.AddLog(conn, fmt.Sprintf("已添加 digest 认证账号 %s", conn.User))
	}

	s.AddLog(conn, "执行命令: ls /")
	children, stat, err := zkConn.Children("/")
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("执行 ls / 失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	var builder strings.Builder
	builder.WriteString("=== ls / 结果 ===\n")
	builder.WriteString(fmt.Sprintf("根节点包含 %d 个子节点\n", len(children)))
	if len(children) == 0 {
		builder.WriteString("(无子节点)\n")
	} else {
		for _, child := range children {
			builder.WriteString("- ")
			builder.WriteString(child)
			builder.WriteString("\n")
		}
	}
	if stat != nil {
		builder.WriteString(fmt.Sprintf("\nstat: czxid=%d, mzxid=%d, version=%d, ctime=%s, mtime=%s\n",
			stat.Czxid, stat.Mzxid, stat.Version,
			time.UnixMilli(stat.Ctime).Format(time.RFC3339),
			time.UnixMilli(stat.Mtime).Format(time.RFC3339)))
	}
	s.AddLog(conn, "✓ ls / 执行完成")

	// 执行 srvr 四字命令获取服务器信息
	s.AddLog(conn, "执行命令: srvr")
	srvrResult, err := s.executeZKFourLetterCommand(conn.IP, conn.Port, "srvr")
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("⚠ srvr 命令执行失败: %v", err))
		builder.WriteString("\n=== srvr 结果 ===\n")
		builder.WriteString(fmt.Sprintf("执行失败: %v\n", err))
	} else {
		s.AddLog(conn, "✓ srvr 执行完成")
		builder.WriteString("\n=== srvr 结果 ===\n")
		builder.WriteString(srvrResult)
	}

	conn.Status = "success"
	conn.Message = "连接成功（ls / 和 srvr 完成）"
	s.SetConnectionResult(conn, builder.String())
	conn.ConnectedAt = time.Now()
}

// ExecuteZookeeperCommand 执行 Zookeeper 四字命令
func (s *ConnectorService) ExecuteZookeeperCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("Zookeeper 连接未建立")
	}

	// 支持的四字命令列表
	validCommands := map[string]bool{
		"envi": true, // 环境变量
		"conf": true, // 配置信息
		"cons": true, // 连接信息
		"crst": true, // 重置连接统计
		"dump": true, // 会话和临时节点
		"ruok": true, // 服务器是否运行
		"stat": true, // 服务器统计信息
		"srvr": true, // 服务器信息
		"mntr": true, // 监控信息
	}

	cmd := strings.TrimSpace(strings.ToLower(command))
	if !validCommands[cmd] {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: envi, conf, cons, crst, dump, ruok, stat, srvr, mntr", command)
	}

	port := conn.Port
	if port == "" {
		port = "2181"
	}

	return s.executeZKFourLetterCommand(conn.IP, port, cmd)
}

// executeZKFourLetterCommand 执行 Zookeeper 四字命令的底层实现
func (s *ConnectorService) executeZKFourLetterCommand(host, port, command string) (string, error) {
	// 连接到 Zookeeper 服务器
	address := net.JoinHostPort(host, port)
	
	// 使用代理或直连
	var netConn net.Conn
	var err error
	
	if s.Config.Proxy.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		netConn, err = s.DialContextWithProxy(ctx, "tcp", address)
	} else {
		netConn, err = net.DialTimeout("tcp", address, 10*time.Second)
	}
	
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer netConn.Close()

	// 设置读写超时
	netConn.SetDeadline(time.Now().Add(10 * time.Second))

	// 发送四字命令
	_, err = netConn.Write([]byte(command))
	if err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 读取响应
	response := make([]byte, 4096)
	n, err := netConn.Read(response)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	result := string(response[:n])
	if result == "" {
		return "命令执行成功，但无返回内容", nil
	}

	return result, nil
}
