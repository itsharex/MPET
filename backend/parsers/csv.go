package parsers

import (
	"MPET/backend/models"
	"encoding/csv"
	"fmt"
	"strings"
)

// ParseCSV 解析 CSV 内容
func ParseCSV(content string) ([]*models.ConnectionRequest, error) {
	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV 解析失败: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV 文件至少需要包含表头和数据行")
	}

	// 解析表头
	header := records[0]
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// 检查必需的列
	requiredFields := []string{"type", "ip", "port"}
	for _, field := range requiredFields {
		if _, exists := headerMap[field]; !exists {
			return nil, fmt.Errorf("CSV 文件缺少必需的列: %s", field)
		}
	}

	var connections []*models.ConnectionRequest
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) <= headerMap["type"] || len(record) <= headerMap["ip"] || len(record) <= headerMap["port"] {
			continue
		}

		connType := strings.TrimSpace(record[headerMap["type"]])
		ip := strings.TrimSpace(record[headerMap["ip"]])
		port := strings.TrimSpace(record[headerMap["port"]])

		if connType == "" || ip == "" || port == "" {
			continue
		}

		user := ""
		pass := ""
		if idx, exists := headerMap["user"]; exists && idx < len(record) {
			user = strings.TrimSpace(record[idx])
		}
		if idx, exists := headerMap["pass"]; exists && idx < len(record) {
			pass = strings.TrimSpace(record[idx])
		}

		connections = append(connections, &models.ConnectionRequest{
			Type: connType,
			IP:   ip,
			Port: port,
			User: user,
			Pass: pass,
		})
	}

	return connections, nil
}
