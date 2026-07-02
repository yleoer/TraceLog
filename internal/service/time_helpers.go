package service

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"tracelog/internal/tempo"
)

func normalizeLogTimeRequest(input LogTimeRequest, loc *time.Location) (LogTimeRequest, []string, error) {
	input.WorkItemKey = strings.ToUpper(strings.TrimSpace(input.WorkItemKey))
	input.Description = strings.TrimSpace(input.Description)
	input.StartDate = strings.TrimSpace(input.StartDate)
	input.EndDate = strings.TrimSpace(input.EndDate)
	if input.EndDate == "" {
		input.EndDate = input.StartDate
	}
	if loc == nil {
		loc = time.UTC
	}
	if !dayDatePattern.MatchString(input.StartDate) || !dayDatePattern.MatchString(input.EndDate) {
		return LogTimeRequest{}, nil, badRequest("date must use YYYY-MM-DD format")
	}
	if input.Hours < 1 || input.Hours > 8 {
		return LogTimeRequest{}, nil, badRequest("hours must be between 1 and 8")
	}
	if input.Description == "" {
		return LogTimeRequest{}, nil, badRequest("description is required")
	}
	start, err := time.ParseInLocation("2006-01-02", input.StartDate, loc)
	if err != nil {
		return LogTimeRequest{}, nil, badRequest("start_date must use YYYY-MM-DD format")
	}
	end, err := time.ParseInLocation("2006-01-02", input.EndDate, loc)
	if err != nil {
		return LogTimeRequest{}, nil, badRequest("end_date must use YYYY-MM-DD format")
	}
	if end.Before(start) {
		return LogTimeRequest{}, nil, badRequest("end_date must be on or after start_date")
	}
	dates := []string{}
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		dates = append(dates, current.Format("2006-01-02"))
		if len(dates) > maxLogTimeDays {
			return LogTimeRequest{}, nil, badRequest("date range must be 31 days or fewer")
		}
	}
	return input, dates, nil
}

func allowedTimeWorkItem(key string) bool {
	for _, item := range timeWorkItems {
		if item.Key == key {
			return true
		}
	}
	return false
}

func buildTimeWeekView(week string, start time.Time, worklogs []TimeWorklog) TimeWeekView {
	view := TimeWeekView{
		Week:      week,
		StartDate: start.Format("2006-01-02"),
		EndDate:   start.AddDate(0, 0, 6).Format("2006-01-02"),
		Days:      make([]TimeDay, 7),
		WorkItems: append([]TimeWorkItem(nil), timeWorkItems...),
	}
	dayIndex := map[string]int{}
	for i := range view.Days {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")
		dayIndex[dateStr] = i
		view.Days[i] = TimeDay{Date: dateStr, Weekday: weekdayLabels[int(date.Weekday())], Worklogs: []TimeWorklog{}}
	}
	for _, worklog := range worklogs {
		if worklog.StartDate == "" {
			continue
		}
		worklog.WorkItemLabel = timeWorkItemLabel(worklog.WorkItemKey)
		worklog.Hours = float64(worklog.TimeSpentSeconds) / 3600
		if worklog.EndTime == "" {
			worklog.EndTime = endTimeFromSeconds(worklog.StartTime, worklog.TimeSpentSeconds)
		}
		view.Worklogs = append(view.Worklogs, worklog)
		view.TotalHours += worklog.Hours
		if index, ok := dayIndex[worklog.StartDate]; ok {
			view.Days[index].Worklogs = append(view.Days[index].Worklogs, worklog)
			view.Days[index].TotalHours += worklog.Hours
		}
	}
	sort.SliceStable(view.Worklogs, func(i, j int) bool {
		return view.Worklogs[i].StartDate+" "+view.Worklogs[i].StartTime < view.Worklogs[j].StartDate+" "+view.Worklogs[j].StartTime
	})
	for index := range view.Days {
		sort.SliceStable(view.Days[index].Worklogs, func(i, j int) bool {
			return view.Days[index].Worklogs[i].StartTime < view.Days[index].Worklogs[j].StartTime
		})
	}
	return view
}

func timeWorklogsFromTempo(worklogs []tempo.Worklog) []TimeWorklog {
	result := make([]TimeWorklog, 0, len(worklogs))
	for _, worklog := range worklogs {
		result = append(result, timeWorklogFromTempo(worklog))
	}
	return result
}

func timeWorklogFromTempo(worklog tempo.Worklog) TimeWorklog {
	key := strings.ToUpper(strings.TrimSpace(worklog.Issue.Key))
	hours := float64(worklog.TimeSpentSeconds) / 3600
	return TimeWorklog{
		TempoWorklogID:   worklog.TempoWorklogID,
		WorkItemKey:      key,
		WorkItemLabel:    timeWorkItemLabel(key),
		Description:      worklog.Description,
		StartDate:        worklog.StartDate,
		StartTime:        normalizeTempoStartTime(worklog.StartTime),
		EndTime:          endTimeFromSeconds(normalizeTempoStartTime(worklog.StartTime), worklog.TimeSpentSeconds),
		TimeSpentSeconds: worklog.TimeSpentSeconds,
		Hours:            hours,
		Self:             worklog.Self,
	}
}

func timeWorkItemLabel(key string) string {
	for _, item := range timeWorkItems {
		if item.Key == key {
			return item.Label
		}
	}
	return ""
}

func occupiedSecondsByDate(worklogs []TimeWorklog) map[string]int64 {
	result := map[string]int64{}
	for _, worklog := range worklogs {
		if worklog.StartDate == "" || worklog.TimeSpentSeconds <= 0 {
			continue
		}
		result[worklog.StartDate] += worklog.TimeSpentSeconds
	}
	return result
}

func startTimeAfterLoggedSeconds(seconds int64) string {
	return endTimeFromSeconds(workdayStartTime, seconds)
}

func endTimeFromStart(startTime string, hours int) string {
	return endTimeFromSeconds(startTime, int64(hours*3600))
}

func endTimeFromSeconds(startTime string, seconds int64) string {
	parsed, err := time.Parse("15:04:05", normalizeTempoStartTime(startTime))
	if err != nil {
		parsed, _ = time.Parse("15:04:05", workdayStartTime)
	}
	return parsed.Add(time.Duration(seconds) * time.Second).Format("15:04:05")
}

func normalizeTempoStartTime(value string) string {
	value = strings.TrimSpace(value)
	if regexp.MustCompile(`^\d{1,2}:\d{2}:\d{2}$`).MatchString(value) {
		parts := strings.Split(value, ":")
		if len(parts[0]) == 1 {
			return "0" + value
		}
		return value
	}
	if regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(value) {
		parts := strings.Split(value, ":")
		if len(parts[0]) == 1 {
			return "0" + value + ":00"
		}
		return value + ":00"
	}
	return workdayStartTime
}
