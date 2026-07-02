package service

import (
	"context"
	"time"

	"tracelog/internal/tempo"
)

const maxLogTimeDays = 31
const workdayStartTime = "08:00:00"

var timeWorkItems = []TimeWorkItem{
	{Key: "CORETIME-47", Label: "法定节假日"},
	{Key: "CORETIME-11", Label: "个人请假"},
	{Key: "CORETIME-80", Label: "内部SMF需求"},
	{Key: "CORETIME-36", Label: "SE的SMF需求"},
	{Key: "CORETIME-16", Label: "面对客户的文档工作"},
}

func (s *TimeService) ListTimeWorkItems(ctx context.Context) ([]TimeWorkItem, error) {
	_ = ctx
	items := make([]TimeWorkItem, len(timeWorkItems))
	copy(items, timeWorkItems)
	return items, nil
}

func (s *TimeService) GetTimeWeek(ctx context.Context, week string) (TimeWeekView, error) {
	return s.getTimeWeek(ctx, week, false)
}

func (s *TimeService) RefreshTimeWeek(ctx context.Context, week string) (TimeWeekView, error) {
	return s.getTimeWeek(ctx, week, true)
}

func (s *TimeService) getTimeWeek(ctx context.Context, week string, forceRefresh bool) (TimeWeekView, error) {
	if week == "" {
		week = CurrentWeek(s.loc)
	}
	startStr, endStr, err := weekRange(week, s.loc)
	if err != nil {
		return TimeWeekView{}, badRequest("week must use YYYY-Www format")
	}
	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return TimeWeekView{}, err
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return TimeWeekView{}, err
	}
	startDate := start.In(s.loc).Format("2006-01-02")
	endDate := end.In(s.loc).AddDate(0, 0, -1).Format("2006-01-02")

	settings, authorAccountID, err := s.tempoSettingsWithAuthor(ctx)
	if err != nil {
		return TimeWeekView{}, err
	}
	if !forceRefresh {
		cached, ok, err := s.cachedTimeWorklogs(ctx, authorAccountID, startDate, endDate)
		if err != nil {
			return TimeWeekView{}, err
		}
		if ok {
			return buildTimeWeekView(week, start.In(s.loc), cached), nil
		}
	}
	tempoClient := tempo.New(tempo.Config{BaseURL: settings.Tempo.BaseURL, APIToken: settings.Tempo.APIToken})
	if !tempoClient.Configured() {
		return TimeWeekView{}, &AppError{Code: 400, Message: "tempo integration is not configured", Err: tempo.ErrNotConfigured}
	}
	worklogs, err := tempoClient.ListUserWorklogs(ctx, authorAccountID, tempo.WorklogFilter{From: startDate, To: endDate, Limit: 100})
	if err != nil {
		return TimeWeekView{}, mapTempoListError(err)
	}
	cachedWorklogs := timeWorklogsFromTempo(worklogs)
	if err := s.repo.ReplaceCachedTimeWorklogs(ctx, authorAccountID, startDate, endDate, cachedWorklogs, nowString()); err != nil {
		return TimeWeekView{}, err
	}
	return buildTimeWeekView(week, start.In(s.loc), cachedWorklogs), nil
}

func (s *TimeService) LogTempoTime(ctx context.Context, input LogTimeRequest) (LogTimeResult, error) {
	request, dates, err := normalizeLogTimeRequest(input, s.loc)
	if err != nil {
		return LogTimeResult{}, err
	}
	if !allowedTimeWorkItem(request.WorkItemKey) {
		return LogTimeResult{}, badRequest("work_item_key is not allowed")
	}

	settings, authorAccountID, err := s.tempoSettingsWithAuthor(ctx)
	if err != nil {
		return LogTimeResult{}, err
	}
	jiraClient, err := s.jiraClient()
	if err != nil {
		return LogTimeResult{}, err
	}
	issueID, err := jiraClient.GetIssueID(ctx, request.WorkItemKey)
	if err != nil {
		return LogTimeResult{}, mapJiraLookupError("get jira issue id", err)
	}

	tempoClient := tempo.New(tempo.Config{
		BaseURL:  settings.Tempo.BaseURL,
		APIToken: settings.Tempo.APIToken,
	})
	if !tempoClient.Configured() {
		return LogTimeResult{}, &AppError{Code: 400, Message: "tempo integration is not configured", Err: tempo.ErrNotConfigured}
	}
	existing, err := s.timeWorklogsForRange(ctx, tempoClient, authorAccountID, request.StartDate, request.EndDate)
	if err != nil {
		return LogTimeResult{}, err
	}
	occupiedSeconds := occupiedSecondsByDate(existing)

	result := LogTimeResult{
		WorkItemKey: request.WorkItemKey,
		Description: request.Description,
		Hours:       request.Hours,
		StartDate:   request.StartDate,
		EndDate:     request.EndDate,
		Total:       len(dates),
		Entries:     make([]LogTimeEntry, 0, len(dates)),
	}
	for _, date := range dates {
		startTime := startTimeAfterLoggedSeconds(occupiedSeconds[date])
		entry := LogTimeEntry{
			Date:      date,
			StartTime: startTime,
			EndTime:   endTimeFromStart(startTime, request.Hours),
		}
		worklog, err := tempoClient.CreateWorklog(ctx, tempo.WorklogInput{
			AuthorAccountID: authorAccountID,
			IssueID:         issueID,
			TimeSpent:       int64(request.Hours * 3600),
			BillableSeconds: int64(request.Hours * 3600),
			StartDate:       date,
			StartTime:       startTime,
			Description:     request.Description,
		})
		if err != nil {
			entry.Error = mapTempoWorklogError(err).Error()
			result.Failed++
			result.Entries = append(result.Entries, entry)
			continue
		}
		entry.TempoWorklogID = worklog.TempoWorklogID
		entry.Self = worklog.Self
		if err := s.repo.UpsertCachedTimeWorklog(ctx, authorAccountID, TimeWorklog{
			TempoWorklogID:   worklog.TempoWorklogID,
			WorkItemKey:      request.WorkItemKey,
			WorkItemLabel:    timeWorkItemLabel(request.WorkItemKey),
			Description:      request.Description,
			StartDate:        date,
			StartTime:        startTime,
			EndTime:          entry.EndTime,
			TimeSpentSeconds: int64(request.Hours * 3600),
			Hours:            float64(request.Hours),
			Self:             worklog.Self,
		}, nowString()); err != nil {
			return LogTimeResult{}, err
		}
		result.Successful++
		result.Entries = append(result.Entries, entry)
		occupiedSeconds[date] += int64(request.Hours * 3600)
	}
	return result, nil
}
