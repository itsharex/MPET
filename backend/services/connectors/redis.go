package connectors

import (
	"MPET/backend/models"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/proxy"
)

// ConnectRedis 连接 Redis
func (s *ConnectorService) ConnectRedis(conn *models.Connection) {
	ctx := context.Background()
	addr := net.JoinHostPort(conn.IP, conn.Port)
	s.AddLog(conn, fmt.Sprintf("连接地址: %s", addr))

	// 检查是否使用代理
	if s.Config.Proxy.Enabled {
		s.AddLog(conn, fmt.Sprintf("使用 SOCKS5 代理: %s:%s", s.Config.Proxy.Host, s.Config.Proxy.Port))
	}

	// 创建 Dialer
	var dialer func(ctx context.Context, network, addr string) (net.Conn, error)
	if s.Config.Proxy.Enabled {
		proxyDialer, err := s.GetProxyDialer()
		if err != nil {
			s.AddLog(conn, fmt.Sprintf("✗ 创建代理 Dialer 失败: %v", err))
			conn.Status = "failed"
			conn.Message = fmt.Sprintf("代理配置错误: %v", err)
			return
		}
		if contextDialer, ok := proxyDialer.(proxy.ContextDialer); ok {
			dialer = contextDialer.DialContext
		} else {
			dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return proxyDialer.Dial(network, addr)
			}
		}
	}

	// 如果用户提供了密码，直接使用密码连接，跳过未授权访问
	if conn.Pass != "" {
		s.AddLog(conn, "尝试使用密码连接")
		opts := &redis.Options{
			Addr:     addr,
			Password: conn.Pass,
			DB:       0,
		}
		if dialer != nil {
			opts.Dialer = dialer
		}
		rdb := redis.NewClient(opts)
		_, err := rdb.Ping(ctx).Result()
		if err == nil {
			s.AddLog(conn, "✓ 密码认证成功")
			s.AddLog(conn, "获取数据库信息")
			result := s.GetRedisDatabases(addr, conn.Pass, ctx)
			conn.Status = "success"
			conn.Message = "连接成功（使用密码）"
			s.SetConnectionResult(conn, result)
			conn.ConnectedAt = time.Now()
			rdb.Close()
			return
		}
		s.AddLog(conn, fmt.Sprintf("✗ 密码认证失败: %v", err))
		rdb.Close()
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("连接失败: %v", err)
		s.AddLog(conn, "密码认证失败")
		return
	}

	// 如果没有提供密码，尝试未授权访问
	s.AddLog(conn, "尝试未授权访问（无密码）")
	opts := &redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	}
	if dialer != nil {
		opts.Dialer = dialer
	}
	rdb := redis.NewClient(opts)

	_, err := rdb.Ping(ctx).Result()
	if err == nil {
		s.AddLog(conn, "✓ 未授权访问成功")
		s.AddLog(conn, "获取数据库信息")
		result := s.GetRedisDatabases(addr, "", ctx)
		conn.Status = "success"
		conn.Message = "连接成功（未授权访问）"
		s.SetConnectionResult(conn, result)
		conn.ConnectedAt = time.Now()
		rdb.Close()
		return
	}
	s.AddLog(conn, fmt.Sprintf("✗ 未授权访问失败: %v", err))
	rdb.Close()

	conn.Status = "failed"
	conn.Message = fmt.Sprintf("连接失败: %v", err)
	s.AddLog(conn, "所有连接尝试均失败")
}

// GetRedisDatabases 获取 Redis 数据库信息
func (s *ConnectorService) GetRedisDatabases(addr, password string, ctx context.Context) string {
	var results []string

	// 创建临时客户端用于获取信息
	tempRdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	defer tempRdb.Close()

	// 获取 keyspace 信息（显示有数据的数据库）
	info, err := tempRdb.Info(ctx, "keyspace").Result()
	if err == nil && info != "" {
		results = append(results, fmt.Sprintf("Keyspace 信息:\n%s", info))
	} else {
		results = append(results, fmt.Sprintf("获取 Keyspace 信息失败: %v", err))
	}

	// 获取配置的数据库数量
	config, err := tempRdb.ConfigGet(ctx, "databases").Result()
	if err == nil && len(config) > 0 {
		results = append(results, fmt.Sprintf("配置的数据库数量: %s", config[1]))
	}

	// 尝试检查每个数据库（0-15）是否有数据
	var databasesWithData []string
	for i := 0; i < 16; i++ {
		testRdb := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       i,
		})
		keys, err := testRdb.DBSize(ctx).Result()
		testRdb.Close()
		if err == nil && keys > 0 {
			databasesWithData = append(databasesWithData, fmt.Sprintf("db%d (%d keys)", i, keys))
		}
	}

	if len(databasesWithData) > 0 {
		results = append(results, fmt.Sprintf("有数据的数据库: %s", strings.Join(databasesWithData, ", ")))
	} else {
		results = append(results, "未发现包含数据的数据库")
	}

	return strings.Join(results, "\n")
}

// ExecuteRedisCommand 执行 Redis 命令
func (s *ConnectorService) ExecuteRedisCommand(conn *models.Connection, command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := net.JoinHostPort(conn.IP, conn.Port)
	
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: conn.Pass,
		DB:       0,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.DialContextWithProxy(ctx, network, addr)
		},
	})
	defer client.Close()

	// 将命令字符串分割并转换为接口切片
	fields := strings.Fields(command)
	args := make([]interface{}, len(fields))
	for i, field := range fields {
		args[i] = field
	}

	result, err := client.Do(ctx, args...).Result()
	if err != nil {
		return "", fmt.Errorf("命令执行失败: %v", err)
	}

	return fmt.Sprintf("%v", result), nil
}
