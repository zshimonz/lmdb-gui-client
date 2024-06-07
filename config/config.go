package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type ConnectionConfig struct {
	Name         string `yaml:"name"`
	DatabasePath string `yaml:"database_path"`
	MapSize      int64  `yaml:"map_size"` // GB
}

type AppConfig struct {
	Connections []ConnectionConfig `yaml:"connections"`
}

var configPath = "lmdb-gui-client.yaml"
var Config AppConfig

// 读取配置文件
func LoadConfig() error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // 文件不存在时，不进行任何操作
	}
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &Config)
}

// 保存配置文件
func SaveConfig() error {
	data, err := yaml.Marshal(&Config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, data, 0644)
}
