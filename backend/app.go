package backend

import (
	"MPET/backend/handlers"
	"MPET/backend/models"
	"MPET/backend/services"
	"context"
	"fmt"
	"log"
)

// App 应用结构
type App struct {
	ctx context.Context
	
	// Handlers
	connHandler    *handlers.ConnectionHandler
	importHandler  *handlers.ImportHandler
	proxyHandler   *handlers.ProxyHandler
	commandHandler *handlers.CommandHandler
	fileHandler    *handlers.FileHandler
	reportHandler  *handlers.ReportHandler
	
	// Service
	service     *services.ConnectorService
	vulnService *services.VulnerabilityService
	logger      *services.Logger
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{}
}

// Startup 应用启动
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger = services.GetLogger()
	
	a.logger.Info("========================================")
	a.logger.Info("MPET - Multi-Protocol Exploitation Toolkit")
	a.logger.Info("多协议漏洞利用与攻击模拟平台")
	a.logger.Info("公众号：浅梦安全")
	a.logger.Info("开发者：onewin")
	a.logger.Info("========================================")
	
	service, err := services.NewConnectorService()
	if err != nil {
		a.logger.Error(fmt.Sprintf("初始化服务失败: %v", err))
		log.Fatal("初始化服务失败:", err)
	}
	a.service = service
	
	// 初始化 handlers
	a.connHandler = handlers.NewConnectionHandler(service)
	a.proxyHandler = handlers.NewProxyHandler(service)
	a.commandHandler = handlers.NewCommandHandler(service)
	a.fileHandler = handlers.NewFileHandler(ctx, service)
	a.importHandler = handlers.NewImportHandler(ctx, service, a.connHandler)
	
	// 初始化报告服务
	reportService := services.NewReportService(service)
	a.reportHandler = handlers.NewReportHandler(reportService)
	
	// 初始化漏洞信息服务
	a.vulnService = services.NewVulnerabilityService(service.GetDB())
	if err := a.vulnService.InitDefaultVulnerabilities(); err != nil {
		a.logger.Error(fmt.Sprintf("初始化漏洞信息失败: %v", err))
	}
	
	a.logger.Info("服务初始化成功")
	
	log.Println("========================================")
	log.Println("MPET - Multi-Protocol Exploitation Toolkit")
	log.Println("多协议漏洞利用与攻击模拟平台")
	log.Println("公众号：浅梦安全")
	log.Println("开发者：onewin")
	log.Println("========================================")
}

// ===== 连接管理 =====
func (a *App) GetConnections(connType string) []*models.Connection {
	return a.connHandler.GetConnections(connType)
}

func (a *App) AddConnection(req models.ConnectionRequest) (*models.Connection, error) {
	return a.connHandler.AddConnection(req)
}

func (a *App) ConnectSingle(id string) error {
	return a.connHandler.ConnectSingle(id)
}

func (a *App) ConnectBatch(ids []string) (int, error) {
	return a.connHandler.ConnectBatch(ids)
}

func (a *App) UpdateConnection(id string, req models.ConnectionRequest) error {
	return a.connHandler.UpdateConnection(id, req)
}

func (a *App) DeleteConnection(id string) error {
	return a.connHandler.DeleteConnection(id)
}

func (a *App) DeleteBatchConnections(ids []string) (int, error) {
	return a.connHandler.DeleteBatchConnections(ids)
}

func (a *App) TestConnection(req models.ConnectionRequest) (string, error) {
	return a.connHandler.TestConnection(req)
}

// ===== 导入功能 =====
func (a *App) ImportCSV() (int, error) {
	return a.importHandler.ImportCSV()
}

// ===== 代理配置 =====
func (a *App) GetProxySettings() models.ProxyConfig {
	return a.proxyHandler.GetProxySettings()
}

func (a *App) UpdateProxySettings(proxy models.ProxyConfig) error {
	return a.proxyHandler.UpdateProxySettings(proxy)
}

// ===== 命令执行 =====
func (a *App) ExecuteCommand(id string, command string) (string, error) {
	return a.commandHandler.ExecuteCommand(id, command)
}

// ExecuteContainerCommand 在 Docker 容器中执行命令
func (a *App) ExecuteContainerCommand(id string, containerID string, command string) (string, error) {
	return a.commandHandler.ExecuteContainerCommand(id, containerID, command)
}

// GetDockerContainers 获取 Docker 容器列表
func (a *App) GetDockerContainers(id string) (string, error) {
	conn, exists := a.service.GetConnection(id)
	if !exists {
		return "", fmt.Errorf("连接不存在")
	}
	
	if conn.Type != "Docker" {
		return "", fmt.Errorf("仅支持 Docker 类型的连接")
	}
	
	return a.service.GetDockerContainersJSON(id)
}

