package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	AppEnv                   string
	UploadDir                string
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
	MailInsecureSkipVerify   bool
	ResetTokenExpiresMinutes int
	AgentCommissionPercent   float64
	DefaultAdminEmail        string
	DefaultAdminPassword     string
	DefaultSalesEmail        string
	DefaultSalesPassword     string
	DatabaseURL              string
	SSOCodeExpiresSeconds    int
	SSOClientRedirects       map[string]string
	CORSAllowedOrigins       []string
	GoogleClientID           string
	PancakeWebhookSecret     string
	AnalyticsTimezone        string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppPort:                  getEnv("APP_PORT", getEnv("PORT", "8080")),
		AppEnv:                   getEnv("APP_ENV", "development"),
		UploadDir:                getEnv("UPLOAD_DIR", "uploads"),
		DBHost:                   getEnv("DB_HOST", getEnv("MYSQLHOST", getEnv("MYSQL_HOST", "127.0.0.1"))),
		DBPort:                   getEnv("DB_PORT", getEnv("MYSQLPORT", getEnv("MYSQL_PORT", "3306"))),
		DBUser:                   getEnv("DB_USER", getEnv("MYSQLUSER", getEnv("MYSQL_USER", "root"))),
		DBPassword:               getEnv("DB_PASSWORD", getEnv("MYSQLPASSWORD", getEnv("MYSQL_PASSWORD", ""))),
		DBName:                   getEnv("DB_NAME", getEnv("MYSQLDATABASE", getEnv("MYSQL_DATABASE", "begmt2"))),
		JWTSecret:                getEnv("JWT_SECRET", "change-this-secret"),
		JWTExpiresHours:          getEnvAsInt("JWT_EXPIRES_HOURS", 24),
		MailHost:                 getEnv("MAIL_HOST", "smtp.gmail.com"),
		MailPort:                 getEnvAsInt("MAIL_PORT", 587),
		MailUsername:             getEnv("MAIL_USERNAME", ""),
		MailPassword:             getEnv("MAIL_PASSWORD", ""),
		MailFromName:             getEnv("MAIL_FROM_NAME", "BeGMT2"),
		MailInsecureSkipVerify:   getEnvAsBool("MAIL_INSECURE_SKIP_VERIFY", false),
		ResetTokenExpiresMinutes: getEnvAsInt("RESET_TOKEN_EXPIRES_MINUTES", 15),
		AgentCommissionPercent:   getEnvAsFloat("AGENT_COMMISSION_PERCENT", 5),
		DefaultAdminEmail:        getEnv("DEFAULT_ADMIN_EMAIL", "superadmin@example.com"),
		DefaultAdminPassword:     getEnv("DEFAULT_ADMIN_PASSWORD", "password123"),
		DefaultSalesEmail:        getEnv("DEFAULT_SALES_EMAIL", "sales@example.com"),
		DefaultSalesPassword:     getEnv("DEFAULT_SALES_PASSWORD", "password123"),
		DatabaseURL:              getEnv("DATABASE_URL", getEnv("MYSQL_URL", getEnv("MYSQL_PUBLIC_URL", ""))),
		SSOCodeExpiresSeconds:    getEnvAsInt("SSO_CODE_EXPIRES_SECONDS", 60),
		SSOClientRedirects:       getEnvAsMap("SSO_CLIENTS", ""),
		CORSAllowedOrigins:       getEnvAsList("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:5000"),
		GoogleClientID:           getEnv("GOOGLE_CLIENT_ID", ""),
		PancakeWebhookSecret:     getEnv("PANCAKE_WEBHOOK_SECRET", ""),
		AnalyticsTimezone:        getEnv("ANALYTICS_TIMEZONE", "Asia/Jakarta"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return cleanEnvValue(value)
	}
	return fallback
}

func cleanEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
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

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
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

func MySQLDSNFromURL(rawURL string) (string, bool) {
	if rawURL == "" {
		return "", false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "mysql" || parsed.User == nil || parsed.Host == "" {
		return "", false
	}

	password, _ := parsed.User.Password()
	database := strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		return "", false
	}

	query := parsed.Query()
	if query.Get("charset") == "" {
		query.Set("charset", "utf8mb4")
	}
	if query.Get("parseTime") == "" {
		query.Set("parseTime", "True")
	}
	if query.Get("loc") == "" {
		query.Set("loc", "Local")
	}

	return parsed.User.Username() + ":" + password + "@tcp(" + parsed.Host + ")/" + database + "?" + query.Encode(), true
}
