package connectors

import (
	"MPET/backend/models"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// ConnectEtcd 连接 etcd
func (s *ConnectorService) ConnectEtcd(conn *models.Connection) {
	port := conn.Port
	if port == "" {
		port = "2379"
	}

	endpoint := fmt.Sprintf("http://%s", net.JoinHostPort(conn.IP, port))
	s.AddLog(conn, fmt.Sprintf("目标地址: %s", endpoint))
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("⚠ 注意: etcd 客户端暂不支持 SOCKS5 代理"))
	}

	// 创建 etcd 客户端配置
	config := clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 10 * time.Second,
	}

	// 如果提供了认证信息
	if conn.User != "" {
		config.Username = conn.User
		config.Password = conn.Pass
		s.AddLog(conn, fmt.Sprintf("使用认证: %s", conn.User))
	}

	// 创建客户端
	client, err := clientv3.New(config)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}
	defer client.Close()

	s.AddLog(conn, "✓ etcd 客户端创建成功")

	// 测试连接 - 获取集群成员信息
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	memberList, err := client.MemberList(ctx)
	if err != nil {
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("获取集群信息失败: %v", err)
		s.AddLog(conn, conn.Message)
		return
	}

	s.AddLog(conn, fmt.Sprintf("✓ 发现 %d 个集群成员", len(memberList.Members)))

	// 获取版本信息
	statusResp, err := client.Status(ctx, endpoint)
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("⚠ 获取版本信息失败: %v", err))
	} else {
		s.AddLog(conn, fmt.Sprintf("✓ etcd 版本: %s", statusResp.Version))
	}

	// 尝试列出根目录的键
	getResp, err := client.Get(ctx, "/", clientv3.WithPrefix(), clientv3.WithLimit(10))
	if err != nil {
		s.AddLog(conn, fmt.Sprintf("⚠ 获取键列表失败: %v", err))
	} else {
		s.AddLog(conn, fmt.Sprintf("✓ 发现 %d 个键（前10个）", len(getResp.Kvs)))
	}

	// 构建结果信息
	var result strings.Builder
	result.WriteString("=== etcd 集群信息 ===\n")
	result.WriteString(fmt.Sprintf("集群成员数量: %d\n", len(memberList.Members)))
	if statusResp != nil {
		result.WriteString(fmt.Sprintf("etcd 版本: %s\n", statusResp.Version))
		result.WriteString(fmt.Sprintf("数据库大小: %d bytes\n", statusResp.DbSize))
		result.WriteString(fmt.Sprintf("Leader ID: %d\n", statusResp.Leader))
	}
	result.WriteString("\n集群成员:\n")
	for _, member := range memberList.Members {
		result.WriteString(fmt.Sprintf("  - ID: %d, Name: %s\n", member.ID, member.Name))
		result.WriteString(fmt.Sprintf("    Peer URLs: %v\n", member.PeerURLs))
		result.WriteString(fmt.Sprintf("    Client URLs: %v\n", member.ClientURLs))
	}

	if getResp != nil && len(getResp.Kvs) > 0 {
		result.WriteString("\n键值示例（前10个）:\n")
		for _, kv := range getResp.Kvs {
			result.WriteString(fmt.Sprintf("  - %s\n", string(kv.Key)))
		}
	}

	conn.Status = "success"
	conn.Message = "连接成功"
	s.SetConnectionResult(conn, result.String())
	conn.ConnectedAt = time.Now()
	s.AddLog(conn, "✓ etcd 连接测试完成")
}

