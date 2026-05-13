package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type MySQLConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type Config struct {
	MySQL      MySQLConfig `json:"mysql"`
	JWTSecret  string      `json:"jwt_secret"`
	AIApiKey   string      `json:"ai_api_key"`
	AIApiURL   string      `json:"ai_api_url"`
	AIModel    string      `json:"ai_model"`
	ServerPort string      `json:"server_port"`
	UseMockAI  bool        `json:"use_mock_ai"`
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, c.MySQL.Database)
}

func Load() *Config {
	cfg := &Config{
		MySQL: MySQLConfig{
			Host:     "115.190.57.118",
			Port:     3306,
			User:     "root",
			Password: "4ay1nkal3u8ed77y",
			Database: "Do-Practice-Questions",
		},
		JWTSecret:  "your-secret-key-change-in-production",
		AIApiKey:   "",
		AIApiURL:   "https://api.deepseek.com",
		AIModel:    "deepseek-chat",
		ServerPort: "8080",
		UseMockAI:  true,
	}

	if data, err := os.ReadFile("config.json"); err == nil {
		var fileCfg Config
		if json.Unmarshal(data, &fileCfg) == nil {
			if fileCfg.MySQL.Host != "" {
				cfg.MySQL.Host = fileCfg.MySQL.Host
			}
			if fileCfg.MySQL.Port != 0 {
				cfg.MySQL.Port = fileCfg.MySQL.Port
			}
			if fileCfg.MySQL.User != "" {
				cfg.MySQL.User = fileCfg.MySQL.User
			}
			if fileCfg.MySQL.Password != "" {
				cfg.MySQL.Password = fileCfg.MySQL.Password
			}
			if fileCfg.MySQL.Database != "" {
				cfg.MySQL.Database = fileCfg.MySQL.Database
			}
			if fileCfg.JWTSecret != "" {
				cfg.JWTSecret = fileCfg.JWTSecret
			}
			if fileCfg.AIApiKey != "" {
				cfg.AIApiKey = fileCfg.AIApiKey
			}
			if fileCfg.AIApiURL != "" {
				cfg.AIApiURL = fileCfg.AIApiURL
			}
			if fileCfg.AIModel != "" {
				cfg.AIModel = fileCfg.AIModel
			}
			if fileCfg.ServerPort != "" {
				cfg.ServerPort = fileCfg.ServerPort
			}
			cfg.UseMockAI = fileCfg.UseMockAI
		}
	}

	if v := os.Getenv("MYSQL_HOST"); v != "" {
		cfg.MySQL.Host = v
	}
	if v := os.Getenv("MYSQL_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.MySQL.Port)
	}
	if v := os.Getenv("MYSQL_USER"); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		cfg.MySQL.Password = v
	}
	if v := os.Getenv("MYSQL_DATABASE"); v != "" {
		cfg.MySQL.Database = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("AI_API_KEY"); v != "" {
		cfg.AIApiKey = v
	}
	if v := os.Getenv("AI_API_URL"); v != "" {
		cfg.AIApiURL = v
	}
	if v := os.Getenv("AI_MODEL"); v != "" {
		cfg.AIModel = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.ServerPort = v
	}
	if v := os.Getenv("USE_MOCK_AI"); v == "true" {
		cfg.UseMockAI = true
	} else if v := os.Getenv("USE_MOCK_AI"); v == "false" {
		cfg.UseMockAI = false
	}

	return cfg
}
