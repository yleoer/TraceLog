package service

import (
	"context"
	"strings"
	"time"
)

func (s *DashboardService) GetMonthGantt(ctx context.Context, month string) (MonthGanttView, error) {
	month, start, end, err := monthRange(month, s.loc)
	if err != nil {
		return MonthGanttView{}, badRequest("month must use YYYY-MM format")
	}
	items, err := s.repo.ListMonthGanttItemsBetween(ctx, start, end)
	if err != nil {
		return MonthGanttView{}, err
	}
	return MonthGanttView{Month: month, Items: items}, nil
}

func monthRange(month string, loc *time.Location) (string, string, string, error) {
	month = strings.TrimSpace(month)
	if month == "" {
		month = time.Now().In(loc).Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil || start.Format("2006-01") != month {
		return "", "", "", err
	}
	end := start.AddDate(0, 1, 0)
	return month, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), nil
}
