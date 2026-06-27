package app

import (
	"database/sql"
	"os"
	"path/filepath"

	"tracelog/internal/appsettings"
	"tracelog/internal/config"
	"tracelog/internal/jira"
	"tracelog/internal/service"
	"tracelog/internal/store"
)

type Runtime struct {
	Service   *service.Service
	UploadDir string
}

func NewRuntime(cfg config.Config, database *sql.DB) (*Runtime, error) {
	repo := store.New(database)
	jiraClient := jira.New(jira.Config{BaseURL: cfg.JiraBaseURL, Email: cfg.JiraEmail, APIToken: cfg.JiraAPIToken})
	settingsStore := appsettings.New(filepath.Join(cfg.DataDir, "tracelog-settings.json"), appsettings.Settings{
		Jira: appsettings.JiraSettings{
			BaseURL:  cfg.JiraBaseURL,
			Email:    cfg.JiraEmail,
			APIToken: cfg.JiraAPIToken,
		},
		AI: appsettings.AISettings{Provider: "openai"},
		OpenAI: appsettings.ProviderSettings{
			BaseURL: cfg.OpenAIBaseURL,
			APIKey:  cfg.OpenAIAPIKey,
			Model:   cfg.OpenAIModel,
		},
		DeepSeek: appsettings.ProviderSettings{
			BaseURL: cfg.DeepSeekBaseURL,
			APIKey:  cfg.DeepSeekAPIKey,
			Model:   cfg.DeepSeekModel,
		},
	})
	uploadDir := filepath.Join(cfg.DataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, err
	}

	svc := service.New(repo, jiraClient, settingsStore, uploadDir)
	svc.SetLocation(cfg.Location)
	return &Runtime{Service: svc, UploadDir: uploadDir}, nil
}
