package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config Agent配置
type Config struct {
	AgentID    string `yaml:"agent_id"`
	ServerAddr string `yaml:"server_addr"`
	CertDir    string `yaml:"cert_dir"`
	LogFile    string `yaml:"log_file"`
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
	return cfg, nil
}
