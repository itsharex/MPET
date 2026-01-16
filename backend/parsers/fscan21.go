package parsers

import (
	"MPET/backend/models"
	"fmt"
	"strings"
)

// ParseFscan21 解析 fscan 2.1.1 结果文件
// 格式: 127.0.0.1:9200 elasticsearch admin/123456
func ParseFscan21(content string) []*models.ConnectionRequest {
	lines := strings.Split(content, "\n")
	var connections []*models.ConnectionRequest
	inVulnSection := false
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "# ===== 漏洞信息 =====") {
			inVulnSection = true
			continue
		}
		
		if strings.HasPrefix(line, "# =====") && !strings.Contains(line, "漏洞信息") {
			inVulnSection = false
			continue
		}
		
		if !inVulnSection || line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// 解析 IP:Port
		ipPortParts := strings.Split(parts[0], ":")
		if len(ipPortParts) != 2 {
			continue
		}

		ip := strings.TrimSpace(ipPortParts[0])
		port := strings.TrimSpace(ipPortParts[1])
		connType := NormalizeFscanServiceType(strings.TrimSpace(parts[1]))

		if connType == "" {
			continue
		}

		user := ""
		pass := ""

		if len(parts) >= 3 {
			credInfo := parts[2]
			
			if !strings.Contains(strings.ToLower(line), "未授权") && 
			   !strings.Contains(strings.ToLower(line), "unauthorized") &&
			   credInfo != "/" {
				if strings.Contains(credInfo, "/") {
					credParts := strings.SplitN(credInfo, "/", 2)
					if len(credParts) >= 1 {
						user = strings.TrimSpace(credParts[0])
					}
					if len(credParts) >= 2 {
						pass = strings.TrimSpace(credParts[1])
					}
				} else {
					user = strings.TrimSpace(credInfo)
				}
			}
		}

		// 去重
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
