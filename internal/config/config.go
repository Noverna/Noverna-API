package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	"noverna.de/m/v2/internal/logger"
)

//! IMPORTANT - BETTER ERROR HANDLING NEEDED

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

type Advanced struct {
	CacheEndpoint string `toml:"cache_endpoint"`
	CacheNodes    []string `toml:"cache_nodes"`
	CDNEndpoint string `toml:"cdn_endpoint"`
	CDNNodes    []string `toml:"cdn_nodes"`
}

type Debug struct {
	Enabled bool `toml:"enabled"`
}

var (
	config *Config
	log *logger.Logger
	once   sync.Once
)

func GetConfig() *Config {
	once.Do(func() {
		if err := Init(); err != nil {
			logger.Error("Failed to initialize config", map[string]any{"error": err})
			// Fallback zu Default-Config
			config = getDefaultConfig()
		}
	})
	return config
}

func Init() error {
	cfg := &Config{}
	// Create Logger
	log = logger.NewLogger()
	log.WithField("SERVICE", "API")
	log.WithField("PART", "config")
	
	configFile, err := findConfigFile()
	if err != nil {
		log.Error("config file not found", map[string]any{"error": err})
		return nil
	}
	
	if _, err := toml.DecodeFile(configFile, cfg); err != nil {
		log.Error("failed to decode config file", map[string]any{"error": err})
		return nil
	}
	
	// Validierung der Konfiguration
	if err := validateConfig(cfg); err != nil {
		log.Error("invalid config", map[string]any{"error": err})
		return nil
	}
	
	// Defaults setzen
	applyDefaults(cfg)
	
	// Logger Level setzen
	if err := setLogLevel(cfg.Server.LogLevel); err != nil {
		log.Error("failed to set log level", map[string]any{"error": err})
		return nil
	}
	
	config = cfg
	return nil
}

func findConfigFile() (string, error) {
	candidates := []string{
		"noverna.toml",
		"assets/noverna.toml",
		"config/noverna.toml",
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	log.Error("config file not found", map[string]any{"error": "config file not found"})
	return "", nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		log.Error("invalid port", map[string]any{"error": "port must be between 1 and 65535"})
		return nil
	}
	
	if cfg.Uploads.MAX_FILE_SIZE <= 0 {
	  log.Error("invalid max file size", map[string]any{"error": "max file size must be greater than 0"})
		return nil
	}
	
	if cfg.Security.RateLimitPerMinute < 0 {
		log.Error("invalid rate limit per minute", map[string]any{"error": "rate limit per minute must be greater than 0"})
		return nil
	}
	
	return nil
}

// applyDefaults sets default values for missing configuration
func applyDefaults(cfg *Config) {
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "info"
	}
	
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	
	if cfg.Uploads.MAX_FILE_SIZE == 0 {
		cfg.Uploads.MAX_FILE_SIZE = 10
	}
	
	if cfg.Security.RateLimitPerMinute == 0 {
		cfg.Security.RateLimitPerMinute = 60
	}
}

// setLogLevel sets the logger level based on the config
func setLogLevel(level string) error {
	switch level {
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
	default:
		return fmt.Errorf("invalid log level: %s", level)
	}
	return nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *Config {
	return &Config{
		Server: Server{
			Host:     "localhost",
			Port:     8080,
			LogLevel: "info",
			DataDir:  "./data",
			TempDir:  "./tmp",
		},
		Uploads: Uploads{
			MAX_FILE_SIZE: 10,
			AllowedTypes:  []string{"image/jpeg", "image/png", "text/plain"},
		},
		Security: Security{
			TokenRequired:      false,
			RateLimitPerMinute: 60,
		},
		Debug: Debug{
			Enabled: false,
		},
	}
}

// IsDebugEnabled returns true if debug mode is enabled
func IsDebugEnabled() bool {
	cfg := GetConfig()
	return cfg.Debug.Enabled
}

// GetServerAddress returns the formatted server address
func GetServerAddress() string {
	cfg := GetConfig()
  return fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
}