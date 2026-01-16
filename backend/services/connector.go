package services

import (
	"MPET/backend/config"
	"MPET/backend/models"
	"MPET/backend/services/connectors"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ConnectorService struct {
	db     *sql.DB
	config *models.Config
	// 嵌入 connectors.ConnectorService 以使用其方法
	*connectors.ConnectorService
}

func NewConnectorService() (*ConnectorService, error) {
	db, err := initDatabase()
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %v", err)
	}

	service := &ConnectorService{
		db:     db,
		config: cfg,
		ConnectorService: &connectors.ConnectorService{
			DB:     db,
			Config: cfg,
		},
	}

	// 应用启动时重置所有连接状态为待测试
	service.ResetAllConnectionStatus()

	return service, nil
}

func (s *ConnectorService) UpdateConfig(cfg *models.Config) {
	if cfg != nil {
		s.config = cfg
		s.ConnectorService.Config = cfg
	}
}

// GetDB 获取数据库连接
func (s *ConnectorService) GetDB() *sql.DB {
	return s.db
}

// ResetAllConnectionStatus 重置所有连接状态为待测试（应用启动时调用）
func (s *ConnectorService) ResetAllConnectionStatus() error {
	updateSQL := `UPDATE connections SET status = 'pending', message = '等待连接测试'`
	_, err := s.db.Exec(updateSQL)
	if err != nil {
		log.Printf("重置连接状态失败: %v", err)
		return err
	}
	log.Println("已重置所有连接状态为待测试")
	return nil
}

