package parsers

import (
	"MPET/backend/models"
	"fmt"
	"strings"
)

// ParseLightx 解析 lightx 结果文件
// 格式: [2025-12-14 21:00:31] [Plugin:MySQL:SUCCESS] MySQL:127.0.0.1:3306 root/root
func ParseLightx(content string) []*models.ConnectionRequest {
	lines := strings.Split(content, "\n")
	var connections []*models.ConnectionRequest
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if !strings.Contains(line, "[Plugin:") || !strings.Contains(line, ":SUCCESS]") {
			continue
		}

		pluginStart := strings.Index(line, "[Plugin:")
		pluginEnd := strings.Index(line, ":SUCCESS]")
		if pluginStart == -1 || pluginEnd == -1 {
			continue
		}

		serviceType := line[pluginStart+8 : pluginEnd]
		if serviceType == "NetInfo" {
			continue
		}

		contentStart := pluginEnd + 9
		if contentStart >= len(line) {
			continue
		}
		mainContent := strings.TrimSpace(line[contentStart:])

		parts := strings.Fields(mainContent)
		if len(parts) < 1 {
			continue
		}

		serviceParts := strings.Split(parts[0], ":")
		if len(serviceParts) < 3 {
			continue
		}

		connType := NormalizeLightxServiceType(strings.TrimSpace(serviceParts[0]))
		ip := strings.TrimSpace(serviceParts[1])
		port := strings.TrimSpace(serviceParts[2])

		if connType == "" {
			continue
		}

		user := ""
		pass := ""

		if len(parts) >= 2 {
			credInfo := parts[1]
			
			if !strings.Contains(strings.ToLower(line), "未授权访问") && 
			   !strings.Contains(strings.ToLower(line), "unauthorized") &&
			   !strings.Contains(strings.ToLower(line), "匿名登录") {
				if strings.Contains(credInfo, "/") {
					credParts := strings.SplitN(credInfo, "/", 2)
					if len(credParts) >= 1 {
						user = strings.TrimSpace(credParts[0])
					}
					if len(credParts) >= 2 {
						pass = strings.TrimSpace(credParts[1])
					}
				}
			} else if strings.Contains(credInfo, "anonymous") {
				user = "anonymous"
			}
		}

		uniqueKey := fmt.Sprintf("%s:%s:%s:%s:%s", connType, ip, port, user, pass)
		if seen[uniqueKey] {
			continue
		}
		seen[uniqueKey] = true

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

// NormalizeLightxServiceType 标准化 lightx 服务类型名称
func NormalizeLightxServiceType(lightxType string) string {
	lowerType := strings.ToLower(lightxType)
	
	typeMap := map[string]string{
		"redis": "Redis", "mysql": "MySQL", "mssql": "SQLServer", "sqlserver": "SQLServer",
		"oracle": "Oracle", "postgres": "PostgreSQL", "postgresql": "PostgreSQL",
		"mongodb": "MongoDB", "mongo": "MongoDB", "ssh": "SSH", "ftp": "FTP",
		"sftp": "SFTP", "smb": "SMB", "rdp": "RDP", "memcached": "Memcached",
		"elasticsearch": "Elasticsearch", "es": "Elasticsearch",
		"rabbitmq": "RabbitMQ", "mqtt": "MQTT", "zookeeper": "Zookeeper",
		"etcd": "Etcd", "kafka": "Kafka", "vnc": "VNC", "adb": "ADB", "jdwp": "JDWP","docker": "Docker",
	}
	
	if normalized, exists := typeMap[lowerType]; exists {
		return normalized
	}
	
	if len(lightxType) > 0 {
		return strings.ToUpper(lightxType[:1]) + strings.ToLower(lightxType[1:])
	}
	
	return ""
}
