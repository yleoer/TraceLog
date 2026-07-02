package service

import (
	"context"
	"time"

	"tracelog/internal/appsettings"
	"tracelog/internal/jira"
)

type serviceCore struct {
	repo      Repository
	jira      *jira.Client
	settings  *appsettings.Store
	uploadDir string
	loc       *time.Location
}

// Service preserves the Wails-facing API while delegating each domain to a
// smaller service with its own receiver and collaborator set.
type Service struct {
	*IssueService
	*TempTaskService
	*WeeklyService
	*DashboardService
	*DayService
	*SettingsService
	*TimeService
	*UploadService
	*IntegrationService
	*ExportService
	*SearchService

	core *serviceCore
}

type IssueService struct {
	*serviceCore
}

type TempTaskService struct {
	*serviceCore
}

type WeeklyService struct {
	*serviceCore
}

type DashboardService struct {
	*serviceCore

	issues    *IssueService
	tempTasks *TempTaskService
	weekly    *WeeklyService
}

type DayService struct {
	*serviceCore
}

type SettingsService struct {
	*serviceCore
}

type TimeService struct {
	*serviceCore
}

type UploadService struct {
	*serviceCore
}

type IntegrationService struct {
	*serviceCore

	issues *IssueService
}

type ExportService struct {
	*serviceCore

	issues    *IssueService
	tempTasks *TempTaskService
	weekly    *WeeklyService
}

type SearchService struct {
	*serviceCore
}

func (s *serviceCore) withRepository(ctx context.Context, fn func(Repository) error) error {
	if txRepo, ok := s.repo.(transactionRepository); ok {
		return txRepo.WithTransaction(ctx, fn)
	}
	return fn(s.repo)
}

func New(repo Repository, jiraClient *jira.Client, settingsStore *appsettings.Store, uploadDir ...string) *Service {
	dir := ""
	if len(uploadDir) > 0 {
		dir = uploadDir[0]
	}
	core := &serviceCore{repo: repo, jira: jiraClient, settings: settingsStore, uploadDir: dir, loc: time.UTC}
	issues := &IssueService{serviceCore: core}
	tempTasks := &TempTaskService{serviceCore: core}
	weekly := &WeeklyService{serviceCore: core}
	return &Service{
		IssueService:       issues,
		TempTaskService:    tempTasks,
		WeeklyService:      weekly,
		DashboardService:   &DashboardService{serviceCore: core, issues: issues, tempTasks: tempTasks, weekly: weekly},
		DayService:         &DayService{serviceCore: core},
		SettingsService:    &SettingsService{serviceCore: core},
		TimeService:        &TimeService{serviceCore: core},
		UploadService:      &UploadService{serviceCore: core},
		IntegrationService: &IntegrationService{serviceCore: core, issues: issues},
		ExportService:      &ExportService{serviceCore: core, issues: issues, tempTasks: tempTasks, weekly: weekly},
		SearchService:      &SearchService{serviceCore: core},
		core:               core,
	}
}

// SetLocation overrides the timezone used for day and week boundary calculations.
// Timestamps stay stored in UTC; only bucketing/range math uses this location.
func (s *Service) SetLocation(loc *time.Location) {
	if loc != nil {
		s.core.loc = loc
	}
}
