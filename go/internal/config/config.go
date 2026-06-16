package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBName     string
	DBUsername string
	DBPassword string

	RabbitMQHost     string
	RabbitMQPort     string
	RabbitMQVHost    string
	RabbitMQUsername string
	RabbitMQPassword string
	RabbitMQUseTLS   bool

	RateLimitPerMinute int
	RateLimitPerSecond int
	RateLimitBurst     int

	OAuth2Enabled   bool
	OAuth2IssuerURI string
	OAuth2JWKSetURI string

	VaultEnabled bool
}

func Load() Config {
	return Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "orders"),
		DBUsername: getEnv("DB_USERNAME", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),

		RabbitMQHost:     getEnv("RABBITMQ_HOST", "localhost"),
		RabbitMQPort:     getEnv("RABBITMQ_PORT", "5672"),
		RabbitMQVHost:    getEnv("RABBITMQ_VHOST", "/"),
		RabbitMQUsername: getEnv("RABBITMQ_USERNAME", "guest"),
		RabbitMQPassword: getEnv("RABBITMQ_PASSWORD", "guest"),
		RabbitMQUseTLS:   getEnvAsBool("RABBITMQ_USE_TLS", false),

		RateLimitPerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 100),
		RateLimitPerSecond: getEnvAsInt("RATE_LIMIT_PER_SECOND", 20),
		RateLimitBurst:     getEnvAsInt("RATE_LIMIT_BURST", 50),

		OAuth2Enabled:   getEnvAsBool("OAUTH2_ENABLED", false),
		OAuth2IssuerURI: getEnv("OAUTH2_ISSUER_URI", ""),
		OAuth2JWKSetURI: getEnv("OAUTH2_JWK_SET_URI", ""),

		VaultEnabled: getEnvAsBool("VAULT_ENABLED", false),
	}
}

func (c Config) DatabaseURI() string {
	return (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUsername, c.DBPassword),
		Host:   fmt.Sprintf("%s:%s", c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
	}).String() + "?sslmode=disable"
}

func (c Config) RabbitMQURI() string {
	scheme := "amqp"
	if c.RabbitMQUseTLS {
		scheme = "amqps"
	}
	escapedVHost := url.PathEscape(c.RabbitMQVHost)
	if escapedVHost == "" {
		escapedVHost = "%2F"
	}
	return (&url.URL{
		Scheme: scheme,
		User:   url.UserPassword(c.RabbitMQUsername, c.RabbitMQPassword),
		Host:   fmt.Sprintf("%s:%s", c.RabbitMQHost, c.RabbitMQPort),
		Path:   "/" + escapedVHost,
	}).String()
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