// GetK8sPods 获取 Kubernetes Pod 列表
func (a *App) GetK8sPods(id string) (string, error) {
	conn, exists := a.service.GetConnection(id)
	if !exists {
		return "", fmt.Errorf("连接不存在")
	}
	
	if conn.Type != "Kubernetes" {
		return "", fmt.Errorf("仅支持 Kubernetes 类型的连接")
	}
	
	return a.service.GetK8sPodsJSON(id)
}

// ===== 文件操作 =====
func (a *App) BrowseFTPDirectory(id string, dirPath string) (string, error) {
	return a.fileHandler.BrowseFTPDirectory(id, dirPath)
}

func (a *App) DownloadFTPFile(id string, filePath string) error {
	return a.fileHandler.DownloadFTPFile(id, filePath)
}

func (a *App) BrowseSMBDirectory(id string, shareName string, dirPath string) (string, error) {
	return a.fileHandler.BrowseSMBDirectory(id, shareName, dirPath)
}

func (a *App) DownloadSMBFile(id string, shareName string, filePath string) error {
	return a.fileHandler.DownloadSMBFile(id, shareName, filePath)
}

func (a *App) BrowseSFTPDirectory(id string, dirPath string) (string, error) {
	return a.fileHandler.BrowseSFTPDirectory(id, dirPath)
}

func (a *App) DownloadSFTPFile(id string, filePath string) error {
	return a.fileHandler.DownloadSFTPFile(id, filePath)
}

// ===== 系统日志 =====
func (a *App) GetSystemLogs() []string {
	return a.logger.GetLogs()
}

func (a *App) ClearSystemLogs() error {
	a.logger.Clear()
	a.logger.Info("系统日志已清空")
	return nil
}

// ===== 报告导出 =====
func (a *App) ExportReport(req services.ExportReportRequest) (string, error) {
	return a.reportHandler.ExportReport(req)
}

// ===== 服务类型 =====
func (a *App) GetServiceTypes() []map[string]interface{} {
	return []map[string]interface{}{
		{"value": "Redis", "label": "Redis", "port": "6379"},
		{"value": "Memcached", "label": "Memcached", "port": "11211"},
		{"value": "MySQL", "label": "MySQL", "port": "3306"},
		{"value": "PostgreSQL", "label": "PostgreSQL", "port": "5432"},
		{"value": "SQLServer", "label": "SQL Server", "port": "1433"},
		{"value": "Oracle", "label": "Oracle", "port": "1521"},
		{"value": "MongoDB", "label": "MongoDB", "port": "27017"},
		{"value": "FTP", "label": "FTP", "port": "21"},
		{"value": "SFTP", "label": "SFTP", "port": "22"},
		{"value": "SSH", "label": "SSH", "port": "22"},
		{"value": "SMB", "label": "SMB", "port": "445"},
		{"value": "RabbitMQ", "label": "RabbitMQ", "port": "5672"},
		{"value": "MQTT", "label": "MQTT", "port": "1883"},
		{"value": "WMI", "label": "WMI", "port": "135"},
		{"value": "Elasticsearch", "label": "Elasticsearch", "port": "9200"},
		{"value": "Zookeeper", "label": "Zookeeper", "port": "2181"},
		{"value": "Etcd", "label": "Etcd", "port": "2379"},
		{"value": "ADB", "label": "ADB", "port": "5555"},
		{"value": "Kafka", "label": "Kafka", "port": "9092"},
		{"value": "JDWP", "label": "JDWP", "port": "8000"},
		{"value": "RMI", "label": "RMI", "port": "1099"},
		{"value": "RDP", "label": "RDP", "port": "3389"},
		{"value": "VNC", "label": "VNC", "port": "5900"},
		{"value": "Docker", "label": "Docker API", "port": "2375"},
		{"value": "Kubernetes", "label": "Kubernetes API", "port": "8080"},
	}
}

// ===== 漏洞信息管理 =====
func (a *App) GetAllVulnerabilities() ([]models.VulnerabilityInfo, error) {
	return a.vulnService.GetAll()
}

func (a *App) GetVulnerabilityByServiceType(serviceType string) (*models.VulnerabilityInfo, error) {
	return a.vulnService.GetByServiceType(serviceType)
}

func (a *App) SaveVulnerability(vuln models.VulnerabilityInfo) error {
	return a.vulnService.Save(&vuln)
}

func (a *App) DeleteVulnerability(id string) error {
	return a.vulnService.Delete(id)
}
