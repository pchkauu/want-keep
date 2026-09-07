package domain

import calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"

type EventKind string
type EventOrigin string

const (
	Created            EventKind   = "created"
	OpeningCorrected   EventKind   = "opening_corrected"
	OwnershipChanged   EventKind   = "ownership_changed"
	Interactive        EventOrigin = "interactive"
	LiveSync           EventOrigin = "live_sync"
	HistoricalBackfill EventOrigin = "historical_backfill"
)

type Event struct {
	AccountID string
	Revision  uint64
	Kind      EventKind
	Origin    EventOrigin
	Reason    string
	At        calendar.Instant
}

func (e Event) Validate() error {
	if e.AccountID == "" || e.Revision < 1 || e.Revision > 9007199254740991 || len(e.Reason) < 1 || len(e.Reason) > 2000 || e.At.String() == "" {
		return ErrInvalidAccount
	}
	if e.Kind != Created && e.Kind != OpeningCorrected && e.Kind != OwnershipChanged {
		return ErrInvalidAccount
	}
	if e.Origin != Interactive && e.Origin != LiveSync && e.Origin != HistoricalBackfill {
		return ErrInvalidAccount
	}
	return nil
}

func (e Event) CelebrationEligible() bool {
	return e.Kind == Created && (e.Origin == Interactive || e.Origin == LiveSync)
}
