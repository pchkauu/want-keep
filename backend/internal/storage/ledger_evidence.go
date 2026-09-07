package storage

import (
	"context"
	"encoding/json"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) RevisionEvidence(ctx context.Context, p household.Principal, id string, revision uint64) ([]ledger.Evidence, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT 'source',source_id,max(source_revision) FROM want_keep.ledger_revision_sources WHERE household_id=$1 AND operation_id=$2 AND revision=$3 GROUP BY source_id
 UNION SELECT 'attachment',attachment_id,1 FROM want_keep.transaction_details WHERE household_id=$1 AND operation_id=$2 AND revision=$3 AND attachment_id IS NOT NULL
 UNION SELECT 'review',operation_id,revision FROM want_keep.ledger_review_results WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY 1,2,3`, p.HouseholdID(), id, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ledger.Evidence{}
	for rows.Next() {
		var e ledger.Evidence
		if err = rows.Scan(&e.Kind, &e.ID, &e.Revision); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) TransactionSourceFacts(ctx context.Context, p household.Principal, id string, revision uint64) ([]application.SourceFact, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT DISTINCT ON(f.source_id) f.source_id,f.source_revision,f.fact,f.conflict FROM want_keep.ledger_source_facts f JOIN want_keep.ledger_revision_sources r USING(household_id,source_id,source_revision) WHERE r.household_id=$1 AND r.operation_id=$2 AND r.revision=$3 ORDER BY f.source_id,f.source_revision DESC`, p.HouseholdID(), id, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []application.SourceFact{}
	for rows.Next() {
		var v application.SourceFact
		var data []byte
		if err = rows.Scan(&v.SourceID, &v.Revision, &data, &v.Conflict); err != nil {
			return nil, err
		}
		var stored storedSourceFact
		if err = json.Unmarshal(data, &stored); err != nil {
			return nil, ErrStorage
		}
		v.Fact, err = stored.domain()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
