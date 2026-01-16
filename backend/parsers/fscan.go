package parsers

import (
	"MPET/backend/models"
	"strings"
)

// ParseFscan 解析 fscan 1.8.4 结果文件
// 格式: [+] Redis:192.168.1.100:6379 unauthorized
func ParseFscan(content string) []*models.ConnectionRequest {
	lines := strings.Split(content, "\n")
	var connections []*models.ConnectionRequest

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "[+]") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// 提取服务信息 (格式: Type:IP:Port)
		serviceInfo := parts[1]
		serviceParts := strings.Split(serviceInfo, ":")
		if len(serviceParts) < 3 {
			continue
		}

		connType := NormalizeFscanServiceType(strings.TrimSpace(serviceParts[0]))
		ip := strings.TrimSpace(serviceParts[1])
		port := strings.TrimSpace(serviceParts[2])

		if connType == "" {
			continue
		}

		user := ""
		pass := ""

		// 提取凭据信息
		if len(parts) >= 3 {
			credInfo := strings.Join(parts[2:], " ")
			
			// 检查是否为未授权访问
			if !strings.Contains(strings.ToLower(credInfo), "unauthorized") && 
			   !strings.Contains(strings.ToLower(credInfo), "unauth") &&
			   !strings.Contains(strings.ToLower(credInfo), "no auth") {
				if strings.Contains(credInfo, ":") {
					credParts := strings.SplitN(credInfo, ":", 2)
					if len(credParts) == 2 {
						user = strings.TrimSpace(credParts[0])
						pass = strings.TrimSpace(credParts[1])
					}
				} else {
					user = strings.TrimSpace(credInfo)
				}
			}
		}

		connections = append(connections, &models.ConnectionRequest{
			Type: connType,
			IP:   ip,
			Port: port,
			User: user,
			Pass: pass,
		})
	}

	return connections
}

// NormalizeFscanServiceType 标准化 fscan 服务类型名称
func NormalizeFscanServiceType(fscanType string) string {
	lowerType := strings.ToLower(fscanType)
	
	typeMap := map[string]string{
		"redis": "Redis", "mysql": "MySQL", "mssql": "SQLServer", "sqlserver": "SQLServer",
		"oracle": "Oracle", "postgres": "PostgreSQL", "postgresql": "PostgreSQL",
		"mongodb": "MongoDB", "mongo": "MongoDB", "ssh": "SSH", "ftp": "FTP",
		"smb": "SMB", "rdp": "RDP", "memcached": "Memcached",
		"elasticsearch": "Elasticsearch", "es": "Elasticsearch",
		"rabbitmq": "RabbitMQ", "mqtt": "MQTT", "zookeeper": "Zookeeper",
		"etcd": "Etcd", "kafka": "Kafka", "vnc": "VNC","docker": "Docker",
	}
	
	if normalized, exists := typeMap[lowerType]; exists {
		return normalized
	}
	
	if len(fscanType) > 0 {
		return strings.ToUpper(fscanType[:1]) + strings.ToLower(fscanType[1:])
	}
	
	return ""
}
