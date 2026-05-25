// Package config loads runtime configuration from environment variables.
//
// .env files are read on startup (if present); explicit environment variables
// always win. Missing values fall back to sensible development defaults, with
// a final validation pass that fails fast on anything critical.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	LiveKit     LiveKitConfig
	Log         LogConfig
}

type ServerConfig struct {
	Port         int
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxConnections  int
	MaxIdleTime     time.Duration
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	Issuer        string
}

type LiveKitConfig struct {
	URL         string // Internal URL used by the backend to call LiveKit.
	FrontendURL string // Public URL handed to browser clients.
	APIKey      string
	APISecret   string
	HostIP      string // Host IP advertised to LiveKit for ICE candidates.
	Port        string // External LiveKit port exposed to clients.
}

type LogConfig struct {
	Level string
}

// Load reads configuration from the environment (and optional .env file)
// and returns a validated Config or an error.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		Server: ServerConfig{
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			DSN:             getEnv("DATABASE_DSN", "postgres://vibemeet:vibemeetpass@localhost:5432/vibemeet?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DATABASE_MAX_CONNECTIONS", 25),
			MaxIdleTime:     getEnvAsDuration("DATABASE_MAX_IDLE_TIME", 5*time.Minute),
			ConnMaxLifetime: getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", 1*time.Hour),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
			AccessTTL:     getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    getEnvAsDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
			Issuer:        getEnv("JWT_ISSUER", "vibemeet"),
		},
		LiveKit: LiveKitConfig{
			URL:         getEnv("LIVEKIT_URL", "ws://localhost:7880"),
			FrontendURL: getEnv("LIVEKIT_FRONTEND_URL", ""),
			APIKey:      getEnv("LIVEKIT_API_KEY", "devkey"),
			APISecret:   getEnv("LIVEKIT_API_SECRET", "devsecret"),
			HostIP:      getEnv("HOST_IP", GetLocalIP()),
			Port:        getEnv("LIVEKIT_PORT", "7880"),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.JWT.AccessSecret == "" || c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT secrets must be set")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN must be set")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// GetLocalIP returns the first non-loopback IPv4 address of the host, preferring
// private LAN ranges (192.168.*, 10.*, 172.*). Used as a fallback when HOST_IP
// is not set in the environment — convenient for LAN-only development.
func GetLocalIP() string {
	if ip := os.Getenv("HOST_IP"); ip != "" {
		return ip
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}

	priorityPrefixes := []string{"192.168.", "10.", "172."}
	var fallbackIP string

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		ip := ipnet.IP.String()

		for _, prefix := range priorityPrefixes {
			if strings.HasPrefix(ip, prefix) {
				return ip
			}
		}
		if fallbackIP == "" {
			fallbackIP = ip
		}
	}

	if fallbackIP != "" {
		return fallbackIP
	}
	return "localhost"
}
