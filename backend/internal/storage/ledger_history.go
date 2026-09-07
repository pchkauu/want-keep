package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) HistoryRevisions(ctx context.Context, p household.Principal, id string, before uint64, limit int) ([]ledger.Revision, uint64, error) {
	if limit < 1 || limit > 100 || before > 9007199254740991 {
		return nil, 0, ledger.ErrInvalidRevision
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	rows, err := q.Query(ctx, `SELECT revision FROM want_keep.operation_revisions WHERE household_id=$1 AND operation_id=$2 AND ($3::bigint=0 OR revision<$3) ORDER BY revision DESC LIMIT $4`, p.HouseholdID(), id, before, limit+1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	refs := []uint64{}
	for rows.Next() {
		var v uint64
		if err = rows.Scan(&v); err != nil {
			return nil, 0, err
		}
		refs = append(refs, v)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	rows.Close()
	var next uint64
	if len(refs) > limit {
		refs = refs[:limit]
		next = refs[len(refs)-1]
	}
	out := []ledger.Revision{}
	for _, v := range refs {
		r, err := s.LedgerRevision(ctx, p, id, v)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, next, nil
}
