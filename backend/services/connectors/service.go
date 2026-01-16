package connectors

import (
	"MPET/backend/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// ConnectorService 连接器服务
type ConnectorService struct {
	DB     *sql.DB
	Config *models.Config
}

// AddLog 添加日志
func (s *ConnectorService) AddLog(conn *models.Connection, message string) {
	if conn.Logs == nil {
		conn.Logs = []string{}
	}
	timestamp := time.Now().Format("15:04:05")
	logMsg := fmt.Sprintf("[%s] %s", timestamp, message)
	conn.Logs = append(conn.Logs, logMsg)
	log.Printf("[%s %s:%s] %s", conn.Type, conn.IP, conn.Port, logMsg)
}

// SetConnectionResult 设置连接结果（保留历史记录）
func (s *ConnectorService) SetConnectionResult(conn *models.Connection, newResult string) {
	// 如果已有历史结果，添加分隔符后追加
	if conn.Result != "" {
		separator := fmt.Sprintf("\n\n%s\n重新连接时间: %s\n%s\n\n", 
			strings.Repeat("=", 60),
			time.Now().Format("2006-01-02 15:04:05"),
			strings.Repeat("=", 60))
		conn.Result += separator + newResult
	} else {
		// 首次连接，直接设置
		conn.Result = newResult
	}
}

// GetProxyDialer 获取代理拨号器
func (s *ConnectorService) GetProxyDialer() (proxy.Dialer, error) {
	if !s.Config.Proxy.Enabled {
		return nil, nil
	}

	if s.Config.Proxy.Type != "socks5" {
		return nil, fmt.Errorf("不支持的代理类型: %s", s.Config.Proxy.Type)
	}

	proxyAddr := net.JoinHostPort(s.Config.Proxy.Host, s.Config.Proxy.Port)

	var auth *proxy.Auth
	if s.Config.Proxy.User != "" {
		auth = &proxy.Auth{
			User:     s.Config.Proxy.User,
			Password: s.Config.Proxy.Pass,
		}
	}

	return proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
}

// DialWithProxy 使用代理拨号
func (s *ConnectorService) DialWithProxy(network, address string) (net.Conn, error) {
	proxyDialer, err := s.GetProxyDialer()
	if err != nil {
		return nil, err
	}

	if proxyDialer != nil {
		return proxyDialer.Dial(network, address)
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.Dial(network, address)
}

// DialContextWithProxy 使用代理拨号（带上下文）
func (s *ConnectorService) DialContextWithProxy(ctx context.Context, network, address string) (net.Conn, error) {
	proxyDialer, err := s.GetProxyDialer()
	if err != nil {
		return nil, err
	}

	if proxyDialer != nil {
		if contextDialer, ok := proxyDialer.(proxy.ContextDialer); ok {
			return contextDialer.DialContext(ctx, network, address)
		}
		return proxyDialer.Dial(network, address)
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, network, address)
}

// Connect 执行连接测试
func (s *ConnectorService) Connect(conn *models.Connection) {
	conn.Status = "pending"
	conn.Message = "连接中..."
	
	// 不清空历史日志，而是追加分隔符
	if len(conn.Logs) > 0 {
		conn.Logs = append(conn.Logs, "")
		conn.Logs = append(conn.Logs, strings.Repeat("-", 60))
	}
	
	// 如果 Logs 为 nil，初始化为空数组
	if conn.Logs == nil {
		conn.Logs = []string{}
	}

	s.AddLog(conn, fmt.Sprintf("开始连接 %s 服务", conn.Type))
	s.AddLog(conn, fmt.Sprintf("目标地址: %s:%s", conn.IP, conn.Port))
	if conn.User != "" {
		s.AddLog(conn, fmt.Sprintf("用户名: %s", conn.User))
	}
	if conn.Pass != "" {
		s.AddLog(conn, "使用密码认证")
	} else {
		s.AddLog(conn, "尝试未授权访问或无密码连接")
	}

	switch strings.ToLower(conn.Type) {
	case "redis":
		s.ConnectRedis(conn)
	case "memcached":
		s.ConnectMemcached(conn)
	case "mysql":
		s.ConnectMySQL(conn)
	case "postgresql", "postgres":
		s.ConnectPostgreSQL(conn)
	case "mongodb", "mongo":
		s.ConnectMongoDB(conn)
	case "ssh":
		s.ConnectSSH(conn)
	case "sftp":
		s.ConnectSFTP(conn)
	case "sqlserver", "mssql", "sql":
		s.ConnectSQLServer(conn)
	case "oracle":
		s.ConnectOracle(conn)
	case "ftp":
		s.ConnectFTP(conn)
	case "smb", "samba", "cifs":
		s.ConnectSMB(conn)
	case "rabbitmq":
		s.ConnectRabbitMQ(conn)
	case "mqtt":
		s.ConnectMQTT(conn)
	case "wmi":
		s.ConnectWMI(conn)
	case "elasticsearch", "es":
		s.ConnectElasticsearch(conn)
	case "zookeeper", "zk":
		s.ConnectZookeeper(conn)
	case "adb":
		s.ConnectADB(conn)
	case "kafka":
		s.ConnectKafka(conn)
	case "etcd":
		s.ConnectEtcd(conn)
	case "jdwp":
		s.ConnectJDWP(conn)
	case "rmi":
		s.ConnectRMI(conn)
	case "vnc":
		s.ConnectVNC(conn)
	case "rdp":
		s.ConnectRDP(conn)
	case "docker":
		s.ConnectDocker(conn)
	case "kubernetes", "k8s":
		s.ConnectKubernetes(conn)
	default:
		conn.Status = "failed"
		conn.Message = fmt.Sprintf("服务类型 %s 尚未迁移到新架构", conn.Type)
		s.AddLog(conn, fmt.Sprintf("错误: 服务类型 %s 尚未迁移", conn.Type))
	}
}

// ExecuteCommand 执行自定义命令
func (s *ConnectorService) ExecuteCommand(conn *models.Connection, command string) (string, error) {
	switch conn.Type {
	case "Redis":
		return s.ExecuteRedisCommand(conn, command)
	case "Memcached":
		return s.ExecuteMemcachedCommand(conn, command)
	case "MySQL":
		return s.ExecuteMySQLCommand(conn, command)
	case "PostgreSQL":
		return s.ExecutePostgreSQLCommand(conn, command)
	case "MongoDB":
		return s.ExecuteMongoDBCommand(conn, command)
	case "SSH":
		return s.ExecuteSSHCommand(conn, command)
	case "SQLServer":
		return s.ExecuteSQLServerCommand(conn, command)
	case "Oracle":
		return s.ExecuteOracleCommand(conn, command)
	case "Zookeeper":
		return s.ExecuteZookeeperCommand(conn, command)
	case "Elasticsearch":
		return s.ExecuteElasticsearchCommand(conn, command)
	case "ADB":
		return s.ExecuteADBCommand(conn, command)
	case "Kafka":
		return s.ExecuteKafkaCommand(conn, command)
	case "Etcd":
		return s.ExecuteEtcdCommand(conn, command)
	case "JDWP":
		return s.ExecuteJDWPCommand(conn, command)
	case "VNC":
		return s.ExecuteVNCCommand(conn, command)
	case "RDP":
		return s.ExecuteRDPCommand(conn, command)
	case "Docker":
		return s.ExecuteDockerCommand(conn, command)
	case "Kubernetes":
		return s.ExecuteKubernetesCommand(conn, command)
	default:
		return "", fmt.Errorf("不支持的服务类型: %s", conn.Type)
	}
}
