package config

import (
	"MPET/backend/models"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

var (
	instance *models.Config
	lock     sync.RWMutex
)

func defaultConfig() *models.Config {
	return &models.Config{
		Password: "admin123",
		Proxy: models.ProxyConfig{
			Type: "socks5",
		},
	}
}

func normalizeConfig(cfg *models.Config) {
	if cfg == nil {
		return
	}
	if cfg.Password == "" {
		cfg.Password = "admin123"
	}
	if cfg.Proxy.Type == "" {
		cfg.Proxy.Type = "socks5"
	}
}

func loadFromFile() (*models.Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile("config.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			normalizeConfig(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	normalizeConfig(cfg)
	return cfg, nil
}

// LoadConfig 加载配置文件
func LoadConfig() (*models.Config, error) {
	lock.Lock()
	defer lock.Unlock()

	if instance != nil {
		return instance, nil
	}

	cfg, err := loadFromFile()
	if err != nil {
		return nil, err
	}

	instance = cfg
	return cfg, nil
}

// GetConfig 获取配置实例
func GetConfig() *models.Config {
	lock.RLock()
	if instance != nil {
		cfg := instance
		lock.RUnlock()
		return cfg
	}
	lock.RUnlock()
	cfg, _ := LoadConfig()
	return cfg
}

// SaveConfig 保存配置并更新内存实例
func SaveConfig(cfg *models.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	normalizeConfig(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return err
	}
	lock.Lock()
	instance = cfg
	lock.Unlock()
	return nil
}