// ExecuteEtcdCommand 执行 etcd 命令
func (s *ConnectorService) ExecuteEtcdCommand(conn *models.Connection, command string) (string, error) {
	if conn.Status != "success" {
		return "", fmt.Errorf("etcd 连接未建立")
	}

	port := conn.Port
	if port == "" {
		port = "2379"
	}

	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	endpoint := fmt.Sprintf("http://%s", net.JoinHostPort(conn.IP, port))

	// 创建 etcd 客户端配置
	config := clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 10 * time.Second,
	}

	// 如果提供了认证信息
	if conn.User != "" {
		config.Username = conn.User
		config.Password = conn.Pass
	}

	// 创建客户端
	client, err := clientv3.New(config)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer client.Close()

	// 解析命令
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("无效的命令")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmdType := strings.ToLower(parts[0])
	var result strings.Builder

	switch cmdType {
	case "get":
		// 获取键值
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: get <key> [--prefix]")
		}
		key := parts[1]
		
		opts := []clientv3.OpOption{}
		if len(parts) > 2 && parts[2] == "--prefix" {
			opts = append(opts, clientv3.WithPrefix())
		}

		resp, err := client.Get(ctx, key, opts...)
		if err != nil {
			return "", fmt.Errorf("获取失败: %v", err)
		}

		if len(resp.Kvs) == 0 {
			result.WriteString("未找到键\n")
		} else {
			result.WriteString(fmt.Sprintf("找到 %d 个键:\n\n", len(resp.Kvs)))
			for _, kv := range resp.Kvs {
				result.WriteString(fmt.Sprintf("Key: %s\n", string(kv.Key)))
				result.WriteString(fmt.Sprintf("Value: %s\n", string(kv.Value)))
				result.WriteString(fmt.Sprintf("Version: %d\n", kv.Version))
				result.WriteString(fmt.Sprintf("CreateRevision: %d\n", kv.CreateRevision))
				result.WriteString(fmt.Sprintf("ModRevision: %d\n\n", kv.ModRevision))
			}
		}

	case "put":
		// 设置键值
		if len(parts) < 3 {
			return "", fmt.Errorf("用法: put <key> <value>")
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")

		_, err := client.Put(ctx, key, value)
		if err != nil {
			return "", fmt.Errorf("设置失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("✓ 成功设置键: %s\n", key))

	case "del", "delete":
		// 删除键
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: del <key> [--prefix]")
		}
		key := parts[1]

		opts := []clientv3.OpOption{}
		if len(parts) > 2 && parts[2] == "--prefix" {
			opts = append(opts, clientv3.WithPrefix())
		}

		resp, err := client.Delete(ctx, key, opts...)
		if err != nil {
			return "", fmt.Errorf("删除失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("✓ 成功删除 %d 个键\n", resp.Deleted))

	case "list", "ls":
		// 列出键
		prefix := "/"
		if len(parts) > 1 {
			prefix = parts[1]
		}

		resp, err := client.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
		if err != nil {
			return "", fmt.Errorf("列出键失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("前缀 '%s' 下的键（共 %d 个）:\n\n", prefix, len(resp.Kvs)))
		for _, kv := range resp.Kvs {
			result.WriteString(fmt.Sprintf("%s\n", string(kv.Key)))
		}

	case "member", "members":
		// 列出集群成员
		memberList, err := client.MemberList(ctx)
		if err != nil {
			return "", fmt.Errorf("获取成员列表失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("集群成员（共 %d 个）:\n\n", len(memberList.Members)))
		for _, member := range memberList.Members {
			result.WriteString(fmt.Sprintf("ID: %d\n", member.ID))
			result.WriteString(fmt.Sprintf("Name: %s\n", member.Name))
			result.WriteString(fmt.Sprintf("Peer URLs: %v\n", member.PeerURLs))
			result.WriteString(fmt.Sprintf("Client URLs: %v\n\n", member.ClientURLs))
		}

	case "status":
		// 获取状态信息
		statusResp, err := client.Status(ctx, endpoint)
		if err != nil {
			return "", fmt.Errorf("获取状态失败: %v", err)
		}

		result.WriteString("=== etcd 状态信息 ===\n\n")
		result.WriteString(fmt.Sprintf("版本: %s\n", statusResp.Version))
		result.WriteString(fmt.Sprintf("数据库大小: %d bytes\n", statusResp.DbSize))
		result.WriteString(fmt.Sprintf("Leader ID: %d\n", statusResp.Leader))
		result.WriteString(fmt.Sprintf("Raft Term: %d\n", statusResp.RaftTerm))
		result.WriteString(fmt.Sprintf("Raft Index: %d\n", statusResp.RaftIndex))

	case "watch":
		return "", fmt.Errorf("watch 命令需要持续监听，请使用其他命令")

	case "compact":
		// 压缩历史版本
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: compact <revision>")
		}
		var revision int64
		fmt.Sscanf(parts[1], "%d", &revision)

		_, err := client.Compact(ctx, revision)
		if err != nil {
			return "", fmt.Errorf("压缩失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("✓ 成功压缩到版本: %d\n", revision))

	case "version":
		// 获取版本信息
		statusResp, err := client.Status(ctx, endpoint)
		if err != nil {
			return "", fmt.Errorf("获取版本失败: %v", err)
		}

		result.WriteString(fmt.Sprintf("etcd 版本: %s\n", statusResp.Version))

	default:
		return "", fmt.Errorf("不支持的命令: %s\n\n支持的命令:\n  - get <key> [--prefix]: 获取键值\n  - put <key> <value>: 设置键值\n  - del <key> [--prefix]: 删除键\n  - list [prefix]: 列出键\n  - member: 列出集群成员\n  - status: 获取状态信息\n  - version: 获取版本信息\n  - compact <revision>: 压缩历史版本", cmdType)
	}

	return result.String(), nil
}
