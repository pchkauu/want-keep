package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) historicalLedgerState(ctx context.Context, q reader, p household.Principal, current ledger.Revision, asOf calendar.Instant) (ledger.Revision, bool, bool, error) {
	effective := current.Clone()
	at, ns := splitInstant(asOf)
	var revision uint64
	err := q.QueryRow(ctx, `SELECT r.revision FROM want_keep.ledger_revision_audit r WHERE r.household_id=$1 AND r.operation_id=$2 AND (r.recorded_at,r.recorded_ns)<=($3,$4) ORDER BY r.revision DESC LIMIT 1`, p.HouseholdID(), current.OperationID, at, ns).Scan(&revision)
	known := err == nil
	if known {
		state, err := s.LedgerRevision(ctx, p, current.OperationID, revision)
		if err != nil {
			return ledger.Revision{}, false, false, err
		}
		effective.State, effective.PostedAt = state.State, state.PostedAt
		if effective.State != ledger.Posted && effective.State != ledger.Reversed {
			effective.PostedAt = calendar.Instant{}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ledger.Revision{}, false, false, err
	}
	postedBySource := current.Origin == "source" && current.State == ledger.Posted && current.PostedAt.String() != "" && !current.PostedAt.Time().After(asOf.Time())
	if postedBySource && (!known || effective.State == ledger.Draft || effective.State == ledger.Pending) {
		effective.State, effective.PostedAt, known = ledger.Posted, current.PostedAt, true
	}
	reversedUnknown := current.Origin == "source" && current.State == ledger.Reversed && current.PostedAt.String() != "" && !current.PostedAt.Time().After(asOf.Time()) && (!known || effective.State != ledger.Reversed)
	return effective, known, reversedUnknown, nil
}

func (s *Store) historicalMatchingEffects(ctx context.Context, q reader, p household.Principal, current ledger.Revision, accountID string, asset money.Asset, openingAt, asOf calendar.Instant) ([]account.Effect, bool, error) {
	components := map[string]bool{}
	if current.Accounting() == ledger.ExcludedFromAccounting {
		return nil, false, nil
	}
	for i, part := range current.Participation.Parts {
		if current.Contributes(i) && current.Postings[i].AccountID == accountID && !part.At.Time().After(asOf.Time()) {
			components[part.ComponentID] = true
		}
	}
	if len(components) == 0 {
		return nil, false, nil
	}
	group, err := s.MatchingGroup(ctx, p, current.Participation.GroupID)
	if err != nil {
		return nil, false, err
	}
	facts := []ledger.Revision{}
	unknown := false
	for _, member := range group.Members {
		fact, found, err := s.CurrentLedgerRevision(ctx, p, member.OperationID)
		if err != nil {
			return nil, false, err
		}
		if !found {
			unknown = true
			continue
		}
		effective, known, reversedUnknown, err := s.historicalLedgerState(ctx, q, p, fact, asOf)
		if err != nil {
			return nil, false, err
		}
		relevant := false
		for _, part := range fact.Participation.Parts {
			relevant = relevant || components[part.ComponentID]
		}
		if relevant && (!known || reversedUnknown) {
			unknown = true
		}
		facts = append(facts, effective)
	}
	if !unknown {
		_, assigned, err := group.Assign(facts, true)
		if err != nil {
			unknown = true
		} else {
			for _, fact := range assigned {
				if fact.OperationID != current.OperationID {
					continue
				}
				changes, err := fact.BalanceEffects()
				if err != nil {
					return nil, false, err
				}
				effects := []account.Effect{}
				for _, change := range changes {
					if change.AccountID != accountID || change.At.Time().After(asOf.Time()) {
						continue
					}
					values := account.Amounts{Owned: change.Owned, Available: change.Available, Locked: change.Locked, Debt: change.Debt}
					effects = append(effects, account.Effect{OperationID: current.OperationID, Revision: current.Revision, At: change.At, Changes: &values})
				}
				return effects, false, nil
			}
			unknown = true
		}
	}
	// A missing historical lifecycle or a partially appended compound group cannot
	// authorize a balancing adjustment. Keep the evidence and explicit uncertainty.
	effect, err := s.historicalUncertainty(ctx, q, p, current, accountID, asset, openingAt, asOf, false)
	if err != nil {
		return nil, false, err
	}
	if effect != nil {
		return []account.Effect{*effect}, true, nil
	}
	values := account.UnknownAmounts("transaction_state_at_source_unknown")
	return []account.Effect{{OperationID: current.OperationID, Revision: current.Revision, At: asOf, Changes: &values}}, true, nil
}
