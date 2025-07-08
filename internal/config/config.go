package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"noverna.de/m/v2/internal/logger"
)

type Config struct {
	Server   Server   `toml:"server"`
	Uploads  Uploads  `toml:"uploads"`
	Security Security `toml:"security"`
	Debug    Debug    `toml:"debug"`
}

type Server struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	LogLevel string `toml:"log_level"`
	DataDir  string `toml:"data_dir"`
	TempDir  string `toml:"temp_dir"`
}

type Uploads struct {
	MAX_FILE_SIZE int      `toml:"max_file_size_mb"`
	AllowedTypes  []string `toml:"allowed_types"`
}

type Security struct {
	TokenRequired      bool   `toml:"token_required"`
	ApiKey             string `toml:"api_key"`
	RateLimitPerMinute int    `toml:"rate_limit_per_minute"`
}

type Debug struct {
	Enabled bool `toml:"enabled"`
}

var config *Config

func GetConfig() *Config {
	if config == nil {
		config = &Config{}
		
		f := "noverna.toml"
		if _, err := os.Stat(f); err != nil {
			f = "assets/noverna.toml"
		}
		
		if _, err := toml.DecodeFile(f, config); err != nil {
			logger.Error("Error while loading config file", map[string]any{"error": err})
			return nil
		}
		
		if config.Server.LogLevel == "" {
			config.Server.LogLevel = "info"
		}

		switch config.Server.LogLevel {
		case "debug":
			logger.SetLevel(logger.DEBUG)
		case "info":
			logger.SetLevel(logger.INFO)
		case "warn":
			logger.SetLevel(logger.WARN)
		case "error":
			logger.SetLevel(logger.ERROR)
		case "fatal": 
			logger.SetLevel(logger.FATAL)
		}
	}
	return config
}

func Init() error {
	config = &Config{}
	
	f := "noverna.toml"
	if _, err := os.Stat(f); err != nil {
		f = "assets/noverna.toml"
	}
	
	if _, err := toml.DecodeFile(f, config); err != nil {
		return err
	}
	
	// If no LogLevel is set, set it to "info"
	if config.Server.LogLevel == "" {
		config.Server.LogLevel = "info"
	}
	
	// Log Level set // TODO: Refactor
	switch config.Server.LogLevel {
	case "debug":
		logger.SetLevel(logger.DEBUG)
	case "info":
		logger.SetLevel(logger.INFO)
	case "warn":
		logger.SetLevel(logger.WARN)
	case "error":
		logger.SetLevel(logger.ERROR)
	case "fatal": 
		logger.SetLevel(logger.FATAL)
	}
	
	return nil
}