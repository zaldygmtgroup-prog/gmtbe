package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	AppEnv                   string
	DBHost                   string
	DBPort                   string
	DBUser                   string
	DBPassword               string
	DBName                   string
	JWTSecret                string
	JWTExpiresHours          int
	MailHost                 string
	MailPort                 int
	MailUsername             string
	MailPassword             string
	MailFromName             string
	ResetTokenExpiresMinutes int
	AgentCommissionPercent   float64
	DefaultAdminEmail        string
	DefaultAdminPassword     string
	DefaultSalesEmail        string
	DefaultSalesPassword     string
	SSOCodeExpiresSeconds    int
	SSOClientRedirects       map[string]string
	CORSAllowedOrigins       []string
	MidtransMerchantID       string
	MidtransClientKey        string
	MidtransServerKey        string
	MidtransEnvironment      string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppPort:                  getEnv("APP_PORT", getEnv("PORT", "8080")),
		AppEnv:                   getEnv("APP_ENV", "development"),
		DBHost:                   getEnv("DB_HOST", getEnv("MYSQLHOST", "127.0.0.1")),
		DBPort:                   getEnv("DB_PORT", getEnv("MYSQLPORT", "3306")),
		DBUser:                   getEnv("DB_USER", getEnv("MYSQLUSER", "root")),
		DBPassword:               getEnv("DB_PASSWORD", getEnv("MYSQLPASSWORD", "")),
		DBName:                   getEnv("DB_NAME", getEnv("MYSQLDATABASE", "begmt2")),
		JWTSecret:                getEnv("JWT_SECRET", "change-this-secret"),
		JWTExpiresHours:          getEnvAsInt("JWT_EXPIRES_HOURS", 24),
		MailHost:                 getEnv("MAIL_HOST", "smtp.gmail.com"),
		MailPort:                 getEnvAsInt("MAIL_PORT", 587),
		MailUsername:             getEnv("MAIL_USERNAME", ""),
		MailPassword:             getEnv("MAIL_PASSWORD", ""),
		MailFromName:             getEnv("MAIL_FROM_NAME", "BeGMT2"),
		ResetTokenExpiresMinutes: getEnvAsInt("RESET_TOKEN_EXPIRES_MINUTES", 15),
		AgentCommissionPercent:   getEnvAsFloat("AGENT_COMMISSION_PERCENT", 5),
		DefaultAdminEmail:        getEnv("DEFAULT_ADMIN_EMAIL", "superadmin@example.com"),
		DefaultAdminPassword:     getEnv("DEFAULT_ADMIN_PASSWORD", "password123"),
		DefaultSalesEmail:        getEnv("DEFAULT_SALES_EMAIL", "sales@example.com"),
		DefaultSalesPassword:     getEnv("DEFAULT_SALES_PASSWORD", "password123"),
		SSOCodeExpiresSeconds:    getEnvAsInt("SSO_CODE_EXPIRES_SECONDS", 60),
		SSOClientRedirects:       getEnvAsMap("SSO_CLIENTS", ""),
		CORSAllowedOrigins:       getEnvAsList("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:5000"),
		MidtransMerchantID:       getEnv("MIDTRANS_MERCHANT_ID", ""),
		MidtransClientKey:        getEnv("MIDTRANS_CLIENT_KEY", ""),
		MidtransServerKey:        getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransEnvironment:      getEnv("MIDTRANS_ENVIRONMENT", "sandbox"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvAsFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvAsMap(key string, fallback string) map[string]string {
	value := getEnv(key, fallback)
	result := make(map[string]string)
	if value == "" {
		return result
	}

	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		redirectURI := strings.TrimSpace(parts[1])
		if name != "" && redirectURI != "" {
			result[name] = redirectURI
		}
	}

	return result
}

func getEnvAsList(key string, fallback string) []string {
	value := getEnv(key, fallback)
	result := make([]string, 0)
	if value == "" {
		return result
	}

	items := strings.Split(value, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}
