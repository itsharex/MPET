package handlers

import (
	"MPET/backend/models"
	"MPET/backend/parsers"
	"MPET/backend/services"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ImportHandler struct {
	ctx     context.Context
	service *services.ConnectorService
	logger  *services.Logger
	connHandler *ConnectionHandler
}

func NewImportHandler(ctx context.Context, service *services.ConnectorService, connHandler *ConnectionHandler) *ImportHandler {
	return &ImportHandler{
		ctx:     ctx,
		service: service,
		logger:  services.GetLogger(),
		connHandler: connHandler,
	}
}

// ImportCSV 导入文件
func (h *ImportHandler) ImportCSV() (int, error) {
	filePath, err := runtime.OpenFileDialog(h.ctx, runtime.OpenDialogOptions{
		Title: "选择 CSV 或 Fscan 结果文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil {
		h.logger.Error(fmt.Sprintf("打开文件对话框失败: %v", err))
		return 0, err
	}

	if filePath == "" {
		h.logger.Warn("导入文件: 未选择文件")
		return 0, fmt.Errorf("未选择文件")
	}

	h.logger.Info(fmt.Sprintf("开始导入文件: %s", filePath))

	content, err := os.ReadFile(filePath)
	if err != nil {
		h.logger.Error(fmt.Sprintf("读取文件失败: %v", err))
		return 0, fmt.Errorf("读取文件失败: %v", err)
	}

	// 判断文件类型
	if strings.HasSuffix(strings.ToLower(filePath), ".txt") {
		return h.importTextFile(string(content))
	}

	return h.importCSVContent(string(content))
}

func (h *ImportHandler) importTextFile(content string) (int, error) {
	var connections []*models.ConnectionRequest
	
	if strings.Contains(content, "[Plugin:") && strings.Contains(content, ":SUCCESS]") {
		connections = parsers.ParseLightx(content)
	} else if strings.Contains(content, "# ===== 漏洞信息 =====") {
		connections = parsers.ParseFscan21(content)
	} else {
		connections = parsers.ParseFscan(content)
	}

	return h.addConnections(connections)
}

func (h *ImportHandler) importCSVContent(content string) (int, error) {
	connections, err := parsers.ParseCSV(content)
	if err != nil {
		h.logger.Error(fmt.Sprintf("CSV 解析失败: %v", err))
		return 0, err
	}

	return h.addConnections(connections)
}

func (h *ImportHandler) addConnections(connections []*models.ConnectionRequest) (int, error) {
	count := 0
	skipped := 0

	for _, req := range connections {
		conn := h.service.CreateConnectionFromCSV(req.Type, req.IP, req.Port, req.User, req.Pass)
		if err := h.service.AddConnection(conn); err != nil {
			skipped++
			continue
		}
		count++
	}

	h.logger.Info(fmt.Sprintf("导入完成: 成功 %d 条, 跳过 %d 条", count, skipped))
	return count, nil
}
