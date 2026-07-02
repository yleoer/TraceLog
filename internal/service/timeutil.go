package service

import (
	"fmt"
	"regexp"
	"time"
)

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func CurrentWeek(loc *time.Location) string {
	return weekFromTime(time.Now().In(loc))
}

func weekFromTime(value time.Time) string {
	year, week := value.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func weekRange(week string, loc *time.Location) (string, string, error) {
	match := regexp.MustCompile(`^(\d{4})-W(\d{2})$`).FindStringSubmatch(week)
	if match == nil {
		return "", "", fmt.Errorf("invalid week")
	}
	var year, weekNumber int
	if _, err := fmt.Sscanf(week, "%04d-W%02d", &year, &weekNumber); err != nil {
		return "", "", err
	}
	if weekNumber < 1 || weekNumber > isoWeeksInYear(year, loc) {
		return "", "", fmt.Errorf("invalid week")
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := jan4.AddDate(0, 0, 1-weekday)
	start := week1Monday.AddDate(0, 0, (weekNumber-1)*7)
	end := start.AddDate(0, 0, 7)
	return start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), nil
}

func isoWeeksInYear(year int, loc *time.Location) int {
	_, week := time.Date(year, 12, 28, 0, 0, 0, 0, loc).ISOWeek()
	return week
}

var weekdayLabels = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

var dayDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
