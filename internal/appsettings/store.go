package appsettings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Settings struct {
	Jira     JiraSettings     `json:"jira"`
	AI       AISettings       `json:"ai"`
	OpenAI   ProviderSettings `json:"openai"`
	DeepSeek ProviderSettings `json:"deepseek"`
	Prompts  PromptSettings   `json:"prompts"`
}

type JiraSettings struct {
	BaseURL     string `json:"base_url"`
	Email       string `json:"email"`
	APIToken    string `json:"api_token,omitempty"`
	HasAPIToken bool   `json:"has_api_token"`
}

type AISettings struct {
	Provider string `json:"provider"`
}

type ProviderSettings struct {
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
	APIKey    string `json:"api_key,omitempty"`
	HasAPIKey bool   `json:"has_api_key"`
}

type PromptSettings struct {
	IssueSummary  string `json:"issue_summary"`
	WeeklySummary string `json:"weekly_summary"`
}

const DefaultIssueSummaryPrompt = "你是一个资深工程师助手。请根据 Jira 信息和阶段评论，总结该 issue 当前的工作简述。输出中文，控制在 2-4 句话，直接给结论，不要写标题、列表或寒暄。"

const DefaultWeeklySummaryPrompt = "你是一个资深工程师助手。请根据本周 Jira、评论、临时需求和 TODO，生成中文周总结。结构清晰，包含本周完成、关键进展、风险阻塞、下周建议。直接输出 Markdown，不要寒暄。"

type Store struct {
	path     string
	defaults Settings
	mu       sync.Mutex
}

func New(path string, defaults Settings) *Store {
	return &Store{path: path, defaults: normalize(defaults)}
}

func (s *Store) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) Save(input Settings) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	merged := normalize(input)
	if merged.Jira.APIToken == "" {
		merged.Jira.APIToken = existing.Jira.APIToken
	}
	if merged.OpenAI.APIKey == "" {
		merged.OpenAI.APIKey = existing.OpenAI.APIKey
	}
	if merged.DeepSeek.APIKey == "" {
		merged.DeepSeek.APIKey = existing.DeepSeek.APIKey
	}
	merged = normalize(merged)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Settings{}, err
	}
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return Settings{}, err
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return Settings{}, err
	}
	return merged, nil
}

func Public(settings Settings) Settings {
	settings.Jira.HasAPIToken = settings.Jira.APIToken != ""
	settings.Jira.APIToken = ""
	settings.OpenAI.HasAPIKey = settings.OpenAI.APIKey != ""
	settings.OpenAI.APIKey = ""
	settings.DeepSeek.HasAPIKey = settings.DeepSeek.APIKey != ""
	settings.DeepSeek.APIKey = ""
	return settings
}

func (s *Store) loadLocked() (Settings, error) {
	settings := s.defaults
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return normalize(settings), nil
	}
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	return normalize(settings), nil
}

func normalize(settings Settings) Settings {
	settings.Jira.BaseURL = strings.TrimRight(strings.TrimSpace(settings.Jira.BaseURL), "/")
	settings.Jira.Email = strings.TrimSpace(settings.Jira.Email)
	settings.Jira.APIToken = strings.TrimSpace(settings.Jira.APIToken)
	settings.AI.Provider = strings.TrimSpace(strings.ToLower(settings.AI.Provider))
	if settings.AI.Provider == "" {
		settings.AI.Provider = "openai"
	}
	settings.OpenAI.BaseURL = strings.TrimRight(strings.TrimSpace(settings.OpenAI.BaseURL), "/")
	if settings.OpenAI.BaseURL == "" {
		settings.OpenAI.BaseURL = "https://api.openai.com/v1"
	}
	settings.OpenAI.Model = strings.TrimSpace(settings.OpenAI.Model)
	if settings.OpenAI.Model == "" {
		settings.OpenAI.Model = "gpt-4.1-mini"
	}
	settings.OpenAI.APIKey = strings.TrimSpace(settings.OpenAI.APIKey)
	settings.DeepSeek.BaseURL = strings.TrimRight(strings.TrimSpace(settings.DeepSeek.BaseURL), "/")
	if settings.DeepSeek.BaseURL == "" {
		settings.DeepSeek.BaseURL = "https://api.deepseek.com"
	}
	settings.DeepSeek.Model = strings.TrimSpace(settings.DeepSeek.Model)
	if settings.DeepSeek.Model == "" {
		settings.DeepSeek.Model = "deepseek-v4-flash"
	}
	settings.DeepSeek.APIKey = strings.TrimSpace(settings.DeepSeek.APIKey)
	settings.Prompts.IssueSummary = strings.TrimSpace(settings.Prompts.IssueSummary)
	if settings.Prompts.IssueSummary == "" {
		settings.Prompts.IssueSummary = DefaultIssueSummaryPrompt
	}
	settings.Prompts.WeeklySummary = strings.TrimSpace(settings.Prompts.WeeklySummary)
	if settings.Prompts.WeeklySummary == "" {
		settings.Prompts.WeeklySummary = DefaultWeeklySummaryPrompt
	}
	return settings
}
