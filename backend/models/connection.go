package models

import "time"

// Connection 连接信息
type Connection struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // Redis, FTP, PostgreSQL, MySQL, SQLServer, RabbitMQ, SSH, MongoDB, SMB, WMI, MQTT, Oracle, Elasticsearch, Zookeeper
	IP          string    `json:"ip"`
	Port        string    `json:"port"`
	User        string    `json:"user"`
	Pass        string    `json:"pass"`
	Status      string    `json:"status"`  // success, failed, pending
	Message     string    `json:"message"` // 连接结果消息
	Result      string    `json:"result"`  // SSH 执行结果或其他详细信息
	Logs        []string  `json:"logs"`    // 详细连接日志
	CreatedAt   time.Time `json:"created_at"`
	ConnectedAt time.Time `json:"connected_at,omitempty"`
}

// ConnectionRequest 连接请求
type ConnectionRequest struct {
	Type string `json:"type"`
	IP   string `json:"ip"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // socks5
	Host    string `json:"host"`
	Port    string `json:"port"`
	User    string `json:"user"`
	Pass    string `json:"pass"`
}

// Config 应用配置
type Config struct {
	Password string      `json:"password"`
	Proxy    ProxyConfig `json:"proxy"`
}
