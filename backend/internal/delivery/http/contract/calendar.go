package contract

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

type CalendarConverter struct{}

func (CalendarConverter) InstantFromDTO(value generated.Instant) (calendar.Instant, error) {
	return calendar.ParseInstant(value)
}
func (CalendarConverter) DateFromDTO(value generated.Date) (calendar.Date, error) {
	return calendar.ParseDate(value)
}
func (CalendarConverter) MonthFromDTO(value generated.Month) (calendar.Month, error) {
	return calendar.ParseMonth(value)
}
func (CalendarConverter) TimezoneFromDTO(value generated.Timezone) (calendar.Timezone, error) {
	return calendar.ParseTimezone(value)
}

func (CalendarConverter) InstantToDTO(value calendar.Instant) (generated.Instant, error) {
	checked, err := calendar.ParseInstant(value.String())
	if err != nil {
		return "", err
	}
	return checked.String(), nil
}
func (CalendarConverter) DateToDTO(value calendar.Date) (generated.Date, error) {
	checked, err := calendar.ParseDate(value.String())
	if err != nil {
		return "", err
	}
	return checked.String(), nil
}
func (CalendarConverter) MonthToDTO(value calendar.Month) (generated.Month, error) {
	checked, err := calendar.ParseMonth(value.String())
	if err != nil {
		return "", err
	}
	return checked.String(), nil
}
func (CalendarConverter) TimezoneToDTO(value calendar.Timezone) (generated.Timezone, error) {
	checked, err := calendar.ParseTimezone(value.String())
	if err != nil {
		return "", err
	}
	return checked.String(), nil
}
