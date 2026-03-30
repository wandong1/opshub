package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config Agent配置
type Config struct {
	AgentID       string `yaml:"agent_id"`
	ServerAddr    string `yaml:"server_addr"`
	CertDir       string `yaml:"cert_dir"`
	LogFile       string `yaml:"log_file"`
	LogMaxSize    int    `yaml:"log_max_size"`    // 日志文件最大大小（MB），默认 100MB
	LogMaxBackups int    `yaml:"log_max_backups"` // 保留的旧日志文件数量，默认 3
	LogMaxAge     int    `yaml:"log_max_age"`     // 日志文件最大保留天数，默认 30 天
	LogLevel      string `yaml:"log_level"`       // 日志级别：debug, info, warn, error，默认 info
}

// Load 加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.LogMaxSize == 0 {
		cfg.LogMaxSize = 100 // 默认 100MB
	}
	if cfg.LogMaxBackups == 0 {
		cfg.LogMaxBackups = 3 // 默认保留 3 个备份
	}
	if cfg.LogMaxAge == 0 {
		cfg.LogMaxAge = 30 // 默认保留 30 天
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info" // 默认 info 级别
	}

	return cfg, nil
}