func (s *ConnectorService) AddConnection(conn *models.Connection) error {
	values, err := connectionToValues(conn)
	if err != nil {
		return fmt.Errorf("序列化连接数据失败: %v", err)
	}

	insertSQL := `INSERT INTO connections 
		(id, type, ip, port, user, pass, status, message, result, logs, created_at, connected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(insertSQL, values...)
	return err
}

func (s *ConnectorService) GetConnection(id string) (*models.Connection, bool) {
	querySQL := `SELECT id, type, ip, port, user, pass, status, message, result, logs, created_at, connected_at
		FROM connections WHERE id = ?`

	row := s.db.QueryRow(querySQL, id)
	conn, err := connectionFromRow(row)
	if err != nil {
		return nil, false
	}
	return conn, true
}

func (s *ConnectorService) GetAllConnections() []*models.Connection {
	querySQL := `SELECT id, type, ip, port, user, pass, status, message, result, logs, created_at, connected_at
		FROM connections ORDER BY created_at DESC`

	rows, err := s.db.Query(querySQL)
	if err != nil {
		return []*models.Connection{}
	}
	defer rows.Close()

	connections := []*models.Connection{}
	for rows.Next() {
		conn, err := connectionFromRows(rows)
		if err != nil {
			continue
		}
		connections = append(connections, conn)
	}
	return connections
}

func (s *ConnectorService) GetConnectionsByType(connType string) []*models.Connection {
	querySQL := `SELECT id, type, ip, port, user, pass, status, message, result, logs, created_at, connected_at
		FROM connections WHERE type = ? ORDER BY created_at DESC`

	rows, err := s.db.Query(querySQL, connType)
	if err != nil {
		return []*models.Connection{}
	}
	defer rows.Close()

	connections := []*models.Connection{}
	for rows.Next() {
		conn, err := connectionFromRows(rows)
		if err != nil {
			continue
		}
		connections = append(connections, conn)
	}
	return connections
}

func (s *ConnectorService) DeleteConnection(id string) bool {
	deleteSQL := `DELETE FROM connections WHERE id = ?`
	result, err := s.db.Exec(deleteSQL, id)
	if err != nil {
		return false
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0
}

func (s *ConnectorService) UpdateConnection(conn *models.Connection) error {
	logsJSON := "[]"
	if conn.Logs != nil && len(conn.Logs) > 0 {
		jsonData, _ := json.Marshal(conn.Logs)
		logsJSON = string(jsonData)
	}

	connectedAtStr := ""
	if !conn.ConnectedAt.IsZero() {
		connectedAtStr = conn.ConnectedAt.Format(time.RFC3339)
	}

	updateSQL := `UPDATE connections SET 
		status = ?, message = ?, result = ?, logs = ?, connected_at = ?
		WHERE id = ?`

	_, err := s.db.Exec(updateSQL, conn.Status, conn.Message, conn.Result, logsJSON, connectedAtStr, conn.ID)
	return err
}

func (s *ConnectorService) UpdateConnectionInfo(id, connType, ip, port, user, pass string) error {
	updateSQL := `UPDATE connections SET 
		type = ?, ip = ?, port = ?, user = ?, pass = ?
		WHERE id = ?`

	_, err := s.db.Exec(updateSQL, connType, ip, port, user, pass, id)
	return err
}

func (s *ConnectorService) DeleteBatchConnections(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]

	deleteSQL := fmt.Sprintf("DELETE FROM connections WHERE id IN (%s)", placeholders)

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	result, err := s.db.Exec(deleteSQL, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

func (s *ConnectorService) CreateConnectionFromCSV(connType, ip, port, user, pass string) *models.Connection {
	return &models.Connection{
		ID:        uuid.New().String(),
		Type:      connType,
		IP:        ip,
		Port:      port,
		User:      user,
		Pass:      pass,
		Status:    "pending",
		CreatedAt: time.Now(),
		Logs:      []string{},
	}
}

// Connect 包装 connectors.Connect 并更新数据库
func (s *ConnectorService) Connect(conn *models.Connection) {
	// 从数据库加载现有的日志和结果，保留历史记录
	existingConn, exists := s.GetConnection(conn.ID)
	if exists {
		// 保留历史日志和结果
		conn.Logs = existingConn.Logs
		conn.Result = existingConn.Result
	}
	
	// 更新数据库状态
	s.UpdateConnection(conn)
	
	// 调用实际的连接逻辑
	s.ConnectorService.Connect(conn)
	
	// 连接完成后更新数据库
	s.UpdateConnection(conn)
}

// UpdateConnectionResult 更新连接的执行结果和日志
func (s *ConnectorService) UpdateConnectionResult(id string, result string, logs []string) error {
	logsJSON := "[]"
	if logs != nil && len(logs) > 0 {
		jsonData, _ := json.Marshal(logs)
		logsJSON = string(jsonData)
	}

	updateSQL := `UPDATE connections SET result = ?, logs = ? WHERE id = ?`
	_, err := s.db.Exec(updateSQL, result, logsJSON, id)
	return err
}

// ExecuteContainerCommand 在 Docker 容器中执行命令
func (s *ConnectorService) ExecuteContainerCommand(conn *models.Connection, containerID string, command string) (string, error) {
	return s.ConnectorService.ExecuteContainerCommand(conn, containerID, command)
}

// GetDockerContainersJSON 获取 Docker 容器列表的 JSON 数据
func (s *ConnectorService) GetDockerContainersJSON(id string) (string, error) {
	conn, exists := s.GetConnection(id)
	if !exists {
		return "", fmt.Errorf("连接不存在")
	}
	
	if conn.Type != "Docker" {
		return "", fmt.Errorf("仅支持 Docker 类型的连接")
	}
	
	return s.ConnectorService.GetDockerContainersJSON(conn)
}

// GetK8sPodsJSON 获取 Kubernetes Pod 列表的 JSON 数据
func (s *ConnectorService) GetK8sPodsJSON(id string) (string, error) {
	conn, exists := s.GetConnection(id)
	if !exists {
		return "", fmt.Errorf("连接不存在")
	}
	
	if conn.Type != "Kubernetes" {
		return "", fmt.Errorf("仅支持 Kubernetes 类型的连接")
	}
	
	return s.ConnectorService.GetK8sPodsJSON(conn)
}
