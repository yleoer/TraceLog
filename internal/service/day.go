package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *DayService) CreateDayEntry(ctx context.Context, date string, content string) (DayEntry, error) {
	date = strings.TrimSpace(date)
	content = strings.TrimSpace(content)
	if !dayDatePattern.MatchString(date) {
		return DayEntry{}, badRequest("date must use YYYY-MM-DD format")
	}
	if content == "" {
		return DayEntry{}, badRequest("content_md is required")
	}
	now := nowString()
	return s.repo.CreateDayEntry(ctx, DayEntry{Date: date, ContentMD: content, CreatedAt: now, UpdatedAt: now})
}

func (s *DayService) DeleteDayEntry(ctx context.Context, id int64) error {
	return mapError(s.repo.DeleteDayEntry(ctx, id))
}

func (s *serviceCore) buildDays(ctx context.Context, startStr string, endStr string, dayCount int) ([]DayWork, error) {
	first, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		return nil, err
	}
	firstLoc := first.In(s.loc)
	days := make([]time.Time, dayCount)
	for i := range days {
		days[i] = firstLoc.AddDate(0, 0, i)
	}
	issueComments, err := s.repo.ListIssueCommentsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	tempComments, err := s.repo.ListTempTaskCommentsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	activityEvents, err := s.repo.ListActivityEventsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	todoComments, err := s.repo.ListCompletedIssueTodoCommentsBetween(ctx, startStr, endStr)
	if err != nil {
		return nil, err
	}
	dayEntries, err := s.repo.ListDayEntriesBetween(ctx, days[0].Format("2006-01-02"), days[len(days)-1].Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	comments := append(append(append(issueComments, tempComments...), activityEvents...), todoComments...)
	return bucketByDay(s.loc, days, comments, dayEntries), nil
}

func bucketByDay(loc *time.Location, days []time.Time, comments []DayComment, entries []DayEntry) []DayWork {
	result := make([]DayWork, len(days))
	index := map[string]int{}
	for i, d := range days {
		dateStr := d.Format("2006-01-02")
		result[i] = DayWork{Date: dateStr, Weekday: weekdayLabels[int(d.Weekday())], Comments: []DayComment{}, Entries: []DayEntry{}}
		index[dateStr] = i
	}
	for _, c := range comments {
		if i, ok := index[commentDayString(c.HappenedAt, loc)]; ok {
			c.URL = commentURL(c)
			result[i].Comments = append(result[i].Comments, c)
		}
	}
	for i := range result {
		sort.SliceStable(result[i].Comments, func(left, right int) bool {
			return result[i].Comments[left].HappenedAt < result[i].Comments[right].HappenedAt
		})
		result[i].Activities = groupDayActivities(result[i].Comments)
	}
	for _, e := range entries {
		if i, ok := index[e.Date]; ok {
			result[i].Entries = append(result[i].Entries, e)
		}
	}
	return result
}

func groupDayActivities(comments []DayComment) []DayActivity {
	activities := []DayActivity{}
	index := map[string]int{}
	for _, c := range comments {
		key := fmt.Sprintf("%s:%d:%s", c.Source, c.RefID, c.RefKey)
		i, ok := index[key]
		if !ok {
			activity := DayActivity{
				Source:   c.Source,
				RefID:    c.RefID,
				RefKey:   c.RefKey,
				RefTitle: c.RefTitle,
				URL:      c.URL,
				Comments: []DayComment{},
			}
			activities = append(activities, activity)
			i = len(activities) - 1
			index[key] = i
		}
		activities[i].Comments = append(activities[i].Comments, c)
		if activities[i].StartedAt == "" || c.HappenedAt < activities[i].StartedAt {
			activities[i].StartedAt = c.HappenedAt
		}
	}
	return activities
}

func commentDayString(happenedAt string, loc *time.Location) string {
	t, err := time.Parse(time.RFC3339, happenedAt)
	if err != nil {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}

func commentURL(c DayComment) string {
	if c.Source == "temp_task" {
		return fmt.Sprintf("/temp-tasks/%d", c.RefID)
	}
	return "/issues/" + c.RefKey
}
