package handlers

import (
	"MPET/backend/models"
	"MPET/backend/services"
	"fmt"
)

type ConnectionHandler struct {
	service *services.ConnectorService
	logger  *services.Logger
}

func NewConnectionHandler(service *services.ConnectorService) *ConnectionHandler {
	return &ConnectionHandler{
		service: service,
		logger:  services.GetLogger(),
	}
}

// GetConnections 获取连接列表
func (h *ConnectionHandler) GetConnections(connType string) []*models.Connection {
	if connType != "" && connType != "all" {
		return h.service.GetConnectionsByType(connType)
	}
	return h.service.GetAllConnections()
}

// AddConnection 添加连接
func (h *ConnectionHandler) AddConnection(req models.ConnectionRequest) (*models.Connection, error) {
	if req.Type == "" || req.IP == "" || req.Port == "" {
		h.logger.Warn(fmt.Sprintf("添加连接失败: 缺少必需字段 (type=%s, ip=%s, port=%s)", req.Type, req.IP, req.Port))
		return nil, fmt.Errorf("缺少必需字段: type, ip, port")
	}

	conn := h.service.CreateConnectionFromCSV(req.Type, req.IP, req.Port, req.User, req.Pass)
	if err := h.service.AddConnection(conn); err != nil {
		h.logger.Error(fmt.Sprintf("添加连接失败: %v", err))
		return nil, err
	}

	h.logger.Info(fmt.Sprintf("添加连接: %s %s:%s", req.Type, req.IP, req.Port))
	go h.service.Connect(conn)

	return conn, nil
}

// ConnectSingle 单个连接
func (h *ConnectionHandler) ConnectSingle(id string) error {
	conn, exists := h.service.GetConnection(id)
	if !exists {
		h.logger.Warn(fmt.Sprintf("单个连接失败: 连接不存在 (id=%s)", id))
		return fmt.Errorf("连接不存在")
	}

	h.logger.Info(fmt.Sprintf("单个连接: %s %s:%s", conn.Type, conn.IP, conn.Port))
	go h.service.Connect(conn)

	return nil
}

// ConnectBatch 批量连接
func (h *ConnectionHandler) ConnectBatch(ids []string) (int, error) {
	count := 0
	for _, id := range ids {
		conn, exists := h.service.GetConnection(id)
		if !exists {
			continue
		}
		count++
		go h.service.Connect(conn)
	}

	h.logger.Info(fmt.Sprintf("批量连接: 启动 %d 个连接任务", count))
	return count, nil
}

// UpdateConnection 更新连接
func (h *ConnectionHandler) UpdateConnection(id string, req models.ConnectionRequest) error {
	if req.Type == "" || req.IP == "" || req.Port == "" {
		h.logger.Warn(fmt.Sprintf("更新连接失败: 缺少必需字段"))
		return fmt.Errorf("缺少必需字段: type, ip, port")
	}

	existingConn, exists := h.service.GetConnection(id)
	if !exists {
		h.logger.Warn(fmt.Sprintf("更新连接失败: 连接不存在 (id=%s)", id))
		return fmt.Errorf("连接不存在")
	}

	password := req.Pass
	if password == "" {
		password = existingConn.Pass
	}

	if err := h.service.UpdateConnectionInfo(id, req.Type, req.IP, req.Port, req.User, password); err != nil {
		h.logger.Error(fmt.Sprintf("更新连接失败: %v", err))
		return err
	}

	h.logger.Info(fmt.Sprintf("更新连接: %s %s:%s", req.Type, req.IP, req.Port))
	return nil
}

// DeleteConnection 删除连接
func (h *ConnectionHandler) DeleteConnection(id string) error {
	if !h.service.DeleteConnection(id) {
		h.logger.Warn(fmt.Sprintf("删除连接失败: 连接不存在 (id=%s)", id))
		return fmt.Errorf("连接不存在")
	}
	
	h.logger.Info(fmt.Sprintf("删除连接: id=%s", id))
	return nil
}

// DeleteBatchConnections 批量删除
func (h *ConnectionHandler) DeleteBatchConnections(ids []string) (int, error) {
	count, err := h.service.DeleteBatchConnections(ids)
	if err != nil {
		h.logger.Error(fmt.Sprintf("批量删除失败: %v", err))
		return count, err
	}
	h.logger.Info(fmt.Sprintf("批量删除: 删除 %d 个连接", count))
	return count, nil
}

// TestConnection 测试连接
func (h *ConnectionHandler) TestConnection(req models.ConnectionRequest) (string, error) {
	if req.Type == "" || req.IP == "" || req.Port == "" {
		h.logger.Warn("测试连接失败: 缺少必需字段")
		return "", fmt.Errorf("缺少必需字段: type, ip, port")
	}

	h.logger.Info(fmt.Sprintf("测试连接: %s %s:%s", req.Type, req.IP, req.Port))

	tempConn := h.service.CreateConnectionFromCSV(req.Type, req.IP, req.Port, req.User, req.Pass)
	h.service.Connect(tempConn)
	
	if tempConn.Status == "success" {
		h.logger.Info(fmt.Sprintf("测试连接成功: %s %s:%s", req.Type, req.IP, req.Port))
		return tempConn.Message, nil
	}
	
	h.logger.Warn(fmt.Sprintf("测试连接失败: %s %s:%s - %s", req.Type, req.IP, req.Port, tempConn.Message))
	return "", fmt.Errorf(tempConn.Message)
}
