package storage

import (
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

func splitInstant(i calendar.Instant) (time.Time, int16) {
	return i.Time().Truncate(time.Microsecond), int16(i.Time().Nanosecond() % 1000)
}
func restoreInstant(at time.Time, ns int16) (calendar.Instant, error) {
	if ns < 0 || ns > 999 || at.Nanosecond()%1000 != 0 {
		return calendar.Instant{}, calendar.ErrInvalidTime
	}
	return calendar.ParseInstant(at.UTC().Add(time.Duration(ns)).Format(time.RFC3339Nano))
}
