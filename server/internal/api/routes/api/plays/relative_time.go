package plays

import (
	"fmt"
	"math"
	"time"
)

const (
	playHistoryMinutesInDay           int64 = 24 * 60
	playHistoryMinutesInAlmostTwoDays int64 = 42 * 60
	playHistoryMinutesInMonth         int64 = 30 * playHistoryMinutesInDay
	playHistoryMinutesInTwoMonths     int64 = 2 * playHistoryMinutesInMonth
	playHistoryMinutesInYear          int64 = 365 * playHistoryMinutesInDay
)

type playHistoryRelativeTimeOptions struct {
	AddSuffix bool
}

// formatPlayHistoryRelativeTime mirrors date-fns formatDistanceToNow's
// non-strict English thresholds.
func formatPlayHistoryRelativeTime(value, now time.Time, options playHistoryRelativeTimeOptions) string {
	future := value.After(now)
	if value.After(now) {
		value, now = now, value
	}

	minutes := int64(math.Round(now.Sub(value).Seconds() / 60))
	distance := ""
	switch {
	case minutes < 1:
		distance = "less than a minute"
	case minutes < 2:
		distance = "1 minute"
	case minutes < 45:
		distance = pluralPlayHistoryDistance(minutes, "minute")
	case minutes < 90:
		distance = "about 1 hour"
	case minutes < playHistoryMinutesInDay:
		hours := int64(math.Round(float64(minutes) / 60))
		distance = "about " + pluralPlayHistoryDistance(hours, "hour")
	case minutes < playHistoryMinutesInAlmostTwoDays:
		distance = "1 day"
	case minutes < playHistoryMinutesInMonth:
		days := int64(math.Round(float64(minutes) / float64(playHistoryMinutesInDay)))
		distance = pluralPlayHistoryDistance(days, "day")
	case minutes < playHistoryMinutesInTwoMonths:
		months := int64(math.Round(float64(minutes) / float64(playHistoryMinutesInMonth)))
		distance = "about " + pluralPlayHistoryDistance(months, "month")
	case minutes < playHistoryMinutesInYear:
		months := int64(math.Round(float64(minutes) / float64(playHistoryMinutesInMonth)))
		distance = pluralPlayHistoryDistance(months, "month")
	default:
		years := minutes / playHistoryMinutesInYear
		remainingMinutes := minutes - years*playHistoryMinutesInYear
		if remainingMinutes < 3*playHistoryMinutesInMonth {
			distance = "about " + pluralPlayHistoryDistance(years, "year")
		} else if remainingMinutes < 9*playHistoryMinutesInMonth {
			distance = "over " + pluralPlayHistoryDistance(years, "year")
		} else {
			distance = "almost " + pluralPlayHistoryDistance(years+1, "year")
		}
	}

	if !options.AddSuffix {
		return distance
	}
	if future {
		return "in " + distance
	}
	return distance + " ago"
}

func pluralPlayHistoryDistance(value int64, unit string) string {
	if value == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", value, unit)
}
