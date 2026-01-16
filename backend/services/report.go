package services

import (
	"MPET/backend/models"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReportService 报告服务
type ReportService struct {
	connectorService *ConnectorService
	vulnService      *VulnerabilityService
}

// NewReportService 创建报告服务
func NewReportService(connectorService *ConnectorService) *ReportService {
	return &ReportService{
		connectorService: connectorService,
		vulnService:      NewVulnerabilityService(connectorService.GetDB()),
	}
}

// VulnerabilityData 漏洞数据
type VulnerabilityData struct {
	Name     string   `json:"name"`     // 漏洞名称
	Level    string   `json:"level"`    // 风险等级
	Target   string   `json:"target"`   // URL/目标
	Describe string   `json:"describe"` // 漏洞说明
	Images   []string `json:"images"`   // 漏洞截图（base64）
	Repair   string   `json:"repair"`   // 修复建议
}

// ExportReportRequest 导出报告请求
type ExportReportRequest struct {
	ConnectionIDs   []string            `json:"connectionIds"`   // 要导出的连接ID列表
	Vulnerabilities []VulnerabilityData `json:"vulnerabilities"` // 漏洞数据列表
}

// ExportReport 导出 Markdown 格式报告
func (s *ReportService) ExportReport(req ExportReportRequest) (string, error) {
	fmt.Printf("开始导出报告，连接数: %d, 漏洞数: %d\n", len(req.ConnectionIDs), len(req.Vulnerabilities))

	// 补充漏洞数据
	if len(req.Vulnerabilities) == 0 {
		fmt.Println("没有漏洞数据，从连接生成")
		req.Vulnerabilities = s.generateVulnerabilitiesFromConnections(req.ConnectionIDs)
	} else {
		fmt.Println("补充漏洞数据")
		for i := range req.Vulnerabilities {
			if req.Vulnerabilities[i].Name == "" && i < len(req.ConnectionIDs) {
				conn, exists := s.connectorService.GetConnection(req.ConnectionIDs[i])
				if exists {
					fmt.Printf("补充连接 %s (类型: %s)\n", req.ConnectionIDs[i], conn.Type)
					req.Vulnerabilities[i].Name = s.getVulnerabilityName(conn)
					req.Vulnerabilities[i].Level = s.getVulnerabilityLevel(conn)
					req.Vulnerabilities[i].Target = fmt.Sprintf("%s:%s", conn.IP, conn.Port)
					req.Vulnerabilities[i].Describe = s.getVulnerabilityDescription(conn)
					req.Vulnerabilities[i].Repair = s.getRepairSuggestion(conn)
				}
			}
		}
	}

	if len(req.Vulnerabilities) == 0 {
		return "", fmt.Errorf("没有可导出的漏洞数据")
	}

	// 创建输出目录
	timestamp := time.Now().Format("20060102_150405")
	reportDir := filepath.Join("reports", fmt.Sprintf("report_%s", timestamp))
	assetsDir := filepath.Join(reportDir, "assets")
	
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	// 生成 Markdown 内容
	var md strings.Builder
	
	// 报告头部
	md.WriteString("# 漏洞扫描报告\n\n")
	md.WriteString(fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**漏洞总数**: %d\n\n", len(req.Vulnerabilities)))
	md.WriteString("---\n\n")

	// 目录
	md.WriteString("## 目录\n\n")
	for i, vul := range req.Vulnerabilities {
		md.WriteString(fmt.Sprintf("%d. [%s](#漏洞-%d-%s)\n", i+1, vul.Name, i+1, strings.ReplaceAll(vul.Name, " ", "-")))
	}
	md.WriteString("\n---\n\n")

	// 漏洞详情
	for i, vul := range req.Vulnerabilities {
		fmt.Printf("处理漏洞 %d: %s\n", i+1, vul.Name)
		
		md.WriteString(fmt.Sprintf("## 漏洞 %d: %s\n\n", i+1, vul.Name))
		
		// 基本信息表格
		md.WriteString("### 基本信息\n\n")
		md.WriteString("| 项目 | 内容 |\n")
		md.WriteString("|------|------|\n")
		md.WriteString(fmt.Sprintf("| **漏洞名称** | %s |\n", vul.Name))
		md.WriteString(fmt.Sprintf("| **风险等级** | %s |\n", s.getLevelBadge(vul.Level)))
		md.WriteString(fmt.Sprintf("| **目标地址** | `%s` |\n", vul.Target))
		md.WriteString("\n")
		
		// 漏洞说明
		md.WriteString("### 漏洞说明\n\n")
		// 将 \\n 替换为真正的换行符
		description := strings.ReplaceAll(vul.Describe, "\\n", "\n")
		md.WriteString(description)
		md.WriteString("\n\n")
		
		// 漏洞截图
		if len(vul.Images) > 0 {
			md.WriteString("### 漏洞截图\n\n")
			for j, img := range vul.Images {
				imgPath, err := s.saveBase64ImageToAssets(img, assetsDir, i, j)
				if err == nil {
					// 使用相对路径
					relPath := filepath.Join("assets", filepath.Base(imgPath))
					md.WriteString(fmt.Sprintf("![漏洞截图 %d](%s)\n\n", j+1, relPath))
					fmt.Printf("截图已保存: %s\n", imgPath)
				} else {
					md.WriteString(fmt.Sprintf("*截图 %d 保存失败*\n\n", j+1))
					fmt.Printf("截图保存失败: %v\n", err)
				}
			}
		} else {
			md.WriteString("### 漏洞截图\n\n")
			md.WriteString("*无截图*\n\n")
		}
		
		// 修复建议
		md.WriteString("### 修复建议\n\n")
		// 先将 \\n 替换为真正的换行符，然后转换为 Markdown 列表
		repair := strings.ReplaceAll(vul.Repair, "\\n", "\n")
		repairLines := strings.Split(repair, "\n")
		for _, line := range repairLines {
			line = strings.TrimSpace(line)
			if line != "" {
				md.WriteString(line)
				md.WriteString("\n")
			}
		}
		md.WriteString("\n")
		
		md.WriteString("---\n\n")
	}

	// 报告尾部
	md.WriteString("## 报告说明\n\n")
	md.WriteString("本报告由 MPET (Multi-Protocol Exploitation Toolkit) 自动生成。\n\n")
	md.WriteString("**注意事项**:\n\n")
	md.WriteString("- 本报告仅供安全测试和漏洞修复参考\n")
	md.WriteString("- 请勿将本报告用于非法用途\n")
	md.WriteString("- 建议尽快修复报告中列出的安全漏洞\n")
	md.WriteString("- 修复后请重新进行安全测试验证\n\n")
	md.WriteString("---\n\n")
	md.WriteString(fmt.Sprintf("*报告生成时间: %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	// 保存 Markdown 文件
	reportPath := filepath.Join(reportDir, "漏洞报告.md")
	if err := os.WriteFile(reportPath, []byte(md.String()), 0644); err != nil {
		return "", fmt.Errorf("保存报告失败: %v", err)
	}

	absPath, _ := filepath.Abs(reportPath)
	fmt.Printf("报告已保存: %s\n", absPath)
	return absPath, nil
}

// getLevelBadge 获取风险等级徽章
func (s *ReportService) getLevelBadge(level string) string {
	badges := map[string]string{
		"高危": "🔴 **高危**",
		"中危": "🟡 **中危**",
		"低危": "🟢 **低危**",
	}
	if badge, ok := badges[level]; ok {
		return badge
	}
	return level
}

// saveBase64ImageToAssets 保存 base64 图片到 assets 目录
func (s *ReportService) saveBase64ImageToAssets(base64Str string, assetsDir string, vulIndex, imgIndex int) (string, error) {
	// 移除 data:image/png;base64, 前缀
	if strings.Contains(base64Str, ",") {
		base64Str = strings.Split(base64Str, ",")[1]
	}

	// 解码 base64
	imgData, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", fmt.Errorf("解码图片失败: %v", err)
	}

	// 保存图片
	imgPath := filepath.Join(assetsDir, fmt.Sprintf("vuln_%d_screenshot_%d.png", vulIndex+1, imgIndex+1))
	if err := os.WriteFile(imgPath, imgData, 0644); err != nil {
		return "", fmt.Errorf("保存图片失败: %v", err)
	}

	return imgPath, nil
}

// generateVulnerabilitiesFromConnections 从连接生成漏洞数据
func (s *ReportService) generateVulnerabilitiesFromConnections(connectionIDs []string) []VulnerabilityData {
	var vulnerabilities []VulnerabilityData

	for _, id := range connectionIDs {
		conn, exists := s.connectorService.GetConnection(id)
		if !exists {
			fmt.Printf("连接 %s 不存在\n", id)
			continue
		}

		fmt.Printf("处理连接: 类型=%s, IP=%s:%s\n", conn.Type, conn.IP, conn.Port)

		vul := VulnerabilityData{
			Name:     s.getVulnerabilityName(conn),
			Level:    s.getVulnerabilityLevel(conn),
			Target:   fmt.Sprintf("%s:%s", conn.IP, conn.Port),
			Describe: s.getVulnerabilityDescription(conn),
			Images:   []string{},
			Repair:   s.getRepairSuggestion(conn),
		}

		vulnerabilities = append(vulnerabilities, vul)
	}

	return vulnerabilities
}

// getVulnerabilityName 获取漏洞名称（从数据库读取，根据是否有用户名密码判断）
func (s *ReportService) getVulnerabilityName(conn *models.Connection) string {
	// 判断是弱口令还是未授权
	serviceType := conn.Type
	if conn.User == "" && conn.Pass == "" {
		// 未授权访问
		serviceType = conn.Type + "_Unauth"
	} else {
		// 弱口令
		serviceType = conn.Type + "_Weak"
	}

	vuln, err := s.vulnService.GetByServiceType(serviceType)
	if err == nil && vuln != nil {
		return vuln.Name
	}
	// 降级到默认值
	if conn.User == "" && conn.Pass == "" {
		return fmt.Sprintf("%s 未授权访问漏洞", conn.Type)
	}
	return fmt.Sprintf("%s 弱口令漏洞", conn.Type)
}

// getVulnerabilityLevel 获取风险等级（从数据库读取）
func (s *ReportService) getVulnerabilityLevel(conn *models.Connection) string {
	// 判断是弱口令还是未授权
	serviceType := conn.Type
	if conn.User == "" && conn.Pass == "" {
		serviceType = conn.Type + "_Unauth"
	} else {
		serviceType = conn.Type + "_Weak"
	}

	vuln, err := s.vulnService.GetByServiceType(serviceType)
	if err == nil && vuln != nil {
		return vuln.Level
	}
	// 降级到默认值：所有弱口令都是高危
	return "高危"
}

// getVulnerabilityDescription 获取漏洞说明（从数据库读取并填充用户名密码）
func (s *ReportService) getVulnerabilityDescription(conn *models.Connection) string {
	// 判断是弱口令还是未授权
	serviceType := conn.Type
	if conn.User == "" && conn.Pass == "" {
		serviceType = conn.Type + "_Unauth"
	} else {
		serviceType = conn.Type + "_Weak"
	}

	vuln, err := s.vulnService.GetByServiceType(serviceType)
	if err == nil && vuln != nil {
		// 替换占位符
		desc := vuln.Description
		desc = strings.ReplaceAll(desc, "{username}", conn.User)
		desc = strings.ReplaceAll(desc, "{password}", conn.Pass)
		return desc
	}
	// 降级到默认值
	if conn.User == "" && conn.Pass == "" {
		return fmt.Sprintf("目标 %s 服务未启用认证保护，允许任意用户未授权访问，可能被攻击者利用。", conn.Type)
	}
	return fmt.Sprintf("目标 %s 服务使用了弱口令（用户名：%s，密码：%s），攻击者可以通过暴力破解获取访问权限。", conn.Type, conn.User, conn.Pass)
}

// getRepairSuggestion 获取修复建议（从数据库读取）
func (s *ReportService) getRepairSuggestion(conn *models.Connection) string {
	// 判断是弱口令还是未授权
	serviceType := conn.Type
	if conn.User == "" && conn.Pass == "" {
		serviceType = conn.Type + "_Unauth"
	} else {
		serviceType = conn.Type + "_Weak"
	}

	vuln, err := s.vulnService.GetByServiceType(serviceType)
	if err == nil && vuln != nil {
		return vuln.Repair
	}
	// 降级到默认值
	return "1. 启用服务认证\n2. 设置复杂密码（至少 16 位，包含大小写字母、数字和特殊字符）\n3. 限制网络访问\n4. 定期更新版本\n5. 审计安全日志"
}
