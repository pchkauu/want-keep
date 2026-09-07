package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

var ErrInvalidTime = errors.New("invalid time")
var instantSyntax = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`)

type Instant struct {
	value time.Time
	valid bool
}
type Date struct{ value string }
type Month struct{ value string }
type Timezone struct{ value string }

func ParseInstant(value string) (Instant, error) {
	if !instantSyntax.MatchString(value) {
		return Instant{}, ErrInvalidTime
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Year() < 1 {
		return Instant{}, ErrInvalidTime
	}
	return Instant{value: parsed.UTC(), valid: true}, nil
}

func (i Instant) String() string {
	if !i.valid {
		return ""
	}
	return i.value.Format(time.RFC3339Nano)
}
func (i Instant) Time() time.Time { return i.value }

func ParseDate(value string) (Date, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Year() < 1 || parsed.Format(time.DateOnly) != value {
		return Date{}, ErrInvalidTime
	}
	return Date{value: value}, nil
}
func (d Date) String() string { return d.value }

func ParseMonth(value string) (Month, error) {
	parsed, err := time.Parse("2006-01", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01") != value {
		return Month{}, ErrInvalidTime
	}
	return Month{value: value}, nil
}
func (m Month) String() string { return m.value }

func ParseTimezone(value string) (Timezone, error) {
	if value != "UTC" && (!strings.Contains(value, "/") || strings.HasPrefix(value, "/") || strings.Contains(value, "..")) {
		return Timezone{}, ErrInvalidTime
	}
	if _, err := time.LoadLocation(value); err != nil {
		return Timezone{}, ErrInvalidTime
	}
	return Timezone{value: value}, nil
}
func (z Timezone) String() string { return z.value }

func (i Instant) DateIn(zone Timezone) (Date, error) {
	if !i.valid {
		return Date{}, ErrInvalidTime
	}
	validated, err := ParseTimezone(zone.value)
	if err != nil {
		return Date{}, err
	}
	location, err := time.LoadLocation(validated.value)
	if err != nil {
		return Date{}, ErrInvalidTime
	}
	return ParseDate(i.value.In(location).Format(time.DateOnly))
}
