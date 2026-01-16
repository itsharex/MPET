package handlers

import (
	"MPET/backend/services"
)

// ReportHandler 报告处理器
type ReportHandler struct {
	reportService *services.ReportService
}

// NewReportHandler 创建报告处理器
func NewReportHandler(reportService *services.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// ExportReport 导出报告
func (h *ReportHandler) ExportReport(req services.ExportReportRequest) (string, error) {
	return h.reportService.ExportReport(req)
}
