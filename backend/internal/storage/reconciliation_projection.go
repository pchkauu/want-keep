package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func (s *Store) SyncJob(ctx context.Context, p household.Principal, id string) (jobs.Job, error) {
	return s.Job(ctx, p, id)
}

func (s *Store) HistoricalProjection(ctx context.Context, p household.Principal, accountID string, asOf calendar.Instant) (account.Amounts, reporting.Coverage, []string, error) {
	entry, err := s.Account(ctx, p, accountID)
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	opening, found, err := s.Opening(ctx, p, accountID)
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	if !found {
		coverage, _ := reporting.NewCoverage(reporting.Partial, []string{"opening_missing"})
		return account.UnknownAmounts("opening_missing"), coverage, nil, nil
	}
	openingAt, err := opening.Instant()
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	if openingAt.Time().After(asOf.Time()) {
		coverage, _ := reporting.NewCoverage(reporting.Partial, []string{"opening_after_source"})
		return account.UnknownAmounts("opening_after_source"), coverage, nil, nil
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	rows, err := q.Query(ctx, `SELECT o.id,o.revision FROM want_keep.operations o WHERE o.household_id=$1 AND EXISTS(SELECT 1 FROM want_keep.postings p WHERE (p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision) AND p.account_id=$2) ORDER BY o.id`, p.HouseholdID(), accountID)
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	type reference struct {
		id       string
		revision uint64
	}
	references := []reference{}
	for rows.Next() {
		var reference reference
		if err = rows.Scan(&reference.id, &reference.revision); err != nil {
			rows.Close()
			return account.Amounts{}, reporting.Coverage{}, nil, err
		}
		references = append(references, reference)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	effects := []account.Effect{}
	related := []string{}
	uncertain := false
	at, ns := splitInstant(asOf)
	for _, reference := range references {
		current, err := s.LedgerRevision(ctx, p, reference.id, reference.revision)
		if err != nil {
			return account.Amounts{}, reporting.Coverage{}, nil, err
		}
		// Opening is the projection anchor below, not an additional movement.
		if current.Type == ledger.Opening {
			continue
		}
		if current.OccurredAt.Time().After(asOf.Time()) {
			continue
		}
		effective := current.Clone()
		if current.Type != ledger.Adjustment {
			var historical uint64
			err = q.QueryRow(ctx, `SELECT r.revision FROM want_keep.ledger_revision_audit r WHERE r.household_id=$1 AND r.operation_id=$2 AND (r.recorded_at,r.recorded_ns)<=($3,$4) ORDER BY r.revision DESC LIMIT 1`, p.HouseholdID(), reference.id, at, ns).Scan(&historical)
			historicalKnown := err == nil
			if historicalKnown {
				state, loadErr := s.LedgerRevision(ctx, p, reference.id, historical)
				if loadErr != nil {
					return account.Amounts{}, reporting.Coverage{}, nil, loadErr
				}
				effective.State = state.State
				effective.PostedAt = state.PostedAt
				if effective.State != ledger.Posted && effective.State != ledger.Reversed {
					effective.PostedAt = calendar.Instant{}
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return account.Amounts{}, reporting.Coverage{}, nil, err
			}
			postedBySource := current.Origin == "source" && current.State == ledger.Posted && current.PostedAt.String() != "" && !current.PostedAt.Time().After(asOf.Time())
			if postedBySource && (!historicalKnown || effective.State == ledger.Draft || effective.State == ledger.Pending) {
				effective.State = ledger.Posted
				effective.PostedAt = current.PostedAt
				historicalKnown = true
			}
			if current.Origin == "source" && current.State == ledger.Reversed && current.PostedAt.String() != "" && !current.PostedAt.Time().After(asOf.Time()) && (!historicalKnown || effective.State != ledger.Reversed) {
				effect, uncertaintyErr := s.historicalUncertainty(ctx, q, p, current, accountID, entry.Asset, openingAt, asOf, true)
				if uncertaintyErr != nil {
					return account.Amounts{}, reporting.Coverage{}, nil, uncertaintyErr
				}
				if effect != nil {
					effects = append(effects, *effect)
					related = append(related, current.OperationID)
					uncertain = true
				}
				continue
			}
			if !historicalKnown {
				effect, uncertaintyErr := s.historicalUncertainty(ctx, q, p, current, accountID, entry.Asset, openingAt, asOf, false)
				if uncertaintyErr != nil {
					return account.Amounts{}, reporting.Coverage{}, nil, uncertaintyErr
				}
				if effect != nil {
					effects = append(effects, *effect)
					related = append(related, current.OperationID)
					uncertain = true
				}
				continue
			}
		}
		balanceEffects, err := effective.BalanceEffects()
		if err != nil {
			return account.Amounts{}, reporting.Coverage{}, nil, err
		}
		for _, effect := range balanceEffects {
			if effect.AccountID != accountID {
				continue
			}
			changes := account.Amounts{Owned: effect.Owned, Available: effect.Available, Locked: effect.Locked, Debt: effect.Debt}
			effects = append(effects, account.Effect{OperationID: current.OperationID, Revision: current.Revision, At: effect.At, Changes: &changes})
			related = append(related, current.OperationID)
		}
	}
	values, err := opening.Project(entry.Asset, effects)
	if err != nil {
		return account.Amounts{}, reporting.Coverage{}, nil, err
	}
	reasons := []string{}
	if !opening.Confirmed {
		reasons = append(reasons, "opening_unconfirmed")
	}
	if uncertain {
		reasons = append(reasons, "transaction_state_at_source_unknown")
	}
	for _, value := range values.Fields() {
		if _, known := value.Value(); !known {
			reasons = append(reasons, "balance_components_incomplete")
			break
		}
	}
	coverage := reporting.Coverage{}
	if len(reasons) == 0 {
		coverage, _ = reporting.NewCoverage(reporting.Complete, nil)
	} else {
		coverage, _ = reporting.NewCoverage(reporting.Partial, reasons)
	}
	return values, coverage, related, nil
}

func (s *Store) historicalUncertainty(ctx context.Context, q reader, p household.Principal, current ledger.Revision, accountID string, asset money.Asset, openingAt, asOf calendar.Instant, forcePosted bool) (*account.Effect, error) {
	rows, err := q.Query(ctx, `SELECT revision FROM want_keep.operation_revisions WHERE household_id=$1 AND operation_id=$2 AND revision<=$3 ORDER BY revision`, p.HouseholdID(), current.OperationID, current.Revision)
	if err != nil {
		return nil, err
	}
	revisions := []uint64{}
	for rows.Next() {
		var revision uint64
		if err = rows.Scan(&revision); err != nil {
			rows.Close()
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}

	affected := [4]bool{}
	for _, revision := range revisions {
		candidate, loadErr := s.LedgerRevision(ctx, p, current.OperationID, revision)
		if loadErr != nil {
			return nil, loadErr
		}
		if candidate.OccurredAt.Time().Before(openingAt.Time()) || candidate.OccurredAt.Time().After(asOf.Time()) {
			continue
		}
		states := []ledger.State{}
		if forcePosted {
			states = append(states, ledger.Posted)
		} else {
			switch candidate.State {
			case ledger.Pending, ledger.Cancelled:
				states = append(states, ledger.Pending)
			case ledger.Posted, ledger.Reversed:
				states = append(states, ledger.Posted)
			}
		}
		for _, state := range states {
			possible := candidate.Clone()
			possible.AccountingState = ledger.IncludedInAccounting
			possible.State = state
			if state == ledger.Pending {
				possible.PostedAt = calendar.Instant{}
			} else if possible.PostedAt.String() == "" {
				possible.PostedAt = possible.OccurredAt
			}
			balanceEffects, effectErr := possible.BalanceEffects()
			if effectErr != nil {
				return nil, effectErr
			}
			for _, effect := range balanceEffects {
				if effect.AccountID != accountID {
					continue
				}
				for index, amount := range []reporting.Amount{effect.Owned, effect.Available, effect.Locked, effect.Debt} {
					value, known := amount.Value()
					if !known || value.Sign() != 0 {
						affected[index] = true
					}
				}
			}
		}
	}
	if !affected[0] && !affected[1] && !affected[2] && !affected[3] {
		return nil, nil
	}
	zero, err := money.NewMoney("0", asset)
	if err != nil {
		return nil, err
	}
	knownZero, err := reporting.KnownAmount(zero)
	if err != nil {
		return nil, err
	}
	unknown, err := reporting.MissingAmount(reporting.Unknown, "transaction_state_at_source_unknown")
	if err != nil {
		return nil, err
	}
	changes := account.Amounts{Owned: knownZero, Available: knownZero, Locked: knownZero, Debt: knownZero}
	fields := []*reporting.Amount{&changes.Owned, &changes.Available, &changes.Locked, &changes.Debt}
	for index := range affected {
		if affected[index] {
			*fields[index] = unknown
		}
	}
	return &account.Effect{OperationID: current.OperationID, Revision: current.Revision, At: asOf, Changes: &changes}, nil
}
