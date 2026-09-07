package domain

import (
	"slices"
	"strconv"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
)

// Participation assigns each observed monetary component to one carrier without
// changing the provider lifecycle or the user's accounting exclusion.
type ParticipationKind string
type ParticipationState string
type ContributionRole string

type Participation struct {
	GroupID string
	Kind    ParticipationKind
	State   ParticipationState
	Parts   []Contribution
}

type Contribution struct {
	ComponentID     string
	State           State
	At              calendar.Instant
	Position        int
	CarrierID       string
	CarrierPosition int
	Role            ContributionRole
}

func (p Participation) SameCarriers(other Participation) bool {
	if p.GroupID != other.GroupID || p.Kind != other.Kind || p.State != other.State || len(p.Parts) != len(other.Parts) {
		return false
	}
	for i, part := range p.Parts {
		q := other.Parts[i]
		if part.ComponentID != q.ComponentID || part.Position != q.Position || part.CarrierID != q.CarrierID || part.CarrierPosition != q.CarrierPosition || part.Role != q.Role {
			return false
		}
	}
	return true
}

func (p Participation) Validate(r Revision) error {
	if p.GroupID == "" {
		if p.Kind != "" || p.State != "" || len(p.Parts) != 0 {
			return ErrInvalidRevision
		}
		return nil
	}
	if !slices.Contains([]ParticipationKind{"payment", "transfer", "exchange"}, p.Kind) || !slices.Contains([]ParticipationState{"waiting", "linked"}, p.State) {
		return ErrInvalidRevision
	}
	if p.State == "waiting" {
		if len(p.Parts) != 0 {
			return ErrInvalidRevision
		}
		return nil
	}
	if len(p.Parts) != len(r.Postings) {
		return ErrInvalidRevision
	}
	for i, part := range p.Parts {
		if part.ComponentID != part.CarrierID+":"+strconv.Itoa(part.CarrierPosition) || len(part.ComponentID) > 100 {
			return ErrInvalidRevision
		}
		if !part.State.Valid() || part.At.String() == "" || part.Position != i || part.CarrierID == "" || part.CarrierPosition < 0 || part.CarrierPosition >= 1000 || !slices.Contains([]ContributionRole{"payment", "outgoing", "incoming", "fee"}, part.Role) {
			return ErrInvalidRevision
		}
		posting := r.Postings[i]
		if !posting.MovesMoney() || posting.Role != Principal && posting.Role != Fee || part.Role == "fee" && posting.Role != Fee || part.Role != "fee" && posting.Role != Principal {
			return ErrInvalidRevision
		}
		if p.Kind == "payment" && part.Role != "payment" && part.Role != "fee" || p.Kind != "payment" && part.Role == "payment" {
			return ErrInvalidRevision
		}
		if part.Role == "outgoing" && posting.Money.Sign() >= 0 || part.Role == "incoming" && posting.Money.Sign() <= 0 || part.CarrierID == r.OperationID && part.CarrierPosition != i {
			return ErrInvalidRevision
		}
	}
	return nil
}

func (r Revision) Contributes(position int) bool {
	if r.Participation.GroupID == "" {
		return true
	}
	return r.Participation.State == "linked" && position < len(r.Participation.Parts) && r.Participation.Parts[position].CarrierID == r.OperationID
}

func (r Revision) InternalPrincipal(position int) bool {
	return r.Participation.State == "linked" && (r.Participation.Kind == "transfer" || r.Participation.Kind == "exchange") && r.Postings[position].Role == Principal
}

func (r Revision) ContributionState(position int) State {
	if r.Participation.State == "linked" && position < len(r.Participation.Parts) {
		return r.Participation.Parts[position].State
	}
	return r.State
}
func (r Revision) ContributionAt(position int) calendar.Instant {
	if r.Participation.State == "linked" && position < len(r.Participation.Parts) {
		return r.Participation.Parts[position].At
	}
	return r.OccurredAt
}
