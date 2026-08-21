package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	RedisAddr  string
	LoggerAddr string
	LoggerPort string
	GRPCPort   string
}

func Load() *Config {
	loadDotEnv(".env")

	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "task_1_hamrah"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		LoggerAddr: getEnv("LOGGER_ADDR", "localhost:9090"),
		LoggerPort: getEnv("LOGGER_GRPC_PORT", "9090"),
		GRPCPort:   getEnv("GRPC_PORT", "8080"),
	}
}

func (c *Config) DBDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
