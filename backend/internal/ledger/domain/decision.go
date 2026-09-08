package domain

import (
	"errors"
	"maps"
	"slices"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrDecisionConflict = errors.New("decision conflicts with subsequent changes")
var ErrNoChange = errors.New("correction has no effect")

type Field string

const (
	PrincipalField    Field = "principal"
	FeesField         Field = "fees"
	DateField         Field = "occurred_at"
	PayerField        Field = "payer"
	MerchantField     Field = "merchant"
	NoteField         Field = "note"
	AccountingField   Field = "accounting"
	CategoryField     Field = "category"
	MerchantIDField   Field = "merchant_identity"
	ReceiptItemsField Field = "receipt_items"
	LegacyField       Field = "legacy_all"
	MatchingField     Field = "matching"
	ContributionField Field = "contribution"
)

func (f Field) Valid() bool {
	return slices.Contains([]Field{PrincipalField, FeesField, DateField, PayerField, MerchantField, NoteField, AccountingField, CategoryField, MerchantIDField, ReceiptItemsField, LegacyField, MatchingField, ContributionField}, f)
}

type AccountingState string

const (
	IncludedInAccounting   AccountingState = "included"
	ExcludedFromAccounting AccountingState = "excluded"
)

type Protection struct {
	DecisionID string
	Revision   uint64
}

type PayerChange struct {
	State    string
	MemberID household.MembershipID
}
type Correction struct {
	Principal              *[]Posting
	Fees                   *[]Posting
	OccurredAt             *calendar.Instant
	Payer                  *PayerChange
	Merchant, Note         *string
	CategoryID, MerchantID *string
	ReceiptItems           *ReceiptItemsCorrection
}

type DecisionEntry struct {
	OperationID   string
	Before, After uint64
	Fields        []Field
}
type Decision struct {
	ID, Kind, Reason, UndoOf string
	ActorID                  household.UserID
	At                       calendar.Instant
	Entries                  []DecisionEntry
	Evidence                 []Evidence
}
type Evidence struct {
	Kind, ID string
	Revision uint64
}

func (e Evidence) Validate() error {
	if !slices.Contains([]string{"source", "attachment", "review"}, e.Kind) || e.ID == "" || e.Revision < 1 || e.Revision > 9007199254740991 {
		return ErrInvalidRevision
	}
	return nil
}

func (d Decision) Validate() error {
	if d.ID == "" || d.ActorID == "" || d.At.String() == "" || utf8.RuneCountInString(d.Reason) < 1 || utf8.RuneCountInString(d.Reason) > 2000 || len(d.Entries) < 1 || len(d.Entries) > 100 {
		return ErrInvalidRevision
	}
	if !slices.Contains([]string{"correction", "exclusion", "undo", "automated", "matching"}, d.Kind) || (d.Kind == "undo") != (d.UndoOf != "") {
		return ErrInvalidRevision
	}
	seen := map[string]bool{}
	for _, e := range d.Entries {
		if e.OperationID == "" || seen[e.OperationID] || e.Before < 1 || e.After != e.Before+1 || e.After > 9007199254740991 || len(e.Fields) == 0 {
			return ErrInvalidRevision
		}
		seen[e.OperationID] = true
		fields := map[Field]bool{}
		for _, f := range e.Fields {
			if !f.Valid() || f == LegacyField || fields[f] {
				return ErrInvalidRevision
			}
			fields[f] = true
		}
	}
	for _, e := range d.Evidence {
		if e.Validate() != nil {
			return ErrInvalidRevision
		}
	}
	return nil
}

func (r Revision) Clone() Revision {
	if r.Correspondence != nil {
		c := *r.Correspondence
		r.Correspondence = &c
	}
	r.Postings = slices.Clone(r.Postings)
	r.Participation.Parts = slices.Clone(r.Participation.Parts)
	r.ReceiptItems = slices.Clone(r.ReceiptItems)
	r.Protections = maps.Clone(r.Protections)
	if r.Protections == nil {
		r.Protections = map[Field]Protection{}
	}
	r.FieldVersions = maps.Clone(r.FieldVersions)
	if r.FieldVersions == nil {
		r.FieldVersions = map[Field]uint64{}
	}
	return r
}

func (r Revision) Accounting() AccountingState {
	if r.AccountingState == "" {
		return IncludedInAccounting
	}
	return r.AccountingState
}

func (r Revision) WithDecision(d Decision, fields []Field) Revision {
	r = r.Clone()
	r.Revision++
	r.ActorID = d.ActorID
	r.Reason = d.Reason
	r.DecisionID = d.ID
	r.RecordedAt = d.At
	if r.Protections == nil {
		r.Protections = map[Field]Protection{}
	}
	if r.FieldVersions == nil {
		r.FieldVersions = map[Field]uint64{}
	}
	for _, f := range fields {
		r.FieldVersions[f] = r.Revision
		if d.Kind != "automated" && d.Kind != "undo" && f != ContributionField {
			r.Protections[f] = Protection{d.ID, r.Revision}
		}
	}
	r.HumanOverride = len(r.Protections) > 0
	return r
}

func (r Revision) UndoFields(entry DecisionEntry, before Revision, source *Revision) (Revision, error) {
	if r.OperationID != entry.OperationID || before.OperationID != entry.OperationID || before.Revision != entry.Before {
		return r, ErrInvalidRevision
	}
	next := r.Clone()
	for _, f := range entry.Fields {
		// Derived contribution state/time is rebuilt by its owner from current
		// facts. It cannot supersede an independent association or date decision.
		if f == ContributionField {
			continue
		}
		if r.FieldVersions[f] != entry.After {
			return r, ErrDecisionConflict
		}
		basis := before
		_, legacy := before.Protections[LegacyField]
		if _, protected := before.Protections[f]; !protected && !legacy && source != nil && f != AccountingField && f != MatchingField {
			basis = *source
		}
		if err := next.CopyField(basis, f); err != nil {
			return r, err
		}
		if prior, ok := before.Protections[f]; ok {
			next.Protections[f] = prior
		} else {
			delete(next.Protections, f)
		}
	}
	next.HumanOverride = len(next.Protections) > 0
	return next, nil
}

// ReapplySource resolves retained source updates without replacing independent later decisions.
func (r Revision) ReapplySource(source Revision, independent []Field) (Revision, error) {
	merged, _, err := r.MergeSource(source)
	if err != nil {
		return r, err
	}
	for _, field := range independent {
		if err := merged.CopyField(r, field); err != nil {
			return r, err
		}
	}
	merged, err = merged.InTimezone(r.Timezone)
	if err != nil || merged.Validate() != nil || r.State.RequireNext(merged.State) != nil {
		return r.Clone(), nil
	}
	return merged, nil
}

// MergeSource retains only explicit human choices; provider state remains authoritative.
func (r Revision) MergeSource(source Revision) (Revision, bool, error) {
	if _, legacy := r.Protections[LegacyField]; legacy || r.HumanOverride && len(r.Protections) == 0 {
		return r.Clone(), true, nil
	}
	next := source.Clone()
	next.Revision = r.Revision
	next.Protections = maps.Clone(r.Protections)
	next.FieldVersions = maps.Clone(r.FieldVersions)
	next.AccountingState = r.Accounting()
	next.Participation = r.Clone().Participation
	next.HumanOverride = len(next.Protections) > 0
	conflict := false
	for f := range r.Protections {
		if f != AccountingField && f != MatchingField && !r.FieldEqual(source, f) {
			conflict = true
		}
		if err := next.CopyField(r, f); err != nil {
			return r, false, err
		}
	}
	return next, conflict, nil
}

func (p Posting) SameMoney(q Posting) bool {
	value, err := p.Money.Compare(q.Money)
	return err == nil && value == 0
}

func (r Revision) ConflictsWithSource(source *Revision) bool {
	if source == nil {
		return false
	}
	if _, legacy := r.Protections[LegacyField]; legacy {
		return !r.SameFacts(*source)
	}
	for f := range r.Protections {
		if f != AccountingField && f != MatchingField && !r.FieldEqual(*source, f) {
			return true
		}
	}
	merged, _, err := r.MergeSource(*source)
	if err != nil {
		return true
	}
	merged, err = merged.InTimezone(r.Timezone)
	return err != nil || merged.Validate() != nil || r.State.RequireNext(source.State) != nil
}
