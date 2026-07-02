package service

import (
	"context"
	"strings"

	"tracelog/internal/appsettings"
)

func (s *SettingsService) GetSettings(ctx context.Context) (AppSettings, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return AppSettings{}, err
	}
	return toServiceSettings(appsettings.Public(settings)), nil
}

func (s *SettingsService) UpdateSettings(ctx context.Context, input AppSettings) (AppSettings, error) {
	nextSettings := fromServiceSettings(input)
	existing, err := s.settings.Load()
	if err != nil {
		return AppSettings{}, err
	}
	nextSettings = mergeSettingsSecrets(nextSettings, existing)
	tempoToken := strings.TrimSpace(input.Tempo.APIToken)
	if tempoToken != "" && tempoToken != existing.Tempo.APIToken {
		accountID, err := s.fetchTempoAuthorAccountID(ctx, nextSettings)
		if err != nil {
			return AppSettings{}, err
		}
		nextSettings.Tempo.AuthorAccountID = accountID
	}
	settings, err := s.settings.Save(nextSettings)
	if err != nil {
		return AppSettings{}, err
	}
	return toServiceSettings(appsettings.Public(settings)), nil
}

func toServiceSettings(settings appsettings.Settings) AppSettings {
	return AppSettings{
		Jira: JiraSettings{
			BaseURL:     settings.Jira.BaseURL,
			Email:       settings.Jira.Email,
			APIToken:    settings.Jira.APIToken,
			HasAPIToken: settings.Jira.HasAPIToken,
		},
		Tempo: TempoSettings{
			BaseURL:         settings.Tempo.BaseURL,
			APIToken:        settings.Tempo.APIToken,
			HasAPIToken:     settings.Tempo.HasAPIToken,
			AuthorAccountID: settings.Tempo.AuthorAccountID,
		},
		AI: AISettings{Provider: settings.AI.Provider},
		OpenAI: ProviderSettings{
			BaseURL:   settings.OpenAI.BaseURL,
			Model:     settings.OpenAI.Model,
			APIKey:    settings.OpenAI.APIKey,
			HasAPIKey: settings.OpenAI.HasAPIKey,
		},
		DeepSeek: ProviderSettings{
			BaseURL:   settings.DeepSeek.BaseURL,
			Model:     settings.DeepSeek.Model,
			APIKey:    settings.DeepSeek.APIKey,
			HasAPIKey: settings.DeepSeek.HasAPIKey,
		},
		Prompts: PromptSettings{
			IssueSummary:  settings.Prompts.IssueSummary,
			WeeklySummary: settings.Prompts.WeeklySummary,
		},
	}
}

func fromServiceSettings(settings AppSettings) appsettings.Settings {
	return appsettings.Settings{
		Jira: appsettings.JiraSettings{
			BaseURL:  settings.Jira.BaseURL,
			Email:    settings.Jira.Email,
			APIToken: settings.Jira.APIToken,
		},
		Tempo: appsettings.TempoSettings{
			BaseURL:         settings.Tempo.BaseURL,
			APIToken:        settings.Tempo.APIToken,
			AuthorAccountID: settings.Tempo.AuthorAccountID,
		},
		AI: appsettings.AISettings{Provider: settings.AI.Provider},
		OpenAI: appsettings.ProviderSettings{
			BaseURL: settings.OpenAI.BaseURL,
			Model:   settings.OpenAI.Model,
			APIKey:  settings.OpenAI.APIKey,
		},
		DeepSeek: appsettings.ProviderSettings{
			BaseURL: settings.DeepSeek.BaseURL,
			Model:   settings.DeepSeek.Model,
			APIKey:  settings.DeepSeek.APIKey,
		},
		Prompts: appsettings.PromptSettings{
			IssueSummary:  settings.Prompts.IssueSummary,
			WeeklySummary: settings.Prompts.WeeklySummary,
		},
	}
}

func mergeSettingsSecrets(settings appsettings.Settings, existing appsettings.Settings) appsettings.Settings {
	if settings.Jira.APIToken == "" {
		settings.Jira.APIToken = existing.Jira.APIToken
	}
	if settings.Tempo.APIToken == "" {
		settings.Tempo.APIToken = existing.Tempo.APIToken
	}
	if settings.Tempo.AuthorAccountID == "" {
		settings.Tempo.AuthorAccountID = existing.Tempo.AuthorAccountID
	}
	if settings.OpenAI.APIKey == "" {
		settings.OpenAI.APIKey = existing.OpenAI.APIKey
	}
	if settings.DeepSeek.APIKey == "" {
		settings.DeepSeek.APIKey = existing.DeepSeek.APIKey
	}
	return settings
}
