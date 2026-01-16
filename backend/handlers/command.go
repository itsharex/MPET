package handlers

import (
	"MPET/backend/services"
	"fmt"
	"strings"
	"time"
)

type CommandHandler struct {
	service *services.ConnectorService
	logger  *services.Logger
}

func NewCommandHandler(service *services.ConnectorService) *CommandHandler {
	return &CommandHandler{
		service: service,
		logger:  services.GetLogger(),
	}
}

// ExecuteCommand 执行命令
func (h *CommandHandler) ExecuteCommand(id string, command string) (string, error) {
	conn, exists := h.service.GetConnection(id)
	if !exists {
		h.logger.Warn(fmt.Sprintf("执行命令失败: 连接不存在 (id=%s)", id))
		return "", fmt.Errorf("连接不存在")
	}

	if conn.Status != "success" {
		h.logger.Warn(fmt.Sprintf("执行命令失败: 连接未成功 (id=%s)", id))
		return "", fmt.Errorf("连接未成功，无法执行命令")
	}

	h.logger.Info(fmt.Sprintf("执行命令: %s %s:%s - %s", conn.Type, conn.IP, conn.Port, command))

	result, err := h.service.ExecuteCommand(conn, command)
	if err != nil {
		h.logger.Error(fmt.Sprintf("命令执行失败: %v", err))
		return "", err
	}

	// 追加日志和结果
	timestamp := time.Now().Format("15:04:05")
	logEntry := fmt.Sprintf("[%s] 执行命令: %s", timestamp, command)
	conn.Logs = append(conn.Logs, logEntry)
	
	separator := fmt.Sprintf("\n%s\n命令: %s\n执行时间: %s\n%s\n\n", 
		strings.Repeat("=", 60), 
		command, 
		time.Now().Format("2006-01-02 15:04:05"),
		strings.Repeat("=", 60))
	conn.Result += separator + result + "\n"
	
	conn.Logs = append(conn.Logs, fmt.Sprintf("[%s] ✓ 命令执行完成", timestamp))
	
	if err := h.service.UpdateConnectionResult(id, conn.Result, conn.Logs); err != nil {
		h.logger.Error(fmt.Sprintf("保存命令结果失败: %v", err))
	}

	h.logger.Info(fmt.Sprintf("命令执行成功: %s %s:%s", conn.Type, conn.IP, conn.Port))
	return result, nil
}

// ExecuteContainerCommand 在 Docker 容器中执行命令
func (h *CommandHandler) ExecuteContainerCommand(id string, containerID string, command string) (string, error) {
	conn, exists := h.service.GetConnection(id)
	if !exists {
		h.logger.Warn(fmt.Sprintf("执行容器命令失败: 连接不存在 (id=%s)", id))
		return "", fmt.Errorf("连接不存在")
	}

	if conn.Status != "success" {
		h.logger.Warn(fmt.Sprintf("执行容器命令失败: 连接未成功 (id=%s)", id))
		return "", fmt.Errorf("连接未成功，无法执行命令")
	}

	if conn.Type != "Docker" {
		return "", fmt.Errorf("仅支持 Docker 类型的连接")
	}

	h.logger.Info(fmt.Sprintf("在容器中执行命令: %s %s:%s [容器: %s] - %s", 
		conn.Type, conn.IP, conn.Port, containerID, command))

	result, err := h.service.ExecuteContainerCommand(conn, containerID, command)
	if err != nil {
		h.logger.Error(fmt.Sprintf("容器命令执行失败: %v", err))
		return "", err
	}

	h.logger.Info(fmt.Sprintf("容器命令执行成功: %s %s:%s [容器: %s]", 
		conn.Type, conn.IP, conn.Port, containerID))
	
	return result, nil
}
