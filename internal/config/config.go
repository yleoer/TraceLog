package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DataDir         string
	DatabasePath    string
	JiraBaseURL     string
	JiraEmail       string
	JiraAPIToken    string
	OpenAIBaseURL   string
	OpenAIAPIKey    string
	OpenAIModel     string
	DeepSeekBaseURL string
	DeepSeekAPIKey  string
	DeepSeekModel   string
	Timezone        string
	Location        *time.Location
}

func Load() Config {
	dataDir := getEnv("APP_DATA_DIR", desktopDataDir())
	return loadWithDataDir(dataDir)
}

func loadWithDataDir(dataDir string) Config {
	timezone := getEnv("APP_TIMEZONE", "Asia/Shanghai")
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}

	return Config{
		DataDir:         dataDir,
		DatabasePath:    getEnv("DATABASE_PATH", filepath.Join(dataDir, "tracelog.db")),
		JiraBaseURL:     getEnv("JIRA_BASE_URL", ""),
		JiraEmail:       getEnv("JIRA_EMAIL", ""),
		JiraAPIToken:    getEnv("JIRA_API_TOKEN", ""),
		OpenAIBaseURL:   getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:     getEnv("OPENAI_MODEL", "gpt-4.1-mini"),
		DeepSeekBaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekAPIKey:  getEnv("DEEPSEEK_API_KEY", ""),
		DeepSeekModel:   getEnv("DEEPSEEK_MODEL", "deepseek-v4-flash"),
		Timezone:        timezone,
		Location:        location,
	}
}

func desktopDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "TraceLog")
	}
	return filepath.Join(".", ".tracelog")
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
