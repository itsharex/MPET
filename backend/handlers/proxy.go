package handlers

import (
	"MPET/backend/config"
	"MPET/backend/models"
	"MPET/backend/services"
	"fmt"
	"strings"
)

type ProxyHandler struct {
	service *services.ConnectorService
	logger  *services.Logger
}

func NewProxyHandler(service *services.ConnectorService) *ProxyHandler {
	return &ProxyHandler{
		service: service,
		logger:  services.GetLogger(),
	}
}

// GetProxySettings 获取代理配置
func (h *ProxyHandler) GetProxySettings() models.ProxyConfig {
	cfg := config.GetConfig()
	if cfg == nil {
		return models.ProxyConfig{}
	}
	return cfg.Proxy
}

// UpdateProxySettings 更新代理配置
func (h *ProxyHandler) UpdateProxySettings(proxy models.ProxyConfig) error {
	if proxy.Type == "" {
		proxy.Type = "socks5"
	}

	if proxy.Enabled {
		if strings.TrimSpace(proxy.Host) == "" || strings.TrimSpace(proxy.Port) == "" {
			h.logger.Warn("更新代理配置失败: 启用代理时必须填写主机和端口")
			return fmt.Errorf("启用代理时必须填写主机和端口")
		}
	}

	current := config.GetConfig()
	if current == nil {
		h.logger.Error("更新代理配置失败: 无法加载配置")
		return fmt.Errorf("无法加载配置")
	}

	updated := *current
	updated.Proxy = proxy

	if err := config.SaveConfig(&updated); err != nil {
		h.logger.Error(fmt.Sprintf("保存代理配置失败: %v", err))
		return err
	}

	h.service.UpdateConfig(&updated)
	
	if proxy.Enabled {
		h.logger.Info(fmt.Sprintf("代理配置已更新: %s://%s:%s", proxy.Type, proxy.Host, proxy.Port))
	} else {
		h.logger.Info("代理已禁用")
	}
	
	return nil
}
